package contracts

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestHealthStateStringNamesEachState(t *testing.T) {
	want := map[HealthState]string{HealthOK: "ok", HealthDegraded: "degraded", HealthDown: "down", HealthOff: "off"}
	for state, text := range want {
		if got := state.String(); got != text {
			t.Errorf("%d.String() = %q, want %q", int(state), got, text)
		}
	}
	if len(HealthStates()) != len(want) {
		t.Fatalf("HealthStates() = %v, want the %d states", HealthStates(), len(want))
	}
}

func TestParseHealthStateReadsEveryStateBack(t *testing.T) {
	for _, want := range HealthStates() {
		got, err := ParseHealthState(want.String())
		if err != nil || got != want {
			t.Fatalf("ParseHealthState(%q) = %v, %v, want %v", want.String(), got, err, want)
		}
	}
}

func TestParseHealthStateRefusesUnknownText(t *testing.T) {
	for _, text := range []string{"", "OK", "up", "HealthState(0)"} {
		got, err := ParseHealthState(text)
		if !errors.Is(err, ErrUnknownHealthState) {
			t.Errorf("ParseHealthState(%q) error = %v, want ErrUnknownHealthState", text, err)
		}
		if got != 0 {
			t.Errorf("ParseHealthState(%q) = %v, want the zero value on error", text, got)
		}
	}
}

func TestHealthStateRoundTripsThroughJSON(t *testing.T) {
	type envelope struct {
		State HealthState `json:"state"`
	}
	raw, err := json.Marshal(envelope{State: HealthDegraded})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(raw) != `{"state":"degraded"}` {
		t.Fatalf("wire JSON = %s, want the text form", raw)
	}
	var back envelope
	if err := json.Unmarshal(raw, &back); err != nil || back.State != HealthDegraded {
		t.Fatalf("json.Unmarshal() = %+v, %v, want HealthDegraded", back, err)
	}
}

func TestHealthStateRefusesToMarshalAnUnknownValue(t *testing.T) {
	for _, state := range []HealthState{0, 9} {
		if _, err := state.MarshalText(); !errors.Is(err, ErrUnknownHealthState) {
			t.Errorf("MarshalText(%d) error = %v, want ErrUnknownHealthState", int(state), err)
		}
	}
}

func TestHealthStateUnmarshalRefusesUnknownText(t *testing.T) {
	var state HealthState
	if err := json.Unmarshal([]byte(`"fine"`), &state); !errors.Is(err, ErrUnknownHealthState) {
		t.Fatalf("json.Unmarshal(\"fine\") error = %v, want ErrUnknownHealthState", err)
	}
}

func TestWorstHealthOrdersDownDegradedOKOff(t *testing.T) {
	cases := []struct {
		states []HealthState
		want   HealthState
	}{
		{nil, HealthOff},
		{[]HealthState{HealthOff}, HealthOff},
		{[]HealthState{HealthOff, HealthOK}, HealthOK},
		{[]HealthState{HealthOK, HealthDegraded, HealthOff}, HealthDegraded},
		{[]HealthState{HealthDegraded, HealthDown, HealthOK}, HealthDown},
		{[]HealthState{HealthDown, HealthOff}, HealthDown},
	}
	for _, tc := range cases {
		if got := WorstHealth(tc.states...); got != tc.want {
			t.Errorf("WorstHealth(%v) = %v, want %v", tc.states, got, tc.want)
		}
	}
}

func TestWorstHealthIgnoresUnknownValues(t *testing.T) {
	if got := WorstHealth(HealthState(9), HealthOK, 0); got != HealthOK {
		t.Fatalf("WorstHealth(9, ok, 0) = %v, want HealthOK", got)
	}
	if got := WorstHealth(HealthState(9)); got != HealthOff {
		t.Fatalf("WorstHealth(9) = %v, want HealthOff", got)
	}
}
