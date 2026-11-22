package contracts

import (
	"errors"
	"fmt"
	"time"
)

// ToolCall is one transition of a tool call a backend follows through a turn. ID is
// the backend's identifier of the call, unique within the turn: the started and
// finished transitions of one call carry the same ID. Tool names the tool and Detail
// describes the call in human words. State says which transition this is. At is when
// the backend observed the transition, and its zero value means the host stamps it
// on receipt.
type ToolCall struct {
	ID     string        `json:"id"`
	Tool   string        `json:"tool"`
	Detail string        `json:"detail,omitempty"`
	State  ToolCallState `json:"state"`
	At     time.Time     `json:"at,omitzero"`
}

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
