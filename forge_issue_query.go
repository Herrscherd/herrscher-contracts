package contracts

import "fmt"

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
	}
	return fmt.Sprintf("SortDirection(%d)", int(d))
}

// Validate returns an error matching [ErrInvalidIssueQuery] when the state, the
// sort or the direction is outside its closed set. A plugin calls it before it
// calls the forge.
func (q IssueQuery) Validate() error {
	if q.State < IssueStateUnknown || q.State > IssueClosed {
		return fmt.Errorf("state %d: %w", int(q.State), ErrInvalidIssueQuery)
	}
	if q.Sort < IssueSortDefault || q.Sort > IssueSortUpdated {
		return fmt.Errorf("sort %d: %w", int(q.Sort), ErrInvalidIssueQuery)
	}
	if q.Direction < SortDirectionDefault || q.Direction > SortDescending {
		return fmt.Errorf("direction %d: %w", int(q.Direction), ErrInvalidIssueQuery)
	}
	return nil
}
