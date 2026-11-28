package contracts

import (
	"context"
	"time"
)

// ForgeCredentials is the port the host implements and hands to a forge plugin.
// The host owns the credentials and the agent never sees a token: a plugin asks
// for the token of its forge host each time it calls the forge, and stores
// none. Token returns a token only for a host the user connected, and
// [ErrForgeNotConnected] otherwise. Invalidate drops the token the host cached
// for host; a plugin calls it after the forge answered [ErrForgeUnauthorized]
// to that token.
type ForgeCredentials interface {
	Token(ctx context.Context, host ForgeHost) (ForgeToken, error)
	Invalidate(ctx context.Context, host ForgeHost)
}

// Forge is the port a code forge plugin implements, one instance per forge
// host: a plugin is built for one [ForgeHost] and serves only repositories on
// it. Every method takes a context and stops when it is done. A method that
// takes a repository, a pull request, a job, a thread or an issue returns
// [ErrForgeWrongHost] for one on another host, as [RequireRepoHost] reports it.
// The forgetest package holds CheckRefusesForeignHost, which a plugin's tests
// run to prove that refusal for every such method. Listings take a
// [PageRequest] and a plugin refuses a limit outside its range with
// [ErrInvalidPageLimit]. [Forge.JobLog] returns at most the last maxBytes bytes
// of the job's log, the tail; maxBytes must be between 1 and [MaxJobLogBytes],
// and otherwise JobLog returns [ErrInvalidLogLimit].
//
// A failure a caller tells apart wraps one of the Err sentinels of this package
// or one of the error types [ForgeRateLimitedError] and
// [ForgeMergeRefusedError]: [ErrForgeNotConnected] when the host holds no
// credential, [ErrForgeUnauthorized] when the forge refuses it,
// [ErrForgeForbidden] when it lacks a permission, [ErrForgeNotFound] when the
// reference names nothing, and [ErrInvalidRepoRef] for a reference a plugin
// cannot map to its forge.
//
// [PullRequest] with a non-empty since returns [ErrForgeNotModified] when the
// pull request has not changed since that [Revision]. The event stream of a
// forge is the optional [PullRequestFeed] interface, so a plugin that gains it
// does not change this one. The checks a branch requires are the optional
// [RequiredChecksSource] for the same reason.
type Forge interface {
	Manifest() Manifest
	Connection(ctx context.Context) (ForgeConnection, error)
	RepoAccess(ctx context.Context, repo ForgeRepoRef) (RepoPermission, error)
	// Repositories lists the repositories the connected account can read on the
	// host, a page at a time. A plugin builds each one with [NewForgeRepo] and
	// skips a repository that NewForgeRepo refuses, such as one whose clone URL
	// is not on the host, instead of failing the page. A forge served under a
	// base path is not supported, since [ForgeHost] has no path. Like every
	// listing, it refuses a limit outside its range with [ErrInvalidPageLimit].
	Repositories(ctx context.Context, page PageRequest) (Page[ForgeRepo], error)
	OpenPullRequest(ctx context.Context, draft PullRequestDraft) (PullRequest, error)
	PullRequest(ctx context.Context, ref PullRequestRef, since Revision) (PullRequest, error)
	MergePullRequest(ctx context.Context, ref PullRequestRef, method MergeMethod) error
	Checks(ctx context.Context, repo ForgeRepoRef, commit CommitSHA, page PageRequest) (Page[Check], error)
	JobLog(ctx context.Context, job JobRef, maxBytes int) ([]byte, error)
	ReviewThreads(ctx context.Context, ref PullRequestRef, page PageRequest) (Page[ReviewThread], error)
	ReplyToThread(ctx context.Context, thread ThreadRef, body string) error
	Issue(ctx context.Context, ref IssueRef) (Issue, error)
	// Issues lists the issues of repo that match query, a page at a time, in the
	// order query names. The listing includes pull requests the forge lists among
	// issues, each marked by [Issue.IsPullRequest]. It refuses a query outside its
	// closed sets with [ErrInvalidIssueQuery] and a limit outside its range with
	// [ErrInvalidPageLimit].
	Issues(ctx context.Context, repo ForgeRepoRef, query IssueQuery, page PageRequest) (Page[Issue], error)
}

// Revision is the opaque version of a pull request or an issue listing the forge
// reports: its ETag or the equivalent. Compare revisions only for equality, and
// pass the one of an earlier read back as since. A revision of an issue listing
// is valid only for the repository, query and page that produced it; see
// [ConditionalIssues].
type Revision string

// ForgeConnection describes the account a forge plugin is connected as on its
// host: who it is, where its token came from, the scopes the forge reports for
// it, and the rate limit left.
type ForgeConnection struct {
	Login  string
	Source TokenSource
	// Permissions are the scopes the forge reports as granted, in the forge's
	// own names.
	Permissions []string
	RateLimit   ForgeRateLimit
}

// ForgeRateLimit is the request budget the forge reports for the token. Reset
// is when Remaining is refilled, and the zero time when the forge does not say.
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

// PullRequestDraft is what a plugin needs to open a pull request: the
// repository, the branch that carries the change, the branch it merges into,
// and the text.
type PullRequestDraft struct {
	Repo  ForgeRepoRef
	Head  string
	Base  string
	Title string
	Body  string
	Draft bool
}

// PullRequest is the state of a pull request as the forge reports it. Revision
// names the version of this read, to pass back as since on the next one. Merge
// is [MergeStateUnknown] while the forge computes it. Checks summarizes the
// checks of HeadSHA, and is [ChecksUnknown] when the forge did not report them.
// LatestReview is the [ReviewID] of the last review submitted on the request.
// It is empty while there is none, and also when the forge does not report
// reviews, so a host must not read an empty LatestReview as the absence of a
// review. A value different from the one of the last read means a review was
// submitted since: compare it for equality only, as with Revision.
type PullRequest struct {
	Ref          PullRequestRef
	Title        string
	URL          string
	Head         string
	Base         string
	HeadSHA      CommitSHA
	State        PullRequestState
	Draft        bool
	Merge        MergeState
	Checks       ChecksState
	LatestReview ReviewID
	Author       ForgeActor
	Revision     Revision
}

// ReviewID is the opaque identifier of a review the forge reports, compared for
// equality only, as with [Revision]. The zero ReviewID means no review is reported,
// or that the forge reports no reviews, and a host cannot tell the two apart.
type ReviewID string

// PullRequestFeed is the optional event stream of a forge, which a plugin
// implements when its forge grants one. A caller finds it with a type assertion
// and reads [Forge.PullRequest] on a cadence otherwise.
//
// The feed exists in process only. The host compiles the plugin that serves a
// forge into its own binary, and no wire transport carries the feed, so a
// caller in another process reads [Forge.PullRequest] on a cadence instead.
//
// PullRequestEvents returns a channel of signals for one pull request. The
// channel closes in one of two ways, and the signal before the close tells them
// apart:
//   - when ctx is done. The plugin sends no new signal once it has observed
//     the cancellation, but a signal may still arrive if the cancellation lands
//     between its context check and its send, so the caller treats the close,
//     not the last signal, as the end of the feed;
//   - when the stream fails while ctx is live, after a signal whose Err is set
//     and whose Kind is zero. Err matches the cause, so a refusal mid-stream
//     matches [ErrFeedRefused] with errors.Is. The caller then reads
//     [Forge.PullRequest] on a cadence.
//
// A caller that receives a signal with Err stops reading the channel and cancels
// the ctx it passed to PullRequestEvents.
//
// A refusal at open returns an error matching [ErrFeedRefused] and no channel.
//
// Sending never blocks past ctx: each send selects on ctx.Done(). A signal of a
// kind the caller has not read yet is merged, not queued again, so the channel
// holds at most one pending signal per [PullRequestEventKind], and the signal
// with an error is never dropped. A caller that falls behind still learns of
// every kind of change, and reads the pull request to see its current state.
//
// Two obligations belong to the plugin and are not verified by
// forgetest.CheckPullRequestFeed: no new signal is sent once the cancellation
// has been observed, and no send blocks without a select on ctx.Done().
type PullRequestFeed interface {
	PullRequestEvents(ctx context.Context, ref PullRequestRef) (<-chan PullRequestSignal, error)
}

// IssueWriter is the optional interface of a forge that opens issues and
// comments on them, which a plugin implements beside [Forge]. A caller finds it
// with a type assertion. Like the methods of [Forge], each method returns
// [ErrForgeWrongHost] for a repository on another host, and forgetest's
// CheckRefusesForeignHost proves it when the forge implements the interface.
// OpenIssue refuses a draft with an empty title with [ErrEmptyIssueTitle], and
// CommentOnIssue an empty body with [ErrEmptyComment], before calling the forge.
type IssueWriter interface {
	OpenIssue(ctx context.Context, draft IssueDraft) (Issue, error)
	CommentOnIssue(ctx context.Context, ref IssueRef, body string) error
}

// IssueDraft is what a plugin needs to open an issue: the repository, the text,
// and the labels to put on it.
type IssueDraft struct {
	Repo   ForgeRepoRef
	Title  string
	Body   string
	Labels []string
}

// PullRequestSignal is one event of a [PullRequestFeed]. It carries either the
// Kind of a change of the pull request, or an Err when the stream failed. A
// signal with Err set has a zero Kind, and it is the last signal before the
// channel closes. A reader reads the pull request to see what changed.
type PullRequestSignal struct {
	Kind PullRequestEventKind
	Err  error
}

// Check is one status of a commit: check runs and commit statuses are both
// mapped onto Check. Conclusion is meaningful only when Status is
// [CheckCompleted]; before that it is [ConclusionNone]. Consumers read the
// conclusion through [Check.Concluded] rather than from the field. Job is set
// only when the check comes from a CI job whose log [Forge.JobLog] can read,
// and is nil otherwise. AppID is the app or integration that reported the
// check, zero when the forge reports none. Attempt orders the runs of a check
// with the same name and app: a larger value is a later run, so a re-run
// supersedes the run it repeats. A plugin sets it to a value that strictly
// increases with each re-run of the same check, such as the id of the GitHub
// check run, so that a successful re-run supersedes an earlier failure. It sets
// 0 when the forge gives no such value, and a value of 0 keeps the earlier
// failures of the same name and app: at equal attempts the more severe run is
// read. [SummarizeRequiredChecks] reads only the latest attempt of each check.
type Check struct {
	Name       string
	Commit     CommitSHA
	Status     CheckStatus
	Conclusion CheckConclusion
	URL        string
	Job        *JobRef
	AppID      int64
	Attempt    int64
}

// RequiredCheck is a check that a branch requires to merge: its name, and the
// AppID of the app or integration that must report it. Its fields are
// unexported, so a value is built only by [NewRequiredCheck]. The zero value has
// no name, and [SummarizeRequiredChecks] refuses it with
// [ErrEmptyRequiredCheckName].
type RequiredCheck struct {
	name  string
	appID int64
}

// NewRequiredCheck returns the requirement of the check name, reported by the
// app appID. A zero appID accepts a check from any source, as a rule that names
// no integration does. It returns an error matching [ErrEmptyRequiredCheckName]
// when name is empty.
func NewRequiredCheck(name string, appID int64) (RequiredCheck, error) {
	if name == "" {
		return RequiredCheck{}, ErrEmptyRequiredCheckName
	}
	return RequiredCheck{name: name, appID: appID}, nil
}

// Name is the name of the required check.
func (r RequiredCheck) Name() string { return r.name }

// AppID is the app that must report the check, zero when any source does.
func (r RequiredCheck) AppID() int64 { return r.appID }

// RequiredChecksSource is the optional rule set of a forge, which a plugin
// implements when its forge reports the checks a branch requires to merge. A
// caller finds it with a type assertion, as with [PullRequestFeed].
//
// RequiredChecks reads the rulesets of repo first, then its classic branch
// protection, and returns the checks that either names on branch, merged. A rule
// set that does not exist is an absence, and a nil list with a nil error means
// that the absence of any requirement is proven. A read that is refused or fails
// with any other error returns an error wrapping [ErrRequiredChecksUnknown], and
// a caller must not read it as an empty list. It returns [ErrForgeWrongHost] for
// a repository on another host, as the methods of [Forge] do. Pass the result to
// [SummarizeRequiredChecks] with the checks of a commit.
type RequiredChecksSource interface {
	RequiredChecks(ctx context.Context, repo ForgeRepoRef, branch BranchName) ([]RequiredCheck, error)
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

// ReviewComment is one message of a review thread, in the order the forge holds
// them.
type ReviewComment struct {
	Author    ForgeActor
	Body      string
	CreatedAt time.Time
}

// Issue is the state of an issue as the forge reports it. State is
// [IssueStateUnknown] when the forge did not report one, and consumers treat it
// as not open. Author carries the association of the account toward the
// repository. CreatedAt and UpdatedAt are the times the forge reports, and the
// zero time where it reports none. IsPullRequest is set when the forge lists a
// pull request among its issues, as GitHub does; the plugin reports it and does
// not drop the pull request, so a caller that wants issues alone skips it. That
// a pull request stays listed with IsPullRequest set is proved by the behavior
// tests of the plugin, not by this contract.
type Issue struct {
	Ref           IssueRef
	Title         string
	Body          string
	URL           string
	Author        ForgeActor
	Labels        []string
	State         IssueState
	CreatedAt     time.Time
	UpdatedAt     time.Time
	IsPullRequest bool
}

// IssueQuery selects the issues of a listing: the labels every issue must
// carry, the login an issue must mention, the time from which an issue must
// have been updated, the state to list and the order. The zero value applies no
// filter and lists open issues in the forge's own order, the safe default of a
// listing. [IssueQuery.Listed] returns the state that is read, and
// [IssueQuery.Validate] refuses a value outside the closed sets.
//
// Mentioned selects the issues that mention the login, as the mentioned
// parameter of the REST issues endpoint describes it: a user mentioned in the
// issue. The contract does not fix what counts as a mention; the plugin states
// the meaning it gives to it in its documentation. A plugin whose forge cannot
// filter by mention refuses such a query with [ErrUnsupportedIssueFilter] rather
// than ignoring the login.
//
// UpdatedSince selects the issues updated at or after the instant. Implementations
// must include issues updated exactly at UpdatedSince; a forge whose API filters
// strictly after the bound must widen the request and filter locally. The zero
// time applies no filter.
type IssueQuery struct {
	Labels       []string
	Mentioned    ForgeLogin
	UpdatedSince time.Time
	State        IssueState
	Sort         IssueSort
	Direction    SortDirection
}

// Listed is the state a listing reads: [IssueOpen] when State is the zero
// value, and State otherwise.
func (q IssueQuery) Listed() IssueState {
	if q.State == IssueStateUnknown {
		return IssueOpen
	}
	return q.State
}

// IssueListing is one page of an issue listing read with a revision. Revision
// names the version of this read, built with [IssueListingRevision], to pass back
// as since on the next read of the same repository, query and page. Compare
// revisions only for equality, as with [Revision].
type IssueListing struct {
	Page     Page[Issue]
	Revision Revision
}

// ConditionalIssues is the optional interface of a forge that reports the
// revision of an issue listing, which a plugin implements beside [Forge]. A
// caller finds it with a type assertion and reads [Forge.Issues] otherwise.
//
// A since is valid only for the repository, the [IssueQuery] and the
// [PageRequest] of the read that produced it. A plugin builds each revision with
// [IssueListingRevision] and reads a since back with [IssueListingETag], which
// accepts only a revision built for the same three. A since that
// [IssueListingETag] does not accept, including the revision of another query or
// page, reads the listing in full, and IssuesSince never returns
// [ErrForgeNotModified] for it. A since it accepts returns [ErrForgeNotModified]
// when the listing has not changed. With an empty since, IssuesSince reads the
// listing as Issues does.
//
// IssuesSince returns [ErrForgeWrongHost] for a repository on another host,
// [ErrInvalidIssueQuery] for a query outside the closed sets,
// [ErrUnsupportedIssueFilter] for a filter the forge cannot apply, and
// [ErrInvalidPageLimit] for a limit outside its range.
type ConditionalIssues interface {
	IssuesSince(ctx context.Context, repo ForgeRepoRef, query IssueQuery, page PageRequest, since Revision) (IssueListing, error)
}
