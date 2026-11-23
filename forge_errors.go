package contracts

import (
	"errors"
	"time"
)

// Errors a forge plugin returns and a caller tells apart, matched with errors.Is.
var (
	// ErrForgeNotConnected reports that the host holds no credential for the forge host.
	ErrForgeNotConnected = errors.New("no forge credential for this host")
	// ErrForgeUnauthorized reports that the forge refused the credential it was given.
	ErrForgeUnauthorized = errors.New("forge refused the credential")
	// ErrForgeForbidden reports that the credential lacks a permission the call needs.
	ErrForgeForbidden = errors.New("forge credential lacks a required permission")
	// ErrForgeNotFound reports that the forge has no repository, pull request, job,
	// thread or issue under the reference given.
	ErrForgeNotFound = errors.New("not found on the forge")
	// ErrInvalidRepoRef reports text that names no repository a forge can hold.
	ErrInvalidRepoRef = errors.New("not a forge repository reference")
	// ErrInvalidLogLimit reports a log size limit that is not positive.
	ErrInvalidLogLimit = errors.New("log size limit must be positive")
	// ErrUnknownTokenSource reports a token source that names no source.
	ErrUnknownTokenSource = errors.New("unknown forge token source")
)

// ForgeRateLimitedError reports that the forge refused a call for its rate limit.
// Reset is when the limit lifts. It is an error type rather than a sentinel because
// the caller acts on the time, so match it with errors.AsType.
type ForgeRateLimitedError struct {
	Reset time.Time
}

// Error names the reset time in RFC 3339, in UTC.
func (e *ForgeRateLimitedError) Error() string {
	return "forge rate limit reached, resets at " + e.Reset.UTC().Format(time.RFC3339)
}
