package contracts

import (
	"errors"
	"strings"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseBranchNameAcceptsValidNames(t *testing.T) {
	for _, text := range []string{
		"main",
		"release/1.2",
		"feature/x-y",
		"a",
		"v1.2.3",
		"feature/no.lockfile",
		strings.Repeat("b", maxBranchNameBytes),
	} {
		branch, err := ParseBranchName(text)
		if err != nil {
			t.Errorf("ParseBranchName(%q) error = %v, want none", text, err)
			continue
		}
		if branch.String() != text || branch.IsZero() {
			t.Errorf("ParseBranchName(%q) = %q, IsZero() = %v, want the name back", text, branch.String(), branch.IsZero())
		}
	}
}

func TestParseBranchNameRefusesHostileNames(t *testing.T) {
	for _, text := range []string{
		"",
		"@",
		"HEAD",
		"--upload-pack=touch /tmp/x",
		"-",
		"-x",
		"/main",
		"main/",
		"release//1",
		"main.",
		"main.lock",
		"feature.lock/x",
		"a..b",
		"a@{b",
		".hidden",
		"a/.b",
		"a/",
		"a b",
		"a\tb",
		"a\nb",
		"a\x00b",
		"a\x1fb",
		"a\x7fb",
		"a~1",
		"a^",
		"a:b",
		"a?",
		"a*",
		"a[b",
		`a\b`,
		strings.Repeat("b", maxBranchNameBytes+1),
	} {
		branch, err := ParseBranchName(text)
		if !errors.Is(err, ErrInvalidBranchName) {
			t.Errorf("ParseBranchName(%q) error = %v, want ErrInvalidBranchName", text, err)
		}
		if !branch.IsZero() {
			t.Errorf("ParseBranchName(%q) = %q, want the zero branch on error", text, branch.String())
		}
	}
}

func TestParseBranchNameRefusalRepeatsNoPartOfTheText(t *testing.T) {
	_, err := ParseBranchName("--" + receivedProbe + " /x")
	echotest.RequireRefusalWithoutEcho(t, err, ErrInvalidBranchName, receivedProbe)
}

func TestBranchNameZeroValueNamesNoBranch(t *testing.T) {
	var zero BranchName
	if !zero.IsZero() || zero.String() != "" {
		t.Fatalf("zero BranchName = %q, IsZero() = %v, want empty and true", zero.String(), zero.IsZero())
	}
}

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
	for _, limit := range []int{1, 4096, 1048576} {
		if err := ValidateLogLimit(limit); err != nil {
			t.Errorf("ValidateLogLimit(%d) error = %v, want none", limit, err)
		}
	}
}

func TestValidateLogLimitRefusesOutsideTheRange(t *testing.T) {
	for _, limit := range []int{0, -1, 1048577} {
		if err := ValidateLogLimit(limit); !errors.Is(err, ErrInvalidLogLimit) {
			t.Errorf("ValidateLogLimit(%d) error = %v, want ErrInvalidLogLimit", limit, err)
		}
	}
}
