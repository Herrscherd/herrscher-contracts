package contracts

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestCompactOutcomeTextForms(t *testing.T) {
	cases := map[CompactOutcome]string{
		CompactDone:    "done",
		CompactShort:   "short",
		CompactFailed:  "failed",
		CompactDryRun:  "dry_run",
		CompactSkipped: "skipped",
	}
	if len(cases) != len(CompactOutcomes()) {
		t.Fatalf("%d text forms for %d outcomes", len(cases), len(CompactOutcomes()))
	}
	for outcome, want := range cases {
		if got := outcome.String(); got != want {
			t.Fatalf("%d.String() = %q, want %q", int(outcome), got, want)
		}
		parsed, err := ParseCompactOutcome(want)
		if err != nil || parsed != outcome {
			t.Fatalf("ParseCompactOutcome(%q) = %v, %v, want %v", want, parsed, err, outcome)
		}
	}
}

func TestCompactOutcomeRoundTripsThroughJSON(t *testing.T) {
	for _, want := range CompactOutcomes() {
		raw, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("Marshal(%v): %v", want, err)
		}
		var got CompactOutcome
		if err := json.Unmarshal(raw, &got); err != nil || got != want {
			t.Fatalf("Unmarshal(%s) = %v, %v, want %v", raw, got, err, want)
		}
	}
}

func TestCompactOutcomeRefusesUnknownText(t *testing.T) {
	for _, text := range []string{"", "complete", "Done", "dry-run"} {
		if _, err := ParseCompactOutcome(text); !errors.Is(err, ErrUnknownCompactOutcome) {
			t.Fatalf("ParseCompactOutcome(%q) error = %v, want ErrUnknownCompactOutcome", text, err)
		}
		var outcome CompactOutcome
		if err := outcome.UnmarshalText([]byte(text)); !errors.Is(err, ErrUnknownCompactOutcome) {
			t.Fatalf("UnmarshalText(%q) error = %v, want ErrUnknownCompactOutcome", text, err)
		}
	}
	if _, err := CompactOutcome(0).MarshalText(); !errors.Is(err, ErrUnknownCompactOutcome) {
		t.Fatalf("MarshalText(zero outcome) error = %v, want ErrUnknownCompactOutcome", err)
	}
}
