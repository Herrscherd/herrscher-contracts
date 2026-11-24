package contracts

import (
	"bytes"
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

func TestApprovalAnswerUnmarshalKeepsFutureRefusal(t *testing.T) {
	var answer ApprovalAnswer
	if err := json.Unmarshal([]byte(`{"ID":"1","Decision":"deny","Refusal":"quota_exceeded"}`), &answer); err != nil {
		t.Fatalf("unmarshal with future refusal: %v", err)
	}
	if answer.Decision != DecisionDeny {
		t.Fatalf("Decision = %q, want deny preserved", answer.Decision)
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

func TestApprovalAnswerReadsRefusedAllowAsDeny(t *testing.T) {
	var answer ApprovalAnswer
	if err := json.Unmarshal([]byte(`{"ID":"1","Decision":"allow","Refusal":"too_many"}`), &answer); err != nil {
		t.Fatalf("unmarshal of refused allow: %v", err)
	}
	if answer.Decision != DecisionDeny {
		t.Fatalf("Decision = %q, want deny for a refused answer", answer.Decision)
	}
	if answer.Refusal != RefusalTooMany {
		t.Fatalf("Refusal = %q, want too_many kept", answer.Refusal)
	}
}

func TestApprovalAnswerMarshalRefusesRefusedAllow(t *testing.T) {
	if _, err := json.Marshal(ApprovalAnswer{ID: "1", Decision: DecisionAllow, Refusal: RefusalTooMany}); !errors.Is(err, ErrRefusedAllow) {
		t.Fatalf("marshal of refused allow error = %v, want ErrRefusedAllow", err)
	}
}

func TestApprovalAnswerMarshalRefusesUnknownDecisionWithRefusal(t *testing.T) {
	for _, decision := range []ApprovalDecision{"", "maybe"} {
		if _, err := json.Marshal(ApprovalAnswer{ID: "1", Decision: decision, Refusal: RefusalTooMany}); !errors.Is(err, ErrInvalidDecision) {
			t.Fatalf("marshal with decision %q and refusal error = %v, want ErrInvalidDecision", decision, err)
		}
	}
}

func TestApprovalAnswerMarshalWritesDenyWithRefusalAsIs(t *testing.T) {
	encoded, err := json.Marshal(ApprovalAnswer{ID: "1", Decision: DecisionDeny, Refusal: RefusalTooMany})
	if err != nil {
		t.Fatalf("marshal of refused deny: %v", err)
	}
	if string(encoded) != `{"ID":"1","Decision":"deny","Reason":"","Refusal":"too_many"}` {
		t.Fatalf("marshal = %s, want the deny written unchanged", encoded)
	}
}

func TestApprovalAnswerReadsMissingDecisionWithRefusalAsDeny(t *testing.T) {
	var answer ApprovalAnswer
	if err := json.Unmarshal([]byte(`{"ID":"1","Refusal":"too_many"}`), &answer); err != nil {
		t.Fatalf("unmarshal refusal without Decision: %v", err)
	}
	if answer.Decision != DecisionDeny || answer.Allowed() || answer.Refusal != RefusalTooMany {
		t.Fatalf("answer = %+v, want a deny carrying too_many", answer)
	}
}

func TestApprovalAnswerReadRefusesPresentUnknownDecisionWithRefusal(t *testing.T) {
	for _, input := range []string{`{"ID":"1","Decision":"","Refusal":"too_many"}`, `{"ID":"1","Decision":"maybe","Refusal":"too_many"}`} {
		var answer ApprovalAnswer
		if err := json.Unmarshal([]byte(input), &answer); !errors.Is(err, ErrInvalidDecision) {
			t.Fatalf("unmarshal %s error = %v, want ErrInvalidDecision", input, err)
		}
	}
}

func TestApprovalRequestDecoderUnknownKeyIsNotRefused(t *testing.T) {
	decoder := json.NewDecoder(strings.NewReader(`{"ID":"1","Tool":"Bash","Subject":"ls","Extra":true}`))
	decoder.DisallowUnknownFields()
	var request ApprovalRequest
	if err := decoder.Decode(&request); err != nil {
		t.Fatalf("Decode() error = %v, want the unknown key ignored", err)
	}
	if request.ID != "1" || request.Tool != "Bash" || request.Subject != "ls" {
		t.Fatalf("Decode() = %+v, want the known fields read", request)
	}
}

func TestApprovalAnswerDecoderUnknownKeyIsNotRefused(t *testing.T) {
	decoder := json.NewDecoder(strings.NewReader(`{"ID":"1","Decision":"allow","Extra":true}`))
	decoder.DisallowUnknownFields()
	var answer ApprovalAnswer
	if err := decoder.Decode(&answer); err != nil {
		t.Fatalf("Decode() error = %v, want the unknown key ignored", err)
	}
	if answer.ID != "1" || answer.Decision != DecisionAllow {
		t.Fatalf("Decode() = %+v, want the known fields read", answer)
	}
}

func TestApprovalAnswerNullLeavesAnswerUnchanged(t *testing.T) {
	answer := ApprovalAnswer{ID: "1", Decision: DecisionAllow, Reason: "ok"}
	if err := json.Unmarshal([]byte("null"), &answer); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	want := ApprovalAnswer{ID: "1", Decision: DecisionAllow, Reason: "ok"}
	if answer != want {
		t.Fatalf("answer after null = %+v, want %+v unchanged", answer, want)
	}
}

type approvalAnswerV034 struct {
	ID       string           `json:"ID"`
	Decision ApprovalDecision `json:"Decision"`
	Reason   string           `json:"Reason"`
}

func TestApprovalAnswerOrdinaryBytesMatchV034(t *testing.T) {
	reasons := []string{"", "ok", `quote " and backslash \`, "line\nbreak\tand tab", "accents é et <tags>&amp;", "  separator"}
	for _, decision := range []ApprovalDecision{DecisionAllow, DecisionDeny} {
		for _, reason := range reasons {
			for _, id := range []string{"", "1", `a"b`} {
				assertBytesMatchV034(t, ApprovalAnswer{ID: id, Decision: decision, Reason: reason})
			}
		}
	}
}

func assertBytesMatchV034(t *testing.T, answer ApprovalAnswer) {
	t.Helper()
	got, err := json.Marshal(answer)
	if err != nil {
		t.Fatalf("marshal %+v: %v", answer, err)
	}
	want, err := json.Marshal(approvalAnswerV034{ID: answer.ID, Decision: answer.Decision, Reason: answer.Reason})
	if err != nil {
		t.Fatalf("marshal v0.34 of %+v: %v", answer, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("bytes = %s, want v0.34 bytes %s", got, want)
	}
}

func TestApprovalAnswerReadsRefusedAllowAsDenyInContainers(t *testing.T) {
	const input = `{"ID":"1","Decision":"allow","Refusal":"too_many"}`
	check := func(t *testing.T, decision ApprovalDecision) {
		t.Helper()
		if decision != DecisionDeny {
			t.Fatalf("Decision inside container = %q, want deny", decision)
		}
	}
	t.Run("pointer", func(t *testing.T) {
		var got *ApprovalAnswer
		if err := json.Unmarshal([]byte(input), &got); err != nil {
			t.Fatalf("unmarshal pointer: %v", err)
		}
		check(t, got.Decision)
	})
	t.Run("slice", func(t *testing.T) {
		var got []ApprovalAnswer
		if err := json.Unmarshal([]byte("["+input+"]"), &got); err != nil {
			t.Fatalf("unmarshal slice: %v", err)
		}
		check(t, got[0].Decision)
	})
	t.Run("struct field", func(t *testing.T) {
		var got struct {
			Answer ApprovalAnswer `json:"answer"`
		}
		if err := json.Unmarshal([]byte(`{"answer":`+input+`}`), &got); err != nil {
			t.Fatalf("unmarshal struct field: %v", err)
		}
		check(t, got.Answer.Decision)
	})
	t.Run("map", func(t *testing.T) {
		var got map[string]ApprovalAnswer
		if err := json.Unmarshal([]byte(`{"k":`+input+`}`), &got); err != nil {
			t.Fatalf("unmarshal map: %v", err)
		}
		check(t, got["k"].Decision)
	})
}

func TestApprovalAnswerOrdinaryJSONUnchanged(t *testing.T) {
	want := ApprovalAnswer{ID: "1", Decision: DecisionAllow, Reason: "ok"}
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal ordinary answer: %v", err)
	}
	if string(encoded) != `{"ID":"1","Decision":"allow","Reason":"ok"}` {
		t.Fatalf("marshal = %s, want the ordinary wire form", encoded)
	}
	var got ApprovalAnswer
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal ordinary answer: %v", err)
	}
	if got != want {
		t.Fatalf("decoded = %+v, want %+v", got, want)
	}
}

func TestApprovalAnswerReadsEmptyRefusalAsAllow(t *testing.T) {
	var answer ApprovalAnswer
	if err := json.Unmarshal([]byte(`{"ID":"1","Decision":"allow","Refusal":""}`), &answer); err != nil {
		t.Fatalf("unmarshal with empty refusal: %v", err)
	}
	want := ApprovalAnswer{ID: "1", Decision: DecisionAllow}
	if answer != want {
		t.Fatalf("decoded = %+v, want %+v", answer, want)
	}
	if !answer.Allowed() {
		t.Fatalf("Allowed() with empty refusal = false, want true")
	}
}

func TestRefusedAnswerBuildsDenyForKnownCode(t *testing.T) {
	got, err := RefusedAnswer("1", RefusalTooMany)
	if err != nil {
		t.Fatalf("RefusedAnswer with known code: %v", err)
	}
	want := ApprovalAnswer{ID: "1", Decision: DecisionDeny, Refusal: RefusalTooMany}
	if got != want {
		t.Fatalf("RefusedAnswer = %+v, want %+v", got, want)
	}
}

func TestRefusedAnswerRefusesEmptyAndUnknownCodes(t *testing.T) {
	for _, refusal := range []ApprovalRefusal{"", "quota_exceeded"} {
		if _, err := RefusedAnswer("1", refusal); !errors.Is(err, ErrInvalidRefusal) {
			t.Fatalf("RefusedAnswer(%q) error = %v, want ErrInvalidRefusal", refusal, err)
		}
	}
}

func TestApprovalAnswerTextTagsMatchWireFields(t *testing.T) {
	got := jsonKeys(t, approvalAnswerText{})
	want := approvalAnswerWireFields()
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("json keys of approvalAnswerText = %v, approvalAnswerWireFields() = %v", got, want)
	}
}
