package contracts

import (
	"errors"
	"fmt"
	"strings"
	"testing"
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
	for state := IssueStateUnknown; state <= IssueClosed; state++ {
		for sort := IssueSortDefault; sort <= IssueSortUpdated; sort++ {
			for direction := SortDirectionDefault; direction <= SortDescending; direction++ {
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
