package contracts

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// LimitWindow is the period a vendor usage limit counts over. It is a closed set
// whose zero value is not a window: [ParseLimitWindow] never returns it, and the
// zero value does not marshal.
type LimitWindow int

const (
	// LimitFiveHour is a rolling window of five hours.
	LimitFiveHour LimitWindow = iota + 1
	// LimitWeekly is a window of seven days.
	LimitWeekly
	// LimitMonthly is a window of one month.
	LimitMonthly
)

// LimitWindows lists every window, in declaration order.
func LimitWindows() []LimitWindow {
	return []LimitWindow{LimitFiveHour, LimitWeekly, LimitMonthly}
}

// String renders the window as its text form, the one [ParseLimitWindow] reads
// back. A value that names no window prints its raw number.
func (w LimitWindow) String() string {
	switch w {
	case LimitFiveHour:
		return "five-hour"
	case LimitWeekly:
		return "weekly"
	case LimitMonthly:
		return "monthly"
	}
	return fmt.Sprintf("LimitWindow(%d)", int(w))
}

// ErrUnknownLimitWindow is returned for text, or a value, that names no limit
// window. Match it with errors.Is.
var ErrUnknownLimitWindow = errors.New("unknown limit window")

// ParseLimitWindow reads a window from the text form [LimitWindow.String]
// writes. Any other text, the empty string included, returns an error matching
// [ErrUnknownLimitWindow].
func ParseLimitWindow(text string) (LimitWindow, error) {
	for _, window := range LimitWindows() {
		if window.String() == text {
			return window, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownLimitWindow, text)
}

// MarshalText writes the text form of the window. A value that names no window
// returns an error matching [ErrUnknownLimitWindow], so it never reaches the
// wire.
func (w LimitWindow) MarshalText() ([]byte, error) {
	if !w.isKnown() {
		return nil, fmt.Errorf("%w: %d", ErrUnknownLimitWindow, int(w))
	}
	return []byte(w.String()), nil
}

// UnmarshalText reads the text form of a window, as [ParseLimitWindow] does.
func (w *LimitWindow) UnmarshalText(text []byte) error {
	parsed, err := ParseLimitWindow(string(text))
	if err != nil {
		return err
	}
	*w = parsed
	return nil
}

func (w LimitWindow) isKnown() bool {
	for _, window := range LimitWindows() {
		if window == w {
			return true
		}
	}
	return false
}

// Limit is one vendor usage limit of an account, as the vendor reports it now.
type Limit struct {
	Window LimitWindow
	// Scope is empty for a limit on the whole account, else a sub-scope the vendor
	// names, such as a model family ("opus", "sonnet").
	Scope string
	// UsedPercent is the share of the limit already used, from 0 to 100.
	UsedPercent float64
	// ResetsAt is when the window starts over, the zero time when the vendor
	// gives none.
	ResetsAt time.Time
	// Span is the length of the window, so a client can place the pace marker,
	// zero when unknown.
	Span time.Duration
}

// LimitReader reads the usage limits of one account without a live instance of
// the backend and without spending a turn. It stops when ctx is done. A vendor
// that offers no way to read them returns an error matching
// [ErrLimitsUnavailable].
type LimitReader func(ctx context.Context, access ProbeAccess) ([]Limit, error)

// ErrLimitsUnavailable is returned by a [LimitReader] whose vendor exposes no
// limits for the account. Match it with errors.Is.
var ErrLimitsUnavailable = errors.New("limits unavailable")

// ModelUsage is the usage of one model within a turn, so a ledger can split a
// turn whose backend ran several models. In, Out, CacheRead and CacheCreate have
// the meaning of TokensIn, Tokens, CacheRead and CacheCreate on [Event].
type ModelUsage struct {
	Model       string `json:"model"`
	In          int    `json:"in,omitempty"`
	Out         int    `json:"out,omitempty"`
	CacheRead   int    `json:"cache_read,omitempty"`
	CacheCreate int    `json:"cache_create,omitempty"`
	// Cost is the cost of the model's share of the turn in USD, zero when the
	// backend reports none.
	Cost float64 `json:"cost,omitempty"`
}
