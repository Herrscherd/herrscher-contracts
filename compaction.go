package contracts

import (
	"context"
	"errors"
	"fmt"
)

// Compactor is an OPTIONAL capability of a live instance: it shortens the
// conversation of one session so the context window has room again. A backend
// that implements it is the default compaction of its own sessions, and a plugin
// that implements it offers a custom stage, which the host runs in the order it
// configures (see [RunCompactStack]). The host type-asserts for it, and the
// manifest says so ahead of time through Capabilities.Compactor.
type Compactor interface {
	Compact(ctx context.Context, req CompactRequest) (CompactResult, error)
}

// CompactTrigger says why a compaction runs. It is a closed set whose zero value
// is not a trigger: [ParseCompactTrigger] never returns it.
type CompactTrigger int

const (
	// TriggerManual is a compaction the operator asked for, with /compact.
	TriggerManual CompactTrigger = iota + 1
	// TriggerAutomatic is a compaction the host ran because the session reached
	// its AutoCompactPercent.
	TriggerAutomatic
)

// CompactTriggers lists every trigger, in declaration order.
func CompactTriggers() []CompactTrigger {
	return []CompactTrigger{TriggerManual, TriggerAutomatic}
}

// String renders the trigger as its text form, the one [ParseCompactTrigger]
// reads back. A value that names no trigger prints its raw number.
func (t CompactTrigger) String() string {
	switch t {
	case TriggerManual:
		return "manual"
	case TriggerAutomatic:
		return "automatic"
	}
	return fmt.Sprintf("CompactTrigger(%d)", int(t))
}

// ErrUnknownCompactTrigger is returned by [ParseCompactTrigger] for text that names
// no trigger. Match it with errors.Is.
var ErrUnknownCompactTrigger = errors.New("unknown compaction trigger")

// ParseCompactTrigger reads a trigger from the text form [CompactTrigger.String]
// writes. Any other text, the empty string included, returns an error matching
// [ErrUnknownCompactTrigger].
func ParseCompactTrigger(text string) (CompactTrigger, error) {
	for _, trigger := range CompactTriggers() {
		if trigger.String() == text {
			return trigger, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownCompactTrigger, text)
}

// CompactRequest is one compaction a stage is asked to run.
//
// TokensUsed and TokenLimit describe the context window at the moment this stage
// runs. A stage that follows another one receives the usage that one left, so
// they are not the usage at the start of the stack.
type CompactRequest struct {
	Session    SessionInfo
	Trigger    CompactTrigger
	TokensUsed int
	TokenLimit int
	// Instruction is the free text the operator gave, as in "/compact <instruction>".
	// It is empty when none was given, and a stage may ignore it.
	Instruction string
}

// CompactResult is what a stage reports once it has run.
//
// TokensAfter is meaningful only when TokensMeasured is true. A stage that cannot
// count its tokens leaves TokensMeasured false, and the zero TokensAfter then says
// nothing: it is never read as "no tokens left". Note is one short line for the
// operator, empty when there is nothing to say. Done stops the stack after this
// stage, for a stage that judges the context compacted enough.
type CompactResult struct {
	TokensAfter    int
	TokensMeasured bool
	Note           string
	Done           bool
}
