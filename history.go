package contracts

import (
	"context"
	"errors"
	"fmt"
)

// HistoryEditor is an OPTIONAL capability of a live backend instance: it reads the
// conversation the model sees on its next turn, and prunes tool calls from it in
// place. A compaction stage uses it to shorten a conversation without a summary:
// what it keeps stays verbatim. The host type-asserts for it, and the manifest says
// so ahead of time through Capabilities.HistoryEditor.
//
// The host calls both methods only while it holds the turn of the session, under
// the same exclusion as Compact. PruneHistory applies every edit or none: on an
// error the conversation is left as it was. Text the operator or the agent wrote is
// never an edit target, only tool calls and their results are.
type HistoryEditor interface {
	ReadHistory(ctx context.Context) (History, error)
	PruneHistory(ctx context.Context, edits []HistoryEdit) (PruneOutcome, error)
}

// History is the conversation of one session, oldest message first, from the
// point the model sees it: a conversation compacted earlier starts after that
// compaction.
type History struct {
	Messages []HistoryMessage
}

// Calls returns every tool call of the history, in order.
func (h History) Calls() []HistoryCall {
	var calls []HistoryCall
	for index := range h.Messages {
		calls = append(calls, h.Messages[index].Calls...)
	}
	return calls
}

// HistoryMessage is one message of a history. Text is what the message says
// besides its tool calls, and Calls are the tool calls an assistant message made,
// each with its result. A user message carries no calls.
type HistoryMessage struct {
	Role  HistoryRole
	Text  string
	Calls []HistoryCall
}

// HistoryCall is one tool call and its result. ID is the backend's identifier of
// the call, unique in the history, the one a [HistoryEdit] names. Input is the call's input as JSON text.
// Result is empty when the call has no result yet.
type HistoryCall struct {
	ID            string
	Tool          string
	Input         string
	Result        string
	ResultIsError bool
}

// HistoryRole says who wrote a message. It is a closed set whose zero value is not
// a role: [ParseHistoryRole] never returns it, and the zero value does not marshal.
type HistoryRole int

const (
	// RoleUser is a message from the operator, or a tool result the backend feeds
	// back as one.
	RoleUser HistoryRole = iota + 1
	// RoleAssistant is a message from the model.
	RoleAssistant
)

// HistoryRoles lists every role, in declaration order.
func HistoryRoles() []HistoryRole {
	return []HistoryRole{RoleUser, RoleAssistant}
}

// String renders the role as its text form, the one [ParseHistoryRole] reads back.
// A value that names no role prints its raw number.
func (r HistoryRole) String() string {
	switch r {
	case RoleUser:
		return "user"
	case RoleAssistant:
		return "assistant"
	}
	return fmt.Sprintf("HistoryRole(%d)", int(r))
}

func (r HistoryRole) isKnown() bool {
	return r == RoleUser || r == RoleAssistant
}

// ErrUnknownHistoryRole is returned for text, or a value, that names no role. Match
// it with errors.Is.
var ErrUnknownHistoryRole = errors.New("unknown history role")

// ParseHistoryRole reads a role from the text form [HistoryRole.String] writes. Any
// other text returns an error matching [ErrUnknownHistoryRole].
func ParseHistoryRole(text string) (HistoryRole, error) {
	for _, role := range HistoryRoles() {
		if role.String() == text {
			return role, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownHistoryRole, text)
}

// MarshalText writes the text form of the role. A value that names no role returns
// an error matching [ErrUnknownHistoryRole].
func (r HistoryRole) MarshalText() ([]byte, error) {
	if !r.isKnown() {
		return nil, fmt.Errorf("%w: %d", ErrUnknownHistoryRole, int(r))
	}
	return []byte(r.String()), nil
}

// UnmarshalText reads the text form of a role, as [ParseHistoryRole] does.
func (r *HistoryRole) UnmarshalText(text []byte) error {
	parsed, err := ParseHistoryRole(string(text))
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}

// PruneAction is what a [HistoryEdit] does to a tool call. It is a closed set whose
// zero value is not an action: [ParsePruneAction] never returns it.
type PruneAction int

const (
	// PruneResult keeps the call and the first KeepChars runes of its result,
	// followed by a note that the rest was pruned.
	PruneResult PruneAction = iota + 1
	// PruneCall keeps only a trace of the call: its result becomes a note, and the
	// backend may shorten its input. The model can still tell that the call was made.
	PruneCall
)

// PruneActions lists every action, in declaration order.
func PruneActions() []PruneAction {
	return []PruneAction{PruneResult, PruneCall}
}

// String renders the action as its text form, the one [ParsePruneAction] reads
// back. A value that names no action prints its raw number.
func (a PruneAction) String() string {
	switch a {
	case PruneResult:
		return "prune_result"
	case PruneCall:
		return "prune_call"
	}
	return fmt.Sprintf("PruneAction(%d)", int(a))
}

func (a PruneAction) isKnown() bool {
	return a == PruneResult || a == PruneCall
}

// ErrUnknownPruneAction is returned for text, or a value, that names no action.
// Match it with errors.Is.
var ErrUnknownPruneAction = errors.New("unknown prune action")

// ParsePruneAction reads an action from the text form [PruneAction.String] writes.
// Any other text returns an error matching [ErrUnknownPruneAction].
func ParsePruneAction(text string) (PruneAction, error) {
	for _, action := range PruneActions() {
		if action.String() == text {
			return action, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownPruneAction, text)
}

// MarshalText writes the text form of the action. A value that names no action
// returns an error matching [ErrUnknownPruneAction].
func (a PruneAction) MarshalText() ([]byte, error) {
	if !a.isKnown() {
		return nil, fmt.Errorf("%w: %d", ErrUnknownPruneAction, int(a))
	}
	return []byte(a.String()), nil
}

// UnmarshalText reads the text form of an action, as [ParsePruneAction] does.
func (a *PruneAction) UnmarshalText(text []byte) error {
	parsed, err := ParsePruneAction(string(text))
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}

// HistoryEdit is one pruning of one tool call. KeepChars is the number of runes
// (Unicode code points) of the result kept verbatim: PruneResult reads it,
// PruneCall ignores it, and it is never negative.
type HistoryEdit struct {
	CallID    string
	Action    PruneAction
	KeepChars int
}

// PruneOutcome is what [HistoryEditor.PruneHistory] reports. Edited counts the
// calls it changed. CharsBefore and CharsAfter count the runes of tool
// inputs and results in the whole history, before and after the edits, so a
// stage can judge what the pruning saved without a tokenizer.
type PruneOutcome struct {
	Edited      int
	CharsBefore int
	CharsAfter  int
}

// ErrInvalidHistoryEdit is returned by [ValidateHistoryEdits] for an edit that
// names no call, names a call twice, has an unknown action or a negative
// KeepChars. Match it with errors.Is.
var ErrInvalidHistoryEdit = errors.New("invalid history edit")

// ErrUnknownHistoryCall is returned by a [HistoryEditor] for an edit whose CallID
// is not a call of the history. Match it with errors.Is.
var ErrUnknownHistoryCall = errors.New("unknown history call")

// ValidateHistoryEdits checks edits before they are applied: each names a call
// once, with a known action and a KeepChars that is not negative. A
// [HistoryEditor] calls it before touching the history, and every failure matches
// [ErrInvalidHistoryEdit].
func ValidateHistoryEdits(edits []HistoryEdit) error {
	seen := make(map[string]bool, len(edits))
	for _, edit := range edits {
		switch {
		case edit.CallID == "":
			return fmt.Errorf("%w: an edit names no call", ErrInvalidHistoryEdit)
		case seen[edit.CallID]:
			return fmt.Errorf("%w: call %q edited twice", ErrInvalidHistoryEdit, edit.CallID)
		case edit.KeepChars < 0:
			return fmt.Errorf("%w: call %q keeps %d characters", ErrInvalidHistoryEdit, edit.CallID, edit.KeepChars)
		}
		if !edit.Action.isKnown() {
			return fmt.Errorf("%w: call %q: %w: %d", ErrInvalidHistoryEdit, edit.CallID, ErrUnknownPruneAction, int(edit.Action))
		}
		seen[edit.CallID] = true
	}
	return nil
}
