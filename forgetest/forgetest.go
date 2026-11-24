// Package forgetest holds the checks a forge plugin's tests run against its
// implementation of [contracts.Forge]. It is test support: import it from
// _test.go files only. It departs from the rule that every goroutine is joined
// by its owner: a call that outlives its deadline and ignores its context
// cannot be stopped from Go, so the check reports it and leaves its goroutine
// running rather than block the whole test binary.
package forgetest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

type reporter interface {
	Helper()
	Errorf(format string, args ...any)
}

type foreignRefs struct {
	commit contracts.CommitSHA
	pull   contracts.PullRequestRef
	issue  contracts.IssueRef
	job    contracts.JobRef
	thread contracts.ThreadRef
}

type foreignCall struct {
	name string
	run  func(ctx context.Context) error
}

// CheckRefusesForeignHost calls every method of forge that takes a repository,
// a pull request, a job, a thread or an issue, each with a reference on the
// foreign host, and fails the test for each call that does not return an error
// matching [contracts.ErrForgeWrongHost]. foreign must name a repository on a
// host other than the one forge serves. Each call gets ten seconds: a call
// still running then is reported as a failure and left running, whether or not
// it watches its context. The check proves only the refusal: a forge that
// refuses every call passes it, so the plugin's own tests prove the calls on
// its own host.
func CheckRefusesForeignHost(tb testing.TB, forge contracts.Forge, foreign contracts.ForgeRepoRef) {
	tb.Helper()
	checkRefusesForeignHost(tb, forge, foreign, callTimeout)
}

func checkRefusesForeignHost(report reporter, forge contracts.Forge, foreign contracts.ForgeRepoRef, timeout time.Duration) {
	report.Helper()
	refs, err := newForeignRefs(foreign)
	if err != nil {
		report.Errorf("forgetest: foreign repository %v: %v", foreign, err)
		return
	}
	for _, call := range foreignCalls(forge, foreign, refs) {
		callErr := runWithDeadline(call, timeout)
		if !errors.Is(callErr, contracts.ErrForgeWrongHost) {
			report.Errorf("%s with a repository on another host: error = %v, want ErrForgeWrongHost", call.name, callErr)
		}
	}
}

func runWithDeadline(call foreignCall, timeout time.Duration) error {
	_, err := withDeadline(timeout, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, call.run(ctx)
	})
	return err
}

type outcome[Value any] struct {
	value Value
	err   error
}

func withDeadline[Value any](timeout time.Duration, run func(ctx context.Context) (Value, error)) (Value, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result := make(chan outcome[Value], 1)
	go func() {
		value, err := run(ctx)
		result <- outcome[Value]{value: value, err: err}
	}()
	select {
	case done := <-result:
		return done.value, done.err
	case <-ctx.Done():
		var none Value
		return none, fmt.Errorf("no answer within %v: %w", timeout, ctx.Err())
	}
}

// CheckListsRepositories lists the repositories of forge one at a time, and
// fails the test unless the listing refuses a limit below 0 or above
// [contracts.MaxPageLimit] with [contracts.ErrInvalidPageLimit], ends with an
// empty Next, and holds only repositories on host, the forge host the plugin
// serves. A listing that never ends fails the check after a bounded number of
// pages, and each page gets the same ten seconds as the other calls.
func CheckListsRepositories(tb testing.TB, forge contracts.Forge, host contracts.ForgeHost) {
	tb.Helper()
	checkListsRepositories(tb, forge, host)
}

const maxListedPages = 100

func checkListsRepositories(report reporter, forge contracts.Forge, host contracts.ForgeHost) {
	report.Helper()
	for _, limit := range []int{-1, contracts.MaxPageLimit + 1} {
		_, err := listRepositoryPage(forge, contracts.PageRequest{Limit: limit})
		if !errors.Is(err, contracts.ErrInvalidPageLimit) {
			report.Errorf("Repositories with limit %d: error = %v, want ErrInvalidPageLimit", limit, err)
		}
	}
	var cursor contracts.PageCursor
	for range maxListedPages {
		page, err := listRepositoryPage(forge, contracts.PageRequest{Cursor: cursor, Limit: 1})
		if err != nil {
			report.Errorf("Repositories: error = %v, want a page", err)
			return
		}
		for _, repo := range page.Items {
			if repo.Repo().Host() != host {
				report.Errorf("Repositories listed %q, want only repositories on %v", repo.Repo(), host)
			}
		}
		if page.Next == "" {
			return
		}
		cursor = page.Next
	}
	report.Errorf("Repositories: the listing did not end within %d pages", maxListedPages)
}

func listRepositoryPage(forge contracts.Forge, page contracts.PageRequest) (contracts.Page[contracts.ForgeRepo], error) {
	return withDeadline(callTimeout, func(ctx context.Context) (contracts.Page[contracts.ForgeRepo], error) {
		return forge.Repositories(ctx, page)
	})
}

func newForeignRefs(foreign contracts.ForgeRepoRef) (foreignRefs, error) {
	commit, err := contracts.ParseCommitSHA(foreignCommit)
	if err != nil {
		return foreignRefs{}, err
	}
	pull, err := contracts.NewPullRequestRef(foreign, 1)
	if err != nil {
		return foreignRefs{}, err
	}
	issue, err := contracts.NewIssueRef(foreign, 1)
	if err != nil {
		return foreignRefs{}, err
	}
	job, err := contracts.NewJobRef(foreign, 1)
	if err != nil {
		return foreignRefs{}, err
	}
	thread, err := contracts.NewThreadRef(pull, "foreign-thread")
	if err != nil {
		return foreignRefs{}, err
	}
	return foreignRefs{commit: commit, pull: pull, issue: issue, job: job, thread: thread}, nil
}

const callTimeout = 10 * time.Second

const foreignCommit = "0000000000000000000000000000000000000000"

func foreignCalls(forge contracts.Forge, foreign contracts.ForgeRepoRef, refs foreignRefs) []foreignCall {
	return []foreignCall{
		{"RepoAccess", func(ctx context.Context) error {
			_, err := forge.RepoAccess(ctx, foreign)
			return err
		}},
		{"OpenPullRequest", func(ctx context.Context) error {
			_, err := forge.OpenPullRequest(ctx, contracts.PullRequestDraft{Repo: foreign, Head: "topic", Base: "main", Title: "t"})
			return err
		}},
		{"PullRequest", func(ctx context.Context) error {
			_, err := forge.PullRequest(ctx, refs.pull, "")
			return err
		}},
		{"MergePullRequest", func(ctx context.Context) error {
			return forge.MergePullRequest(ctx, refs.pull, contracts.MergeMethodMerge)
		}},
		{"Checks", func(ctx context.Context) error {
			_, err := forge.Checks(ctx, foreign, refs.commit, contracts.PageRequest{})
			return err
		}},
		{"JobLog", func(ctx context.Context) error {
			_, err := forge.JobLog(ctx, refs.job, 1)
			return err
		}},
		{"ReviewThreads", func(ctx context.Context) error {
			_, err := forge.ReviewThreads(ctx, refs.pull, contracts.PageRequest{})
			return err
		}},
		{"ReplyToThread", func(ctx context.Context) error {
			return forge.ReplyToThread(ctx, refs.thread, "reply")
		}},
		{"Issue", func(ctx context.Context) error {
			_, err := forge.Issue(ctx, refs.issue)
			return err
		}},
		{"Issues", func(ctx context.Context) error {
			_, err := forge.Issues(ctx, foreign, contracts.IssueQuery{}, contracts.PageRequest{})
			return err
		}},
	}
}
