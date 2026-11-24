package forgetest

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

const listedHost = "github.example.com"

type repositoryListing struct {
	fakeForge

	repos   []contracts.ForgeRepo
	lax     bool
	endless bool
}

func (l repositoryListing) Repositories(_ context.Context, page contracts.PageRequest) (contracts.Page[contracts.ForgeRepo], error) {
	if !l.lax {
		if err := page.Validate(); err != nil {
			return contracts.Page[contracts.ForgeRepo]{}, err
		}
	}
	if l.endless {
		return contracts.Page[contracts.ForgeRepo]{Next: "more"}, nil
	}
	start := 0
	if page.Cursor != "" {
		parsed, err := strconv.Atoi(string(page.Cursor))
		if err != nil {
			return contracts.Page[contracts.ForgeRepo]{}, err
		}
		start = parsed
	}
	size := page.Limit
	if size < 1 {
		size = 2
	}
	end := min(start+size, len(l.repos))
	var next contracts.PageCursor
	if end < len(l.repos) {
		next = contracts.PageCursor(strconv.Itoa(end))
	}
	return contracts.Page[contracts.ForgeRepo]{Items: l.repos[start:end], Next: next}, nil
}

func listedRepo(t *testing.T, repoText string) contracts.ForgeRepo {
	t.Helper()
	host, err := contracts.ParseForgeHost(listedHost)
	if err != nil {
		t.Fatalf("ParseForgeHost() error = %v", err)
	}
	repo, err := contracts.ParseRepoRef(repoText)
	if err != nil {
		t.Fatalf("ParseRepoRef(%q) error = %v", repoText, err)
	}
	listed, err := contracts.NewForgeRepo(host, repo, "https://"+repoText, "", false)
	if err != nil {
		t.Fatalf("NewForgeRepo(%q) error = %v", repoText, err)
	}
	return listed
}

func TestCheckListsRepositoriesPassesAPagedListing(t *testing.T) {
	rec := &recorder{}
	listing := repositoryListing{repos: []contracts.ForgeRepo{
		listedRepo(t, listedHost+"/octo/one"),
		listedRepo(t, listedHost+"/octo/two"),
		listedRepo(t, listedHost+"/octo/three"),
	}}
	host, err := contracts.ParseForgeHost(listedHost)
	if err != nil {
		t.Fatalf("ParseForgeHost() error = %v", err)
	}
	checkListsRepositories(rec, listing, host)
	if len(rec.failures) != 0 {
		t.Fatalf("paged listing reported failures: %q", rec.failures)
	}
}

func TestCheckListsRepositoriesPassesAnEmptyListing(t *testing.T) {
	rec := &recorder{}
	host, err := contracts.ParseForgeHost(listedHost)
	if err != nil {
		t.Fatalf("ParseForgeHost() error = %v", err)
	}
	checkListsRepositories(rec, repositoryListing{}, host)
	if len(rec.failures) != 0 {
		t.Fatalf("empty listing reported failures: %q", rec.failures)
	}
}

func TestCheckListsRepositoriesCatchesAListingThatIgnoresTheLimit(t *testing.T) {
	rec := &recorder{}
	host, err := contracts.ParseForgeHost(listedHost)
	if err != nil {
		t.Fatalf("ParseForgeHost() error = %v", err)
	}
	checkListsRepositories(rec, repositoryListing{lax: true}, host)
	if len(rec.failures) != 2 {
		t.Fatalf("listing that ignores the limit reported %d failures, want one per refused limit (2): %q", len(rec.failures), rec.failures)
	}
}

func TestCheckListsRepositoriesCatchesAZeroRepository(t *testing.T) {
	rec := &recorder{}
	host, err := contracts.ParseForgeHost(listedHost)
	if err != nil {
		t.Fatalf("ParseForgeHost() error = %v", err)
	}
	listing := repositoryListing{repos: []contracts.ForgeRepo{listedRepo(t, listedHost+"/octo/one"), {}}}
	checkListsRepositories(rec, listing, host)
	if len(rec.failures) != 1 {
		t.Fatalf("listing with a zero repository reported %d failures, want 1: %q", len(rec.failures), rec.failures)
	}
}

func TestCheckListsRepositoriesCatchesAListingThatNeverEnds(t *testing.T) {
	rec := &recorder{}
	host, err := contracts.ParseForgeHost(listedHost)
	if err != nil {
		t.Fatalf("ParseForgeHost() error = %v", err)
	}
	checkListsRepositories(rec, repositoryListing{endless: true}, host)
	if len(rec.failures) != 1 {
		t.Fatalf("endless listing reported %d failures, want 1: %q", len(rec.failures), rec.failures)
	}
}

type fakeForge struct {
	refuses bool
	stall   <-chan struct{}
}

func (f fakeForge) refusal() error {
	if f.stall != nil {
		<-f.stall
	}
	if f.refuses {
		return contracts.ErrForgeWrongHost
	}
	return nil
}

func (fakeForge) Manifest() contracts.Manifest { return contracts.Manifest{} }

func (fakeForge) Connection(context.Context) (contracts.ForgeConnection, error) {
	return contracts.ForgeConnection{}, nil
}

func (f fakeForge) RepoAccess(context.Context, contracts.ForgeRepoRef) (contracts.RepoPermission, error) {
	return contracts.PermissionNone, f.refusal()
}

func (fakeForge) Repositories(context.Context, contracts.PageRequest) (contracts.Page[contracts.ForgeRepo], error) {
	return contracts.Page[contracts.ForgeRepo]{}, nil
}

func (f fakeForge) OpenPullRequest(context.Context, contracts.PullRequestDraft) (contracts.PullRequest, error) {
	return contracts.PullRequest{}, f.refusal()
}

func (f fakeForge) PullRequest(context.Context, contracts.PullRequestRef, contracts.Revision) (contracts.PullRequest, error) {
	return contracts.PullRequest{}, f.refusal()
}

func (f fakeForge) MergePullRequest(context.Context, contracts.PullRequestRef, contracts.MergeMethod) error {
	return f.refusal()
}

func (f fakeForge) Checks(context.Context, contracts.ForgeRepoRef, contracts.CommitSHA, contracts.PageRequest) (contracts.Page[contracts.Check], error) {
	return contracts.Page[contracts.Check]{}, f.refusal()
}

func (f fakeForge) JobLog(context.Context, contracts.JobRef, int) ([]byte, error) {
	return nil, f.refusal()
}

func (f fakeForge) ReviewThreads(context.Context, contracts.PullRequestRef, contracts.PageRequest) (contracts.Page[contracts.ReviewThread], error) {
	return contracts.Page[contracts.ReviewThread]{}, f.refusal()
}

func (f fakeForge) ReplyToThread(context.Context, contracts.ThreadRef, string) error {
	return f.refusal()
}

func (f fakeForge) Issue(context.Context, contracts.IssueRef) (contracts.Issue, error) {
	return contracts.Issue{}, f.refusal()
}

func (f fakeForge) Issues(context.Context, contracts.ForgeRepoRef, contracts.IssueQuery, contracts.PageRequest) (contracts.Page[contracts.Issue], error) {
	return contracts.Page[contracts.Issue]{}, f.refusal()
}

type recorder struct {
	failures []string
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func foreignRepo(t *testing.T) contracts.ForgeRepoRef {
	t.Helper()
	repo, err := contracts.ParseRepoRef("gitlab.example.com/octo/hello")
	if err != nil {
		t.Fatalf("ParseRepoRef() error = %v", err)
	}
	return repo
}

func TestCheckRefusesForeignHostPassesAForgeThatRefuses(t *testing.T) {
	CheckRefusesForeignHost(t, fakeForge{refuses: true}, foreignRepo(t))
}

func TestCheckRefusesForeignHostCatchesAForgetfulForge(t *testing.T) {
	rec := &recorder{}
	checkRefusesForeignHost(rec, fakeForge{}, foreignRepo(t), callTimeout)
	if len(rec.failures) != 10 {
		t.Fatalf("forgetful forge reported %d failures, want one per repository method (10): %q", len(rec.failures), rec.failures)
	}
}

func TestCheckRefusesForeignHostIsSilentForAGuardedForge(t *testing.T) {
	rec := &recorder{}
	checkRefusesForeignHost(rec, fakeForge{refuses: true}, foreignRepo(t), callTimeout)
	if len(rec.failures) != 0 {
		t.Fatalf("guarded forge reported failures: %q", rec.failures)
	}
}

func TestCheckRefusesForeignHostRefusesAZeroForeignRepository(t *testing.T) {
	rec := &recorder{}
	checkRefusesForeignHost(rec, fakeForge{refuses: true}, contracts.ForgeRepoRef{}, callTimeout)
	if len(rec.failures) != 1 {
		t.Fatalf("zero foreign repository reported %d failures, want 1: %q", len(rec.failures), rec.failures)
	}
}

func TestCheckRefusesForeignHostReportsACallThatIgnoresItsDeadline(t *testing.T) {
	stall := make(chan struct{})
	defer close(stall)
	rec := &recorder{}
	start := time.Now()
	checkRefusesForeignHost(rec, fakeForge{refuses: true, stall: stall}, foreignRepo(t), 20*time.Millisecond)
	if len(rec.failures) != 10 {
		t.Fatalf("stalled forge reported %d failures, want one per repository method (10): %q", len(rec.failures), rec.failures)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("check took %v on a forge that ignores its context, want it bounded by the deadline", elapsed)
	}
}

func TestCheckRefusesForeignHostGivesEachCallTenSeconds(t *testing.T) {
	if callTimeout != 10*time.Second {
		t.Fatalf("callTimeout = %v, want the documented ten seconds", callTimeout)
	}
}

func TestCheckRefusesForeignHostLetsALateCallFinish(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stall := make(chan struct{})
		rec := &recorder{}
		checkRefusesForeignHost(rec, fakeForge{refuses: true, stall: stall}, foreignRepo(t), time.Millisecond)
		close(stall)
		synctest.Wait()
		if len(rec.failures) != 10 {
			t.Fatalf("late forge reported %d failures, want one per repository method (10): %q", len(rec.failures), rec.failures)
		}
	})
}
