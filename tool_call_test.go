package contracts

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestToolCallStateParsesItsOwnText(t *testing.T) {
	for _, want := range ToolCallStates() {
		got, err := ParseToolCallState(want.String())
		if err != nil || got != want {
			t.Fatalf("ParseToolCallState(%q) = %v, %v, want %v", want.String(), got, err, want)
		}
	}
}

func TestToolCallStateRoundTripsThroughJSON(t *testing.T) {
	for _, want := range ToolCallStates() {
		raw, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("Marshal(%v): %v", want, err)
		}
		var got ToolCallState
		if err := json.Unmarshal(raw, &got); err != nil || got != want {
			t.Fatalf("Unmarshal(%s) = %v, %v, want %v", raw, got, err, want)
		}
	}
}

func TestToolCallStateZeroValueDoesNotMarshal(t *testing.T) {
	var unset ToolCallState
	if _, err := unset.MarshalText(); !errors.Is(err, ErrUnknownToolCallState) {
		t.Fatalf("MarshalText(zero state) error = %v, want ErrUnknownToolCallState", err)
	}
	if _, err := json.Marshal(ToolCallState(99)); !errors.Is(err, ErrUnknownToolCallState) {
		t.Fatalf("Marshal(unknown state) error = %v, want ErrUnknownToolCallState", err)
	}
}

func TestParseToolCallStateRefusesUnknownText(t *testing.T) {
	if _, err := ParseToolCallState("running"); !errors.Is(err, ErrUnknownToolCallState) {
		t.Fatalf("ParseToolCallState(running) error = %v, want ErrUnknownToolCallState", err)
	}
}

func TestToolCallStateFinishedOnlyWhenResultCame(t *testing.T) {
	cases := map[ToolCallState]bool{
		ToolCallStarted:   false,
		ToolCallSucceeded: true,
		ToolCallFailed:    true,
		ToolCallState(0):  false,
	}
	for state, want := range cases {
		if got := state.Finished(); got != want {
			t.Fatalf("%v.Finished() = %v, want %v", state, got, want)
		}
	}
}

func TestEventWithToolCallKeepsWireForm(t *testing.T) {
	want := Event{T: "status", ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallStarted}}
	const wantJSON = `{"t":"status","tool_call":{"id":"c1","tool":"Bash","state":"started"}}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got Event
		err := json.Unmarshal(raw, &got)
		if err == nil && (got.ToolCall == nil || *got.ToolCall != *want.ToolCall) {
			t.Fatalf("Event decoded ToolCall %+v, want %+v", got.ToolCall, want.ToolCall)
		}
		return got, err
	})
}

func TestEventToolCallKeepsObservedTime(t *testing.T) {
	at := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	want := Event{T: "status", ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallFailed, At: at}}
	const wantJSON = `{"t":"status","tool_call":{"id":"c1","tool":"Bash","state":"failed","at":"2026-11-22T10:00:00Z"}}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got Event
		err := json.Unmarshal(raw, &got)
		if err == nil && (got.ToolCall == nil || !got.ToolCall.At.Equal(at) || got.ToolCall.State != ToolCallFailed) {
			t.Fatalf("Event decoded ToolCall %+v, want %+v", got.ToolCall, want.ToolCall)
		}
		return got, err
	})
}
