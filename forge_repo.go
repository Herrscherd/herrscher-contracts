package contracts

import "strings"

const (
	maxRepoRefBytes  = 512
	maxRepoNameBytes = 100
	minRepoSegments  = 2
	gitSuffix        = ".git"
)

// ForgeRepoRef names one repository of a forge: its host, its namespace and its
// name. A namespace is one or more segments joined by "/", as GitLab subgroups
// need; each forge plugin checks the depth it supports. Its fields are
// unexported, so the only way to a value that names a repository is
// [ParseRepoRef]; the zero value names none. Two references compare equal with
// == only when their namespace and name match in case too, because some forges
// are case-sensitive: match repositories with [ForgeRepoRef.SameRepo].
type ForgeRepoRef struct {
	host      ForgeHost
	namespace string
	name      string
}

// ParseRepoRef reads a repository from "host/namespace/name", an https URL, an
// ssh:// URL, or a git@ address in scp form ("git@host:namespace/name"). The
// first segment of the first form must parse as a [ForgeHost], so "owner/name"
// without a host is refused. Userinfo in an https URL is refused, and the
// ssh:// and scp forms require the git@ user. Port 443 is dropped from the host
// in every form, as [ParseForgeHost] does, and the port of an ssh:// URL is
// dropped too, since it is the SSH service's and not the forge's. A ".git"
// suffix on the name is dropped, and so is one trailing slash of a bare or
// https path. A "?" or "#" anywhere refuses the text, even an empty query or
// fragment, and an "@" fails the segment or host validation. It returns an
// error matching [ErrInvalidRepoRef] for any other userinfo, percent-encoding,
// a "." or ".." segment, an empty segment, a segment that starts with "-" or
// ".", a byte outside ASCII letters, digits, ".", "-" and "_", a name longer
// than 100 bytes, or a text longer than 512 bytes. The error does not repeat
// the text.
func ParseRepoRef(text string) (ForgeRepoRef, error) {
	if len(text) > maxRepoRefBytes || strings.ContainsAny(text, "?#") {
		return ForgeRepoRef{}, ErrInvalidRepoRef
	}
	location, found := splitRepoLocation(text)
	if !found {
		return ForgeRepoRef{}, ErrInvalidRepoRef
	}
	host, err := ParseForgeHost(location.host)
	if err != nil {
		return ForgeRepoRef{}, ErrInvalidRepoRef
	}
	path, found := splitRepoPath(location.path)
	if !found {
		return ForgeRepoRef{}, ErrInvalidRepoRef
	}
	return ForgeRepoRef{host: host, namespace: path.namespace, name: path.name}, nil
}

// RequireRepoHost returns an error matching [ErrInvalidForgeHost] when host is
// the zero value, and one matching [ErrInvalidRepoRef] when repo is. It returns
// an error matching [ErrForgeWrongHost] when repo is not on host. A forge
// plugin calls it first in every method that takes a repository.
func RequireRepoHost(host ForgeHost, repo ForgeRepoRef) error {
	if host == (ForgeHost{}) {
		return ErrInvalidForgeHost
	}
	if repo == (ForgeRepoRef{}) {
		return ErrInvalidRepoRef
	}
	if repo.host != host {
		return ErrForgeWrongHost
	}
	return nil
}

// Host is the forge host the repository lives on.
func (r ForgeRepoRef) Host() ForgeHost { return r.host }

// Namespace is the path of groups or the user that holds the repository, such
// as "octo" or "team/platform" for a GitLab subgroup.
func (r ForgeRepoRef) Namespace() string { return r.namespace }

// Name is the repository name, without the namespace and without ".git".
func (r ForgeRepoRef) Name() string { return r.name }

// SameRepo reports whether r and o name the same repository. The host must
// match exactly, and the namespace and the name match when they are equal
// ignoring case. Use it to match repositories, because == is case-sensitive on
// the namespace and the name.
func (r ForgeRepoRef) SameRepo(o ForgeRepoRef) bool {
	return r.host == o.host && strings.EqualFold(r.namespace, o.namespace) && strings.EqualFold(r.name, o.name)
}

// String renders the reference as "host/namespace/name", the form
// [ParseRepoRef] reads back. The zero value renders as the empty string.
func (r ForgeRepoRef) String() string {
	if r == (ForgeRepoRef{}) {
		return ""
	}
	return r.host.String() + "/" + r.namespace + "/" + r.name
}

// ForgeRepo is a repository a forge lists for the connected account: its
// reference, the https URL that clones it, its description and whether it is
// private. Its fields are unexported, so a value is built only by
// [NewForgeRepo]; the zero value names no repository.
type ForgeRepo struct {
	repo        ForgeRepoRef
	cloneURL    string
	description string
	private     bool
}

// NewForgeRepo describes repo as listed on host. It returns an error matching
// [ErrInvalidForgeHost] for the zero host, [ErrInvalidRepoRef] for the zero
// repository, and [ErrForgeWrongHost] when repo is not on host, as
// [RequireRepoHost] reports them. It returns an error matching
// [ErrInvalidCloneURL] when cloneURL is not an https URL that names repo itself
// on host, with or without a ".git" suffix. The error does not repeat cloneURL.
func NewForgeRepo(host ForgeHost, repo ForgeRepoRef, cloneURL, description string, private bool) (ForgeRepo, error) {
	if err := RequireRepoHost(host, repo); err != nil {
		return ForgeRepo{}, err
	}
	if !clonesRepo(repo, cloneURL) {
		return ForgeRepo{}, ErrInvalidCloneURL
	}
	return ForgeRepo{repo: repo, cloneURL: cloneURL, description: description, private: private}, nil
}

// Repo is the reference of the repository.
func (r ForgeRepo) Repo() ForgeRepoRef { return r.repo }

// CloneURL is the https URL that clones the repository, as the forge gives it.
func (r ForgeRepo) CloneURL() string { return r.cloneURL }

// Description is the text the owner wrote for the repository, empty when there
// is none.
func (r ForgeRepo) Description() string { return r.description }

// Private reports whether the forge holds the repository as private.
func (r ForgeRepo) Private() bool { return r.private }

func clonesRepo(repo ForgeRepoRef, cloneURL string) bool {
	if !strings.HasPrefix(cloneURL, "https://") {
		return false
	}
	cloned, err := ParseRepoRef(cloneURL)
	return err == nil && cloned.SameRepo(repo)
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
	return repoLocation{host: host, path: strings.TrimSuffix(path, "/")}, found
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
	account, isGitUser := strings.CutPrefix(authority, "git@")
	host, port, hasPort := strings.Cut(account, ":")
	if !found || !isGitUser || (hasPort && !isPort(port)) {
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
