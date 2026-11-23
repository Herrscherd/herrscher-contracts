package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

const (
	receivedProbe       = echotest.Probe
	oversizedValueBytes = 70000
	unknownArgKind      = 9
)

func requireNamedWireType(t *testing.T, err, sentinel error, field string) {
	t.Helper()
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want it to match %v", err, sentinel)
	}
	msg := err.Error()
	if want := field + " has the wrong JSON type"; !strings.Contains(msg, want) {
		t.Fatalf("message %q does not contain %q", msg, want)
	}
	for _, leak := range []string{"contracts.", "Go", "of type", "uint16", "string", "float64", "bool"} {
		if strings.Contains(msg, leak) {
			t.Fatalf("message %q leaks %q: it must name the wire field only", msg, leak)
		}
	}
	echotest.RequireNoSecretFragment(t, msg, receivedProbe)
}

func requireNamedWireRange(t *testing.T, err, sentinel error, field, value string) {
	t.Helper()
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want it to match %v", err, sentinel)
	}
	msg := err.Error()
	if want := field + " is out of range"; !strings.Contains(msg, want) {
		t.Fatalf("message %q does not contain %q", msg, want)
	}
	for _, leak := range []string{"contracts.", "Go", "of type", "uint16", "uint8", "int64", "float64"} {
		if strings.Contains(msg, leak) {
			t.Fatalf("message %q leaks %q: it must name the wire field only", msg, leak)
		}
	}
	echotest.RequireNoSecretFragment(t, msg, value)
}

func TestNoticeWrongJSONTypeNamesTheWireField(t *testing.T) {
	tests := []struct {
		name  string
		input string
		field string
	}{
		{
			name:  "code",
			input: fmt.Sprintf(`{"code":%q}`, receivedProbe),
			field: "code",
		},
		{
			name:  "code fraction",
			input: `{"code":1.5}`,
			field: "code",
		},
		{
			name:  "code tiny fraction",
			input: `{"code":1e-400}`,
			field: "code",
		},
		{
			name:  "code fraction beyond 64 bits",
			input: `{"code":1.0000000000000000000001}`,
			field: "code",
		},
		{
			name:  "code fraction rounding to zero",
			input: `{"code":1e-999999999}`,
			field: "code",
		},
		{
			name:  "code exponent",
			input: `{"code":1e5}`,
			field: "code",
		},
		{
			name:  "code small exponent",
			input: `{"code":1e2}`,
			field: "code",
		},
		{
			name:  "code integer written as a decimal",
			input: `{"code":1.0}`,
			field: "code",
		},
		{
			name:  "code negative zero",
			input: `{"code":-0}`,
			field: "code",
		},
		{
			name:  "argument name",
			input: fmt.Sprintf(`{"code":1,"args":[{"name":[%q],"kind":1,"value":"x"}]}`, receivedProbe),
			field: noticeWireArgs,
		},
		{
			name:  "argument kind",
			input: fmt.Sprintf(`{"code":1,"args":[{"name":"n","kind":%q,"value":"x"}]}`, receivedProbe),
			field: noticeWireArgs,
		},
		{
			name:  "args not a list",
			input: fmt.Sprintf(`{"code":1,"args":{%q:1}}`, receivedProbe),
			field: noticeWireArgs,
		},
		{
			name:  "text argument value",
			input: fmt.Sprintf(`{"code":1,"args":[{"name":"n","kind":%d,"value":5}]}`, ArgText),
			field: noticeWireArgs,
		},
		{
			name:  "argument value holds the secret",
			input: fmt.Sprintf(`{"code":1,"args":[{"name":"n","kind":%d,"value":[%q]}]}`, ArgCount, receivedProbe),
			field: noticeWireArgs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n Notice
			err := json.Unmarshal([]byte(tt.input), &n)
			requireNamedWireType(t, err, ErrNoticeArgs, tt.field)
		})
	}
}

func TestNoticeOutOfRangeNamesTheWireFieldNotTheNumber(t *testing.T) {
	tests := []struct {
		name  string
		input string
		field string
		value string
	}{
		{name: "code", input: `{"code":70000}`, field: "code", value: "70000"},
		{name: "code beyond int64", input: `{"code":99999999999999999999999}`, field: "code", value: "99999999999999999999999"},
		{name: "kind", input: `{"code":1,"args":[{"name":"n","kind":300,"value":"x"}]}`, field: noticeWireArgs, value: "300"},
		{name: "kind negative", input: `{"code":1,"args":[{"name":"n","kind":-1,"value":"x"}]}`, field: noticeWireArgs, value: "-1"},
		{name: "args beyond float64", input: `{"code":1,"args":1e400}`, field: noticeWireArgs, value: "1e400"},
		{name: "args integer beyond float64", input: `{"code":1,"args":` + strings.Repeat("9", 400) + `}`, field: noticeWireArgs, value: strings.Repeat("9", 400)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n Notice
			err := json.Unmarshal([]byte(tt.input), &n)
			requireNamedWireRange(t, err, ErrNoticeArgs, tt.field, tt.value)
		})
	}
}

func TestApprovalRequestWrongJSONTypeNamesTheWireField(t *testing.T) {
	tests := []struct {
		name  string
		input string
		field string
	}{
		{name: "ID", input: fmt.Sprintf(`{"ID":[%q]}`, receivedProbe), field: "ID"},
		{name: "Tool", input: `{"Tool":5}`, field: "Tool"},
		{name: "Subject", input: fmt.Sprintf(`{"Subject":{"v":%q}}`, receivedProbe), field: "Subject"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r ApprovalRequest
			err := json.Unmarshal([]byte(tt.input), &r)
			requireNamedWireType(t, err, ErrInvalidApproval, tt.field)
		})
	}
}

func TestApprovalAnswerWrongJSONTypeNamesTheWireField(t *testing.T) {
	tests := []struct {
		name  string
		input string
		field string
	}{
		{name: "ID", input: fmt.Sprintf(`{"ID":[%q]}`, receivedProbe), field: "ID"},
		{name: "Decision", input: `{"Decision":5}`, field: "Decision"},
		{name: "Reason", input: fmt.Sprintf(`{"Reason":{"v":%q}}`, receivedProbe), field: "Reason"},
		{name: "Refusal", input: fmt.Sprintf(`{"Refusal":[%q]}`, receivedProbe), field: "Refusal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a ApprovalAnswer
			err := json.Unmarshal([]byte(tt.input), &a)
			requireNamedWireType(t, err, ErrInvalidApproval, tt.field)
		})
	}
}

func TestApprovalAnswerWrongTypeIsNamedWhateverTheFieldOrder(t *testing.T) {
	tests := []struct {
		name  string
		input string
		field string
	}{
		{name: "ID first", input: fmt.Sprintf(`{"ID":4242,"Decision":%q}`, receivedProbe), field: "ID"},
		{name: "Decision first", input: fmt.Sprintf(`{"Decision":%q,"ID":4242}`, receivedProbe), field: "ID"},
		{name: "Refusal first", input: fmt.Sprintf(`{"Refusal":7373,"Decision":%q}`, receivedProbe), field: "Refusal"},
		{name: "Refusal last", input: fmt.Sprintf(`{"Decision":%q,"Refusal":7373}`, receivedProbe), field: "Refusal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a ApprovalAnswer
			err := json.Unmarshal([]byte(tt.input), &a)
			requireNamedWireType(t, err, ErrInvalidApproval, tt.field)
			if errors.Is(err, ErrInvalidDecision) {
				t.Fatalf("error = %v, a wrong JSON type must not match ErrInvalidDecision", err)
			}
		})
	}
}

func TestApprovalAnswerUnknownDecisionNamesTheFieldNotTheText(t *testing.T) {
	var answer ApprovalAnswer
	err := json.Unmarshal([]byte(`{"ID":"1","Decision":"`+receivedProbe+`"}`), &answer)
	if !errors.Is(err, ErrInvalidDecision) || !errors.Is(err, ErrInvalidApproval) {
		t.Fatalf("error = %v, want it to match ErrInvalidDecision and ErrInvalidApproval", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "Decision") {
		t.Fatalf("message %q does not name Decision", msg)
	}
	echotest.RequireNoSecretFragment(t, msg, receivedProbe)
}

func TestApprovalDecisionUnknownTextIsNotEchoed(t *testing.T) {
	var decision ApprovalDecision
	err := json.Unmarshal([]byte(`"`+receivedProbe+`"`), &decision)
	if !errors.Is(err, ErrInvalidDecision) {
		t.Fatalf("error = %v, want ErrInvalidDecision", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
}

func TestNoticeArgumentErrorNamesTheWireFieldNotTheValue(t *testing.T) {
	oversized := `"` + receivedProbe + strings.Repeat("a", oversizedValueBytes) + `"`
	tests := []struct {
		name     string
		input    string
		sentinel error
	}{
		{
			name:     "value required",
			input:    `{"code":1,"args":[{"name":"nom-fil","kind":1}]}`,
			sentinel: ErrNoticeArgs,
		},
		{
			name:     "wrong JSON type",
			input:    fmt.Sprintf(`{"code":1,"args":[{"name":"nom-fil","kind":%d,"value":5}]}`, ArgText),
			sentinel: ErrNoticeArgs,
		},
		{
			name:     "value too long to encode",
			input:    fmt.Sprintf(`{"code":1,"args":[{"name":"nom-fil","kind":%d,"value":%s}]}`, unknownArgKind, oversized),
			sentinel: ErrNoticeLimit,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n Notice
			err := json.Unmarshal([]byte(tt.input), &n)
			if !errors.Is(err, tt.sentinel) {
				t.Fatalf("error = %v, want it to match %v", err, tt.sentinel)
			}
			msg := err.Error()
			if !strings.HasPrefix(msg, "argument: ") {
				t.Fatalf("message %q does not name the wire field", msg)
			}
			if strings.Contains(msg, "nom-fil") || strings.Contains(msg, "aaa") {
				t.Fatalf("message %q echoes the received value", msg)
			}
			echotest.RequireNoSecretFragment(t, msg, receivedProbe)
		})
	}
}

func TestApprovalAnswerNullLeavesItUnchanged(t *testing.T) {
	a := ApprovalAnswer{ID: "keep", Decision: DecisionAllow}
	if err := json.Unmarshal([]byte(`null`), &a); err != nil {
		t.Fatalf("Unmarshal(null) error = %v", err)
	}
	if a.ID != "keep" || a.Decision != DecisionAllow {
		t.Fatalf("Unmarshal(null) changed the answer to %+v", a)
	}
}

func TestApprovalRequestNullLeavesItUnchanged(t *testing.T) {
	r := ApprovalRequest{ID: "keep"}
	if err := json.Unmarshal([]byte(`null`), &r); err != nil {
		t.Fatalf("Unmarshal(null) error = %v", err)
	}
	if r.ID != "keep" {
		t.Fatalf("Unmarshal(null) changed the request to %+v", r)
	}
}
