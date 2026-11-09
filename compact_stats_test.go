package contracts

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
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

func TestCompactStatsValidate(t *testing.T) {
	valid := CompactStats{Stage: "jev", Outcome: CompactDone, Calls: 3, Kept: 1, Pruned: 2, StateTokens: 812, Duration: time.Second}
	cases := []struct {
		name    string
		edit    func(*CompactStats)
		wantErr error
	}{
		{name: "valid stats", edit: func(*CompactStats) {}},
		{name: "no state sent", edit: func(s *CompactStats) { s.StateTokens = 0 }},
		{name: "outcome unset", edit: func(s *CompactStats) { s.Outcome = 0 }, wantErr: ErrUnknownCompactOutcome},
		{name: "outcome out of the set", edit: func(s *CompactStats) { s.Outcome = CompactSkipped + 1 }, wantErr: ErrUnknownCompactOutcome},
		{name: "negative calls", edit: func(s *CompactStats) { s.Calls = -1 }, wantErr: ErrNegativeCompactCount},
		{name: "negative chars after", edit: func(s *CompactStats) { s.CharsAfter = -1 }, wantErr: ErrNegativeCompactCount},
		{name: "negative request tokens", edit: func(s *CompactStats) { s.RequestTokens = -1 }, wantErr: ErrNegativeCompactCount},
		{name: "negative state tokens", edit: func(s *CompactStats) { s.StateTokens = -1 }, wantErr: ErrNegativeCompactCount},
		{name: "negative duration", edit: func(s *CompactStats) { s.Duration = -time.Second }, wantErr: ErrNegativeCompactCount},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stats := valid
			tc.edit(&stats)
			err := stats.Validate()
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
