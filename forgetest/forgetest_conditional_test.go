package forgetest

import (
	"context"
	"strings"
	"testing"

	contracts "github.com/Herrscherd/herrscher-contracts"
)

type listingForge struct {
	fakeForge
}

func (f listingForge) IssuesSince(context.Context, contracts.ForgeRepoRef, contracts.IssueQuery, contracts.PageRequest, contracts.Revision) (contracts.IssueListing, error) {
	return contracts.IssueListing{}, f.refusal()
}

func TestCheckRefusesForeignHostAlsoChecksTheConditionalListingOfAForgeThatHasIt(t *testing.T) {
	rec := &recorder{}
	checkRefusesForeignHost(rec, listingForge{}, foreignRepo(t), callTimeout)
	if len(rec.failures) != 11 {
		t.Fatalf("forgetful listing forge reported %d failures, want the ten reads and IssuesSince: %q", len(rec.failures), rec.failures)
	}
	if !strings.Contains(rec.failures[10], "IssuesSince") {
		t.Fatalf("last failure %q does not name IssuesSince", rec.failures[10])
	}
	CheckRefusesForeignHost(t, listingForge{fakeForge: fakeForge{refuses: true}}, foreignRepo(t))
}

type bindingLister struct {
	ignoresBinding bool
}

func (f bindingLister) IssuesSince(_ context.Context, repo contracts.ForgeRepoRef, query contracts.IssueQuery, page contracts.PageRequest, since contracts.Revision) (contracts.IssueListing, error) {
	if err := query.Validate(); err != nil {
		return contracts.IssueListing{}, err
	}
	if err := page.Validate(); err != nil {
		return contracts.IssueListing{}, err
	}
	if since != "" && (f.ignoresBinding || validSince(since, repo, query, page)) {
		return contracts.IssueListing{}, contracts.ErrForgeNotModified
	}
	return contracts.IssueListing{Revision: contracts.IssueListingRevision(repo, query, page, "etag")}, nil
}

func validSince(since contracts.Revision, repo contracts.ForgeRepoRef, query contracts.IssueQuery, page contracts.PageRequest) bool {
	_, ok := contracts.IssueListingETag(since, repo, query, page)
	return ok
}

type unvalidatedLister struct{}

func (unvalidatedLister) IssuesSince(context.Context, contracts.ForgeRepoRef, contracts.IssueQuery, contracts.PageRequest, contracts.Revision) (contracts.IssueListing, error) {
	return contracts.IssueListing{}, nil
}

func TestCheckConditionalIssuesBindsRevisionsPassesAForgeThatBindsThem(t *testing.T) {
	CheckConditionalIssuesBindsRevisions(t, bindingLister{}, foreignRepo(t))
}

func TestCheckConditionalIssuesBindsRevisionsCatchesAForgeThatIgnoresTheBinding(t *testing.T) {
	rec := &recorder{}
	checkConditionalIssuesBindsRevisions(rec, bindingLister{ignoresBinding: true}, foreignRepo(t), callTimeout)
	if len(rec.failures) != 2 {
		t.Fatalf("binding-blind forge reported %d failures, want the other query and the other page: %q", len(rec.failures), rec.failures)
	}
}

func TestCheckConditionalIssuesRefusesInvalidInputPassesAValidatingForge(t *testing.T) {
	CheckConditionalIssuesRefusesInvalidInput(t, bindingLister{}, foreignRepo(t))
}

func TestCheckConditionalIssuesRefusesInvalidInputCatchesAForgeThatAcceptsIt(t *testing.T) {
	rec := &recorder{}
	checkConditionalIssuesRefusesInvalidInput(rec, unvalidatedLister{}, foreignRepo(t), callTimeout)
	if len(rec.failures) != 4 {
		t.Fatalf("unvalidated forge reported %d failures, want each of the four invalid calls: %q", len(rec.failures), rec.failures)
	}
}
