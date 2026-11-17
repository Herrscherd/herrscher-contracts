package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func assertWireRoundTrip(t *testing.T, raw, want string) {
	t.Helper()
	var got Notice
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	first, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(first) != want {
		t.Fatalf("encoded:\n got %s\nwant %s", first, want)
	}
	var again Notice
	if err = json.Unmarshal(first, &again); err != nil {
		t.Fatalf("unmarshal own output: %v", err)
	}
	second, err := json.Marshal(again)
	if err != nil {
		t.Fatalf("marshal again: %v", err)
	}
	if !bytes.Equal(second, first) {
		t.Fatalf("second encoding:\n got %s\nwant %s", second, first)
	}
}

func TestNoticeJSONRoundTripsWithinTheWireSize(t *testing.T) {
	escapedAngles := strings.ReplaceAll(strings.Repeat("<", 9000), "<", "\\u003c")
	escapedAmpersands := strings.ReplaceAll(strings.Repeat("&", 9000), "&", "\\u0026")
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "text of angle brackets within the limit once escaped",
			raw:  `{"code":1,"args":[{"name":"Detail","kind":1,"value":"` + strings.Repeat("<", 9000) + `"}]}`,
			want: `{"code":1,"args":[{"name":"Detail","kind":1,"value":"` + escapedAngles + `"}]}`,
		},
		{
			name: "unknown kind of ampersands within the limit once escaped",
			raw:  `{"code":1,"args":[{"name":"Detail","kind":9,"value":"` + strings.Repeat("&", 9000) + `"}]}`,
			want: `{"code":1,"args":[{"name":"Detail","kind":9,"value":"` + escapedAmpersands + `"}]}`,
		},
		{
			name: "unknown kind with internal spaces",
			raw:  `{"code":1,"args":[{"name":"Detail","kind":9,"value":{"a": 1,  "b": [2, 3]}}]}`,
			want: `{"code":1,"args":[{"name":"Detail","kind":9,"value":{"a":1,"b":[2,3]}}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertWireRoundTrip(t, tt.raw, tt.want)
		})
	}
}

func TestNoticeJSONRefusesAtReadWhatTheEncoderRefuses(t *testing.T) {
	tooLargeText := `{"code":1,"args":[{"name":"Detail","kind":1,"value":"` + strings.Repeat("<", 11000) + `"}]}`
	var got Notice
	if err := json.Unmarshal([]byte(tooLargeText), &got); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("unmarshal %d angle brackets: err = %v, want ErrNoticeLimit", 11000, err)
	}
	text := Text("Detail", strings.Repeat("<", 11000))
	if _, err := json.Marshal(text); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("marshal %d angle brackets: err = %v, want ErrNoticeLimit", 11000, err)
	}

	tooLargeUnknown := `{"code":1,"args":[{"name":"Detail","kind":9,"value":"` + strings.Repeat("&", 11000) + `"}]}`
	if err := json.Unmarshal([]byte(tooLargeUnknown), &got); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("unmarshal %d ampersands: err = %v, want ErrNoticeLimit", 11000, err)
	}
	unknown := Arg{name: "Detail", kind: ArgKind(9), value: json.RawMessage(`"` + strings.Repeat("&", 11000) + `"`)}
	if _, err := json.Marshal(unknown); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("marshal %d ampersands of an unknown kind: err = %v, want ErrNoticeLimit", 11000, err)
	}
}

func TestNoticeJSONLimitsAnUnknownValueOfMoreThan64KiB(t *testing.T) {
	const size = 70000
	raw := `{"code":1,"args":[{"name":"Detail","kind":9,"value":"` + strings.Repeat("x", size) + `"}]}`
	var got Notice
	if err := json.Unmarshal([]byte(raw), &got); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("unmarshal an unknown kind of %d bytes: err = %v, want ErrNoticeLimit", size, err)
	}
	unknown := Arg{name: "Detail", kind: ArgKind(9), value: json.RawMessage(`"` + strings.Repeat("x", size) + `"`)}
	if _, err := json.Marshal(unknown); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("marshal an unknown kind of %d bytes: err = %v, want ErrNoticeLimit", size, err)
	}
}

func TestNoticeJSONNullLeavesTheNoticeUnchanged(t *testing.T) {
	var holder struct {
		Notice Notice `json:"Notice"`
	}
	holder.Notice = Notice{Code: CodeSessionNotFound, Args: []Arg{Text("Name", "lead")}}
	if err := json.Unmarshal([]byte(`{"Notice":null}`), &holder); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if holder.Notice.Code != CodeSessionNotFound || len(holder.Notice.Args) != 1 {
		t.Fatalf("null changed the notice to %+v", holder.Notice)
	}
}

func TestNoticeJSONSyntaxErrorIsNotAShapeError(t *testing.T) {
	var notice Notice
	err := notice.UnmarshalJSON([]byte(`{"code":`))
	if errors.Is(err, ErrNoticeArgs) {
		t.Fatalf("a malformed JSON notice: err = %v, must not be ErrNoticeArgs", err)
	}
	if _, ok := errors.AsType[*json.SyntaxError](err); !ok {
		t.Fatalf("a malformed JSON notice: err = %v, want a *json.SyntaxError", err)
	}

	var arg Arg
	err = arg.UnmarshalJSON([]byte(`{"name":`))
	if errors.Is(err, ErrNoticeArgs) {
		t.Fatalf("a malformed JSON argument: err = %v, must not be ErrNoticeArgs", err)
	}
	if _, ok := errors.AsType[*json.SyntaxError](err); !ok {
		t.Fatalf("a malformed JSON argument: err = %v, want a *json.SyntaxError", err)
	}

	err = notice.UnmarshalJSON([]byte(`{"code":"x"}`))
	if !errors.Is(err, ErrNoticeArgs) {
		t.Fatalf("a wrong JSON type: err = %v, want ErrNoticeArgs", err)
	}
}

func TestNoticeJSONLimitsTheNameOnItsDecodedBytes(t *testing.T) {
	withName := func(name string) string {
		return `{"code":1,"args":[{"name":"` + name + `","kind":1,"value":"lead"}]}`
	}
	accepted := strings.Repeat("<", 256)
	refused := strings.Repeat("<", 257)

	var got Notice
	if err := json.Unmarshal([]byte(withName(accepted)), &got); err != nil {
		t.Fatalf("unmarshal a name of 256 bytes: %v", err)
	}
	if err := json.Unmarshal([]byte(withName(refused)), &got); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("unmarshal a name of 257 bytes: err = %v, want ErrNoticeLimit", err)
	}
	if _, err := json.Marshal(Text(accepted, "lead")); err != nil {
		t.Fatalf("marshal a name of 256 bytes: %v", err)
	}
	if _, err := json.Marshal(Text(refused, "lead")); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("marshal a name of 257 bytes: err = %v, want ErrNoticeLimit", err)
	}
}

func TestNoticeJSONLimitsTheRawValueBeforeItsCopy(t *testing.T) {
	raw := `{"code":1,"args":[{"name":"Detail","kind":9,"value":` + strings.Repeat(" ", maxNoticeRawBytes+1) + `"x"}]}`
	var got Notice
	if err := json.Unmarshal([]byte(raw), &got); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("unmarshal a value of %d raw bytes: err = %v, want ErrNoticeLimit", maxNoticeRawBytes+1, err)
	}
}

func TestNoticeJSONAcceptsTheEdgeOfTheValueSize(t *testing.T) {
	const onEdge = 10922
	const overEdge = 10923
	raw := `{"code":1,"args":[{"name":"Detail","kind":1,"value":"` + strings.Repeat("<", onEdge) + `"}]}`
	want := `{"code":1,"args":[{"name":"Detail","kind":1,"value":"` + strings.ReplaceAll(strings.Repeat("<", onEdge), "<", "\\u003c") + `"}]}`
	assertWireRoundTrip(t, raw, want)
	if _, err := json.Marshal(Text("Detail", strings.Repeat("<", onEdge))); err != nil {
		t.Fatalf("marshal %d angle brackets: %v", onEdge, err)
	}

	tooLarge := `{"code":1,"args":[{"name":"Detail","kind":1,"value":"` + strings.Repeat("<", overEdge) + `"}]}`
	var got Notice
	if err := json.Unmarshal([]byte(tooLarge), &got); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("unmarshal %d angle brackets: err = %v, want ErrNoticeLimit", overEdge, err)
	}
	if _, err := json.Marshal(Text("Detail", strings.Repeat("<", overEdge))); !errors.Is(err, ErrNoticeLimit) {
		t.Fatalf("marshal %d angle brackets: err = %v, want ErrNoticeLimit", overEdge, err)
	}
}
