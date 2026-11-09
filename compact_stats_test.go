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

func TestCompactStatsWithoutOutcomeRefusesToMarshal(t *testing.T) {
	_, err := json.Marshal(CompactStats{Stage: "jev", Calls: 3})
	if !errors.Is(err, ErrUnknownCompactOutcome) {
		t.Fatalf("json.Marshal(stats without outcome) error = %v, want ErrUnknownCompactOutcome", err)
	}
}

func TestCompactStatsStateFieldsRoundTripThroughJSON(t *testing.T) {
	want := CompactStats{Stage: "jev", Outcome: CompactDone, StateTokens: 812, StateReduction: "texts abridged"}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal(%+v) error = %v", want, err)
	}
	var got CompactStats
	if err := json.Unmarshal(raw, &got); err != nil || got != want {
		t.Fatalf("json.Unmarshal(%s) = %+v, %v, want %+v", raw, got, err, want)
	}
}

func TestCompactStatsOmitsZeroStateFields(t *testing.T) {
	raw, err := json.Marshal(CompactStats{Stage: "jev", Outcome: CompactShort})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", raw, err)
	}
	for _, key := range []string{"StateTokens", "StateReduction"} {
		if _, ok := fields[key]; ok {
			t.Fatalf("zero %s is written in %s, want it omitted", key, raw)
		}
	}
}

func TestCompactStatsOldJSONReadsBackIdentically(t *testing.T) {
	const oldJSON = `{"Stage":"jev","Outcome":"short","Calls":12,"Kept":7,"Truncated":5,"Pruned":0,"Pinned":0,"CharsBefore":40000,"CharsAfter":22000,"TokensRemoved":4500,"Requests":2,"RequestTokens":9000,"Duration":1500000000}`
	var stats CompactStats
	if err := json.Unmarshal([]byte(oldJSON), &stats); err != nil {
		t.Fatalf("json.Unmarshal(old JSON) error = %v", err)
	}
	if stats.StateTokens != 0 || stats.StateReduction != "" {
		t.Fatalf("old JSON set the state fields to %+v, want zero values", stats)
	}
	raw, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("json.Marshal(old stats) error = %v", err)
	}
	if string(raw) != oldJSON {
		t.Fatalf("old JSON re-encoded as %s, want it unchanged", raw)
	}
}

func TestCompactStatsValidateRefusesNegativeStateTokens(t *testing.T) {
	cases := []struct {
		name    string
		tokens  int
		wantErr bool
	}{
		{name: "no state sent", tokens: 0},
		{name: "state sent", tokens: 812},
		{name: "negative", tokens: -1, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CompactStats{Stage: "jev", Outcome: CompactDone, StateTokens: tc.tokens}.Validate()
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("Validate() with %d tokens error = %v, want nil", tc.tokens, err)
				}
				return
			}
			if !errors.Is(err, ErrNegativeStateTokens) {
				t.Fatalf("Validate() with %d tokens error = %v, want ErrNegativeStateTokens", tc.tokens, err)
			}
		})
	}
}
