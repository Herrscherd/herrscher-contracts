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
//
// The host calls Compact only while it holds the turn of the session, under the
// same exclusion as Respond: never in parallel with a Respond, nor with another
// compaction of the same session. A backend may therefore use its persistent
// process without a lock of its own. Progress goes through onEvent, which the host
// always passes non-nil, as for Respond. The host renders the outcome to the
// gateways (the note, the tokens before and after), not the stage. A backend that
// does not implement Compactor has no default stage, and the host must refuse the
// compaction before sending anything.
type Compactor interface {
	Compact(ctx context.Context, req CompactRequest, onEvent func(BackendEvent)) (CompactResult, error)
}

// CompactTrigger says why a compaction runs. It is a closed set whose zero value
// is not a trigger: [ParseCompactTrigger] never returns it, and the zero value
// does not marshal.
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

// ErrUnknownCompactTrigger is returned for text, or a value, that names no
// trigger. Match it with errors.Is.
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

// MarshalText writes the text form of the trigger. A value that names no trigger
// returns an error matching [ErrUnknownCompactTrigger], so it never reaches the
// wire.
func (t CompactTrigger) MarshalText() ([]byte, error) {
	if !t.isKnown() {
		return nil, fmt.Errorf("%w: %d", ErrUnknownCompactTrigger, int(t))
	}
	return []byte(t.String()), nil
}

// UnmarshalText reads the text form of a trigger, as [ParseCompactTrigger] does.
func (t *CompactTrigger) UnmarshalText(text []byte) error {
	parsed, err := ParseCompactTrigger(string(text))
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

func (t CompactTrigger) isKnown() bool {
	for _, trigger := range CompactTriggers() {
		if trigger == t {
			return true
		}
	}
	return false
}

// CompactRequest is one compaction a stage is asked to run.
//
// TokensUsed and TokenLimit describe the context window at the moment this stage
// runs. A stage that follows another one receives the usage that one left, so
// they are not the usage at the start of the stack. They are read only when
// UsageMeasured is true: without it their zero values say nothing, and a stage
// must not divide by TokenLimit. UsageMeasured true guarantees that TokensUsed is
// measured and that TokenLimit is greater than zero.
type CompactRequest struct {
	Session SessionInfo
	Trigger CompactTrigger
	// UsageMeasured is true when TokensUsed and TokenLimit come from a measured
	// context window. RunCompactStack normalises it on entry, then sets it to
	// whether the request has a TokenLimit once a stage measures its tokens.
	UsageMeasured bool
	TokensUsed    int
	TokenLimit    int
	// Instruction is the free text the operator gave, as in "/compact <instruction>".
	// It is empty when none was given, and a stage may ignore it.
	Instruction string
	// History edits the conversation of the session, nil when its backend does not
	// implement [HistoryEditor]. The host fills it for the stages it runs itself; it
	// never crosses a wire, and a backend running its own stage reaches its history
	// directly. It is valid only for the duration of the Compact call.
	History HistoryEditor
}

// CompactResult is what a stage reports once it has run.
//
// TokensAfter is meaningful only when TokensMeasured is true. A stage that cannot
// count its tokens leaves TokensMeasured false, and the zero TokensAfter then says
// nothing: it is never read as "no tokens left". Note is one short line for the
// operator, empty when there is nothing to say. Done stops the stack after this
// stage, for a stage that judges the context compacted enough.
//
// The host reads only Note as text: its compaction frame carries Note and the token
// counts, and no Notice, so Note is the text the operator sees. Notice is the
// message built from a code, which the host does not read yet. Until the host reads
// it, a producer sets both, with Note in English, so that no note is lost when the
// host moves to Notice.
type CompactResult struct {
	TokensAfter    int
	TokensMeasured bool
	// Note is the English text the host shows today. It is the only note the host
	// reads.
	Note string
	Done bool
	// Notice is the coded message the host will read once it reads codes.
	Notice Notice
	// Stats is what the stage measured for the usage report, nil when it measured
	// nothing.
	Stats *CompactStats
	// StageStats is set by RunCompactStack on the result it returns: the Stats of
	// every stage that ran and reported some, in order. A stage's own result leaves
	// it nil.
	StageStats []CompactStats
}

// ErrAutoCompactPercentOutOfRange is returned by [ValidateAutoCompactPercent] for a
// percentage outside 0 to 100. Match it with errors.Is.
var ErrAutoCompactPercentOutOfRange = errors.New("auto compact percent out of range")

// ValidateAutoCompactPercent checks a SessionInfo.AutoCompactPercent: zero means no
// automatic compaction, and 1 to 100 is the share of the window that triggers it.
// Any other value returns an error matching [ErrAutoCompactPercentOutOfRange]. The
// host calls it where it produces the value, before it emits the session. JSON
// decoding does not validate the value.
func ValidateAutoCompactPercent(percent int) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("%w: %d", ErrAutoCompactPercentOutOfRange, percent)
	}
	return nil
}
