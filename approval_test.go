package contracts

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
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

func TestApprovalAnswerRefusalRoundTrip(t *testing.T) {
	want := ApprovalAnswer{ID: "1", Decision: DecisionDeny, Reason: "too big", Refusal: RefusalTooLarge}
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"Refusal":"too_large"`) {
		t.Fatalf("marshal = %s, want the Refusal key with its short code", encoded)
	}
	var got ApprovalAnswer
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal %s: %v", encoded, err)
	}
	if got != want {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
}

func TestApprovalAnswerOmitsEmptyRefusal(t *testing.T) {
	got := jsonKeys(t, ApprovalAnswer{ID: "1", Decision: DecisionDeny, Reason: "human"})
	if slices.Contains(got, "Refusal") {
		t.Fatalf("ApprovalAnswer keys = %v, want no Refusal key when empty", got)
	}
}

func TestParseApprovalRefusalAcceptsKnownValues(t *testing.T) {
	known := []string{"invalid", "too_large", "too_many", "no_asker", "cancelled", "deadline", "stream_closed", "send_failed", "unclassified"}
	for _, text := range known {
		got, err := ParseApprovalRefusal(text)
		if err != nil {
			t.Fatalf("ParseApprovalRefusal(%q): %v", text, err)
		}
		if string(got) != text {
			t.Fatalf("ParseApprovalRefusal(%q) = %q, want the same text", text, got)
		}
	}
}

func TestParseApprovalRefusalRefusesUnknownValues(t *testing.T) {
	for _, text := range []string{"maybe", "", "TOO_LARGE", "too-large", "Invalid"} {
		if _, err := ParseApprovalRefusal(text); !errors.Is(err, ErrInvalidRefusal) {
			t.Fatalf("ParseApprovalRefusal(%q) error = %v, want ErrInvalidRefusal", text, err)
		}
	}
}

func TestApprovalAnswerRefusedAllowDoesNotAllow(t *testing.T) {
	answer := ApprovalAnswer{ID: "1", Decision: DecisionAllow, Refusal: RefusalSendFailed}
	if answer.Allowed() {
		t.Fatalf("Allowed() with decision allow and refusal %q = true, want false", answer.Refusal)
	}
}

func TestApprovalAnswerUnmarshalKeepsFutureRefusal(t *testing.T) {
	var answer ApprovalAnswer
	if err := json.Unmarshal([]byte(`{"ID":"1","Decision":"allow","Refusal":"quota_exceeded"}`), &answer); err != nil {
		t.Fatalf("unmarshal with future refusal: %v", err)
	}
	if answer.Decision != DecisionAllow {
		t.Fatalf("Decision = %q, want allow preserved", answer.Decision)
	}
	if answer.Refusal != "quota_exceeded" {
		t.Fatalf("Refusal = %q, want quota_exceeded kept", answer.Refusal)
	}
	if answer.Refusal.Known() {
		t.Fatalf("Known() for future refusal = true, want false")
	}
	if answer.Allowed() {
		t.Fatalf("Allowed() with a future refusal = true, want false")
	}
}

func TestApprovalRefusalRelaysFutureCode(t *testing.T) {
	want := ApprovalAnswer{ID: "1", Decision: DecisionDeny, Refusal: "quota_exceeded"}
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal with future refusal: %v", err)
	}
	var got ApprovalAnswer
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal relayed %s: %v", encoded, err)
	}
	if got != want {
		t.Fatalf("relay = %+v, want %+v", got, want)
	}
}

func TestApprovalRefusalTextFormRefusesEmpty(t *testing.T) {
	if _, err := json.Marshal(ApprovalRefusal("")); !errors.Is(err, ErrInvalidRefusal) {
		t.Fatalf("marshal of empty refusal error = %v, want ErrInvalidRefusal", err)
	}
	var refusal ApprovalRefusal
	if err := json.Unmarshal([]byte(`""`), &refusal); !errors.Is(err, ErrInvalidRefusal) {
		t.Fatalf("unmarshal of empty refusal error = %v, want ErrInvalidRefusal", err)
	}
	var answer ApprovalAnswer
	if err := json.Unmarshal([]byte(`{"ID":"1","Decision":"deny","Refusal":""}`), &answer); !errors.Is(err, ErrInvalidRefusal) {
		t.Fatalf("unmarshal of answer with empty refusal error = %v, want ErrInvalidRefusal", err)
	}
}

func TestApprovalRefusalKnownReportsKnownCodes(t *testing.T) {
	known := []ApprovalRefusal{RefusalInvalid, RefusalTooLarge, RefusalTooMany, RefusalNoAsker, RefusalCancelled, RefusalDeadline, RefusalStreamClosed, RefusalSendFailed, RefusalUnclassified}
	for _, refusal := range known {
		if !refusal.Known() {
			t.Fatalf("Known() for %q = false, want true", refusal)
		}
	}
	for _, refusal := range []ApprovalRefusal{"quota_exceeded", "Invalid", ""} {
		if refusal.Known() {
			t.Fatalf("Known() for %q = true, want false", refusal)
		}
	}
}

func TestRefusedAnswerDeniesAndCarriesRefusal(t *testing.T) {
	got := RefusedAnswer("1", RefusalTooMany)
	want := ApprovalAnswer{ID: "1", Decision: DecisionDeny, Refusal: RefusalTooMany}
	if got != want {
		t.Fatalf("RefusedAnswer = %+v, want %+v", got, want)
	}
	if got.Allowed() {
		t.Fatalf("Allowed() of a refused answer = true, want false")
	}
}

func TestApprovalAnswerWithoutRefusalDecodesUnchanged(t *testing.T) {
	var got ApprovalAnswer
	if err := json.Unmarshal([]byte(`{"ID":"1","Decision":"allow","Reason":"ok"}`), &got); err != nil {
		t.Fatalf("unmarshal answer without refusal: %v", err)
	}
	want := ApprovalAnswer{ID: "1", Decision: DecisionAllow, Reason: "ok"}
	if got != want {
		t.Fatalf("decoded = %+v, want %+v", got, want)
	}
	if !got.Allowed() {
		t.Fatalf("Allowed() = false, want true")
	}
}
