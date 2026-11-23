package contracts

import (
	"context"
	"time"
)

// ForgeCredentials is the port the host implements and hands to a forge plugin.
// The host owns the credentials and the agent never sees a token: a plugin asks
// for the token of its forge host each time it calls the forge, and stores none.
// Token returns a token only for a host the user connected, and [ErrForgeNotConnected]
// otherwise. Invalidate drops the token the host cached for host; a plugin calls it
// after the forge answered [ErrForgeUnauthorized] to that token.
type ForgeCredentials interface {
	Token(ctx context.Context, host ForgeHost) (ForgeToken, error)
	Invalidate(ctx context.Context, host ForgeHost)
}

// Forge is the port a code forge plugin implements, one instance per forge host: a
// plugin is built for one [ForgeHost] and serves only repositories on it. Every
// method takes a context and stops when it is done. A method that takes a
// repository, a pull request, a job, a thread or an issue returns [ErrForgeWrongHost]
// for one on another host, as [RequireRepoHost] reports it. The forgetest package
// holds CheckRefusesForeignHost, which a plugin's tests run to prove that refusal
// for every such method. Listings take a [PageRequest] and a plugin refuses a limit
// outside its range with [ErrInvalidPageLimit]. [Forge.JobLog] returns at most the
// last maxBytes bytes of the job's log, the tail; maxBytes must be between 1 and
// [MaxJobLogBytes], and otherwise JobLog returns [ErrInvalidLogLimit].
//
// A failure a caller tells apart wraps one of the Err sentinels of this package or
// one of the error types [ForgeRateLimitedError] and [ForgeMergeRefusedError]:
// [ErrForgeNotConnected] when the host holds no credential, [ErrForgeUnauthorized]
// when the forge refuses it, [ErrForgeForbidden] when it lacks a permission,
// [ErrForgeNotFound] when the reference names nothing, and [ErrInvalidRepoRef] for a
// reference a plugin cannot map to its forge.
//
// [PullRequest] with a non-empty since returns [ErrForgeNotModified] when the pull
// request has not changed since that [Revision]. Following the event stream of a
// forge belongs to an optional interface that part A adds, so it does not change
// this one.
type Forge interface {
	Manifest() Manifest
	Connection(ctx context.Context) (ForgeConnection, error)
	RepoAccess(ctx context.Context, repo ForgeRepoRef) (RepoPermission, error)
	OpenPullRequest(ctx context.Context, draft PullRequestDraft) (PullRequest, error)
	PullRequest(ctx context.Context, ref PullRequestRef, since Revision) (PullRequest, error)
	MergePullRequest(ctx context.Context, ref PullRequestRef, method MergeMethod) error
	Checks(ctx context.Context, repo ForgeRepoRef, commit CommitSHA, page PageRequest) (Page[Check], error)
	JobLog(ctx context.Context, job JobRef, maxBytes int) ([]byte, error)
	ReviewThreads(ctx context.Context, ref PullRequestRef, page PageRequest) (Page[ReviewThread], error)
	ReplyToThread(ctx context.Context, thread ThreadRef, body string) error
	Issue(ctx context.Context, ref IssueRef) (Issue, error)
	Issues(ctx context.Context, repo ForgeRepoRef, query IssueQuery, page PageRequest) (Page[Issue], error)
}

// Revision is the opaque version of a pull request the forge reports: its ETag or
// the equivalent. Compare revisions only for equality, and pass the one of an
// earlier read back as since.
type Revision string

// ForgeConnection describes the account a forge plugin is connected as on its host:
// who it is, where its token came from, the scopes the forge reports for it, and
// the rate limit left.
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

// ForgeActor is an account that wrote something on the forge: an author of a
// comment, an issue or a pull request. Bot is set for an automated account.
type ForgeActor struct {
	Login       string
	ID          int64
	Association AuthorAssociation
	Bot         bool
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

// PullRequest is the state of a pull request as the forge reports it. Revision
// names the version of this read, to pass back as since on the next one. Merge is
// [MergeStateUnknown] while the forge computes it.
type PullRequest struct {
	Ref      PullRequestRef
	Title    string
	URL      string
	Head     string
	Base     string
	HeadSHA  CommitSHA
	State    PullRequestState
	Draft    bool
	Merge    MergeState
	Author   ForgeActor
	Revision Revision
}

// Check is one status of a commit: check runs and commit statuses are both mapped
// onto Check. Conclusion is meaningful only when Status is [CheckCompleted]; before
// that it is [ConclusionNone]. Consumers read the conclusion through [Check.Concluded]
// rather than from the field. Job is set only when the check comes from a CI job
// whose log [Forge.JobLog] can read, and is nil otherwise.
type Check struct {
	Name       string
	Commit     CommitSHA
	Status     CheckStatus
	Conclusion CheckConclusion
	URL        string
	Job        *JobRef
}

// Concluded returns the conclusion of a completed check and true. It returns
// [ConclusionNone] and false for a check that has not completed.
func (c Check) Concluded() (CheckConclusion, bool) {
	if c.Status != CheckCompleted {
		return ConclusionNone, false
	}
	return c.Conclusion, true
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
	Author    ForgeActor
	Body      string
	CreatedAt time.Time
}

// Issue is the state of an issue as the forge reports it. State is
// [IssueStateUnknown] when the forge did not report one, and consumers treat it as
// not open.
type Issue struct {
	Ref    IssueRef
	Title  string
	Body   string
	URL    string
	Author ForgeActor
	Labels []string
	State  IssueState
}

// IssueQuery selects the issues of a listing: the labels every issue must carry, and
// the state to list. The zero State lists open issues, the safe default of a listing;
// [IssueQuery.Listed] returns the state that is read.
type IssueQuery struct {
	Labels []string
	State  IssueState
}

// Listed is the state a listing reads: [IssueOpen] when State is the zero value, and
// State otherwise.
func (q IssueQuery) Listed() IssueState {
	if q.State == IssueStateUnknown {
		return IssueOpen
	}
	return q.State
}
