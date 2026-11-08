package contracts

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// HealthState is the outcome of one health check. It is a closed set whose zero
// value is not a state: [ParseHealthState] never returns it, and the zero value
// does not marshal. From the most to the least severe the states are
// [HealthDown], [HealthDegraded], [HealthOK] and [HealthOff].
type HealthState int

const (
	// HealthOK is a service that answered as expected.
	HealthOK HealthState = iota + 1
	// HealthDegraded is a service that answers, but slowly, partially or under a
	// rate limit.
	HealthDegraded
	// HealthDown is a service that does not answer, or refuses its credentials.
	HealthDown
	// HealthOff is a service that is not set up or not logged in. It never fails a
	// health check on its own.
	HealthOff
)

// HealthStates lists every state, in declaration order.
func HealthStates() []HealthState {
	return []HealthState{HealthOK, HealthDegraded, HealthDown, HealthOff}
}

// String renders the state as its text form, the one [ParseHealthState] reads
// back. A value that names no state prints its raw number.
func (s HealthState) String() string {
	switch s {
	case HealthOK:
		return "ok"
	case HealthDegraded:
		return "degraded"
	case HealthDown:
		return "down"
	case HealthOff:
		return "off"
	}
	return fmt.Sprintf("HealthState(%d)", int(s))
}

// ErrUnknownHealthState is returned for text, or a value, that names no health
// state. Match it with errors.Is.
var ErrUnknownHealthState = errors.New("unknown health state")

// ParseHealthState reads a state from the text form [HealthState.String] writes.
// Any other text, the empty string included, returns an error matching
// [ErrUnknownHealthState].
func ParseHealthState(text string) (HealthState, error) {
	for _, state := range HealthStates() {
		if state.String() == text {
			return state, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownHealthState, text)
}

// MarshalText writes the text form of the state. A value that names no state
// returns an error matching [ErrUnknownHealthState], so it never reaches the
// wire.
func (s HealthState) MarshalText() ([]byte, error) {
	if !s.isKnown() {
		return nil, fmt.Errorf("%w: %d", ErrUnknownHealthState, int(s))
	}
	return []byte(s.String()), nil
}

// UnmarshalText reads the text form of a state, as [ParseHealthState] does.
func (s *HealthState) UnmarshalText(text []byte) error {
	parsed, err := ParseHealthState(string(text))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// WorstHealth returns the most severe of states. It returns [HealthOff] when
// states is empty or names no known state, so an aggregate of nothing fails
// nothing. A value that names no state is ignored.
func WorstHealth(states ...HealthState) HealthState {
	worst := HealthOff
	for _, state := range states {
		if state.severity() > worst.severity() {
			worst = state
		}
	}
	return worst
}

func (s HealthState) isKnown() bool {
	for _, state := range HealthStates() {
		if state == s {
			return true
		}
	}
	return false
}

const (
	severityOff = iota + 1
	severityOK
	severityDegraded
	severityDown
)

func (s HealthState) severity() int {
	switch s {
	case HealthOff:
		return severityOff
	case HealthOK:
		return severityOK
	case HealthDegraded:
		return severityDegraded
	case HealthDown:
		return severityDown
	}
	return 0
}

// ProbeAccess is what the host hands a [HealthProber] or a [LimitReader]: the
// account to check and how far to go.
type ProbeAccess struct {
	// Getenv reads the environment of the account probed: the profile environment
	// merged over the host environment.
	Getenv func(string) string
	// Account is the profile probed, empty for the vendor's native account.
	Account ProfileName
	// Config holds the plugin's own settings, as its factory would receive them,
	// so a probe reads a token or a key the way the live plugin does.
	Config PluginConfig
	// Deep asks for a real, minimal round trip instead of a cheap check.
	Deep bool
}

// ProbeReport is what one health check found. Its zero value has no State and is
// not a report.
type ProbeReport struct {
	State HealthState
	// Latency is the time the check took, zero when it was not measured.
	Latency time.Duration
	// Detail is one line for the operator, a code and its arguments the client
	// renders. The CodeHealth codes cover the usual findings.
	Detail Notice
	// Remedy is what the operator can do about the state, the zero Notice when
	// there is nothing to do.
	Remedy Notice
}

// HealthProber checks one service of a plugin for one account, without a live
// instance of the plugin. It stops when ctx is done, and reports a failure in
// its [ProbeReport] rather than as an error.
type HealthProber func(ctx context.Context, access ProbeAccess) ProbeReport
