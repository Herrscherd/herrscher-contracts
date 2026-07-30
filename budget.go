package contracts

import (
	"fmt"
	"unicode/utf8"
)

// BudgetError is returned by Memory.Record when a node's Body exceeds the
// configured per-node budget. It carries the sizes so the caller (a Learner or
// a CLI verb) knows how much to trim, then consolidates/replaces to fit rather
// than blindly appending — the write-time atomicity forcer (memory roadmap G1).
// A budget of 0 disables the check, so this is never returned in that case.
type BudgetError struct {
	Key   string // node Key that was rejected
	Runes int    // rune count of the offered Body
	Limit int    // configured per-node budget, in runes
}

func (e *BudgetError) Error() string {
	return fmt.Sprintf(
		"memory: node %q body is %d runes, over the %d-rune budget; consolidate before recording",
		e.Key, e.Runes, e.Limit,
	)
}

// EnforceBudget returns a *BudgetError when body exceeds limit runes. A limit
// of zero or less disables the check and returns nil. key labels the rejected
// item in the returned error. Rune count, not byte length, is authoritative.
func EnforceBudget(key, body string, limit int) error {
	if limit <= 0 {
		return nil
	}
	if r := utf8.RuneCountInString(body); r > limit {
		return &BudgetError{Key: key, Runes: r, Limit: limit}
	}
	return nil
}
