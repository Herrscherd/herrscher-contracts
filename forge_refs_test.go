package contracts

import (
	"errors"
	"testing"
)

func TestNewRefsRefuseANonPositiveNumberOrNoRepo(t *testing.T) {
	repo := mustOctoHelloRepo(t)
	for _, number := range []int{0, -1} {
		if _, err := NewPullRequestRef(repo, number); !errors.Is(err, ErrInvalidRef) {
			t.Errorf("NewPullRequestRef(%d) error = %v, want ErrInvalidRef", number, err)
		}
		if _, err := NewIssueRef(repo, number); !errors.Is(err, ErrInvalidRef) {
			t.Errorf("NewIssueRef(%d) error = %v, want ErrInvalidRef", number, err)
		}
	}
	if _, err := NewJobRef(repo, 0); !errors.Is(err, ErrInvalidRef) {
		t.Errorf("NewJobRef(0) error = %v, want ErrInvalidRef", err)
	}
	if _, err := NewPullRequestRef(ForgeRepoRef{}, 1); !errors.Is(err, ErrInvalidRef) {
		t.Errorf("NewPullRequestRef(zero repo) error = %v, want ErrInvalidRef", err)
	}
	if _, err := NewIssueRef(ForgeRepoRef{}, 1); !errors.Is(err, ErrInvalidRef) {
		t.Errorf("NewIssueRef(zero repo) error = %v, want ErrInvalidRef", err)
	}
	if _, err := NewJobRef(ForgeRepoRef{}, 1); !errors.Is(err, ErrInvalidRef) {
		t.Errorf("NewJobRef(zero repo) error = %v, want ErrInvalidRef", err)
	}
}

func TestNewRefsKeepWhatTheyNameAndAccessorsReturnIt(t *testing.T) {
	repo := mustOctoHelloRepo(t)

	pull, err := NewPullRequestRef(repo, 7)
	if err != nil || pull.Repo() != repo || pull.Number() != 7 {
		t.Fatalf("NewPullRequestRef(7) = %+v, %v, want the repo and number back", pull, err)
	}
	issue, err := NewIssueRef(repo, 9)
	if err != nil || issue.Repo() != repo || issue.Number() != 9 {
		t.Fatalf("NewIssueRef(9) = %+v, %v, want the repo and number back", issue, err)
	}
	job, err := NewJobRef(repo, 1<<40)
	if err != nil || job.Repo() != repo || job.ID() != 1<<40 {
		t.Fatalf("NewJobRef(1<<40) = %+v, %v, want the repo and identifier back", job, err)
	}
}

func TestNewThreadRefRefusesAnEmptyIdentifierOrNoPullRequest(t *testing.T) {
	pull, err := NewPullRequestRef(mustOctoHelloRepo(t), 3)
	if err != nil {
		t.Fatalf("NewPullRequestRef() error = %v", err)
	}
	if _, err = NewThreadRef(pull, ""); !errors.Is(err, ErrInvalidRef) {
		t.Errorf("NewThreadRef(empty id) error = %v, want ErrInvalidRef", err)
	}
	if _, err = NewThreadRef(PullRequestRef{}, "PRRT_1"); !errors.Is(err, ErrInvalidRef) {
		t.Errorf("NewThreadRef(zero pull request) error = %v, want ErrInvalidRef", err)
	}
	thread, err := NewThreadRef(pull, "PRRT_1")
	if err != nil || thread.PullRequest() != pull || thread.ID() != "PRRT_1" {
		t.Fatalf("NewThreadRef() = %+v, %v, want the pull request and identifier back", thread, err)
	}
}

func TestValidateLogLimitAcceptsTheRange(t *testing.T) {
	for _, limit := range []int{1, 4096, MaxJobLogBytes} {
		if err := ValidateLogLimit(limit); err != nil {
			t.Errorf("ValidateLogLimit(%d) error = %v, want none", limit, err)
		}
	}
}

func TestValidateLogLimitRefusesOutsideTheRange(t *testing.T) {
	for _, limit := range []int{0, -1, MaxJobLogBytes + 1} {
		if err := ValidateLogLimit(limit); !errors.Is(err, ErrInvalidLogLimit) {
			t.Errorf("ValidateLogLimit(%d) error = %v, want ErrInvalidLogLimit", limit, err)
		}
	}
}
