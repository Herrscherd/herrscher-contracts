package contracts

import "fmt"

// MaxJobLogBytes is the largest log tail [Forge.JobLog] returns, in bytes.
const MaxJobLogBytes = 1 << 20

// PullRequestRef names one pull request of a repository by its number. Build
// one with [NewPullRequestRef]; the zero value names none.
type PullRequestRef struct {
	repo   ForgeRepoRef
	number int
}

// NewPullRequestRef names pull request number of repo. It returns an error
// matching [ErrInvalidRef] when the repository is the zero value or the number
// is not positive.
func NewPullRequestRef(repo ForgeRepoRef, number int) (PullRequestRef, error) {
	if err := checkRepoNumber(repo, int64(number)); err != nil {
		return PullRequestRef{}, err
	}
	return PullRequestRef{repo: repo, number: number}, nil
}

// Repo is the repository the pull request belongs to.
func (r PullRequestRef) Repo() ForgeRepoRef { return r.repo }

// Number is the forge's number of the pull request.
func (r PullRequestRef) Number() int { return r.number }

// IssueRef names one issue of a repository by its number. Build one with
// [NewIssueRef]; the zero value names none.
type IssueRef struct {
	repo   ForgeRepoRef
	number int
}

// NewIssueRef names issue number of repo. It returns an error matching
// [ErrInvalidRef] when the repository is the zero value or the number is not
// positive.
func NewIssueRef(repo ForgeRepoRef, number int) (IssueRef, error) {
	if err := checkRepoNumber(repo, int64(number)); err != nil {
		return IssueRef{}, err
	}
	return IssueRef{repo: repo, number: number}, nil
}

// Repo is the repository the issue belongs to.
func (r IssueRef) Repo() ForgeRepoRef { return r.repo }

// Number is the forge's number of the issue.
func (r IssueRef) Number() int { return r.number }

// JobRef names one CI job of a repository by the forge's own identifier. Build
// one with [NewJobRef]; the zero value names none.
type JobRef struct {
	repo ForgeRepoRef
	id   int64
}

// NewJobRef names the CI job id of repo. It returns an error matching
// [ErrInvalidRef] when the repository is the zero value or the identifier is
// not positive.
func NewJobRef(repo ForgeRepoRef, jobID int64) (JobRef, error) {
	if err := checkRepoNumber(repo, jobID); err != nil {
		return JobRef{}, err
	}
	return JobRef{repo: repo, id: jobID}, nil
}

// Repo is the repository the job belongs to.
func (j JobRef) Repo() ForgeRepoRef { return j.repo }

// ID is the forge's identifier of the job.
func (j JobRef) ID() int64 { return j.id }

// ThreadRef names one review thread of a pull request by the forge's
// identifier. Build one with [NewThreadRef]; the zero value names none.
type ThreadRef struct {
	pullRequest PullRequestRef
	id          string
}

// NewThreadRef names the review thread id of pullRequest. It returns an error
// matching [ErrInvalidRef] when the pull request is the zero value or the
// identifier is empty.
func NewThreadRef(pullRequest PullRequestRef, threadID string) (ThreadRef, error) {
	if pullRequest == (PullRequestRef{}) || threadID == "" {
		return ThreadRef{}, ErrInvalidRef
	}
	return ThreadRef{pullRequest: pullRequest, id: threadID}, nil
}

// PullRequest is the pull request the thread belongs to.
func (t ThreadRef) PullRequest() PullRequestRef { return t.pullRequest }

// ID is the forge's identifier of the thread.
func (t ThreadRef) ID() string { return t.id }

// ValidateLogLimit checks the size a caller asks [Forge.JobLog] for. It returns
// an error matching [ErrInvalidLogLimit] when maxBytes is below 1 or above
// [MaxJobLogBytes].
func ValidateLogLimit(maxBytes int) error {
	if maxBytes < 1 || maxBytes > MaxJobLogBytes {
		return fmt.Errorf("%d: %w", maxBytes, ErrInvalidLogLimit)
	}
	return nil
}

func checkRepoNumber(repo ForgeRepoRef, number int64) error {
	if repo == (ForgeRepoRef{}) || number <= 0 {
		return ErrInvalidRef
	}
	return nil
}
