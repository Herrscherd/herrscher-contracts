package contracts

import (
	"errors"
	"fmt"
	"math"
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

func TestRenderRefusesCodeWithoutMessage(t *testing.T) {
	sources := fstest.MapFS{
		"en.json": {Data: []byte(`[{"id":"session.exists","other":"a session named {{.Name}} already exists"}]`)},
		"fr.json": {Data: []byte(`[{"id":"session.exists","other":"une session nommée « {{.Name}} » existe déjà"}]`)},
	}
	renderer := &Renderer{catalog: func() (*i18n.Catalog, error) { return i18n.NewCatalog(nil, sources) }}
	text, err := renderer.Render(i18n.English, Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "x")}})
	if !errors.Is(err, ErrMissingMessage) || text != "" {
		t.Fatalf("Render of a code without message = %q, %v; want ErrMissingMessage and no text", text, err)
	}
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
			"aucune session nommée « x »",
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
