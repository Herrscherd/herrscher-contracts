// Package forgetest holds the checks a forge plugin's tests run against its
// implementation of [contracts.Forge]. It is test support: import it from _test.go
// files only.
package forgetest

import (
	"context"
	"errors"
	"testing"

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

// CheckRefusesForeignHost calls every method of forge that takes a repository, a pull
// request, a job, a thread or an issue, each with a reference on the foreign host,
// and fails the test for each call that does not return an error matching
// [contracts.ErrForgeWrongHost]. foreign must name a repository on a host other than
// the one forge serves.
func CheckRefusesForeignHost(tb testing.TB, forge contracts.Forge, foreign contracts.ForgeRepoRef) {
	tb.Helper()
	checkRefusesForeignHost(tb, forge, foreign)
}

func checkRefusesForeignHost(report reporter, forge contracts.Forge, foreign contracts.ForgeRepoRef) {
	report.Helper()
	refs, err := newForeignRefs(foreign)
	if err != nil {
		report.Errorf("forgetest: foreign repository %v: %v", foreign, err)
		return
	}
	for _, call := range foreignCalls(forge, foreign, refs) {
		callErr := call.run(context.Background())
		if !errors.Is(callErr, contracts.ErrForgeWrongHost) {
			report.Errorf("%s with a repository on another host: error = %v, want ErrForgeWrongHost", call.name, callErr)
		}
	}
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
