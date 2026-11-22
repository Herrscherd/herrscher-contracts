package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ToolCall is one transition of a tool call a backend follows through a turn. ID is
// the backend's identifier of the call, unique within the turn: the started and
// finished transitions of one call carry the same ID. Tool names the tool and is set
// on every transition, so a consumer can pair and label a finished call without its
// start. Detail describes the call in human words (a command, a path, a query) and is
// set on the start only. It is display text that can hold secrets, so an exporter must
// not send it unless the operator opts in. State says which transition this is. Error
// is a short human reason for a failed call and is empty otherwise. It can carry the
// same content as Detail (stderr, a command, a token), so an exporter must not send it
// unless the operator opts in, and an OpenTelemetry error.type takes the closed value
// tool_failed instead of its text. At is when the host received the transition:
// backends leave it zero and the host stamps it, and it is what travels on Event and
// in the journal.
type ToolCall struct {
	ID     string        `json:"id"`
	Tool   string        `json:"tool"`
	Detail string        `json:"detail,omitempty"`
	State  ToolCallState `json:"state"`
	At     time.Time     `json:"at,omitzero"`
	Error  string        `json:"error,omitempty"`
}

// ErrInvalidToolCall is returned for a tool call that lacks its id, its tool or its
// state, or that carries an error on a state other than failed. Match it with errors.Is.
var ErrInvalidToolCall = errors.New("invalid tool call")

// Validate reports whether the call is well formed: an ID, a Tool and a known State
// are required, and Error is allowed only on a failed call. It returns an error
// matching [ErrInvalidToolCall] otherwise.
func (c ToolCall) Validate() error {
	switch {
	case c.ID == "":
		return fmt.Errorf("%w: empty id", ErrInvalidToolCall)
	case c.Tool == "":
		return fmt.Errorf("%w: empty tool", ErrInvalidToolCall)
	case !c.State.isKnown():
		return fmt.Errorf("%w: state %v", ErrInvalidToolCall, c.State)
	case c.Error != "" && c.State != ToolCallFailed:
		return fmt.Errorf("%w: error on a %v call", ErrInvalidToolCall, c.State)
	}
	return nil
}

// MarshalJSON writes the call as its wire form, refusing it unless [ToolCall.Validate]
// accepts it. A refusal matches [ErrInvalidToolCall].
func (c ToolCall) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(toolCallWire(c))
}

// UnmarshalJSON decodes a call and refuses it unless [ToolCall.Validate] accepts it.
// Every refusal matches [ErrInvalidToolCall].
func (c *ToolCall) UnmarshalJSON(data []byte) error {
	var decoded toolCallWire
	if err := json.Unmarshal(data, &decoded); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidToolCall, err)
	}
	parsed := ToolCall(decoded)
	if err := parsed.Validate(); err != nil {
		return err
	}
	*c = parsed
	return nil
}

type toolCallWire ToolCall

// ToolCallState says where a tool call stands. It is a closed set whose zero value is
// not a state: [ParseToolCallState] never returns it, and the zero value does not
// marshal.
type ToolCallState int

const (
	// ToolCallStarted is a call the backend has begun and whose result has not come back.
	ToolCallStarted ToolCallState = iota + 1
	// ToolCallSucceeded is a call whose result came back without an error.
	ToolCallSucceeded
	// ToolCallFailed is a call whose result came back as an error.
	ToolCallFailed
)

// ToolCallStates lists every state, in declaration order.
func ToolCallStates() []ToolCallState {
	return []ToolCallState{ToolCallStarted, ToolCallSucceeded, ToolCallFailed}
}

// String renders the state as its text form, the one [ParseToolCallState] reads back.
// A value that names no state prints its raw number.
func (s ToolCallState) String() string {
	switch s {
	case ToolCallStarted:
		return "started"
	case ToolCallSucceeded:
		return "succeeded"
	case ToolCallFailed:
		return "failed"
	}
	return fmt.Sprintf("ToolCallState(%d)", int(s))
}

func (s ToolCallState) isKnown() bool {
	return s == ToolCallStarted || s == ToolCallSucceeded || s == ToolCallFailed
}

// Finished reports whether the state ends the call: true for [ToolCallSucceeded] and
// [ToolCallFailed], false for [ToolCallStarted] and for a value that names no state.
func (s ToolCallState) Finished() bool {
	return s == ToolCallSucceeded || s == ToolCallFailed
}

// ErrUnknownToolCallState is returned for text, or a value, that names no tool call
// state. Match it with errors.Is.
var ErrUnknownToolCallState = errors.New("unknown tool call state")

// ParseToolCallState reads a state from the text form [ToolCallState.String] writes.
// Any other text returns an error matching [ErrUnknownToolCallState].
func ParseToolCallState(text string) (ToolCallState, error) {
	for _, state := range ToolCallStates() {
		if state.String() == text {
			return state, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownToolCallState, text)
}

// MarshalText writes the text form of the state. A value that names no state returns
// an error matching [ErrUnknownToolCallState].
func (s ToolCallState) MarshalText() ([]byte, error) {
	if !s.isKnown() {
		return nil, fmt.Errorf("%w: %d", ErrUnknownToolCallState, int(s))
	}
	return []byte(s.String()), nil
}

// UnmarshalText reads the text form of a state, as [ParseToolCallState] does.
func (s *ToolCallState) UnmarshalText(text []byte) error {
	parsed, err := ParseToolCallState(string(text))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// ValidateToolCall reports whether the event's tool call follows the rules of its kind.
// A [BackendKindTool] event may carry a started call whose Tool equals the event's Tool;
// a [BackendKindToolDone] event must carry a finished call, with Tool and Detail empty on
// the event and no Detail on the call; no other kind carries a call. A [BackendKindTool]
// event without a call stays valid, for backends that cannot follow a call. Neither kind
// may carry a stamped At, which the host sets. Every refusal matches [ErrInvalidToolCall].
func (e BackendEvent) ValidateToolCall() error {
	if e.ToolCall == nil {
		if e.Kind == BackendKindToolDone {
			return fmt.Errorf("%w: %s without a call", ErrInvalidToolCall, BackendKindToolDone)
		}
		return nil
	}
	if err := e.ToolCall.Validate(); err != nil {
		return err
	}
	switch e.Kind {
	case BackendKindTool:
		return e.validateToolCallStart()
	case BackendKindToolDone:
		return e.validateToolCallResult()
	}
	return fmt.Errorf("%w: kind %q carries a call", ErrInvalidToolCall, e.Kind)
}

func (e BackendEvent) validateToolCallStart() error {
	call := e.ToolCall
	switch {
	case call.State != ToolCallStarted:
		return fmt.Errorf("%w: %s carries a %v call", ErrInvalidToolCall, BackendKindTool, call.State)
	case e.Tool != call.Tool:
		return fmt.Errorf("%w: event tool %q differs from call tool %q", ErrInvalidToolCall, e.Tool, call.Tool)
	case !call.At.IsZero():
		return fmt.Errorf("%w: %s carries a stamped At", ErrInvalidToolCall, BackendKindTool)
	}
	return nil
}

func (e BackendEvent) validateToolCallResult() error {
	call := e.ToolCall
	switch {
	case !call.State.Finished():
		return fmt.Errorf("%w: %s carries a %v call", ErrInvalidToolCall, BackendKindToolDone, call.State)
	case e.Tool != "":
		return fmt.Errorf("%w: %s carries tool %q", ErrInvalidToolCall, BackendKindToolDone, e.Tool)
	case e.Detail != "":
		return fmt.Errorf("%w: %s carries a detail", ErrInvalidToolCall, BackendKindToolDone)
	case call.Detail != "":
		return fmt.Errorf("%w: %s call carries a detail", ErrInvalidToolCall, BackendKindToolDone)
	case !call.At.IsZero():
		return fmt.Errorf("%w: %s carries a stamped At", ErrInvalidToolCall, BackendKindToolDone)
	}
	return nil
}
