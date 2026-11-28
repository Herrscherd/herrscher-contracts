package contracts

import (
	"errors"
	"fmt"
)

// ChecksState is the summary of the checks on a pull request's head commit. It is
// a closed set whose zero value, [ChecksUnknown], is the state while the forge has
// not reported the checks. A failed check outranks an action-required one, which
// outranks a pending one.
type ChecksState int

const (
	// ChecksUnknown is a pull request whose checks the forge has not reported.
	ChecksUnknown ChecksState = iota
	// ChecksNone is a pull request whose head commit has no check.
	ChecksNone
	// ChecksPending is a pull request with a check that has not completed and no
	// failed or action-required check.
	ChecksPending
	// ChecksPassed is a pull request whose checks all completed successfully.
	ChecksPassed
	// ChecksFailed is a pull request with a completed check that failed.
	ChecksFailed
	// ChecksActionRequired is a pull request with a check that waits for a person
	// to act, and no failed check.
	ChecksActionRequired
)

// String renders the state as its short name. A value that names no state prints
// its raw number.
func (s ChecksState) String() string {
	switch s {
	case ChecksUnknown:
		return unknownName
	case ChecksNone:
		return noneName
	case ChecksPending:
		return "pending"
	case ChecksPassed:
		return "passed"
	case ChecksFailed:
		return failedName
	case ChecksActionRequired:
		return "action_required"
	}
	return fmt.Sprintf("ChecksState(%d)", int(s))
}

// ErrUnknownChecksState is returned for text that names no checks state. Match it
// with errors.Is.
var ErrUnknownChecksState = errors.New("unknown checks state")

// ParseChecksState reads a state from the text form [ChecksState.String] writes.
// Any other text, the empty string included, returns an error matching
// [ErrUnknownChecksState].
func ParseChecksState(text string) (ChecksState, error) {
	for _, state := range checksStates() {
		if state.String() == text {
			return state, nil
		}
	}
	return 0, ErrUnknownChecksState
}

func checksStates() []ChecksState {
	return []ChecksState{ChecksUnknown, ChecksNone, ChecksPending, ChecksPassed, ChecksFailed, ChecksActionRequired}
}

// CheckStatus is how far a check has run. It is a closed set whose zero value
// is not a status.
type CheckStatus int

const (
	// CheckQueued is a check that waits for a runner.
	CheckQueued CheckStatus = iota + 1
	// CheckInProgress is a check that is running.
	CheckInProgress
	// CheckCompleted is a check that has finished, and carries a conclusion.
	CheckCompleted
)

// String renders the status as its short name. A value that names no status
// prints its raw number.
func (s CheckStatus) String() string {
	switch s {
	case CheckQueued:
		return "queued"
	case CheckInProgress:
		return "in_progress"
	case CheckCompleted:
		return "completed"
	}
	return fmt.Sprintf("CheckStatus(%d)", int(s))
}

// CheckConclusion is how a completed check ended. Its zero value,
// [ConclusionNone], is the deliberate "no conclusion yet" of a check that has
// not completed. A completed check whose conclusion the plugin cannot map is
// [ConclusionOther], so a completed check never reads as [ConclusionNone].
type CheckConclusion int

const (
	// ConclusionNone is a check that has not completed, so has no conclusion.
	ConclusionNone CheckConclusion = iota
	// ConclusionSuccess is a check that passed.
	ConclusionSuccess
	// ConclusionFailure is a check that failed.
	ConclusionFailure
	// ConclusionNeutral is a check that completed without a verdict.
	ConclusionNeutral
	// ConclusionCancelled is a check that was cancelled before it finished.
	ConclusionCancelled
	// ConclusionSkipped is a check that did not run.
	ConclusionSkipped
	// ConclusionTimedOut is a check that ran past its time limit.
	ConclusionTimedOut
	// ConclusionActionRequired is a check that needs a person to act before it
	// can pass.
	ConclusionActionRequired
	// ConclusionOther is a completed check whose conclusion the plugin cannot
	// map.
	ConclusionOther
)

// String renders the conclusion as its short name. A value that names no
// conclusion prints its raw number.
func (c CheckConclusion) String() string {
	switch c {
	case ConclusionNone:
		return noneName
	case ConclusionSuccess:
		return "success"
	case ConclusionFailure:
		return "failure"
	case ConclusionNeutral:
		return "neutral"
	case ConclusionCancelled:
		return "cancelled"
	case ConclusionSkipped:
		return "skipped"
	case ConclusionTimedOut:
		return "timed_out"
	case ConclusionActionRequired:
		return "action_required"
	case ConclusionOther:
		return "other"
	}
	return fmt.Sprintf("CheckConclusion(%d)", int(c))
}

// SummarizeChecks reads the checks of a pull request's head commit as one
// [ChecksState]. Every check counts, so a re-run that repeats a failed check
// still counts that failure. [SummarizeRequiredChecks] differs on re-runs: it
// reads only the latest attempt of each check, as described there. A check that has not completed is
// pending. A completed check takes the state of its conclusion:
//   - success, neutral, skipped: passed;
//   - action_required: action required, a person must act before the check
//     can finish;
//   - failure, cancelled, timed_out, other, and any conclusion the table does
//     not name, including [ConclusionNone] on a completed check: failed, so an
//     unmapped outcome never reads as passed.
//
// A failed check outranks an action-required one, which outranks a pending one,
// and a pending check outranks passed. With no check at all the state is
// [ChecksNone]. SummarizeChecks never returns [ChecksUnknown]: that state stays
// the state of a forge that reported nothing.
func SummarizeChecks(checks []Check) ChecksState {
	if len(checks) == 0 {
		return ChecksNone
	}
	return rankChecks(checks)
}

// RequiredChecksSummary is the state of a pull request's checks against the
// checks its branch requires. Cause is nil when the requirements were read. When
// Cause is set, the requirements could not be read: if it wraps
// [ErrRequiredChecksUnknown], State is pending, or failed or action required when
// a check already says so, and a host bounds its wait for the requirements and
// shows Cause. Any other Cause leaves State [ChecksUnknown], which a host must
// not read as a state.
type RequiredChecksSummary struct {
	State ChecksState
	Cause error
}

// SummarizeRequiredChecks reads the checks of a pull request's head commit
// against the checks its branch requires, as [RequiredChecksSource] returns
// them, and the error of that read. Each run of a check is read through its
// latest attempt, as [Check] describes; when two runs have the same attempt, the
// more severe one is read. Only the checks that a requirement matches count, so
// a failing check that the branch does not require does not decide the state. A
// required check that no check matches is pending: it has not been created yet,
// so it has not passed. A requirement with an AppID matches only a check that
// the same app reported. With no requirement the state is that of every check,
// each run read as above.
//
// This differs from [SummarizeChecks] on a re-run. SummarizeChecks counts every
// run as published, so a failed run stays a failure after its re-run passes.
// SummarizeRequiredChecks reads only the latest attempt, since a re-run replaces
// the run it repeats, and that is the reading a host needs to merge.
//
// A requirement with an empty name is an invalid input: the error matches
// [ErrEmptyRequiredCheckName] and the summary is empty. Otherwise, an error
// read from the requirements is returned in the summary's Cause, see
// [RequiredChecksSummary].
func SummarizeRequiredChecks(checks []Check, required []RequiredCheck, requiredErr error) (RequiredChecksSummary, error) {
	for _, requirement := range required {
		if requirement.name == "" {
			return RequiredChecksSummary{}, fmt.Errorf("required check: %w", ErrEmptyRequiredCheckName)
		}
	}
	if requiredErr != nil {
		return summarizeUnknownRequirements(checks, requiredErr), nil
	}
	if len(required) == 0 {
		return RequiredChecksSummary{State: summarizeAll(checks)}, nil
	}
	latest := latestAttempts(checks)
	summary := rankChecks(matchedBy(latest, required))
	if summary == ChecksFailed || summary == ChecksActionRequired {
		return RequiredChecksSummary{State: summary}, nil
	}
	for _, requirement := range required {
		if !requirement.metBy(latest) {
			return RequiredChecksSummary{State: ChecksPending}, nil
		}
	}
	return RequiredChecksSummary{State: summary}, nil
}

func summarizeUnknownRequirements(checks []Check, requiredErr error) RequiredChecksSummary {
	if !errors.Is(requiredErr, ErrRequiredChecksUnknown) {
		return RequiredChecksSummary{State: ChecksUnknown, Cause: requiredErr}
	}
	state := summarizeAll(checks)
	if state != ChecksFailed && state != ChecksActionRequired {
		state = ChecksPending
	}
	return RequiredChecksSummary{State: state, Cause: requiredErr}
}

func summarizeAll(checks []Check) ChecksState {
	if len(checks) == 0 {
		return ChecksNone
	}
	return rankChecks(latestAttempts(checks))
}

func (r RequiredCheck) matches(check Check) bool {
	return check.Name == r.name && (r.appID == 0 || check.AppID == r.appID)
}

func (r RequiredCheck) metBy(checks []Check) bool {
	for _, check := range checks {
		if r.matches(check) {
			return true
		}
	}
	return false
}

func matchedBy(checks []Check, required []RequiredCheck) []Check {
	matched := make([]Check, 0, len(checks))
	for _, check := range checks {
		for _, requirement := range required {
			if requirement.matches(check) {
				matched = append(matched, check)
				break
			}
		}
	}
	return matched
}

type checkRun struct {
	name  string
	appID int64
}

func latestAttempts(checks []Check) []Check {
	index := make(map[checkRun]int, len(checks))
	latest := make([]Check, 0, len(checks))
	for _, check := range checks {
		if check.Name == "" {
			latest = append(latest, check)
			continue
		}
		run := checkRun{name: check.Name, appID: check.AppID}
		position, seen := index[run]
		if !seen {
			index[run] = len(latest)
			latest = append(latest, check)
			continue
		}
		if supersedes(check, latest[position]) {
			latest[position] = check
		}
	}
	return latest
}

func supersedes(check, current Check) bool {
	if check.Attempt != current.Attempt {
		return check.Attempt > current.Attempt
	}
	return severity(checkState(check)) > severity(checkState(current))
}

const (
	severityPassed = iota
	severityPending
	severityActionRequired
	severityFailed
)

func severity(state ChecksState) int {
	switch state {
	case ChecksFailed:
		return severityFailed
	case ChecksActionRequired:
		return severityActionRequired
	case ChecksPending:
		return severityPending
	case ChecksUnknown, ChecksNone, ChecksPassed:
	}
	return severityPassed
}

func rankChecks(checks []Check) ChecksState {
	summary := ChecksPassed
	for _, check := range checks {
		switch checkState(check) {
		case ChecksFailed:
			return ChecksFailed
		case ChecksActionRequired:
			summary = ChecksActionRequired
		case ChecksPending:
			if summary != ChecksActionRequired {
				summary = ChecksPending
			}
		case ChecksUnknown, ChecksNone, ChecksPassed:
		}
	}
	return summary
}

func checkState(check Check) ChecksState {
	conclusion, concluded := check.Concluded()
	if !concluded {
		return ChecksPending
	}
	switch conclusion {
	case ConclusionSuccess, ConclusionNeutral, ConclusionSkipped:
		return ChecksPassed
	case ConclusionActionRequired:
		return ChecksActionRequired
	case ConclusionNone, ConclusionFailure, ConclusionCancelled, ConclusionTimedOut, ConclusionOther:
	}
	return ChecksFailed
}
