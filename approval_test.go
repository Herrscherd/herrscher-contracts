package contracts

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
)

func jsonKeys(t *testing.T, value any) []string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func TestApprovalRequestJSONCarriesNoSessionOrToken(t *testing.T) {
	got := jsonKeys(t, ApprovalRequest{ID: "1", Tool: "Bash", Subject: "ls"})
	want := []string{"ID", "Subject", "Tool"}
	if !slices.Equal(got, want) {
		t.Fatalf("ApprovalRequest keys = %v, want %v", got, want)
	}
}

func TestApprovalAnswerJSONCarriesNoSessionOrToken(t *testing.T) {
	got := jsonKeys(t, ApprovalAnswer{ID: "1", Decision: DecisionAllow, Reason: "ok"})
	want := []string{"Decision", "ID", "Reason"}
	if !slices.Equal(got, want) {
		t.Fatalf("ApprovalAnswer keys = %v, want %v", got, want)
	}
}

func TestApprovalAnswerAllowed(t *testing.T) {
	cases := map[string]struct {
		decision ApprovalDecision
		want     bool
	}{
		"allow":   {decision: DecisionAllow, want: true},
		"deny":    {decision: DecisionDeny, want: false},
		"empty":   {decision: "", want: false},
		"unknown": {decision: "maybe", want: false},
		"upper":   {decision: "ALLOW", want: false},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			answer := ApprovalAnswer{ID: "1", Decision: testCase.decision}
			if got := answer.Allowed(); got != testCase.want {
				t.Fatalf("Allowed() with decision %q = %v, want %v", testCase.decision, got, testCase.want)
			}
		})
	}
}

func TestParseApprovalDecisionAcceptsKnownValues(t *testing.T) {
	for _, text := range []string{"allow", "deny"} {
		got, err := ParseApprovalDecision(text)
		if err != nil {
			t.Fatalf("ParseApprovalDecision(%q): %v", text, err)
		}
		if string(got) != text {
			t.Fatalf("ParseApprovalDecision(%q) = %q, want the same text", text, got)
		}
	}
}

func TestParseApprovalDecisionRefusesUnknownValues(t *testing.T) {
	for _, text := range []string{"maybe", "", "ALLOW", "Deny"} {
		if _, err := ParseApprovalDecision(text); !errors.Is(err, ErrInvalidDecision) {
			t.Fatalf("ParseApprovalDecision(%q) error = %v, want ErrInvalidDecision", text, err)
		}
	}
}

func TestApprovalDecisionMarshalRefusesUnknownValue(t *testing.T) {
	if _, err := json.Marshal(ApprovalAnswer{ID: "1", Decision: "maybe"}); !errors.Is(err, ErrInvalidDecision) {
		t.Fatalf("marshal with unknown decision error = %v, want ErrInvalidDecision", err)
	}
}

func TestApprovalAnswerUnmarshalRefusesUnknownDecision(t *testing.T) {
	var answer ApprovalAnswer
	err := json.Unmarshal([]byte(`{"ID":"1","Decision":"maybe"}`), &answer)
	if !errors.Is(err, ErrInvalidDecision) {
		t.Fatalf("unmarshal with unknown decision error = %v, want ErrInvalidDecision", err)
	}
}
