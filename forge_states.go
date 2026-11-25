package contracts

import (
	"errors"
	"fmt"
)

const (
	noneName    = "none"
	unknownName = "unknown"
	failedName  = "failed"
)

// TokenSource says where a forge token came from. It is a closed set whose zero
// value is not a source: [NewForgeToken] refuses it, and it renders as
// "unknown".
type TokenSource int

const (
	// TokenSourceSaved is a token the operator stored in the host.
	TokenSourceSaved TokenSource = iota + 1
	// TokenSourceApp is a token the host minted for an installed forge App.
	TokenSourceApp
	// TokenSourceEnv is a token read from the environment of the host.
	TokenSourceEnv
	// TokenSourceCLI is a token borrowed from the login of the forge's own
	// command line tool.
	TokenSourceCLI
)

// String renders the source as its short name. A value that names no source,
// the zero value included, renders as "unknown".
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
	return unknownName
}

func (s TokenSource) known() bool {
	switch s {
	case TokenSourceSaved, TokenSourceApp, TokenSourceEnv, TokenSourceCLI:
		return true
	}
	return false
}

// PullRequestState is where a pull request stands. It is a closed set whose
// zero value is not a state. A forge that reports a merged request as closed is
// read as [PullRequestMerged] by the plugin.
type PullRequestState int

const (
	// PullRequestOpen is a pull request that awaits review or merge.
	PullRequestOpen PullRequestState = iota + 1
	// PullRequestClosed is a pull request closed without being merged.
	PullRequestClosed
	// PullRequestMerged is a pull request whose change reached its base branch.
	PullRequestMerged
)

// String renders the state as its short name. A value that names no state
// prints its raw number.
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

// PullRequestStates lists every state, in declaration order.
func PullRequestStates() []PullRequestState {
	return []PullRequestState{PullRequestOpen, PullRequestClosed, PullRequestMerged}
}

// ErrUnknownPullRequestState is returned for text, or a value, that names no pull
// request state. Match it with errors.Is.
var ErrUnknownPullRequestState = errors.New("unknown pull request state")

// ParsePullRequestState reads a state from the text form [PullRequestState.String]
// writes. Any other text, the empty string included, returns an error matching
// [ErrUnknownPullRequestState].
func ParsePullRequestState(text string) (PullRequestState, error) {
	for _, state := range PullRequestStates() {
		if state.String() == text {
			return state, nil
		}
	}
	return 0, ErrUnknownPullRequestState
}

// MarshalText writes the text form of the state. A value that names no state
// returns an error matching [ErrUnknownPullRequestState], so it never reaches the
// wire.
func (s PullRequestState) MarshalText() ([]byte, error) {
	if !s.isKnown() {
		return nil, fmt.Errorf("%w: %d", ErrUnknownPullRequestState, int(s))
	}
	return []byte(s.String()), nil
}

// UnmarshalText reads the text form of a state, as [ParsePullRequestState] does.
func (s *PullRequestState) UnmarshalText(text []byte) error {
	parsed, err := ParsePullRequestState(string(text))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// PullRequestEventKind is the kind of change a [PullRequestSignal] reports. It is
// a closed set whose zero value is not a kind.
type PullRequestEventKind int

const (
	// PullRequestReviewSubmitted is a review submitted on the pull request.
	PullRequestReviewSubmitted PullRequestEventKind = iota + 1
	// PullRequestNewCommit is a commit pushed to the branch of the pull request.
	PullRequestNewCommit
	// PullRequestStateChanged is a pull request opened, closed, reopened or merged.
	PullRequestStateChanged
	// PullRequestMergeConflict is a pull request whose branch came to conflict
	// with its base.
	PullRequestMergeConflict
	// PullRequestChecksChanged is a check of the pull request that started or
	// completed.
	PullRequestChecksChanged
)

// String renders the kind as its short name. A value that names no kind prints
// its raw number.
func (k PullRequestEventKind) String() string {
	switch k {
	case PullRequestReviewSubmitted:
		return "review_submitted"
	case PullRequestNewCommit:
		return "new_commit"
	case PullRequestStateChanged:
		return "state_changed"
	case PullRequestMergeConflict:
		return "merge_conflict"
	case PullRequestChecksChanged:
		return "checks_changed"
	}
	return fmt.Sprintf("PullRequestEventKind(%d)", int(k))
}

// ErrUnknownPullRequestEventKind is returned for text that names no pull request
// event kind. Match it with errors.Is.
var ErrUnknownPullRequestEventKind = errors.New("unknown pull request event kind")

// ParsePullRequestEventKind reads a kind from the text form
// [PullRequestEventKind.String] writes. Any other text, the empty string included,
// returns an error matching [ErrUnknownPullRequestEventKind].
func ParsePullRequestEventKind(text string) (PullRequestEventKind, error) {
	for _, kind := range pullRequestEventKinds() {
		if kind.String() == text {
			return kind, nil
		}
	}
	return 0, ErrUnknownPullRequestEventKind
}

func pullRequestEventKinds() []PullRequestEventKind {
	return []PullRequestEventKind{
		PullRequestReviewSubmitted,
		PullRequestNewCommit,
		PullRequestStateChanged,
		PullRequestMergeConflict,
		PullRequestChecksChanged,
	}
}

// ChecksState is the summary of the checks on a pull request's head commit. It is
// a closed set whose zero value, [ChecksUnknown], is the state while the forge has
// not reported the checks. A failed check outranks a pending one.
type ChecksState int

const (
	// ChecksUnknown is a pull request whose checks the forge has not reported.
	ChecksUnknown ChecksState = iota
	// ChecksNone is a pull request whose head commit has no check.
	ChecksNone
	// ChecksPending is a pull request with a check that has not completed and no
	// failed check.
	ChecksPending
	// ChecksPassed is a pull request whose checks all completed successfully.
	ChecksPassed
	// ChecksFailed is a pull request with a completed check that failed.
	ChecksFailed
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
	return []ChecksState{ChecksUnknown, ChecksNone, ChecksPending, ChecksPassed, ChecksFailed}
}

func (s PullRequestState) isKnown() bool {
	for _, state := range PullRequestStates() {
		if state == s {
			return true
		}
	}
	return false
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

// MergeMethod is how a pull request reaches its base branch. It is a closed set
// whose zero value is not a method.
type MergeMethod int

const (
	// MergeMethodMerge records a merge commit that keeps every commit of the
	// branch.
	MergeMethodMerge MergeMethod = iota + 1
	// MergeMethodSquash folds the branch into one commit on the base branch.
	MergeMethodSquash
	// MergeMethodRebase replays the commits of the branch onto the base branch.
	MergeMethodRebase
)

// String renders the method as its short name. A value that names no method
// prints its raw number.
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

// MergeState is what the forge says about merging a pull request now. It is a
// closed set whose zero value, [MergeStateUnknown], is the state while the
// forge still computes it.
type MergeState int

const (
	// MergeStateUnknown is a pull request whose merge state the forge has not
	// computed yet.
	MergeStateUnknown MergeState = iota
	// MergeStateClean is a pull request that can be merged now.
	MergeStateClean
	// MergeStateConflicting is a pull request whose branch conflicts with its
	// base.
	MergeStateConflicting
	// MergeStateBlocked is a pull request that waits for required checks or
	// reviews.
	MergeStateBlocked
	// MergeStateBehind is a pull request whose base has moved past its branch.
	MergeStateBehind
	// MergeStateDraft is a pull request that is still a draft.
	MergeStateDraft
)

// String renders the state as its short name. A value that names no state
// prints its raw number.
func (s MergeState) String() string {
	switch s {
	case MergeStateUnknown:
		return "unknown"
	case MergeStateClean:
		return "clean"
	case MergeStateConflicting:
		return "conflicting"
	case MergeStateBlocked:
		return "blocked"
	case MergeStateBehind:
		return "behind"
	case MergeStateDraft:
		return "draft"
	}
	return fmt.Sprintf("MergeState(%d)", int(s))
}

// AuthorAssociation is how an account stands toward a repository. It is a
// closed set whose zero value, [AssociationNone], means the forge says nothing
// about it.
type AuthorAssociation int

const (
	// AssociationNone is an account the forge gives no association to.
	AssociationNone AuthorAssociation = iota
	// AssociationOwner is the owner of the repository.
	AssociationOwner
	// AssociationMember is a member of the organization that owns the
	// repository.
	AssociationMember
	// AssociationCollaborator is an account invited to the repository.
	AssociationCollaborator
	// AssociationContributor is an account that has contributed before.
	AssociationContributor
	// AssociationFirstTimer is an account whose first contribution is this one.
	AssociationFirstTimer
	// AssociationOther is an account with no other association.
	AssociationOther
)

// String renders the association as its short name. A value that names no
// association prints its raw number.
func (a AuthorAssociation) String() string {
	switch a {
	case AssociationNone:
		return noneName
	case AssociationOwner:
		return "owner"
	case AssociationMember:
		return "member"
	case AssociationCollaborator:
		return "collaborator"
	case AssociationContributor:
		return "contributor"
	case AssociationFirstTimer:
		return "first_timer"
	case AssociationOther:
		return "other"
	}
	return fmt.Sprintf("AuthorAssociation(%d)", int(a))
}

// IssueState is whether an issue is open. It is a closed set whose zero value,
// [IssueStateUnknown], means the forge did not say, and consumers treat it as
// not open. A listing reads the zero value as [IssueOpen] (see
// [IssueQuery.Listed]).
type IssueState int

const (
	// IssueStateUnknown is an issue whose state the forge did not report.
	IssueStateUnknown IssueState = iota
	// IssueOpen is an issue that awaits work or a decision.
	IssueOpen
	// IssueClosed is an issue that was resolved or dropped.
	IssueClosed
)

// String renders the state as its short name. A value that names no state
// prints its raw number.
func (s IssueState) String() string {
	switch s {
	case IssueStateUnknown:
		return unknownName
	case IssueOpen:
		return "open"
	case IssueClosed:
		return "closed"
	}
	return fmt.Sprintf("IssueState(%d)", int(s))
}

// RepoPermission is the rights an account holds on one repository, from none to
// admin. It is a closed set whose zero value, [PermissionNone], is no right at
// all.
type RepoPermission int

const (
	// PermissionNone is an account with no right on the repository.
	PermissionNone RepoPermission = iota
	// PermissionRead is an account that can read the repository.
	PermissionRead
	// PermissionTriage is an account that can manage issues and pull requests
	// without writing code.
	PermissionTriage
	// PermissionWrite is an account that can push to the repository.
	PermissionWrite
	// PermissionMaintain is an account that can manage the repository without
	// owning it.
	PermissionMaintain
	// PermissionAdmin is an account that holds every right on the repository.
	PermissionAdmin
)

// String renders the permission as its short name. A value that names no
// permission prints its raw number.
func (p RepoPermission) String() string {
	switch p {
	case PermissionNone:
		return noneName
	case PermissionRead:
		return "read"
	case PermissionTriage:
		return "triage"
	case PermissionWrite:
		return "write"
	case PermissionMaintain:
		return "maintain"
	case PermissionAdmin:
		return "admin"
	}
	return fmt.Sprintf("RepoPermission(%d)", int(p))
}
