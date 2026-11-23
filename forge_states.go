package contracts

import "fmt"

// TokenSource says where a forge token came from. It is a closed set whose zero
// value is not a source: [NewForgeToken] refuses it.
type TokenSource int

const (
	// TokenSourceSaved is a token the operator stored in the host.
	TokenSourceSaved TokenSource = iota + 1
	// TokenSourceApp is a token the host minted for an installed forge App.
	TokenSourceApp
	// TokenSourceEnv is a token read from the environment of the host.
	TokenSourceEnv
	// TokenSourceCLI is a token borrowed from the login of the forge's own command line tool.
	TokenSourceCLI
)

// String renders the source as its short name. A value that names no source
// prints its raw number.
func (s TokenSource) String() string {
	switch s {
	case TokenSourceSaved:
		return "saved"
	case TokenSourceApp:
		return "app"
	case TokenSourceEnv:
		return "env"
	case TokenSourceCLI:
		return "cli"
	}
	return fmt.Sprintf("TokenSource(%d)", int(s))
}

func (s TokenSource) known() bool {
	switch s {
	case TokenSourceSaved, TokenSourceApp, TokenSourceEnv, TokenSourceCLI:
		return true
	}
	return false
}

// PullRequestState is where a pull request stands. It is a closed set whose zero
// value is not a state. A forge that reports a merged request as closed is read
// as [PullRequestMerged] by the plugin.
type PullRequestState int

const (
	// PullRequestOpen is a pull request that awaits review or merge.
	PullRequestOpen PullRequestState = iota + 1
	// PullRequestClosed is a pull request closed without being merged.
	PullRequestClosed
	// PullRequestMerged is a pull request whose change reached its base branch.
	PullRequestMerged
)

// String renders the state as its short name. A value that names no state prints
// its raw number.
func (s PullRequestState) String() string {
	switch s {
	case PullRequestOpen:
		return "open"
	case PullRequestClosed:
		return "closed"
	case PullRequestMerged:
		return "merged"
	}
	return fmt.Sprintf("PullRequestState(%d)", int(s))
}

// CheckStatus is how far a check has run. It is a closed set whose zero value is
// not a status.
type CheckStatus int

const (
	// CheckQueued is a check that waits for a runner.
	CheckQueued CheckStatus = iota + 1
	// CheckInProgress is a check that is running.
	CheckInProgress
	// CheckCompleted is a check that has finished, and carries a conclusion.
	CheckCompleted
)

// String renders the status as its short name. A value that names no status prints
// its raw number.
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

// CheckConclusion is how a completed check ended. Its zero value, [ConclusionNone],
// is the deliberate "no conclusion yet" of a check that has not completed.
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
	// ConclusionActionRequired is a check that needs a person to act before it can pass.
	ConclusionActionRequired
)

// String renders the conclusion as its short name. A value that names no conclusion
// prints its raw number.
func (c CheckConclusion) String() string {
	switch c {
	case ConclusionNone:
		return "none"
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
	}
	return fmt.Sprintf("CheckConclusion(%d)", int(c))
}

// MergeMethod is how a pull request reaches its base branch. It is a closed set
// whose zero value is not a method.
type MergeMethod int

const (
	// MergeMethodMerge records a merge commit that keeps every commit of the branch.
	MergeMethodMerge MergeMethod = iota + 1
	// MergeMethodSquash folds the branch into one commit on the base branch.
	MergeMethodSquash
	// MergeMethodRebase replays the commits of the branch onto the base branch.
	MergeMethodRebase
)

// String renders the method as its short name. A value that names no method prints
// its raw number.
func (m MergeMethod) String() string {
	switch m {
	case MergeMethodMerge:
		return "merge"
	case MergeMethodSquash:
		return "squash"
	case MergeMethodRebase:
		return "rebase"
	}
	return fmt.Sprintf("MergeMethod(%d)", int(m))
}
