package contracts

import "fmt"

const (
	sha1HexLength   = 40
	sha256HexLength = 64
)

// CommitSHA is the identifier of a git commit: 40 lowercase hex digits for SHA-1 or
// 64 for SHA-256. Build one with [ParseCommitSHA]; the zero value names no commit.
type CommitSHA struct{ hex string }

// ParseCommitSHA reads a commit identifier. It returns an error matching
// [ErrInvalidCommitSHA] for any text that is not 40 or 64 lowercase hex digits.
func ParseCommitSHA(text string) (CommitSHA, error) {
	lengthOK := len(text) == sha1HexLength || len(text) == sha256HexLength
	if !lengthOK || !allBytes(text, isLowerHexByte) {
		return CommitSHA{}, fmt.Errorf("%q: %w", truncatedForError(text), ErrInvalidCommitSHA)
	}
	return CommitSHA{hex: text}, nil
}

// String renders the identifier as ParseCommitSHA reads it back. The zero value
// renders as the empty string.
func (c CommitSHA) String() string { return c.hex }
