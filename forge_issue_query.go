package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// IssueSort is the field an issue listing is ordered by. It is a closed set
// whose zero value, [IssueSortDefault], leaves the order to the forge.
type IssueSort int

const (
	// IssueSortDefault leaves the order of the listing to the forge.
	IssueSortDefault IssueSort = iota
	// IssueSortCreated orders the listing by the time each issue was created.
	IssueSortCreated
	// IssueSortUpdated orders the listing by the time each issue was last updated.
	IssueSortUpdated
	issueSortEnd
)

// String renders the sort as its short name. A value that names no sort prints
// its raw number.
func (s IssueSort) String() string {
	switch s {
	case IssueSortDefault:
		return "default"
	case IssueSortCreated:
		return "created"
	case IssueSortUpdated:
		return "updated"
	case issueSortEnd:
	}
	return fmt.Sprintf("IssueSort(%d)", int(s))
}

// SortDirection is the way an issue listing is ordered. It is a closed set whose
// zero value, [SortDirectionDefault], leaves the direction to the forge.
type SortDirection int

const (
	// SortDirectionDefault leaves the direction of the listing to the forge.
	SortDirectionDefault SortDirection = iota
	// SortAscending orders the listing from the oldest value to the newest.
	SortAscending
	// SortDescending orders the listing from the newest value to the oldest.
	SortDescending
	sortDirectionEnd
)

// String renders the direction as its short name. A value that names no
// direction prints its raw number.
func (d SortDirection) String() string {
	switch d {
	case SortDirectionDefault:
		return "default"
	case SortAscending:
		return "ascending"
	case SortDescending:
		return "descending"
	case sortDirectionEnd:
	}
	return fmt.Sprintf("SortDirection(%d)", int(d))
}

// Validate returns an error matching [ErrInvalidIssueQuery] when the state, the
// sort or the direction is outside its closed set. A plugin calls it before it
// calls the forge.
func (q IssueQuery) Validate() error {
	if !inClosedSet(int(q.State), int(IssueStateUnknown), int(issueStateEnd)) {
		return fmt.Errorf("state %d: %w", int(q.State), ErrInvalidIssueQuery)
	}
	if !inClosedSet(int(q.Sort), int(IssueSortDefault), int(issueSortEnd)) {
		return fmt.Errorf("sort %d: %w", int(q.Sort), ErrInvalidIssueQuery)
	}
	if !inClosedSet(int(q.Direction), int(SortDirectionDefault), int(sortDirectionEnd)) {
		return fmt.Errorf("direction %d: %w", int(q.Direction), ErrInvalidIssueQuery)
	}
	return nil
}

func inClosedSet(value, first, end int) bool {
	return value >= first && value < end
}

const issueListingSeparator = "/"

// IssueListingRevision builds the [Revision] of an issue listing read for repo,
// query and page, from the etag the forge reported for that read. The revision
// is valid only for the same repository, the same query and the same page: see
// [IssueListingETag]. An empty etag gives the zero Revision, which names no
// revision, so the listing is read in full.
func IssueListingRevision(repo ForgeRepoRef, query IssueQuery, page PageRequest, etag string) Revision {
	if etag == "" {
		return Revision("")
	}
	return Revision(issueListingFingerprint(repo, query, page) + issueListingSeparator + etag)
}

// IssueListingETag returns the etag of since and true when since was built by
// [IssueListingRevision] for the same repository, query and page. Otherwise it
// returns false, and the plugin reads the listing in full.
func IssueListingETag(since Revision, repo ForgeRepoRef, query IssueQuery, page PageRequest) (string, bool) {
	fingerprint, etag, found := strings.Cut(string(since), issueListingSeparator)
	if !found || fingerprint != issueListingFingerprint(repo, query, page) {
		return "", false
	}
	return etag, true
}

func issueListingFingerprint(repo ForgeRepoRef, query IssueQuery, page PageRequest) string {
	var key strings.Builder
	fmt.Fprintf(&key, "repo=%q\n", repo.String())
	fmt.Fprintf(&key, "labels=%d\n", len(query.Labels))
	for _, label := range query.Labels {
		fmt.Fprintf(&key, "label=%q\n", label)
	}
	fmt.Fprintf(&key, "mentioned=%q\n", query.Mentioned.String())
	fmt.Fprintf(&key, "updated_since=%s\n", query.UpdatedSince.UTC().Format(time.RFC3339Nano))
	fmt.Fprintf(&key, "state=%d\n", int(query.Listed()))
	fmt.Fprintf(&key, "sort=%d\n", int(query.Sort))
	fmt.Fprintf(&key, "direction=%d\n", int(query.Direction))
	fmt.Fprintf(&key, "cursor=%q\n", string(page.Cursor))
	fmt.Fprintf(&key, "limit=%d\n", page.Limit)
	sum := sha256.Sum256([]byte(key.String()))
	return hex.EncodeToString(sum[:])
}
