package contracts

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

type compactwireFrame struct {
	Code uint16           `json:"code"`
	Args []compactwireArg `json:"args,omitempty"`
}

type compactwireArg struct {
	Name  string          `json:"name"`
	Kind  uint8           `json:"kind"`
	Value json.RawMessage `json:"value"`
}

type notice030 struct {
	Code Code     `json:"Code"`
	Args []arg030 `json:"Args"`
}

type arg030 struct {
	name  string
	kind  ArgKind
	value any
}

func sameNotice(got, want Notice) bool {
	if got.Code != want.Code || len(got.Args) != len(want.Args) {
		return false
	}
	for i := range want.Args {
		if got.Args[i].Name() != want.Args[i].Name() || got.Args[i].Kind() != want.Args[i].Kind() || got.Args[i].Value() != want.Args[i].Value() {
			return false
		}
	}
	return true
}

func compactwireEncode(t *testing.T, notice Notice) string {
	t.Helper()
	frame := compactwireFrame{Code: uint16(notice.Code), Args: make([]compactwireArg, 0, len(notice.Args))}
	for _, arg := range notice.Args {
		value, err := json.Marshal(arg.Value())
		if err != nil {
			t.Fatalf("marshal value: %v", err)
		}
		frame.Args = append(frame.Args, compactwireArg{Name: arg.Name(), Kind: uint8(arg.Kind()), Value: value})
	}
	encoded, err := json.Marshal(frame)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}
	return string(encoded)
}

func oversizedNotice(args int) string {
	elements := make([]string, args)
	for i := range elements {
		elements[i] = `{}`
	}
	return `{"code":1,"args":[` + strings.Join(elements, ",") + `]}`
}

func TestArgJSONRoundTripsEveryKind(t *testing.T) {
	tests := []struct {
		name string
		arg  Arg
		want string
	}{
		{"text", Text("Name", "lead"), `{"name":"Name","kind":1,"value":"lead"}`},
		{"count", Count(3), `{"name":"Count","kind":2,"value":3}`},
		{"ref", Ref("Channel", "chan-9"), `{"name":"Channel","kind":3,"value":"chan-9"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.arg)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(raw) != tt.want {
				t.Fatalf("wire JSON:\n got %s\nwant %s", raw, tt.want)
			}
			var got Arg
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Name() != tt.arg.Name() || got.Kind() != tt.arg.Kind() || got.Value() != tt.arg.Value() {
				t.Fatalf("decoded %+v, want %+v", got, tt.arg)
			}
		})
	}
}

func TestZeroArgJSONStaysMalformed(t *testing.T) {
	raw, err := json.Marshal(Arg{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	const want = `{"name":"","kind":0,"value":null}`
	if string(raw) != want {
		t.Fatalf("zero argument wire JSON:\n got %s\nwant %s", raw, want)
	}
	var got Arg
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	_, err = NewRenderer().Render(i18n.English, Notice{Code: CodeSessionNotFound, Args: []Arg{got}})
	if !errors.Is(err, ErrNoticeArgs) {
		t.Fatalf("rendering a decoded zero argument: err = %v, want ErrNoticeArgs", err)
	}
}

func TestNoticeJSONLiteralsAreFrozen(t *testing.T) {
	tests := []struct {
		name   string
		notice Notice
		want   string
	}{
		{"zero notice", Notice{}, `{"code":0}`},
		{"no argument", Notice{Code: CodeTurnInterrupted}, `{"code":47}`},
		{"text", Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "lead")}}, `{"code":1,"args":[{"name":"Name","kind":1,"value":"lead"}]}`},
		{"count", Notice{Code: CodeSessionLimit, Args: []Arg{Count(3)}}, `{"code":16,"args":[{"name":"Count","kind":2,"value":3}]}`},
		{"ref and text", Notice{Code: CodeChannelKindUnder, Args: []Arg{Ref("Channel", "chan-9"), Text("Kind", "thread")}}, `{"code":32,"args":[{"name":"Channel","kind":3,"value":"chan-9"},{"name":"Kind","kind":1,"value":"thread"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.notice)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(raw) != tt.want {
				t.Fatalf("wire JSON:\n got %s\nwant %s", raw, tt.want)
			}
			var got Notice
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !sameNotice(got, tt.notice) {
				t.Fatalf("decoded %+v, want %+v", got, tt.notice)
			}
		})
	}
}

func TestNoticeJSONRoundTripRendersTheSameText(t *testing.T) {
	renderer := NewRenderer()
	notices := []Notice{
		{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "lead")}},
		{Code: CodeSessionLimit, Args: []Arg{Count(3)}},
		{Code: CodeChannelKindUnder, Args: []Arg{Ref("Channel", "chan-9"), Text("Kind", "thread")}},
	}
	for _, notice := range notices {
		want, err := renderer.Render(i18n.English, notice)
		if err != nil {
			t.Fatalf("render original %+v: %v", notice, err)
		}
		raw, err := json.Marshal(notice)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var decoded Notice
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		got, err := renderer.Render(i18n.English, decoded)
		if err != nil {
			t.Fatalf("render decoded %s: %v", raw, err)
		}
		if got != want {
			t.Fatalf("decoded notice renders %q, want %q", got, want)
		}
	}
}

func TestNoticeJSONReadsTheV030Form(t *testing.T) {
	var got Notice
	if err := json.Unmarshal([]byte(`{"Code":47,"Args":null}`), &got); err != nil {
		t.Fatalf("unmarshal v0.30 notice without arguments: %v", err)
	}
	if got.Code != CodeTurnInterrupted || got.Args != nil {
		t.Fatalf("v0.30 notice without arguments decoded to %+v", got)
	}

	var withArg Notice
	if err := json.Unmarshal([]byte(`{"Code":1,"Args":[{}]}`), &withArg); err != nil {
		t.Fatalf("unmarshal v0.30 notice with an empty argument: %v", err)
	}
	if withArg.Code != CodeSessionNotFound || len(withArg.Args) != 1 {
		t.Fatalf("v0.30 notice with an empty argument decoded to %+v", withArg)
	}
	_, err := NewRenderer().Render(i18n.English, withArg)
	if !errors.Is(err, ErrNoticeArgs) {
		t.Fatalf("rendering the v0.30 empty argument: err = %v, want ErrNoticeArgs", err)
	}
}

func TestNoticeJSONReadsCompactwireKeys(t *testing.T) {
	const raw = `{"code":1,"args":[{"name":"Name","kind":1,"value":"lead"}]}`
	var got Notice
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "lead")}}
	if !sameNotice(got, want) {
		t.Fatalf("decoded %+v, want %+v", got, want)
	}
}

func TestNoticeJSONIgnoresUnknownFields(t *testing.T) {
	const raw = `{"code":1,"future":true,"args":[{"name":"Name","kind":1,"value":"lead","hint":"x"}]}`
	var got Notice
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("unmarshal with unknown fields: %v", err)
	}
	if !sameNotice(got, Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "lead")}}) {
		t.Fatalf("decoded %+v", got)
	}
}

func TestNoticeJSONKeepsAnUnknownCode(t *testing.T) {
	var got Notice
	if err := json.Unmarshal([]byte(`{"code":999,"args":[{"name":"Name","kind":1,"value":"lead"}]}`), &got); err != nil {
		t.Fatalf("unmarshal unknown code: %v", err)
	}
	if got.Code != 999 {
		t.Fatalf("Code = %d, want 999", got.Code)
	}
	_, err := NewRenderer().Render(i18n.English, got)
	if !errors.Is(err, ErrUnknownCode) {
		t.Fatalf("rendering the unknown code: err = %v, want ErrUnknownCode", err)
	}
}

func TestNoticeJSONKeepsAnUnknownArgKind(t *testing.T) {
	var got Notice
	if err := json.Unmarshal([]byte(`{"code":1,"args":[{"name":"Name","kind":9,"value":"lead"}]}`), &got); err != nil {
		t.Fatalf("unmarshal unknown argument kind: %v", err)
	}
	_, err := NewRenderer().Render(i18n.English, got)
	if !errors.Is(err, ErrNoticeArgs) {
		t.Fatalf("rendering the unknown argument kind: err = %v, want ErrNoticeArgs", err)
	}
}

func TestNoticeJSONRefusesAValueOfTheWrongType(t *testing.T) {
	tests := []string{
		`{"code":1,"args":[{"name":"Name","kind":1,"value":3}]}`,
		`{"code":16,"args":[{"name":"Count","kind":2,"value":"3"}]}`,
		`{"code":3,"args":[{"name":"Name","kind":3,"value":true}]}`,
	}
	for _, raw := range tests {
		var got Notice
		err := json.Unmarshal([]byte(raw), &got)
		if !errors.Is(err, ErrNoticeArgs) {
			t.Fatalf("unmarshal %s: err = %v, want ErrNoticeArgs", raw, err)
		}
	}
}

func TestNoticeJSONLimitsTheArgumentCount(t *testing.T) {
	full := make([]Arg, maxNoticeArgs)
	for i := range full {
		full[i] = Text("Name", "lead")
	}
	raw, err := json.Marshal(Notice{Code: CodeSessionNotFound, Args: full})
	if err != nil {
		t.Fatalf("marshal %d arguments: %v", maxNoticeArgs, err)
	}
	var got Notice
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal %d arguments: %v", maxNoticeArgs, err)
	}

	tooMany := make([]Arg, maxNoticeArgs+1)
	for i := range tooMany {
		tooMany[i] = Text("Name", "lead")
	}
	if _, err := json.Marshal(Notice{Code: CodeSessionNotFound, Args: tooMany}); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("marshal %d arguments: err = %v, want ErrNoticeLimit", len(tooMany), err)
	}
	if err := json.Unmarshal([]byte(oversizedNotice(maxNoticeArgs+1)), &got); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("unmarshal %d arguments: err = %v, want ErrNoticeLimit", maxNoticeArgs+1, err)
	}
}

func TestNoticeJSONLimitsTheArgumentSize(t *testing.T) {
	fits := Text("Detail", strings.Repeat("x", maxNoticeArgBytes-2))
	raw, err := json.Marshal(Notice{Code: CodeUnexpected, Args: []Arg{fits}})
	if err != nil {
		t.Fatalf("marshal an argument of the limit size: %v", err)
	}
	var got Notice
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal an argument of the limit size: %v", err)
	}

	tooLarge := Text("Detail", strings.Repeat("x", maxNoticeArgBytes-1))
	if _, err := json.Marshal(tooLarge); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("marshal an oversized argument: err = %v, want ErrNoticeLimit", err)
	}
	oversized := `{"code":6,"args":[{"name":"Detail","kind":1,"value":"` + strings.Repeat("x", maxNoticeArgBytes) + `"}]}`
	if err := json.Unmarshal([]byte(oversized), &got); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("unmarshal an oversized argument: err = %v, want ErrNoticeLimit", err)
	}
}

func TestNoticeJSONMatchesCompactwire(t *testing.T) {
	notices := []Notice{
		{Code: CodeTurnInterrupted},
		{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "lead")}},
		{Code: CodeSessionLimit, Args: []Arg{Count(3)}},
		{Code: CodeChannelKindUnder, Args: []Arg{Ref("Channel", "chan-9"), Text("Kind", "thread")}},
		{Code: CodeAccountFallback, Args: []Arg{Text("From", `a"b`), Text("To", "é")}},
	}
	for _, notice := range notices {
		want := compactwireEncode(t, notice)
		got, err := json.Marshal(notice)
		if err != nil {
			t.Fatalf("marshal %+v: %v", notice, err)
		}
		if string(got) != want {
			t.Fatalf("contracts encodes %s, compactwire encodes %s", got, want)
		}
		var decoded Notice
		if err := json.Unmarshal([]byte(want), &decoded); err != nil {
			t.Fatalf("contracts decodes compactwire %s: %v", want, err)
		}
		if !sameNotice(decoded, notice) {
			t.Fatalf("contracts decodes compactwire %s to %+v, want %+v", want, decoded, notice)
		}
	}
}

func TestNoticeJSONIsReadByTheV030Decoder(t *testing.T) {
	notices := []Notice{
		{Code: CodeTurnInterrupted},
		{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "lead")}},
		{Code: CodeSessionLimit, Args: []Arg{Count(3)}},
		{Code: CodeChannelKindUnder, Args: []Arg{Ref("Channel", "chan-9"), Text("Kind", "thread")}},
	}
	for _, notice := range notices {
		raw, err := json.Marshal(notice)
		if err != nil {
			t.Fatalf("marshal %+v: %v", notice, err)
		}
		var old notice030
		if err := json.Unmarshal(raw, &old); err != nil {
			t.Fatalf("v0.30 decoder reads %s: %v", raw, err)
		}
		if old.Code != notice.Code || len(old.Args) != len(notice.Args) {
			t.Fatalf("v0.30 decoder reads %s as code %d with %d arguments, want code %d with %d", raw, old.Code, len(old.Args), notice.Code, len(notice.Args))
		}
	}
}
