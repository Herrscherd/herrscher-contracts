package contracts

import (
	"context"
	"time"
)

// ForgeCredentials is the port the host implements and hands to a forge plugin.
// The host owns the credentials and the agent never sees a token: a plugin asks
// for the token of a forge host each time it calls the forge, and stores none.
// Token returns [ErrForgeNotConnected] when the host holds no credential for
// host. Invalidate drops the token the host cached for host; a plugin calls it
// after the forge answered [ErrForgeUnauthorized] to that token.
type ForgeCredentials interface {
	Token(ctx context.Context, host string) (ForgeToken, error)
	Invalidate(ctx context.Context, host string)
}

// Forge is the port a code forge plugin implements, one implementation per forge.
// Every method takes a context and stops when it is done. A failure a caller tells
// apart wraps one of the Err sentinels of this package or a [ForgeRateLimitedError]:
// [ErrForgeNotConnected] when the host holds no credential, [ErrForgeUnauthorized]
// when the forge refuses it, [ErrForgeForbidden] when it lacks a permission,
// [ErrForgeNotFound] when the reference names nothing, [ErrInvalidLogLimit] for a
// log size that is not positive, and [ErrInvalidRepoRef] for a reference a plugin
// cannot map to its forge.
type Forge interface {
	Manifest() Manifest
	Connection(ctx context.Context, host string) (ForgeConnection, error)
	OpenPullRequest(ctx context.Context, draft PullRequestDraft) (PullRequest, error)
	PullRequest(ctx context.Context, ref PullRequestRef) (PullRequest, error)
	MergePullRequest(ctx context.Context, ref PullRequestRef, method MergeMethod) error
	Checks(ctx context.Context, ref PullRequestRef) ([]Check, error)
	JobLog(ctx context.Context, job JobRef, maxBytes int) ([]byte, error)
	ReviewThreads(ctx context.Context, ref PullRequestRef) ([]ReviewThread, error)
	ReplyToThread(ctx context.Context, thread ThreadRef, body string) error
	Issue(ctx context.Context, ref IssueRef) (Issue, error)
}

// ForgeConnection describes the account a forge plugin is connected as on one
// host: who it is, where its token came from, the permissions the forge reports
// for it, and the rate limit left.
type ForgeConnection struct {
	Login  string
	Source TokenSource
	// Permissions are the scopes the forge reports as granted, in the forge's own names.
	Permissions []string
	RateLimit   ForgeRateLimit
}

// ForgeRateLimit is the request budget the forge reports for the token. Reset is
// when Remaining is refilled, and the zero time when the forge does not say.
type ForgeRateLimit struct {
	Remaining int
	Reset     time.Time
}

// PullRequestDraft is what a plugin needs to open a pull request: the repository,
// the branch that carries the change, the branch it merges into, and the text.
type PullRequestDraft struct {
	Repo  ForgeRepoRef
	Head  string
	Base  string
	Title string
	Body  string
	Draft bool
}

// PullRequestRef names one pull request of a repository by its number.
type PullRequestRef struct {
	Repo   ForgeRepoRef
	Number int
}

// PullRequest is the state of a pull request as the forge reports it.
type PullRequest struct {
	Ref   PullRequestRef
	Title string
	URL   string
	Head  string
	Base  string
	State PullRequestState
	Draft bool
}

// Check is one status check or CI run attached to a pull request. Job is set only
// when the check comes from a CI job whose log [Forge.JobLog] can read, and is nil
// otherwise. Conclusion is [ConclusionNone] until the check is completed.
type Check struct {
	Name       string
	Status     CheckStatus
	Conclusion CheckConclusion
	URL        string
	Job        *JobRef
}

// JobRef names one CI job of a repository by the forge's own identifier.
type JobRef struct {
	Repo ForgeRepoRef
	ID   int64
}

// ReviewThread is a conversation anchored to a line of a pull request's diff.
type ReviewThread struct {
	ID       string
	Path     string
	Line     int
	Resolved bool
	Comments []ReviewComment
}

// ReviewComment is one message of a review thread, in the order the forge holds them.
type ReviewComment struct {
	Author    string
	Body      string
	CreatedAt time.Time
}

// ThreadRef names one review thread of a pull request by the forge's identifier.
type ThreadRef struct {
	PullRequest PullRequestRef
	ID          string
}

// IssueRef names one issue of a repository by its number.
type IssueRef struct {
	Repo   ForgeRepoRef
	Number int
}

// Issue is the state of an issue as the forge reports it.
type Issue struct {
	Ref    IssueRef
	Title  string
	Body   string
	URL    string
	Closed bool
}
