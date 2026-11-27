package forgetest

import (
	"context"
	"errors"
	"testing"
	"time"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

const outsideClosedSet = 99

// CheckConditionalIssuesRefusesInvalidInput calls IssuesSince of lister on repo
// with a query outside its closed sets and with a page limit outside its range,
// and fails the test unless the first returns an error matching
// [contracts.ErrInvalidIssueQuery] and the second one matching
// [contracts.ErrInvalidPageLimit]. A plugin's own test proves that no request
// left, since only it sees its transport.
func CheckConditionalIssuesRefusesInvalidInput(tb testing.TB, lister contracts.ConditionalIssues, repo contracts.ForgeRepoRef) {
	tb.Helper()
	checkConditionalIssuesRefusesInvalidInput(tb, lister, repo, callTimeout)
}

func checkConditionalIssuesRefusesInvalidInput(report reporter, lister contracts.ConditionalIssues, repo contracts.ForgeRepoRef, timeout time.Duration) {
	report.Helper()
	calls := []struct {
		name  string
		query contracts.IssueQuery
		page  contracts.PageRequest
		want  error
	}{
		{"state outside its set", contracts.IssueQuery{State: contracts.IssueState(outsideClosedSet)}, contracts.PageRequest{}, contracts.ErrInvalidIssueQuery},
		{"sort outside its set", contracts.IssueQuery{Sort: contracts.IssueSort(outsideClosedSet)}, contracts.PageRequest{}, contracts.ErrInvalidIssueQuery},
		{"direction outside its set", contracts.IssueQuery{Direction: contracts.SortDirection(outsideClosedSet)}, contracts.PageRequest{}, contracts.ErrInvalidIssueQuery},
		{"limit above the range", contracts.IssueQuery{}, contracts.PageRequest{Limit: contracts.MaxPageLimit + 1}, contracts.ErrInvalidPageLimit},
	}
	for i := range calls {
		call := &calls[i]
		_, err := withDeadline(context.Background(), timeout, func(ctx context.Context) (contracts.IssueListing, error) {
			return lister.IssuesSince(ctx, repo, call.query, call.page, "")
		})
		if !errors.Is(err, call.want) {
			report.Errorf("IssuesSince with %s: error = %v, want %v", call.name, err, call.want)
		}
	}
}

// CheckConditionalIssuesBindsRevisions reads the first page of one query of
// lister on repo, then passes its revision back as since on a different query
// and on a different page of the same query. It fails the test unless neither
// read returns [contracts.ErrForgeNotModified], which would mean the plugin
// takes a revision for a read it does not identify. A plugin's own test proves
// that an unchanged listing of the same read is reported as not modified.
func CheckConditionalIssuesBindsRevisions(tb testing.TB, lister contracts.ConditionalIssues, repo contracts.ForgeRepoRef) {
	tb.Helper()
	checkConditionalIssuesBindsRevisions(tb, lister, repo, callTimeout)
}

func checkConditionalIssuesBindsRevisions(report reporter, lister contracts.ConditionalIssues, repo contracts.ForgeRepoRef, timeout time.Duration) {
	report.Helper()
	first := contracts.IssueQuery{Labels: []string{"herrscher"}}
	firstPage := contracts.PageRequest{}
	listing, err := readIssueListing(lister, repo, first, firstPage, "", timeout)
	if err != nil {
		report.Errorf("IssuesSince without a revision: error = %v, want a listing", err)
		return
	}
	if listing.Revision == "" {
		report.Errorf("IssuesSince without a revision returned an empty Revision, want one to bind")
		return
	}
	reads := []struct {
		name  string
		query contracts.IssueQuery
		page  contracts.PageRequest
	}{
		{"another query", contracts.IssueQuery{Labels: []string{"other"}}, firstPage},
		{"another page", first, contracts.PageRequest{Limit: 1}},
	}
	for i := range reads {
		read := &reads[i]
		_, err := readIssueListing(lister, repo, read.query, read.page, listing.Revision, timeout)
		if errors.Is(err, contracts.ErrForgeNotModified) {
			report.Errorf("IssuesSince with the revision of another read (%s): error = %v, want the listing read in full", read.name, err)
		}
	}
}

func readIssueListing(lister contracts.ConditionalIssues, repo contracts.ForgeRepoRef, query contracts.IssueQuery, page contracts.PageRequest, since contracts.Revision, timeout time.Duration) (contracts.IssueListing, error) {
	return withDeadline(context.Background(), timeout, func(ctx context.Context) (contracts.IssueListing, error) {
		return lister.IssuesSince(ctx, repo, query, page, since)
	})
}
