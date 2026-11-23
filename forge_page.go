package contracts

import "fmt"

// MaxPageLimit is the largest number of items one page holds.
const MaxPageLimit = 100

// PageCursor is the opaque position of a page in a listing. The empty cursor is
// the first page; the cursor a page returns as Next reads the page after it.
type PageCursor string

// PageRequest asks for one page of a listing. Limit 0 means the plugin's
// default size, and a limit outside 0 to [MaxPageLimit] is refused by
// [PageRequest.Validate].
type PageRequest struct {
	Cursor PageCursor
	Limit  int
}

// Validate returns an error matching [ErrInvalidPageLimit] when Limit is below
// 0 or above [MaxPageLimit].
func (r PageRequest) Validate() error {
	if r.Limit < 0 || r.Limit > MaxPageLimit {
		return fmt.Errorf("%d: %w", r.Limit, ErrInvalidPageLimit)
	}
	return nil
}

// Page is one page of a listing. An empty Next means the listing ends with this
// page.
type Page[T any] struct {
	Items []T
	Next  PageCursor
}
