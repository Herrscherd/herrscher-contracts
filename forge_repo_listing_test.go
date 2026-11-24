package contracts

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func listingHost(t *testing.T) ForgeHost {
	t.Helper()
	host, err := ParseForgeHost("github.example.com")
	if err != nil {
		t.Fatalf("ParseForgeHost() error = %v", err)
	}
	return host
}

func listingRepo(t *testing.T, text string) ForgeRepoRef {
	t.Helper()
	repo, err := ParseRepoRef(text)
	if err != nil {
		t.Fatalf("ParseRepoRef(%q) error = %v", text, err)
	}
	return repo
}

func TestNewForgeRepoKeepsWhatItIsGiven(t *testing.T) {
	host := listingHost(t)
	repo := listingRepo(t, "github.example.com/octo/hello")
	branch := listingBranch(t, "trunk")
	listed, err := NewForgeRepo(host, ForgeRepoFields{
		Repo:          repo,
		CloneURL:      "https://github.example.com/octo/hello.git",
		Description:   "greets",
		Private:       true,
		Archived:      true,
		DefaultBranch: branch,
	})
	if err != nil {
		t.Fatalf("NewForgeRepo() error = %v, want none", err)
	}
	if listed.Repo() != repo {
		t.Errorf("Repo() = %v, want %v", listed.Repo(), repo)
	}
	if listed.CloneURL() != "https://github.example.com/octo/hello.git" {
		t.Errorf("CloneURL() = %q, want the URL as given", listed.CloneURL())
	}
	if listed.Description() != "greets" {
		t.Errorf("Description() = %q, want %q", listed.Description(), "greets")
	}
	if !listed.Private() {
		t.Errorf("Private() = false, want true")
	}
	if !listed.Archived() {
		t.Errorf("Archived() = false, want true")
	}
	if listed.DefaultBranch() != branch {
		t.Errorf("DefaultBranch() = %q, want %q", listed.DefaultBranch(), branch)
	}
}

func listingBranch(t *testing.T, text string) BranchName {
	t.Helper()
	branch, err := ParseBranchName(text)
	if err != nil {
		t.Fatalf("ParseBranchName(%q) error = %v", text, err)
	}
	return branch
}

func TestNewForgeRepoKeepsTheZeroBranchWhenTheForgeGivesNone(t *testing.T) {
	host := listingHost(t)
	repo := listingRepo(t, "github.example.com/octo/hello")
	listed, err := NewForgeRepo(host, ForgeRepoFields{Repo: repo, CloneURL: "https://github.example.com/octo/hello"})
	if err != nil {
		t.Fatalf("NewForgeRepo() error = %v, want none", err)
	}
	if !listed.DefaultBranch().IsZero() || listed.Archived() {
		t.Errorf("DefaultBranch() = %q, Archived() = %v, want the zero branch and false", listed.DefaultBranch(), listed.Archived())
	}
}

func TestNewForgeRepoAcceptsAnHTTPSCloneURLOfTheRepository(t *testing.T) {
	host := listingHost(t)
	repo := listingRepo(t, "github.example.com/octo/hello")
	urls := []string{
		"https://github.example.com/octo/hello",
		"https://github.example.com/octo/hello.git",
		"https://github.example.com/octo/hello/",
		"https://github.example.com:443/octo/hello",
		"https://GitHub.example.com/octo/hello",
	}
	for _, url := range urls {
		if _, err := NewForgeRepo(host, ForgeRepoFields{Repo: repo, CloneURL: url}); err != nil {
			t.Errorf("NewForgeRepo(clone %q) error = %v, want none", url, err)
		}
	}
}

func TestNewForgeRepoRefusesACloneURLThatIsNotTheRepositoryOnTheHost(t *testing.T) {
	host := listingHost(t)
	repo := listingRepo(t, "github.example.com/octo/hello")
	urls := []string{
		"",
		"github.example.com/octo/hello",
		"http://github.example.com/octo/hello",
		"ssh://git@github.example.com/octo/hello.git",
		"git@github.example.com:octo/hello.git",
		"https://octo@github.example.com/octo/hello",
		"https://github.example.com/octo/hello?ref=main",
		"https://github.example.com/octo/hello#readme",
		"https://github.example.com/octo/other",
		"https://github.example.com/octo",
		"https://gitlab.example.com/octo/hello",
		"https://github.example.com/ghe/octo/hello",
	}
	for _, url := range urls {
		_, err := NewForgeRepo(host, ForgeRepoFields{Repo: repo, CloneURL: url})
		if !errors.Is(err, ErrInvalidCloneURL) {
			t.Errorf("NewForgeRepo(clone %q) error = %v, want ErrInvalidCloneURL", url, err)
		}
	}
}

func TestNewForgeRepoRefusesAnOversizedCloneURL(t *testing.T) {
	host := listingHost(t)
	repo := listingRepo(t, "github.example.com/octo/hello")
	url := "https://github.example.com/octo/hello" + strings.Repeat("x", 2048)
	_, err := NewForgeRepo(host, ForgeRepoFields{Repo: repo, CloneURL: url})
	if !errors.Is(err, ErrInvalidCloneURL) {
		t.Errorf("NewForgeRepo(clone of %d bytes) error = %v, want ErrInvalidCloneURL", len(url), err)
	}
}

func TestNewForgeRepoReplacesInvalidUTF8InADescription(t *testing.T) {
	host := listingHost(t)
	repo := listingRepo(t, "github.example.com/octo/hello")
	description := strings.Repeat("a", 1022) + "\xe2\x82" + "a"
	listed, err := NewForgeRepo(host, ForgeRepoFields{Repo: repo, CloneURL: "https://github.example.com/octo/hello", Description: description})
	if err != nil {
		t.Fatalf("NewForgeRepo() error = %v, want none", err)
	}
	if !utf8.ValidString(listed.Description()) || len(listed.Description()) > 1024 {
		t.Errorf("Description() = %q, want valid UTF-8 of at most 1024 bytes", listed.Description())
	}
	if want := strings.Repeat("a", 1022); listed.Description() != want {
		t.Errorf("Description() has %d bytes, want the %d bytes before the invalid sequence", len(listed.Description()), len(want))
	}
}

func TestNewForgeRepoCutsADescriptionToAWholeRuneWithin1024Bytes(t *testing.T) {
	host := listingHost(t)
	repo := listingRepo(t, "github.example.com/octo/hello")
	tests := []struct {
		name        string
		description string
		want        string
	}{
		{"short description kept", "greets", "greets"},
		{"1024 bytes kept whole", strings.Repeat("a", 1024), strings.Repeat("a", 1024)},
		{"1025 ASCII bytes cut to 1024", strings.Repeat("a", 1025), strings.Repeat("a", 1024)},
		{"two-byte rune starting at 1023 dropped", strings.Repeat("a", 1023) + "é", strings.Repeat("a", 1023)},
		{"two-byte rune starting at 1024 cut away", strings.Repeat("a", 1024) + "é", strings.Repeat("a", 1024)},
		{"three-byte rune spanning 1024 dropped", strings.Repeat("a", 1022) + "€", strings.Repeat("a", 1022)},
		{"three-byte rune before a rune starting at 1024 kept", strings.Repeat("a", 1021) + "€" + "x", strings.Repeat("a", 1021) + "€"},
		{"three-byte rune ending exactly at 1024 kept whole", strings.Repeat("a", 1021) + "€", strings.Repeat("a", 1021) + "€"},
	}
	for _, tt := range tests {
		listed, err := NewForgeRepo(host, ForgeRepoFields{Repo: repo, CloneURL: "https://github.example.com/octo/hello", Description: tt.description})
		if err != nil {
			t.Fatalf("%s: NewForgeRepo() error = %v, want none", tt.name, err)
		}
		if listed.Description() != tt.want {
			t.Errorf("%s: Description() has %d bytes, want the %d bytes expected", tt.name, len(listed.Description()), len(tt.want))
		}
	}
}

func TestNewForgeRepoRefusesAZeroOrForeignRepositoryOrHost(t *testing.T) {
	host := listingHost(t)
	repo := listingRepo(t, "github.example.com/octo/hello")
	foreign := listingRepo(t, "gitlab.example.com/octo/hello")
	tests := []struct {
		name string
		host ForgeHost
		repo ForgeRepoRef
		want error
	}{
		{"zero host", ForgeHost{}, repo, ErrInvalidForgeHost},
		{"zero repository", host, ForgeRepoRef{}, ErrInvalidRepoRef},
		{"repository on another host", host, foreign, ErrForgeWrongHost},
	}
	for _, tt := range tests {
		_, err := NewForgeRepo(tt.host, ForgeRepoFields{Repo: tt.repo, CloneURL: "https://github.example.com/octo/hello"})
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: NewForgeRepo() error = %v, want %v", tt.name, err, tt.want)
		}
	}
}

func TestNewForgeRepoRefusesACloneURLWithoutEcho(t *testing.T) {
	host := listingHost(t)
	repo := listingRepo(t, "github.example.com/octo/hello")
	for _, url := range []string{
		"http://" + echotest.Probe + "/octo/hello",
		"https://github.example.com/octo/" + echotest.Probe,
	} {
		_, err := NewForgeRepo(host, ForgeRepoFields{Repo: repo, CloneURL: url})
		echotest.RequireRefusalWithoutEcho(t, err, ErrInvalidCloneURL, echotest.Probe)
	}
}

func TestForgeRepoZeroValueNamesNoRepository(t *testing.T) {
	var zero ForgeRepo
	if zero.Repo() != (ForgeRepoRef{}) {
		t.Fatalf("zero ForgeRepo Repo() = %v, want the zero reference", zero.Repo())
	}
}
