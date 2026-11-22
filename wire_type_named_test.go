package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const receivedSecret = "s3cret-received-text"

func requireNamedWireType(t *testing.T, err, sentinel error, field string) {
	t.Helper()
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want it to match %v", err, sentinel)
	}
	msg := err.Error()
	if want := field + " has the wrong JSON type"; !strings.Contains(msg, want) {
		t.Fatalf("message %q does not contain %q", msg, want)
	}
	for _, leak := range []string{receivedSecret, "contracts.", "Go", "of type", "uint16", "string", "float64", "bool"} {
		if strings.Contains(msg, leak) {
			t.Fatalf("message %q leaks %q: it must name the wire field only", msg, leak)
		}
	}
}

func TestNoticeWrongJSONTypeNamesTheWireField(t *testing.T) {
	tests := []struct {
		name  string
		input string
		field string
	}{
		{
			name:  "code",
			input: fmt.Sprintf(`{"code":%q}`, receivedSecret),
			field: "code",
		},
		{
			name:  "args not a list",
			input: fmt.Sprintf(`{"code":1,"args":{%q:1}}`, receivedSecret),
			field: noticeWireArgs,
		},
		{
			name:  "text argument value",
			input: fmt.Sprintf(`{"code":1,"args":[{"name":"n","kind":%d,"value":5}]}`, ArgText),
			field: noticeWireArgs,
		},
		{
			name:  "argument value holds the secret",
			input: fmt.Sprintf(`{"code":1,"args":[{"name":"n","kind":%d,"value":[%q]}]}`, ArgCount, receivedSecret),
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

func TestApprovalRequestWrongJSONTypeNamesTheWireField(t *testing.T) {
	tests := []struct {
		name  string
		input string
		field string
	}{
		{name: "ID", input: fmt.Sprintf(`{"ID":[%q]}`, receivedSecret), field: "ID"},
		{name: "Tool", input: `{"Tool":5}`, field: "Tool"},
		{name: "Subject", input: fmt.Sprintf(`{"Subject":{"v":%q}}`, receivedSecret), field: "Subject"},
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
		{name: "ID", input: fmt.Sprintf(`{"ID":[%q]}`, receivedSecret), field: "ID"},
		{name: "Decision", input: `{"Decision":5}`, field: "Decision"},
		{name: "Reason", input: fmt.Sprintf(`{"Reason":{"v":%q}}`, receivedSecret), field: "Reason"},
		{name: "Refusal", input: fmt.Sprintf(`{"Refusal":[%q]}`, receivedSecret), field: "Refusal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a ApprovalAnswer
			err := json.Unmarshal([]byte(tt.input), &a)
			requireNamedWireType(t, err, ErrInvalidApproval, tt.field)
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
