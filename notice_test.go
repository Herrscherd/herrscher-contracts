package contracts

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

func TestCodesAreExactlyTheNamedOnes(t *testing.T) {
	listed := Codes()
	for raw := 1; raw <= math.MaxUint16; raw++ {
		code := Code(raw)
		named := !strings.HasPrefix(code.Key(), "code.")
		_, declared := code.params()
		if inList := slices.Contains(listed, code); named != inList || named != declared {
			t.Fatalf("code %d: named %v, declared %v, listed %v; Key, params and Codes must name the same constants", raw, named, declared, inList)
		}
	}
}

func TestArgKindMatchesName(t *testing.T) {
	cases := []struct {
		name  string
		arg   Arg
		valid bool
	}{
		{"count with the count kind", Count(1), true},
		{"text named Count", Text(countName, "x"), false},
		{"ref named Count", Ref(countName, "1"), false},
		{"count kind under another name", Arg{name: "Name", kind: ArgCount, value: 1}, false},
		{"text under another name", Text("Name", "x"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.arg.checkKind()
			if (err == nil) != tc.valid {
				t.Fatalf("checkKind() = %v, want valid %v", err, tc.valid)
			}
			if err != nil && !errors.Is(err, ErrNoticeArgs) {
				t.Fatalf("checkKind() = %v, want ErrNoticeArgs", err)
			}
		})
	}
}

func TestRenderRefusesCodeWithoutEnglishMessage(t *testing.T) {
	cases := []struct {
		name    string
		sources fstest.MapFS
		refused bool
	}{
		{"defined in no language", fstest.MapFS{
			"en.json": {Data: []byte(`[{"id":"session.exists","other":"a session already exists"}]`)},
			"fr.json": {Data: []byte(`[{"id":"session.exists","other":"une session existe déjà"}]`)},
		}, true},
		{"defined only in French", fstest.MapFS{
			"en.json": {Data: []byte(`[{"id":"session.exists","other":"a session already exists"}]`)},
			"fr.json": {Data: []byte(`[{"id":"session.not_found","other":"session « {{.Name}} » introuvable"}]`)},
		}, true},
		{"defined only in English", fstest.MapFS{
			"en.json": {Data: []byte(`[{"id":"session.not_found","other":"session {{.Name}} not found"}]`)},
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			renderer := &Renderer{catalog: func() (*i18n.Catalog, error) { return i18n.NewCatalog(nil, tc.sources) }}
			text, err := renderer.Render(i18n.French, Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "x")}})
			if tc.refused {
				if !errors.Is(err, ErrMissingMessage) || text != "" {
					t.Fatalf("Render = %q, %v; want ErrMissingMessage and no text", text, err)
				}
				return
			}
			if err != nil || text != "session x not found" {
				t.Fatalf("Render = %q, %v; want the English text through the fallback", text, err)
			}
		})
	}
}

func TestRenderRefusesArgKindMismatchAtRender(t *testing.T) {
	notice := Notice{Code: CodeSessionNotFound, Args: []Arg{{name: "Name", kind: ArgCount, value: 3}}}
	text, err := NewRenderer().Render(i18n.English, notice)
	if !errors.Is(err, ErrNoticeArgs) || text != "" {
		t.Fatalf("Render of a count-kind argument under Name = %q, %v; want ErrNoticeArgs and no text", text, err)
	}
}

func argsWithRefUnder(declared map[string]ArgKind, ref string) []Arg {
	args := make([]Arg, 0, len(declared))
	for name := range declared {
		if name == ref {
			args = append(args, Ref(name, "123"))
		} else {
			args = append(args, argOfKind(name, declared[name]))
		}
	}
	return args
}

func argOfKind(name string, kind ArgKind) Arg {
	switch kind {
	case ArgCount:
		return Count(2)
	case ArgRef:
		return Ref(name, "123")
	case ArgText:
		return Text(name, "x")
	}
	return Arg{}
}

func TestRenderRefusesRefForEveryDeclaredTextParam(t *testing.T) {
	renderer := NewRenderer()
	for _, code := range Codes() {
		declared, _ := code.params()
		for name, kind := range declared {
			if kind != ArgText {
				continue
			}
			text, err := renderer.Render(i18n.English, Notice{Code: code, Args: argsWithRefUnder(declared, name)})
			if !errors.Is(err, ErrNoticeArgs) || text != "" {
				t.Fatalf("Render of code %q with a ref under %s = %q, %v; want ErrNoticeArgs and no text", code.Key(), name, text, err)
			}
		}
	}
}

func TestRenderAcceptsTextForEveryDeclaredParam(t *testing.T) {
	renderer := NewRenderer()
	for _, code := range Codes() {
		declared, _ := code.params()
		var args []Arg
		for name, kind := range declared {
			args = append(args, argOfKind(name, kind))
		}
		for _, lang := range i18n.Languages() {
			if _, err := renderer.Render(lang, Notice{Code: code, Args: args}); err != nil {
				t.Fatalf("Render of code %q in %v with text arguments: %v", code.Key(), lang, err)
			}
		}
	}
}

func TestRenderSettingsTextIsShownAsItIs(t *testing.T) {
	renderer := NewRenderer()
	notice := Notice{Code: CodeSettingMissing, Args: []Arg{Text("Settings", "a, b")}}
	cases := []struct {
		lang i18n.Language
		want string
	}{
		{i18n.English, "these settings must be set: a, b"},
		{i18n.French, "ces réglages doivent être définis : a, b"},
	}
	for _, tc := range cases {
		if got, err := renderer.Render(tc.lang, notice); err != nil || got != tc.want {
			t.Fatalf("Render in %v = %q, %v; want %q", tc.lang, got, err, tc.want)
		}
	}
}

func TestEveryCodeConstantIsListed(t *testing.T) {
	var declared, listed []string
	for _, file := range parsePackageFiles(t) {
		declared = append(declared, codeConstantNames(file)...)
		listed = append(listed, codesListNames(file)...)
	}
	if len(declared) == 0 {
		t.Fatal("no constant of type Code declared in the package")
	}
	for _, name := range declared {
		if !slices.Contains(listed, name) {
			t.Errorf("constant %s has type Code but is missing from Codes()", name)
		}
	}
}

func parsePackageFiles(t *testing.T) []*ast.File {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	return files
}

func codeConstantNames(file *ast.File) []string {
	var names []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if ok && gen.Tok == token.CONST {
			names = append(names, codeNamesInBlock(gen)...)
		}
	}
	return names
}

func codeNamesInBlock(gen *ast.GenDecl) []string {
	var names []string
	typeName := ""
	for _, spec := range gen.Specs {
		value, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if value.Type != nil {
			typeName = typeNameOf(value.Type)
		}
		if typeName == "Code" {
			for _, name := range value.Names {
				names = append(names, name.Name)
			}
		}
	}
	return names
}

func typeNameOf(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func codesListNames(file *ast.File) []string {
	var names []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Codes" || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			lit, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, elt := range lit.Elts {
				if ident, ok := elt.(*ast.Ident); ok {
					names = append(names, ident.Name)
				}
			}
			return true
		})
	}
	return names
}

func TestCodeKeysAreDistinctAndNamed(t *testing.T) {
	seen := make(map[string]Code)
	for _, code := range Codes() {
		key := code.Key()
		if strings.HasPrefix(key, "code.") {
			t.Fatalf("code %d has no named key: %q", uint16(code), key)
		}
		if other, dup := seen[key]; dup {
			t.Fatalf("codes %d and %d share the key %q", uint16(other), uint16(code), key)
		}
		seen[key] = code
	}
}

func TestOutOfSetCodeKeyPrintsItsNumber(t *testing.T) {
	if got := Code(999).Key(); got != "code.999" {
		t.Fatalf("Key of an out-of-set code = %q, want code.999", got)
	}
	if got := Code(0).Key(); got != "code.0" {
		t.Fatalf("Key of the zero code = %q, want code.0", got)
	}
}

func TestArgConstructorsKeepNameKindAndValue(t *testing.T) {
	cases := []struct {
		name  string
		arg   Arg
		want  string
		kind  ArgKind
		value any
	}{
		{"text", Text("Name", "x"), "Name", ArgText, "x"},
		{"count", Count(3), "Count", ArgCount, 3},
		{"ref", Ref("Channel", "123"), "Channel", ArgRef, "123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.arg.Name() != tc.want || tc.arg.Kind() != tc.kind || tc.arg.Value() != tc.value {
				t.Fatalf("arg = %q %d %v, want %q %d %v", tc.arg.Name(), tc.arg.Kind(), tc.arg.Value(), tc.want, tc.kind, tc.value)
			}
		})
	}
}

func TestRenderNoticeInEachLanguage(t *testing.T) {
	cases := []struct {
		notice Notice
		en     string
		fr     string
	}{
		{
			Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "x")}},
			"no session named x",
			"session « x » introuvable",
		},
		{
			Notice{Code: CodeSessionExists, Args: []Arg{Text("Name", "x")}},
			"a session named x already exists",
			"une session nommée « x » existe déjà",
		},
		{
			Notice{Code: CodeUnknownAgent, Args: []Arg{Text("Name", "x")}},
			"unknown agent: x",
			"agent inconnu : x",
		},
		{
			Notice{Code: CodeSessionCreated, Args: []Arg{Text("Name", "x")}},
			"session x created",
			"session « x » créée",
		},
		{
			Notice{Code: CodeSessionClosed, Args: []Arg{Text("Name", "x")}},
			"session x closed",
			"session « x » fermée",
		},
		{
			Notice{Code: CodeUnexpected, Args: []Arg{Text("Detail", "boom")}},
			"unexpected error: boom",
			"erreur inattendue : boom",
		},
		{
			Notice{Code: CodeNodeMerged},
			"this node is folded into an umbrella; restore it with force to detach it",
			"ce nœud est replié dans un regroupement ; pour le détacher, restaure-le en forçant l'opération",
		},
		{
			Notice{Code: CodeCompactNoThread},
			"context compaction unavailable: no turn has run yet, so there is nothing to compact",
			"compaction du contexte indisponible : aucun tour n'a encore été joué, il n'y a rien à compacter",
		},
		{
			Notice{Code: CodeCompactTimeoutInvalid},
			"context compaction unavailable: the compaction timeout must be a positive duration",
			"compaction du contexte indisponible : le délai de compaction doit être une durée positive",
		},
		{
			Notice{Code: CodeSessionLimit, Args: []Arg{Count(1)}},
			"session limit reached: 1 session allowed; close one before creating another",
			"limite de sessions atteinte : 1 session autorisée ; ferme-en une avant d'en créer une autre",
		},
		{
			Notice{Code: CodeSessionLimit, Args: []Arg{Count(0)}},
			"session limit reached: 0 sessions allowed; close one before creating another",
			"limite de sessions atteinte : 0 session autorisée ; ferme-en une avant d'en créer une autre",
		},
		{
			Notice{Code: CodeSessionLimit, Args: []Arg{Count(4)}},
			"session limit reached: 4 sessions allowed; close one before creating another",
			"limite de sessions atteinte : 4 sessions autorisées ; ferme-en une avant d'en créer une autre",
		},
		{
			Notice{Code: CodeNoHome},
			"no home set; run `set home` first",
			"aucun accueil défini ; lance d'abord `set home`",
		},
		{
			Notice{Code: CodeInvalidValue, Args: []Arg{Text("Field", "terminal_only"), Text("Value", "maybe")}},
			"invalid value maybe for terminal_only",
			"valeur invalide maybe pour terminal_only",
		},
		{
			Notice{Code: CodeSharedAgent, Args: []Arg{Text("Agent", "x")}},
			"session create with agent x cannot be shared: an agent needs a place of its own, and shared:true runs in the main checkout",
			"la session ne peut pas être créée avec l'agent « x » en partage : un agent a besoin d'un espace à lui, et shared:true la lance dans le dépôt principal",
		},
		{
			Notice{Code: CodeEntityNotFound, Args: []Arg{Text("Kind", "lead"), Text("Name", "x")}},
			"no lead named x",
			"lead « x » introuvable",
		},
		{
			Notice{Code: CodeEntityUnknown, Args: []Arg{Text("Kind", "backend"), Text("Name", "x")}},
			"unknown backend x",
			"valeur inconnue pour backend : « x »",
		},
		{
			Notice{Code: CodeScheduleNotFound, Args: []Arg{Text("Name", "x")}},
			"no schedule named x",
			"planification « x » introuvable",
		},
		{
			Notice{Code: CodeChannelKindUnder, Args: []Arg{Ref("Channel", "123"), Text("Kind", "voice")}},
			"channel 123 is voice: under must be a category or forum (to run in an existing conversation, pass channel_id)",
			"le salon 123 est de type voice : « under » doit être une catégorie ou un forum (pour lancer une session dans une conversation existante, passe channel_id)",
		},
		{
			Notice{Code: CodeChannelKindHome, Args: []Arg{Ref("Channel", "123"), Text("Kind", "voice")}},
			"channel 123 is voice: home must be a category or forum",
			"le salon 123 est de type voice : l'accueil doit être une catégorie ou un forum",
		},
		{
			Notice{Code: CodeInvalidValueHint, Args: []Arg{Text("Field", "terminal_only"), Text("Value", "maybe"), Text("Hint", "use true or false")}},
			"invalid value maybe for terminal_only: use true or false",
			"valeur invalide maybe pour terminal_only : use true or false",
		},
		{
			Notice{Code: CodeAgentNotFound, Args: []Arg{Text("Name", "x")}},
			"no agent x",
			"agent « x » introuvable",
		},
		{
			Notice{Code: CodeSessionCloseDirty, Args: []Arg{Text("Name", "x")}},
			"session x has uncommitted changes: commit them, or close it with force:true to discard them",
			"la session « x » a des changements non validés : valide-les, ou ferme-la avec force:true pour les abandonner",
		},
		{
			Notice{Code: CodeCompactInstructionRefused},
			"context compaction unavailable: this backend cannot take an instruction for a compaction",
			"compaction du contexte indisponible : ce fournisseur ne peut pas recevoir d'instruction de compaction",
		},
	}
	renderer := NewRenderer()
	for _, tc := range cases {
		if got, err := renderer.Render(i18n.English, tc.notice); err != nil || got != tc.en {
			t.Fatalf("en %q = %q, %v; want %q", tc.notice.Code.Key(), got, err, tc.en)
		}
		if got, err := renderer.Render(i18n.French, tc.notice); err != nil || got != tc.fr {
			t.Fatalf("fr %q = %q, %v; want %q", tc.notice.Code.Key(), got, err, tc.fr)
		}
	}
}

func TestRenderRefusesMalformedNotices(t *testing.T) {
	cases := []struct {
		name   string
		notice Notice
		want   error
	}{
		{"zero code", Notice{Args: []Arg{Text("Name", "x")}}, ErrUnknownCode},
		{"out-of-set code", Notice{Code: Code(999), Args: []Arg{Text("Name", "x")}}, ErrUnknownCode},
		{"zero arg", Notice{Code: CodeSessionNotFound, Args: []Arg{{}}}, ErrNoticeArgs},
		{"empty name", Notice{Code: CodeSessionNotFound, Args: []Arg{Text("", "x")}}, ErrNoticeArgs},
		{"named arg of no kind", Notice{Code: CodeSessionNotFound, Args: []Arg{{name: "Name"}}}, ErrNoticeArgs},
		{"name given twice", Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "a"), Text("Name", "b")}}, ErrNoticeArgs},
		{"missing arg", Notice{Code: CodeSessionNotFound}, ErrNoticeArgs},
		{"wrong name", Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Nom", "x")}}, ErrNoticeArgs},
		{"extra arg", Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "x"), Text("Other", "y")}}, ErrNoticeArgs},
		{"count on a code without one", Notice{Code: CodeSessionNotFound, Args: []Arg{Count(2)}}, ErrNoticeArgs},
	}
	renderer := NewRenderer()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, err := renderer.Render(i18n.English, tc.notice)
			if !errors.Is(err, tc.want) || text != "" {
				t.Fatalf("Render = %q, %v; want an error matching %v", text, err, tc.want)
			}
		})
	}
}

type noticedError struct {
	notice Notice
}

func (w noticedError) Error() string  { return "session gone" }
func (w noticedError) Notice() Notice { return w.notice }

func TestNoticeOfUsesTheNoticeErrorInTheChain(t *testing.T) {
	want := Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "x")}}
	err := fmt.Errorf("close: %w", noticedError{notice: want})
	got := NoticeOf(err)
	if got.Code != want.Code || !equalArgs(got.Args, want.Args) {
		t.Fatalf("NoticeOf = %+v, want %+v", got, want)
	}
}

func TestNoticeOfFallsBackToUnexpected(t *testing.T) {
	got := NoticeOf(errors.New("disk full"))
	if got.Code != CodeUnexpected || !equalArgs(got.Args, []Arg{Text("Detail", "disk full")}) {
		t.Fatalf("NoticeOf plain error = %+v", got)
	}
}

func equalArgs(a, b []Arg) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestResultCarriesNoticeOrData(t *testing.T) {
	notice := Notice{Code: CodeSessionExists, Args: []Arg{Text("Name", "x")}}
	if got, ok := NoticeResult(notice).Notice(); !ok || got.Code != CodeSessionExists {
		t.Fatalf("notice result = %+v, %v", got, ok)
	}
	if NoticeResult(notice).Data() != nil {
		t.Fatal("a notice result carries no data")
	}
	if _, ok := DataResult(3).Notice(); ok {
		t.Fatal("a data result carries no notice")
	}
	if DataResult(3).Data() != 3 {
		t.Fatal("data result must return its data")
	}
	if _, ok := (Result{}).Notice(); ok || (Result{}).Data() != nil {
		t.Fatal("the zero result carries neither")
	}
	if DataResult(nil) != (Result{}) {
		t.Fatal("DataResult(nil) must equal the zero result")
	}
}

func TestCodeValuesAreFrozen(t *testing.T) {
	frozen := map[Code]uint16{
		CodeSessionNotFound:           1,
		CodeSessionExists:             2,
		CodeUnknownAgent:              3,
		CodeSessionCreated:            4,
		CodeSessionClosed:             5,
		CodeUnexpected:                6,
		CodeSettingInvalid:            7,
		CodeSettingMissing:            8,
		CodeNodeMerged:                9,
		CodeCompactUnavailable:        10,
		CodeCompacted:                 11,
		CodeAddressPlaintextRefused:   12,
		CodeCompactTimeoutInvalid:     13,
		CodeCompactInstructionRefused: 14,
		CodeCompactNoThread:           15,
		CodeSessionLimit:              16,
		CodeNoHome:                    17,
		CodeNoHostRegistered:          18,
		CodeServiceControlUnavailable: 19,
		CodeModelRequired:             20,
		CodeExplicitCmdRefused:        21,
		CodeInvalidValue:              22,
		CodeSharedAgent:               23,
		CodeUnknownApprovalMode:       24,
		CodeAgentExists:               25,
		CodeAgentInvalidName:          26,
		CodeAgentNotFound:             27,
		CodeUnsupportedHomeType:       28,
		CodeEntityNotFound:            29,
		CodeEntityUnknown:             30,
		CodeScheduleNotFound:          31,
		CodeChannelKindUnder:          32,
		CodeChannelKindHome:           33,
		CodeInvalidValueHint:          34,
		CodeSessionCloseDirty:         35,
		CodeHealthConnected:           36,
		CodeHealthSignedInAs:          37,
		CodeHealthNotLoggedIn:         38,
		CodeHealthRateLimited:         39,
		CodeHealthCredentialsRefused:  40,
		CodeHealthUnreachable:         41,
		CodeHealthEnvMissing:          42,
		CodeHealthRemedyRun:           43,
		CodeHealthRemedyEnvRestart:    44,
	}
	if len(frozen) != len(Codes()) {
		t.Fatalf("%d codes frozen, %d listed: every code needs its frozen value", len(frozen), len(Codes()))
	}
	for code, value := range frozen {
		if uint16(code) != value {
			t.Errorf("%s = %d, frozen at %d: a code value is part of the wire and must not move", code.Key(), uint16(code), value)
		}
	}
}
