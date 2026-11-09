package contracts

import (
	"errors"
	"fmt"
	"time"
)

// CompactOutcome says how a compaction stage ended. It is a closed set whose zero
// value is not an outcome: [ParseCompactOutcome] never returns it, and the zero
// value does not marshal.
type CompactOutcome int

const (
	// CompactDone is a stage that compacted the context enough to stop the stack.
	CompactDone CompactOutcome = iota + 1
	// CompactShort is a stage that shortened the context without stopping the stack.
	CompactShort
	// CompactFailed is a stage that ran and could not compact the context.
	CompactFailed
	// CompactDryRun is a stage that measured what it would do and changed nothing.
	CompactDryRun
	// CompactSkipped is a stage that had nothing to compact, or was not wired.
	CompactSkipped
)

// CompactOutcomes lists every outcome, in declaration order.
func CompactOutcomes() []CompactOutcome {
	return []CompactOutcome{CompactDone, CompactShort, CompactFailed, CompactDryRun, CompactSkipped}
}

// String renders the outcome as its text form, the one [ParseCompactOutcome] reads
// back. A value that names no outcome prints its raw number.
func (o CompactOutcome) String() string {
	switch o {
	case CompactDone:
		return "done"
	case CompactShort:
		return "short"
	case CompactFailed:
		return "failed"
	case CompactDryRun:
		return "dry_run"
	case CompactSkipped:
		return "skipped"
	}
	return fmt.Sprintf("CompactOutcome(%d)", int(o))
}

func (o CompactOutcome) isKnown() bool {
	for _, outcome := range CompactOutcomes() {
		if outcome == o {
			return true
		}
	}
	return false
}

// ErrUnknownCompactOutcome is returned for text, or a value, that names no outcome.
// Match it with errors.Is.
var ErrUnknownCompactOutcome = errors.New("unknown compaction outcome")

// ParseCompactOutcome reads an outcome from the text form [CompactOutcome.String]
// writes. Any other text, the empty string included, returns an error matching
// [ErrUnknownCompactOutcome].
func ParseCompactOutcome(text string) (CompactOutcome, error) {
	for _, outcome := range CompactOutcomes() {
		if outcome.String() == text {
			return outcome, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownCompactOutcome, text)
}

// MarshalText writes the text form of the outcome. A value that names no outcome
// returns an error matching [ErrUnknownCompactOutcome].
func (o CompactOutcome) MarshalText() ([]byte, error) {
	if !o.isKnown() {
		return nil, fmt.Errorf("%w: %d", ErrUnknownCompactOutcome, int(o))
	}
	return []byte(o.String()), nil
}

// UnmarshalText reads the text form of an outcome, as [ParseCompactOutcome] does.
func (o *CompactOutcome) UnmarshalText(text []byte) error {
	parsed, err := ParseCompactOutcome(string(text))
	if err != nil {
		return err
	}
	*o = parsed
	return nil
}

// CompactStats is what a stage measured while it ran, for the usage report. Stage is
// the name the stage is configured under, which [RunCompactStack] fills when the
// stage left it empty. The counts are of tool calls unless another unit is named, and
// TokensRemoved is an estimate.
type CompactStats struct {
	Stage   string         `json:"Stage"`
	Outcome CompactOutcome `json:"Outcome"`
	// Calls is the number of tool calls the stage examined.
	Calls int `json:"Calls"`
	// Kept is the number of calls left verbatim.
	Kept int `json:"Kept"`
	// Truncated is the number of results cut down to their head and tail.
	Truncated int `json:"Truncated"`
	// Pruned is the number of calls reduced to a trace of themselves.
	Pruned int `json:"Pruned"`
	// Pinned is the number of calls the stage left untouched on purpose.
	Pinned int `json:"Pinned"`
	// CharsBefore and CharsAfter are the runes of the tool inputs and results of the
	// history, before and after the stage.
	CharsBefore int `json:"CharsBefore"`
	CharsAfter  int `json:"CharsAfter"`
	// TokensRemoved is the estimated number of tokens the stage freed.
	TokensRemoved int `json:"TokensRemoved"`
	// Requests is the number of model requests the stage made, and RequestTokens the
	// tokens it sent in them.
	Requests      int `json:"Requests"`
	RequestTokens int `json:"RequestTokens"`
	// Duration is the time the stage took.
	Duration time.Duration `json:"Duration"`
}
