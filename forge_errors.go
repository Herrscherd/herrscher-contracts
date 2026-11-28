package contracts

import (
	"errors"
	"time"
)

// Errors a forge plugin returns and a caller tells apart, matched with
// errors.Is.
var (
	// ErrForgeNotConnected reports that the host holds no credential for the
	// forge host.
	ErrForgeNotConnected = errors.New("no forge credential for this host")
	// ErrForgeUnauthorized reports that the forge refused the credential it was
	// given.
	ErrForgeUnauthorized = errors.New("forge refused the credential")
	// ErrForgeForbidden reports that the credential lacks a permission the call
	// needs.
	ErrForgeForbidden = errors.New("forge credential lacks a required permission")
	// ErrForgeNotFound reports that the forge has no repository, pull request,
	// job, thread or issue under the reference given.
	ErrForgeNotFound = errors.New("not found on the forge")
	// ErrForgeWrongHost reports a reference to a repository on another forge
	// host than the one the plugin serves.
	ErrForgeWrongHost = errors.New("repository is on another forge host")
	// ErrForgeNotModified reports that the pull request or the issue listing has
	// not changed since the revision the caller holds.
	ErrForgeNotModified = errors.New("not modified since the revision given")
	// ErrFeedRefused reports that the forge refused the event feed of a pull
	// request, either at open or mid-stream as the Err of the last
	// [PullRequestSignal]. The caller reads the request on a cadence instead.
	ErrFeedRefused = errors.New("pull request event feed refused by the forge")
	// ErrInvalidForgeHost reports text that names no forge host.
	ErrInvalidForgeHost = errors.New("not a forge host")
	// ErrInvalidRepoRef reports text that names no repository a forge can hold.
	ErrInvalidRepoRef = errors.New("not a forge repository reference")
	// ErrInvalidBranchName reports text that git does not accept as a branch
	// name.
	ErrInvalidBranchName = errors.New("not a git branch name")

	// ErrRequiredChecksUnknown is wrapped by the error of a read of the checks a
	// branch requires that failed or was refused. A caller cannot tell an
	// unreadable rule from an absent one, so [SummarizeRequiredChecks] reads it as
	// pending.
	ErrRequiredChecksUnknown = errors.New("the checks a branch requires are unknown")
	// ErrEmptyRequiredCheckName reports a required check built with no name.
	ErrEmptyRequiredCheckName = errors.New("a required check needs a name")
	// ErrInvalidCommitSHA reports text that is not 40 or 64 lowercase hex
	// digits.
	ErrInvalidCommitSHA = errors.New("not a commit SHA: 40 or 64 lowercase hex digits")
	// ErrInvalidPageLimit reports a page limit below 0 or above [MaxPageLimit].
	ErrInvalidPageLimit = errors.New("page limit must be between 0 and 100")
	// ErrInvalidLogLimit reports a log size outside 1 to [MaxJobLogBytes].
	ErrInvalidLogLimit = errors.New("log size limit must be between 1 and MaxJobLogBytes")
	// ErrInvalidCloneURL reports a clone URL that is not an https URL naming the
	// repository on its forge host.
	ErrInvalidCloneURL = errors.New("not an https clone URL of the repository on its forge host")
	// ErrInvalidRef reports a reference that names no repository, no positive
	// number, or no thread identifier.
	ErrInvalidRef = errors.New("reference names no repository, number or thread")
	// ErrEmptyForgeToken reports a token built from an empty secret.
	ErrEmptyForgeToken = errors.New("forge token secret is empty")
	// ErrEmptyIssueTitle reports an issue draft with no title.
	ErrEmptyIssueTitle = errors.New("an issue needs a title")
	// ErrEmptyComment reports a comment with no text.
	ErrEmptyComment = errors.New("a comment needs a text")
	// ErrInvalidForgeLogin reports text that names no forge login.
	ErrInvalidForgeLogin = errors.New("not a forge login")
	// ErrInvalidIssueQuery reports an issue query whose state, sort or direction
	// is outside its closed set. A filter the forge cannot apply is
	// [ErrUnsupportedIssueFilter] instead.
	ErrInvalidIssueQuery = errors.New("issue query has a value outside its closed set")
	// ErrUnsupportedIssueFilter reports an issue query that filters by a value the
	// forge cannot apply, such as a mention on a forge without that filter.
	ErrUnsupportedIssueFilter = errors.New("issue query filters by a value the forge cannot apply")
	// ErrUnknownTokenSource reports a token source that names no source.
	ErrUnknownTokenSource = errors.New("unknown forge token source")
)

// ForgeRateLimitedError reports that the forge refused a call for its rate
// limit. Reset is when the limit lifts. It is an error type rather than a
// sentinel because the caller acts on the time, so match it with errors.AsType.
type ForgeRateLimitedError struct {
	Reset time.Time
}

// Error names the reset time in RFC 3339, in UTC, or says the reset time is
// unknown when Reset is the zero time.
func (e *ForgeRateLimitedError) Error() string {
	if e.Reset.IsZero() {
		return "forge rate limit reached, reset time unknown"
	}
	return "forge rate limit reached, resets at " + e.Reset.UTC().Format(time.RFC3339)
}

// ForgeMergeRefusedError reports that the forge refused to merge a pull
// request. State is the reason: the state the forge computed for it, such as
// [MergeStateConflicting] or [MergeStateBlocked]. It is an error type rather
// than a sentinel because the caller acts on the state, so match it with
// errors.AsType.
type ForgeMergeRefusedError struct {
	State MergeState
}

// Error names the state the pull request is in.
func (e *ForgeMergeRefusedError) Error() string {
	return "forge refused the merge, the pull request is " + e.State.String()
}
