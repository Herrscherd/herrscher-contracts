package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ToolCall is one transition of a tool call a backend follows through a turn.
// ID is the backend's identifier of the call, unique within one execution of
// the backend: the started and finished transitions of one call carry the same
// ID. A reset opens a new execution within the turn, so the ID of a call may
// come back after one. Tool names the tool and is set on every transition, so
// a consumer can pair and label a
// finished call without its start. Detail describes the call in human words (a
// command, a path, a query) and is set on the start only. It is display text
// that can hold secrets, so an exporter must not send it unless the operator
// opts in. State says which transition this is. Error is a short human reason
// for a failed call and is empty otherwise. It can carry the same content as
// Detail (stderr, a command, a token), so an exporter must not send it unless
// the operator opts in, and an exporter will map error.type to the closed value
// tool_failed instead of its text. At is when the host received the transition:
// backends leave it zero and the host stamps it, and it is what travels on
// Event and in the journal. Its zone offset must be a whole number of minutes,
// since the wire form writes no seconds, and its year must be from 0 to 9999,
// the years the wire form writes. The zero value is not a valid call: its ID is
// empty and its State is zero, and [ToolCall.Validate] refuses it.
type ToolCall struct {
	ID     string        `json:"id"`
	Tool   string        `json:"tool"`
	Detail string        `json:"detail,omitempty"`
	State  ToolCallState `json:"state"`
	At     time.Time     `json:"at,omitzero"`
	Error  string        `json:"error,omitempty"`
}

// ErrInvalidToolCall is returned for every refusal of a tool call. A call is
// refused when it breaks a ToolCall rule: no id, no tool, no known state, an
// error on a call that did not fail, a detail on a finished call, an At whose
// year is outside 0 to 9999, or an At whose zone offset has seconds. A
// BackendEvent is refused by [BackendEvent.ValidateToolCall] when its kind and
// call do not match: a tool_done event without a call, a call whose state does
// not fit its kind (not started on a tool event, not finished on a tool_done
// event), a tool event whose Tool differs from its call's Tool, a tool event
// whose Detail differs from its call's Detail, a tool_done event that carries
// Tool or Detail on the event itself, a kind other than tool or tool_done that
// carries a call, or a call that carries a stamped At. A call is also refused
// when it is decoded, and every such refusal matches ErrInvalidToolCall: a
// field of the wrong JSON type, a state text that names no state (which also
// matches ErrUnknownToolCallState), or an at that is not an RFC 3339 time. For
// a field of the wrong JSON type and for an at that is not an RFC 3339 time, no
// other cause is wrapped: the message names the field, and never the value. A
// document that is not valid JSON is not a refused call: encoding/json reports
// it as a *json.SyntaxError before any UnmarshalJSON runs, and that error does
// not match ErrInvalidToolCall. No refusal echoes the input it refuses: its
// message names the rule that failed or the field that has the wrong JSON type,
// never the text of detail, error, state, at, tool or kind, so a secret in one
// of them stays out of logs. Two exceptions apply. The first is
// *json.SyntaxError, which may quote a character of the input, because it is
// not a refusal of the contract: a logger that must keep the input out logs
// only the type of the error. The second is the *json.MarshalerError that
// json.Marshal wraps around a refusal of ToolCall.MarshalJSON, whose message
// names the Go type *contracts.ToolCall. errors.Is still matches
// ErrInvalidToolCall through that wrap. A direct call to ToolCall.MarshalJSON
// or to ToolCall.Validate gives the message without it. Match a refusal with
// errors.Is.
var ErrInvalidToolCall = errors.New("invalid tool call")

// Validate reports whether the call is well formed: an ID, a Tool and a known
// State are required, Error is allowed only on a failed call, and Detail is
// refused on a finished call. A non-zero At must have a zone offset of whole
// minutes, because the wire form writes no seconds and would drop them, and a
// year from 0 to 9999, the years the wire form writes. It returns an error
// matching [ErrInvalidToolCall] otherwise.
func (c ToolCall) Validate() error {
	switch {
	case c.ID == "":
		return fmt.Errorf("%w: empty id", ErrInvalidToolCall)
	case c.Tool == "":
		return fmt.Errorf("%w: empty tool", ErrInvalidToolCall)
	case c.State == 0:
		return fmt.Errorf("%w: state is missing", ErrInvalidToolCall)
	case !c.State.isKnown():
		return fmt.Errorf("%w: state is not a known state", ErrInvalidToolCall)
	case c.Error != "" && c.State != ToolCallFailed:
		return fmt.Errorf("%w: error on a %v call", ErrInvalidToolCall, c.State)
	case c.Detail != "" && c.State.Finished():
		return fmt.Errorf("%w: detail on a %v call", ErrInvalidToolCall, c.State)
	case c.hasSecondsOffset():
		return fmt.Errorf("%w: at has an offset with seconds", ErrInvalidToolCall)
	case c.hasYearOutOfRange():
		return fmt.Errorf("%w: at has a year outside 0 to 9999", ErrInvalidToolCall)
	}
	return nil
}

func (c ToolCall) hasSecondsOffset() bool {
	if c.At.IsZero() {
		return false
	}
	_, offset := c.At.Zone()
	return offset%60 != 0
}

func (c ToolCall) hasYearOutOfRange() bool {
	year := c.At.Year()
	return year < 0 || year > 9999
}

// MarshalJSON writes the call as its wire form, refusing it unless
// [ToolCall.Validate] accepts it. A refusal matches [ErrInvalidToolCall].
func (c ToolCall) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(toolCallWire(c))
}

// UnmarshalJSON decodes a call and refuses it unless [ToolCall.Validate]
// accepts it. Every refusal matches [ErrInvalidToolCall]. A document that is
// not valid JSON is not a refusal: its *json.SyntaxError is returned unchanged,
// and it may quote a character of the input (see [ErrInvalidToolCall]).
func (c *ToolCall) UnmarshalJSON(data []byte) error {
	var decoded toolCallInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		if _, ok := errors.AsType[*json.SyntaxError](err); ok {
			return err
		}
		if typeErr, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
			return wrongJSONTypeError(typeErr)
		}
		return fmt.Errorf("%w: %w", ErrInvalidToolCall, err)
	}
	parsed := ToolCall(decoded.toolCallWire)
	if decoded.At != nil {
		receivedAt, err := time.Parse(time.RFC3339Nano, *decoded.At)
		if err != nil {
			return fmt.Errorf("%w: at is not an RFC 3339 time", ErrInvalidToolCall)
		}
		parsed.At = receivedAt
	}
	if err := parsed.Validate(); err != nil {
		return err
	}
	*c = parsed
	return nil
}

func wrongJSONTypeError(err *json.UnmarshalTypeError) error {
	field, known := canonicalWireField(err.Field)
	if !known {
		return fmt.Errorf("%w: the call has the wrong JSON type", ErrInvalidToolCall)
	}
	return fmt.Errorf("%w: %s has the wrong JSON type", ErrInvalidToolCall, field)
}

func canonicalWireField(name string) (string, bool) {
	for _, field := range []string{"id", "tool", "detail", "state", "error", "at"} {
		if strings.EqualFold(field, name) {
			return field, true
		}
	}
	return "", false
}

type toolCallWire ToolCall

type toolCallInput struct {
	toolCallWire

	At *string `json:"at"`
}

// ToolCallState says where a tool call stands. It is a closed set whose zero
// value is not a state: [ParseToolCallState] never returns it, and the zero
// value does not marshal.
type ToolCallState int

const (
	// ToolCallStarted is a call the backend has begun and whose result has not
	// come back.
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

// String renders the state as its text form, the one [ParseToolCallState] reads
// back. A value that names no state prints as ToolCallState(N), where N is its
// number, such as ToolCallState(42).
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

// Finished reports whether the state ends the call: true for
// [ToolCallSucceeded] and [ToolCallFailed], false for [ToolCallStarted] and for
// a value that names no state.
func (s ToolCallState) Finished() bool {
	return s == ToolCallSucceeded || s == ToolCallFailed
}

// ErrUnknownToolCallState is returned for text, or a value, that names no tool
// call state. Match it with errors.Is.
var ErrUnknownToolCallState = errors.New("unknown tool call state")

// ParseToolCallState reads a state from the text form [ToolCallState.String]
// writes. Any other text returns an error matching [ErrUnknownToolCallState],
// whose message does not echo the text.
func ParseToolCallState(text string) (ToolCallState, error) {
	for _, state := range ToolCallStates() {
		if state.String() == text {
			return state, nil
		}
	}
	return 0, ErrUnknownToolCallState
}

// MarshalText writes the text form of the state. A value that names no state
// returns an error matching [ErrUnknownToolCallState].
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

// ValidateToolCall reports whether the event's tool call follows the rules of
// its kind. A [BackendKindTool] event may carry a started call whose Tool and
// Detail equal the event's Tool and Detail; a [BackendKindToolDone] event must
// carry a finished call, with Tool and Detail empty on the event and no Detail
// on the call; no other kind carries a call. A [BackendKindTool] event without
// a call stays valid, for backends that cannot follow a call. Neither kind may
// carry a stamped At, which the host sets. Every refusal matches
// [ErrInvalidToolCall].
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
	return fmt.Errorf("%w: only %s and %s events carry a call", ErrInvalidToolCall, BackendKindTool, BackendKindToolDone)
}

func (e BackendEvent) validateToolCallStart() error {
	call := e.ToolCall
	switch {
	case call.State != ToolCallStarted:
		return fmt.Errorf("%w: %s carries a %v call", ErrInvalidToolCall, BackendKindTool, call.State)
	case e.Tool != call.Tool:
		return fmt.Errorf("%w: event tool differs from call tool", ErrInvalidToolCall)
	case e.Detail != call.Detail:
		return fmt.Errorf("%w: event detail differs from call detail", ErrInvalidToolCall)
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
		return fmt.Errorf("%w: %s carries a tool", ErrInvalidToolCall, BackendKindToolDone)
	case e.Detail != "":
		return fmt.Errorf("%w: %s carries a detail", ErrInvalidToolCall, BackendKindToolDone)
	case !call.At.IsZero():
		return fmt.Errorf("%w: %s carries a stamped At", ErrInvalidToolCall, BackendKindToolDone)
	}
	return nil
}
