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

func TestToolCallValidateRefusesMalformedCalls(t *testing.T) {
	cases := map[string]ToolCall{
		"zero state":                {ID: "c1", Tool: "Bash"},
		"empty id":                  {Tool: "Bash", State: ToolCallStarted},
		"empty tool":                {ID: "c1", State: ToolCallSucceeded},
		"error on a started call":   {ID: "c1", Tool: "Bash", State: ToolCallStarted, Error: "exit 1"},
		"error on a succeeded call": {ID: "c1", Tool: "Bash", State: ToolCallSucceeded, Error: "exit 1"},
		"succeeded with detail":     {ID: "c1", Tool: "Bash", Detail: "ls", State: ToolCallSucceeded},
		"failed with detail":        {ID: "c1", Tool: "Bash", Detail: "ls", State: ToolCallFailed, Error: "exit 1"},
		"unknown state value":       {ID: "c1", Tool: "Bash", State: ToolCallState(42)},
	}
	for name, call := range cases {
		if err := call.Validate(); !errors.Is(err, ErrInvalidToolCall) {
			t.Fatalf("%s: Validate() = %v, want ErrInvalidToolCall", name, err)
		}
	}
}

func TestToolCallValidateAcceptsFinishedCallWithTool(t *testing.T) {
	finished := ToolCall{ID: "c1", Tool: "Bash", State: ToolCallSucceeded}
	if err := finished.Validate(); err != nil {
		t.Fatalf("Validate(finished call with tool) = %v, want nil", err)
	}
	failed := ToolCall{ID: "c1", Tool: "Bash", State: ToolCallFailed, Error: "exit 1"}
	if err := failed.Validate(); err != nil {
		t.Fatalf("Validate(failed call with error) = %v, want nil", err)
	}
}

func TestToolCallUnmarshalRefusesMalformedWire(t *testing.T) {
	cases := map[string]string{
		"absent state":          `{"id":"c1","tool":"Bash"}`,
		"zero state":            `{"id":"c1","tool":"Bash","state":""}`,
		"empty id":              `{"id":"","tool":"Bash","state":"started"}`,
		"absent id":             `{"tool":"Bash","state":"started"}`,
		"empty tool":            `{"id":"c1","tool":"","state":"started"}`,
		"absent tool":           `{"id":"c1","state":"started"}`,
		"error on succeeded":    `{"id":"c1","tool":"Bash","state":"succeeded","error":"exit 1"}`,
		"succeeded with detail": `{"id":"c1","tool":"Bash","state":"succeeded","detail":"ls"}`,
		"failed with detail":    `{"id":"c1","tool":"Bash","state":"failed","detail":"ls","error":"exit 1"}`,
	}
	for name, raw := range cases {
		var call ToolCall
		if err := json.Unmarshal([]byte(raw), &call); !errors.Is(err, ErrInvalidToolCall) {
			t.Fatalf("%s: Unmarshal(%s) error = %v, want ErrInvalidToolCall", name, raw, err)
		}
	}
}

func TestToolCallUnmarshalDecodesValidWire(t *testing.T) {
	const raw = `{"id":"c1","tool":"Bash","state":"succeeded"}`
	var call ToolCall
	if err := json.Unmarshal([]byte(raw), &call); err != nil {
		t.Fatalf("Unmarshal(%s): %v", raw, err)
	}
	want := ToolCall{ID: "c1", Tool: "Bash", State: ToolCallSucceeded}
	if call != want {
		t.Fatalf("Unmarshal(%s) = %+v, want %+v", raw, call, want)
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

func TestEventWithFailedToolCallKeepsWireForm(t *testing.T) {
	want := Event{T: "status", ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallFailed, Error: "exit 1"}}
	const wantJSON = `{"t":"status","tool_call":{"id":"c1","tool":"Bash","state":"failed","error":"exit 1"}}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got Event
		err := json.Unmarshal(raw, &got)
		if err == nil && (got.ToolCall == nil || *got.ToolCall != *want.ToolCall) {
			t.Fatalf("Event decoded ToolCall %+v, want %+v", got.ToolCall, want.ToolCall)
		}
		return got, err
	})
}

func TestEventWithoutToolCallOmitsIt(t *testing.T) {
	got, err := json.Marshal(Event{T: "status"})
	if err != nil {
		t.Fatalf("Marshal(Event without ToolCall): %v", err)
	}
	if want := `{"t":"status"}`; string(got) != want {
		t.Fatalf("Event without ToolCall marshals as %s, want %s", got, want)
	}
}

func TestEventToolCallKeepsObservedTime(t *testing.T) {
	at := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	want := Event{T: "status", ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallFailed, At: at, Error: "exit 1"}}
	const wantJSON = `{"t":"status","tool_call":{"id":"c1","tool":"Bash","state":"failed","at":"2026-11-22T10:00:00Z","error":"exit 1"}}`
	assertJSONRoundTrip(t, want, wantJSON, func(raw []byte) (any, error) {
		var got Event
		err := json.Unmarshal(raw, &got)
		if err == nil && (got.ToolCall == nil || !got.ToolCall.At.Equal(at) || got.ToolCall.State != ToolCallFailed) {
			t.Fatalf("Event decoded ToolCall %+v, want %+v", got.ToolCall, want.ToolCall)
		}
		return got, err
	})
}

func TestToolCallMarshalRefusesMalformedCallsInEvent(t *testing.T) {
	cases := map[string]ToolCall{
		"empty id":                  {Tool: "Bash", State: ToolCallStarted},
		"empty tool":                {ID: "c1", State: ToolCallStarted},
		"zero state":                {ID: "c1", Tool: "Bash"},
		"error on a started call":   {ID: "c1", Tool: "Bash", State: ToolCallStarted, Error: "exit 1"},
		"error on a succeeded call": {ID: "c1", Tool: "Bash", State: ToolCallSucceeded, Error: "exit 1"},
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := json.Marshal(Event{T: "status", ToolCall: &call})
			if !errors.Is(err, ErrInvalidToolCall) {
				t.Fatalf("Marshal(Event with %s) error = %v, want ErrInvalidToolCall", name, err)
			}
		})
	}
}

func TestToolCallMarshalRoundTripsValidCalls(t *testing.T) {
	at := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	cases := map[string]ToolCall{
		"started":                {ID: "c1", Tool: "Bash", Detail: "ls", State: ToolCallStarted},
		"succeeded":              {ID: "c1", Tool: "Bash", State: ToolCallSucceeded},
		"failed with error":      {ID: "c1", Tool: "Bash", State: ToolCallFailed, Error: "exit 1"},
		"failed with stamped at": {ID: "c1", Tool: "Bash", State: ToolCallFailed, At: at, Error: "exit 1"},
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			raw, err := json.Marshal(want)
			if err != nil {
				t.Fatalf("Marshal(%s): %v", name, err)
			}
			var got ToolCall
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("Unmarshal(%s): %v", raw, err)
			}
			if !sameToolCall(got, want) {
				t.Fatalf("round trip of %s = %+v, want %+v", raw, got, want)
			}
		})
	}
}

func sameToolCall(a, b ToolCall) bool {
	return a.ID == b.ID && a.Tool == b.Tool && a.Detail == b.Detail && a.State == b.State &&
		a.Error == b.Error && a.At.Equal(b.At)
}

func TestToolCallUnmarshalRefusesUnknownStateWithBothSentinels(t *testing.T) {
	const raw = `{"id":"c1","tool":"Bash","state":"running"}`
	var call ToolCall
	err := json.Unmarshal([]byte(raw), &call)
	if !errors.Is(err, ErrInvalidToolCall) || !errors.Is(err, ErrUnknownToolCallState) {
		t.Fatalf("Unmarshal(%s) error = %v, want ErrInvalidToolCall and ErrUnknownToolCallState", raw, err)
	}
}

func TestToolCallUnmarshalRefusesErrorOnStartedCall(t *testing.T) {
	const raw = `{"id":"c1","tool":"Bash","state":"started","error":"exit 1"}`
	var call ToolCall
	if err := json.Unmarshal([]byte(raw), &call); !errors.Is(err, ErrInvalidToolCall) {
		t.Fatalf("Unmarshal(%s) error = %v, want ErrInvalidToolCall", raw, err)
	}
}

func TestEventUnmarshalRefusesInvalidToolCall(t *testing.T) {
	cases := map[string]string{
		"empty id":      `{"t":"status","tool_call":{"id":"","tool":"Bash","state":"started"}}`,
		"unknown state": `{"t":"status","tool_call":{"id":"c1","tool":"Bash","state":"running"}}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			var event Event
			if err := json.Unmarshal([]byte(raw), &event); !errors.Is(err, ErrInvalidToolCall) {
				t.Fatalf("Unmarshal(%s) error = %v, want ErrInvalidToolCall", raw, err)
			}
		})
	}
}

func TestBackendEventValidateToolCallRefusesBrokenShapes(t *testing.T) {
	started := ToolCall{ID: "c1", Tool: "Bash", State: ToolCallStarted}
	finished := ToolCall{ID: "c1", Tool: "Bash", State: ToolCallSucceeded}
	cases := map[string]BackendEvent{
		"call that fails its own validation": {Kind: BackendKindTool, Tool: "Bash", ToolCall: &ToolCall{Tool: "Bash", State: ToolCallStarted}},
		"start carrying a finished call":     {Kind: BackendKindTool, Tool: "Bash", ToolCall: &finished},
		"start whose tool differs from call": {Kind: BackendKindTool, Tool: "Read", ToolCall: &started},
		"start with a stamped at":            {Kind: BackendKindTool, Tool: "Bash", ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallStarted, At: time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)}},
		"result without a call":              {Kind: BackendKindToolDone},
		"result still started":               {Kind: BackendKindToolDone, ToolCall: &started},
		"result with tool on the event":      {Kind: BackendKindToolDone, Tool: "Bash", ToolCall: &finished},
		"result with detail on the event":    {Kind: BackendKindToolDone, Detail: "ls", ToolCall: &finished},
		"result with detail on the call":     {Kind: BackendKindToolDone, ToolCall: &ToolCall{ID: "c1", Tool: "Bash", Detail: "ls", State: ToolCallSucceeded}},
		"result with a stamped at":           {Kind: BackendKindToolDone, ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallSucceeded, At: time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)}},
		"text carrying a call":               {Kind: "text", ToolCall: &started},
	}
	for name, event := range cases {
		t.Run(name, func(t *testing.T) {
			if err := event.ValidateToolCall(); !errors.Is(err, ErrInvalidToolCall) {
				t.Fatalf("ValidateToolCall() = %v, want ErrInvalidToolCall", err)
			}
		})
	}
}

func TestBackendEventValidateToolCallAcceptsWellFormedShapes(t *testing.T) {
	cases := map[string]BackendEvent{
		"start without a call":           {Kind: BackendKindTool, Tool: "Bash", Detail: "ls"},
		"start with its shim and a call": {Kind: BackendKindTool, Tool: "Bash", Detail: "ls", ToolCall: &ToolCall{ID: "c1", Tool: "Bash", Detail: "ls", State: ToolCallStarted}},
		"result succeeded":               {Kind: BackendKindToolDone, ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallSucceeded}},
		"result failed with error":       {Kind: BackendKindToolDone, ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallFailed, Error: "exit 1"}},
		"text without a call":            {Kind: "text", Detail: "hello"},
	}
	for name, event := range cases {
		t.Run(name, func(t *testing.T) {
			if err := event.ValidateToolCall(); err != nil {
				t.Fatalf("ValidateToolCall() = %v, want nil", err)
			}
		})
	}
}
