package contracts

import (
	"fmt"
	"strings"
)

const (
	maxRepoRefBytes  = 512
	maxRepoNameBytes = 100
	minRepoSegments  = 2
	gitSuffix        = ".git"
)

// ForgeRepoRef names one repository of a forge: its host, its namespace and its
// name. A namespace is one or more segments joined by "/", as GitLab subgroups
// need; each forge plugin checks the depth it supports. Its fields are unexported,
// so the only way to a value that names a repository is [ParseRepoRef]; the zero
// value names none.
type ForgeRepoRef struct {
	host      ForgeHost
	namespace string
	name      string
}

// ParseRepoRef reads a repository from "host/namespace/name", an https URL, an
// ssh:// URL naming the git user, or a git@ address. The first segment of the first
// two forms must parse as a [ForgeHost], so "owner/name" without a host is refused.
// A ".git" suffix on the name is dropped, and so is one trailing slash of an https
// path. It returns an error matching [ErrInvalidRepoRef] for userinfo other than
// git@, a query or a fragment (even an empty one), percent-encoding, a "." or ".."
// segment, an empty segment, a segment that starts with "-" or ".", a byte outside
// ASCII letters, digits, ".", "-" and "_", a name longer than 100 bytes, or a text
// longer than 512 bytes. The error quotes at most the first 64 bytes of the text.
func ParseRepoRef(text string) (ForgeRepoRef, error) {
	if len(text) > maxRepoRefBytes || strings.ContainsAny(text, "?#") {
		return ForgeRepoRef{}, invalidRepoRef(text)
	}
	location, found := splitRepoLocation(text)
	if !found {
		return ForgeRepoRef{}, invalidRepoRef(text)
	}
	host, err := ParseForgeHost(location.host)
	if err != nil {
		return ForgeRepoRef{}, invalidRepoRef(text)
	}
	path, found := splitRepoPath(location.path)
	if !found {
		return ForgeRepoRef{}, invalidRepoRef(text)
	}
	return ForgeRepoRef{host: host, namespace: path.namespace, name: path.name}, nil
}

// RequireRepoHost returns an error matching [ErrForgeWrongHost] when repo is not on
// host. A forge plugin calls it first in every method that takes a repository.
func RequireRepoHost(host ForgeHost, repo ForgeRepoRef) error {
	if repo.host != host {
		return fmt.Errorf("%s is not on %s: %w", repo, host, ErrForgeWrongHost)
	}
	return nil
}

// Host is the forge host the repository lives on.
func (r ForgeRepoRef) Host() ForgeHost { return r.host }

// Namespace is the path of groups or the user that holds the repository, such as
// "octo" or "team/platform" for a GitLab subgroup.
func (r ForgeRepoRef) Namespace() string { return r.namespace }

// Name is the repository name, without the namespace and without ".git".
func (r ForgeRepoRef) Name() string { return r.name }

// String renders the reference as "host/namespace/name", the form [ParseRepoRef]
// reads back. The zero value renders as the empty string.
func (r ForgeRepoRef) String() string {
	if r == (ForgeRepoRef{}) {
		return ""
	}
	return r.host.String() + "/" + r.namespace + "/" + r.name
}

func invalidRepoRef(text string) error {
	return fmt.Errorf("%q: %w", truncatedForError(text), ErrInvalidRepoRef)
}

type repoLocation struct {
	host string
	path string
}

type repoPath struct {
	namespace string
	name      string
}

func splitRepoLocation(text string) (repoLocation, bool) {
	switch {
	case strings.HasPrefix(text, "https://"):
		return splitHTTPSLocation(strings.TrimPrefix(text, "https://"))
	case strings.HasPrefix(text, "ssh://"):
		return splitSSHLocation(strings.TrimPrefix(text, "ssh://"))
	case strings.HasPrefix(text, "git@"):
		host, path, found := strings.Cut(strings.TrimPrefix(text, "git@"), ":")
		return repoLocation{host: host, path: path}, found
	case strings.Contains(text, "://"):
		return repoLocation{}, false
	}
	host, path, found := strings.Cut(text, "/")
	return repoLocation{host: host, path: path}, found
}

func splitHTTPSLocation(rest string) (repoLocation, bool) {
	authority, path, found := strings.Cut(rest, "/")
	if !found || strings.Contains(authority, "@") {
		return repoLocation{}, false
	}
	return repoLocation{host: authority, path: strings.TrimSuffix(path, "/")}, true
}

func splitSSHLocation(rest string) (repoLocation, bool) {
	authority, path, found := strings.Cut(rest, "/")
	host, isGitUser := strings.CutPrefix(authority, "git@")
	if !found || !isGitUser {
		return repoLocation{}, false
	}
	return repoLocation{host: host, path: path}, true
}

func splitRepoPath(path string) (repoPath, bool) {
	segments := strings.Split(path, "/")
	last := len(segments) - 1
	if len(segments) < minRepoSegments {
		return repoPath{}, false
	}
	segments[last] = strings.TrimSuffix(segments[last], gitSuffix)
	name := segments[last]
	if len(name) > maxRepoNameBytes || !allValidRepoSegments(segments) {
		return repoPath{}, false
	}
	return repoPath{namespace: strings.Join(segments[:last], "/"), name: name}, true
}

func allValidRepoSegments(segments []string) bool {
	for _, segment := range segments {
		if !validRepoSegment(segment) {
			return false
		}
	}
	return true
}

func validRepoSegment(segment string) bool {
	return segment != "" && segment[0] != '-' && segment[0] != '.' && allBytes(segment, isRepoSegmentByte)
}

func isRepoSegmentByte(character byte) bool {
	return isLetterOrDigit(character) || character == '.' || character == '-' || character == '_'
}
