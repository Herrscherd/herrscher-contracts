package contracts

import (
	"errors"
	"testing"

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
	listed, err := NewForgeRepo(host, repo, "https://github.example.com/octo/hello.git", "greets", true)
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
		if _, err := NewForgeRepo(host, repo, url, "", false); err != nil {
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
	}
	for _, url := range urls {
		_, err := NewForgeRepo(host, repo, url, "", false)
		if !errors.Is(err, ErrInvalidCloneURL) {
			t.Errorf("NewForgeRepo(clone %q) error = %v, want ErrInvalidCloneURL", url, err)
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
		_, err := NewForgeRepo(tt.host, tt.repo, "https://github.example.com/octo/hello", "", false)
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
		_, err := NewForgeRepo(host, repo, url, "", false)
		echotest.RequireRefusalWithoutEcho(t, err, ErrInvalidCloneURL, echotest.Probe)
	}
}

func TestForgeRepoZeroValueNamesNoRepository(t *testing.T) {
	var zero ForgeRepo
	if zero.Repo() != (ForgeRepoRef{}) {
		t.Fatalf("zero ForgeRepo Repo() = %v, want the zero reference", zero.Repo())
	}
}
