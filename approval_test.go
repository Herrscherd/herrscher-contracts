package contracts

import (
	"encoding/json"
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
