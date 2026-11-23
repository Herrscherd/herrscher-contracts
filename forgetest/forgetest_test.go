package forgetest

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

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
	baseline := runtime.NumGoroutine()
	stall := make(chan struct{})
	rec := &recorder{}
	checkRefusesForeignHost(rec, fakeForge{refuses: true, stall: stall}, foreignRepo(t), time.Millisecond)
	close(stall)
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > baseline {
		if time.Now().After(deadline) {
			t.Fatalf("goroutines = %d after the late calls answered, want %d: a late answer must not block", runtime.NumGoroutine(), baseline)
		}
		time.Sleep(time.Millisecond)
	}
}
