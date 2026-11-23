package contracts

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestToolCallStateParsesItsOwnText(t *testing.T) {
	for _, want := range ToolCallStates() {
		got, err := ParseToolCallState(want.String())
		if err != nil || got != want {
			t.Errorf("ParseToolCallState(%q) = %v, %v, want %v", want.String(), got, err, want)
		}
	}
}

func TestToolCallStateRoundTripsThroughJSON(t *testing.T) {
	for _, want := range ToolCallStates() {
		raw, err := json.Marshal(want)
		if err != nil {
			t.Errorf("Marshal(%v): %v", want, err)
			continue
		}
		var got ToolCallState
		if err := json.Unmarshal(raw, &got); err != nil || got != want {
			t.Errorf("Unmarshal(%s) = %v, %v, want %v", raw, got, err, want)
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
			t.Errorf("%v.Finished() = %v, want %v", state, got, want)
		}
	}
}

func invalidToolCalls() map[string]ToolCall {
	return map[string]ToolCall{
		"zero state":                {ID: "c1", Tool: "Bash"},
		"empty id":                  {Tool: "Bash", State: ToolCallStarted},
		"empty tool":                {ID: "c1", State: ToolCallSucceeded},
		"error on a started call":   {ID: "c1", Tool: "Bash", State: ToolCallStarted, Error: "exit 1"},
		"error on a succeeded call": {ID: "c1", Tool: "Bash", State: ToolCallSucceeded, Error: "exit 1"},
		"succeeded with detail":     {ID: "c1", Tool: "Bash", Detail: "ls", State: ToolCallSucceeded},
		"failed with detail":        {ID: "c1", Tool: "Bash", Detail: "ls", State: ToolCallFailed, Error: "exit 1"},
		"unknown state value":       {ID: "c1", Tool: "Bash", State: ToolCallState(42)},
		"sub-minute offset":         {ID: "c1", Tool: "Bash", State: ToolCallStarted, At: time.Date(2026, 11, 22, 10, 0, 0, 0, time.FixedZone("lmt", 19*60+32))},
		"year 10000":                {ID: "c1", Tool: "Bash", State: ToolCallStarted, At: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)},
		"year -1":                   {ID: "c1", Tool: "Bash", State: ToolCallStarted, At: time.Date(-1, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
}

func TestToolCallValidateRefusesMalformedCalls(t *testing.T) {
	for name, call := range invalidToolCalls() {
		if err := call.Validate(); !errors.Is(err, ErrInvalidToolCall) {
			t.Errorf("%s: Validate() = %v, want ErrInvalidToolCall", name, err)
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
		err := json.Unmarshal([]byte(raw), &call)
		if !errors.Is(err, ErrInvalidToolCall) {
			t.Errorf("%s: Unmarshal(%s) error = %v, want ErrInvalidToolCall", name, raw, err)
			continue
		}
		if strings.Contains(err.Error(), "ToolCallState") {
			t.Errorf("%s: Unmarshal(%s) error = %v, names the Go type ToolCallState", name, raw, err)
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
	for name, call := range invalidToolCalls() {
		t.Run(name, func(t *testing.T) {
			_, err := json.Marshal(Event{T: "status", ToolCall: &call})
			if !errors.Is(err, ErrInvalidToolCall) {
				t.Fatalf("Marshal(Event with %s) error = %v, want ErrInvalidToolCall", name, err)
			}
		})
	}
}

func TestEventMarshalWrapsToolCallRefusalInMarshalerError(t *testing.T) {
	_, err := json.Marshal(Event{T: "status", ToolCall: &ToolCall{Tool: "Bash", State: ToolCallStarted}})
	if !errors.Is(err, ErrInvalidToolCall) {
		t.Fatalf("Marshal(Event with empty id) error = %v, want ErrInvalidToolCall", err)
	}
	if _, ok := errors.AsType[*json.MarshalerError](err); !ok {
		t.Fatalf("Marshal(Event with empty id) error = %v, want a *json.MarshalerError", err)
	}
}

func TestToolCallMarshalDirectRefusalNamesNoGoType(t *testing.T) {
	call := ToolCall{Tool: "Bash", State: ToolCallStarted}
	_, err := call.MarshalJSON()
	if !errors.Is(err, ErrInvalidToolCall) {
		t.Fatalf("MarshalJSON(empty id) error = %v, want ErrInvalidToolCall", err)
	}
	if strings.Contains(err.Error(), "contracts.") {
		t.Fatalf("MarshalJSON(empty id) error = %q, want no Go type name", err)
	}
}

func TestToolCallMarshalRoundTripsValidCalls(t *testing.T) {
	at := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	cases := map[string]ToolCall{
		"started":                            {ID: "c1", Tool: "Bash", Detail: "ls", State: ToolCallStarted},
		"succeeded":                          {ID: "c1", Tool: "Bash", State: ToolCallSucceeded},
		"failed with error":                  {ID: "c1", Tool: "Bash", State: ToolCallFailed, Error: "exit 1"},
		"failed with stamped at":             {ID: "c1", Tool: "Bash", State: ToolCallFailed, At: at, Error: "exit 1"},
		"stamped at +02:00 with nanoseconds": {ID: "c1", Tool: "Bash", State: ToolCallFailed, At: time.Date(2026, 11, 22, 12, 0, 0, 123456789, time.FixedZone("cest", 2*60*60)), Error: "exit 1"},
		"stamped at -05:30":                  {ID: "c1", Tool: "Bash", State: ToolCallFailed, At: time.Date(2026, 11, 22, 4, 30, 0, 0, time.FixedZone("ist", -(5*60+30)*60)), Error: "exit 1"},
		"year 0":                             {ID: "c1", Tool: "Bash", State: ToolCallFailed, At: time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC), Error: "exit 1"},
		"year 9999":                          {ID: "c1", Tool: "Bash", State: ToolCallFailed, At: time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC), Error: "exit 1"},
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
	_, aOffset := a.At.Zone()
	_, bOffset := b.At.Zone()
	return a.ID == b.ID && a.Tool == b.Tool && a.Detail == b.Detail && a.State == b.State &&
		a.Error == b.Error && a.At.Equal(b.At) && aOffset == bOffset
}

func unmarshalErr(raw string) error {
	var call ToolCall
	return json.Unmarshal([]byte(raw), &call)
}

func TestToolCallRefusalsDoNotEchoTheirInput(t *testing.T) {
	const secret = "token=sk-live-7f3a9c"
	started := ToolCall{ID: "c1", Tool: "Bash", Detail: secret, State: ToolCallStarted}
	refusals := map[string]error{
		"event detail differs from call": BackendEvent{Kind: BackendKindTool, Tool: "Bash", Detail: "ls", ToolCall: &started}.ValidateToolCall(),
		"detail on a finished call":      ToolCall{ID: "c1", Tool: "Bash", Detail: secret, State: ToolCallSucceeded}.Validate(),
		"tool_done carrying a detail":    BackendEvent{Kind: BackendKindToolDone, Detail: secret, ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallSucceeded}}.ValidateToolCall(),
		"unmarshal of a finished detail": unmarshalErr(`{"id":"c1","tool":"Bash","state":"succeeded","detail":"` + secret + `"}`),
		"tool on a start":                BackendEvent{Kind: BackendKindTool, Tool: secret, ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallStarted}}.ValidateToolCall(),
		"tool on tool_done":              BackendEvent{Kind: BackendKindToolDone, Tool: secret, ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallSucceeded}}.ValidateToolCall(),
		"error on a started call":        ToolCall{ID: "c1", Tool: "Bash", State: ToolCallStarted, Error: secret}.Validate(),
		"kind of the event":              BackendEvent{Kind: secret, ToolCall: &ToolCall{ID: "c1", Tool: "Bash", State: ToolCallStarted}}.ValidateToolCall(),
	}
	for name, err := range refusals {
		t.Run(name, func(t *testing.T) {
			requireRefusalWithoutEcho(t, err, ErrInvalidToolCall, secret)
		})
	}
	wire := map[string]struct {
		raw  string
		want []error
	}{
		"secret in state": {
			raw:  `{"id":"c1","tool":"Bash","state":"` + secret + `"}`,
			want: []error{ErrInvalidToolCall, ErrUnknownToolCallState},
		},
		"secret in at": {
			raw:  `{"id":"c1","tool":"Bash","state":"started","at":"` + secret + `"}`,
			want: []error{ErrInvalidToolCall},
		},
	}
	for name, tc := range wire {
		t.Run(name, func(t *testing.T) {
			var call ToolCall
			err := json.Unmarshal([]byte(tc.raw), &call)
			for _, want := range tc.want {
				if !errors.Is(err, want) {
					t.Fatalf("Unmarshal(%s) error = %v, want %v", tc.raw, err, want)
				}
			}
			requireNoSecretFragment(t, err.Error(), secret)
		})
	}
	_, err := ParseToolCallState(secret)
	requireBareSentinel(t, err, ErrUnknownToolCallState)
}
