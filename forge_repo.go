package contracts

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	defaultForgeHost   = "github.com"
	hostedRepoSegments = 3
)

// ForgeRepoRef names one repository of a forge: its host, its owner and its name. Its
// fields are unexported, so the only way to a value that names a repository is
// [ParseRepoRef]; the zero value names none.
type ForgeRepoRef struct {
	host  string
	owner string
	name  string
}

// ParseRepoRef reads a repository from "owner/name" (on github.com), "host/owner/name",
// an https URL or a git@ SSH address. A ".git" suffix is dropped, and so is one
// trailing slash of a URL path; the host is kept in lower case. It returns an error
// matching [ErrInvalidRepoRef] for any other text, including an http URL, a URL
// with a port, a query, a fragment or userinfo.
func ParseRepoRef(text string) (ForgeRepoRef, error) {
	location, found := splitRepoLocation(text)
	if !found {
		return ForgeRepoRef{}, fmt.Errorf("%q: %w", text, ErrInvalidRepoRef)
	}
	owner, name, paired := strings.Cut(trimRepoPath(location.path), "/")
	if !paired || !validHost(location.host) || !validRepoSegment(owner) || !validRepoSegment(name) {
		return ForgeRepoRef{}, fmt.Errorf("%q: %w", text, ErrInvalidRepoRef)
	}
	return ForgeRepoRef{host: strings.ToLower(location.host), owner: owner, name: name}, nil
}

// Host is the forge host the repository lives on, such as "github.com".
func (r ForgeRepoRef) Host() string { return r.host }

// Owner is the user or organization that owns the repository.
func (r ForgeRepoRef) Owner() string { return r.owner }

// Name is the repository name, without the owner and without ".git".
func (r ForgeRepoRef) Name() string { return r.name }

// String renders the reference as "host/owner/name", the form [ParseRepoRef]
// reads back. The zero value renders as the empty string.
func (r ForgeRepoRef) String() string {
	if r.name == "" {
		return ""
	}
	return r.host + "/" + r.owner + "/" + r.name
}

type repoLocation struct {
	host string
	path string
}

func splitRepoLocation(text string) (repoLocation, bool) {
	switch {
	case strings.HasPrefix(text, "https://"):
		return splitHTTPSLocation(text)
	case strings.HasPrefix(text, "git@"):
		return splitSSHLocation(text)
	case strings.Contains(text, "://"):
		return repoLocation{}, false
	}
	segments := strings.SplitN(text, "/", hostedRepoSegments)
	if len(segments) == hostedRepoSegments {
		return repoLocation{host: segments[0], path: segments[1] + "/" + segments[2]}, true
	}
	return repoLocation{host: defaultForgeHost, path: text}, true
}

func splitHTTPSLocation(text string) (repoLocation, bool) {
	parsed, err := url.Parse(text)
	if err != nil || parsed.User != nil || parsed.Port() != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return repoLocation{}, false
	}
	return repoLocation{host: parsed.Hostname(), path: strings.TrimPrefix(parsed.Path, "/")}, true
}

func splitSSHLocation(text string) (repoLocation, bool) {
	host, path, found := strings.Cut(strings.TrimPrefix(text, "git@"), ":")
	return repoLocation{host: host, path: path}, found
}

func trimRepoPath(location string) string {
	return strings.TrimSuffix(strings.TrimSuffix(location, "/"), ".git")
}

func validHost(host string) bool {
	return host != "" && allBytes(host, isHostByte)
}

func validRepoSegment(segment string) bool {
	return segment != "" && segment != "." && segment != ".." && allBytes(segment, isRepoSegmentByte)
}

func allBytes(text string, allowed func(byte) bool) bool {
	for index := range len(text) {
		if !allowed(text[index]) {
			return false
		}
	}
	return true
}

func isHostByte(character byte) bool {
	return isLetterOrDigit(character) || character == '.' || character == '-'
}

func isRepoSegmentByte(character byte) bool {
	return isHostByte(character) || character == '_'
}

func isLetterOrDigit(character byte) bool {
	return (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9')
}
