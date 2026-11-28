package contracts

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseForgeLoginAcceptsALoginOfTheForgeAlphabet(t *testing.T) {
	for _, text := range []string{"octo", "Octo-Cat_2.x", "a", strings.Repeat("a", maxForgeLoginBytes)} {
		login, err := ParseForgeLogin(text)
		if err != nil {
			t.Fatalf("ParseForgeLogin(%q) error = %v, want nil", text, err)
		}
		if login.String() != text {
			t.Fatalf("ParseForgeLogin(%q).String() = %q, want the text back", text, login.String())
		}
	}
}

func TestParseForgeLoginRefusesTextThatNamesNoAccount(t *testing.T) {
	for _, text := range []string{"", "-octo", ".octo", "oc to", "octo@cat", "octo/cat", "octo\n", "é", strings.Repeat("a", maxForgeLoginBytes+1)} {
		_, err := ParseForgeLogin(text)
		if !errors.Is(err, ErrInvalidForgeLogin) {
			t.Fatalf("ParseForgeLogin(%q) error = %v, want ErrInvalidForgeLogin", text, err)
		}
	}
}

func TestZeroForgeLoginNamesNoAccount(t *testing.T) {
	var login ForgeLogin
	if !login.IsZero() || login.String() != "" {
		t.Fatalf("zero ForgeLogin = %q, IsZero() = %v, want an empty login", login.String(), login.IsZero())
	}
	parsed, err := ParseForgeLogin("octo")
	if err != nil || parsed.IsZero() {
		t.Fatalf("ParseForgeLogin(octo) = %v, %v, want a login that is not zero", parsed, err)
	}
}

func TestZeroIssueQueryIsValidAndListsOpenIssues(t *testing.T) {
	var query IssueQuery
	if err := query.Validate(); err != nil {
		t.Fatalf("zero IssueQuery.Validate() = %v, want nil", err)
	}
	if query.Listed() != IssueOpen {
		t.Fatalf("zero IssueQuery.Listed() = %v, want IssueOpen", query.Listed())
	}
}

func TestIssueQueryValidateAcceptsEachClosedValue(t *testing.T) {
	for state := IssueStateUnknown; state < issueStateEnd; state++ {
		for sort := IssueSortDefault; sort < issueSortEnd; sort++ {
			for direction := SortDirectionDefault; direction < sortDirectionEnd; direction++ {
				query := IssueQuery{State: state, Sort: sort, Direction: direction}
				if err := query.Validate(); err != nil {
					t.Fatalf("Validate(%v, %v, %v) = %v, want nil", state, sort, direction, err)
				}
			}
		}
	}
}

func TestIssueQueryValidateRefusesAValueOutsideItsClosedSet(t *testing.T) {
	tests := []struct {
		name  string
		query IssueQuery
	}{
		{"state below the set", IssueQuery{State: IssueState(-1)}},
		{"state above the set", IssueQuery{State: IssueClosed + 1}},
		{"sort below the set", IssueQuery{Sort: IssueSort(-1)}},
		{"sort above the set", IssueQuery{Sort: IssueSortUpdated + 1}},
		{"direction below the set", IssueQuery{Direction: SortDirection(-1)}},
		{"direction above the set", IssueQuery{Direction: SortDescending + 1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.query.Validate(); !errors.Is(err, ErrInvalidIssueQuery) {
				t.Fatalf("Validate() = %v, want ErrInvalidIssueQuery", err)
			}
		})
	}
}

func TestIssueSortAndDirectionNameEachValue(t *testing.T) {
	tests := []struct {
		name string
		got  fmt.Stringer
		want string
	}{
		{"sort default", IssueSortDefault, "default"},
		{"sort created", IssueSortCreated, "created"},
		{"sort updated", IssueSortUpdated, "updated"},
		{"sort unnamed", IssueSort(9), "IssueSort(9)"},
		{"direction default", SortDirectionDefault, "default"},
		{"direction ascending", SortAscending, "ascending"},
		{"direction descending", SortDescending, "descending"},
		{"direction unnamed", SortDirection(9), "SortDirection(9)"},
	}
	for _, test := range tests {
		if test.got.String() != test.want {
			t.Fatalf("%s: String() = %q, want %q", test.name, test.got.String(), test.want)
		}
	}
}

func TestEveryClosedValueOfAnIssueQueryIsNamed(t *testing.T) {
	for state := IssueStateUnknown; state < issueStateEnd; state++ {
		if strings.HasPrefix(state.String(), "IssueState(") {
			t.Fatalf("IssueState %d has no name: %s", int(state), state.String())
		}
	}
	for sort := IssueSortDefault; sort < issueSortEnd; sort++ {
		if strings.HasPrefix(sort.String(), "IssueSort(") {
			t.Fatalf("IssueSort %d has no name: %s", int(sort), sort.String())
		}
	}
	for direction := SortDirectionDefault; direction < sortDirectionEnd; direction++ {
		if strings.HasPrefix(direction.String(), "SortDirection(") {
			t.Fatalf("SortDirection %d has no name: %s", int(direction), direction.String())
		}
	}
}

func TestIssueListingETagAcceptsOnlyTheReadThatProducedIt(t *testing.T) {
	repo := mustRepo(t, "gitlab.example.com/octo/hello")
	other := mustRepo(t, "gitlab.example.com/octo/world")
	login := mustLogin(t, "octo")
	since := time.Date(2026, time.November, 27, 9, 0, 0, 0, time.UTC)
	query := IssueQuery{Labels: []string{"herrscher"}, Mentioned: login, UpdatedSince: since, Sort: IssueSortUpdated, Direction: SortAscending}
	page := PageRequest{Cursor: "next", Limit: 20}
	revision := IssueListingRevision(repo, query, page, "etag-1")

	etag, ok := IssueListingETag(revision, repo, query, page)
	if !ok || etag != "etag-1" {
		t.Fatalf("IssueListingETag of its own read = %q, %v, want etag-1, true", etag, ok)
	}
	changed := []struct {
		name  string
		repo  ForgeRepoRef
		query IssueQuery
		page  PageRequest
	}{
		{"another repository", other, query, page},
		{"another label", repo, IssueQuery{Labels: []string{"other"}, Mentioned: login, UpdatedSince: since, Sort: IssueSortUpdated, Direction: SortAscending}, page},
		{"no mention", repo, IssueQuery{Labels: query.Labels, UpdatedSince: since, Sort: IssueSortUpdated, Direction: SortAscending}, page},
		{"another instant", repo, IssueQuery{Labels: query.Labels, Mentioned: login, UpdatedSince: since.Add(time.Second), Sort: IssueSortUpdated, Direction: SortAscending}, page},
		{"another state", repo, IssueQuery{Labels: query.Labels, Mentioned: login, UpdatedSince: since, State: IssueClosed, Sort: IssueSortUpdated, Direction: SortAscending}, page},
		{"another sort", repo, IssueQuery{Labels: query.Labels, Mentioned: login, UpdatedSince: since, Sort: IssueSortCreated, Direction: SortAscending}, page},
		{"another direction", repo, IssueQuery{Labels: query.Labels, Mentioned: login, UpdatedSince: since, Sort: IssueSortUpdated, Direction: SortDescending}, page},
		{"another cursor", repo, query, PageRequest{Cursor: "other", Limit: 20}},
		{"another limit", repo, query, PageRequest{Cursor: "next", Limit: 10}},
	}
	for _, change := range changed {
		if _, ok := IssueListingETag(revision, change.repo, change.query, change.page); ok {
			t.Fatalf("revision of one read accepted for %s, want refused", change.name)
		}
	}
	if _, ok := IssueListingETag("", repo, query, page); ok {
		t.Fatalf("empty revision accepted, want refused")
	}
	if _, ok := IssueListingETag(Revision("not-a-revision"), repo, query, page); ok {
		t.Fatalf("revision without a binding accepted, want refused")
	}
}

func TestIssueListingETagKeepsAnETagThatHasASeparator(t *testing.T) {
	repo := mustRepo(t, "gitlab.example.com/octo/hello")
	revision := IssueListingRevision(repo, IssueQuery{}, PageRequest{}, "W/\"a/b\"")
	etag, ok := IssueListingETag(revision, repo, IssueQuery{}, PageRequest{})
	if !ok || etag != "W/\"a/b\"" {
		t.Fatalf("etag = %q, %v, want the etag back whole", etag, ok)
	}
}

func mustRepo(t *testing.T, text string) ForgeRepoRef {
	t.Helper()
	repo, err := ParseRepoRef(text)
	if err != nil {
		t.Fatalf("ParseRepoRef(%q) error = %v", text, err)
	}
	return repo
}

func mustLogin(t *testing.T, text string) ForgeLogin {
	t.Helper()
	login, err := ParseForgeLogin(text)
	if err != nil {
		t.Fatalf("ParseForgeLogin(%q) error = %v", text, err)
	}
	return login
}

func TestIssueListingRevisionWithoutAnETagIsTheZeroRevision(t *testing.T) {
	repo := mustRepo(t, "gitlab.example.com/octo/hello")
	revision := IssueListingRevision(repo, IssueQuery{}, PageRequest{}, "")
	if revision != Revision("") {
		t.Fatalf("revision = %q, want the zero Revision that reads the listing in full", revision)
	}
	if _, ok := IssueListingETag(revision, repo, IssueQuery{}, PageRequest{}); ok {
		t.Fatal("zero revision accepted as a binding, want it treated as absent")
	}
}
