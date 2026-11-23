package contracts

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRateLimitErrorIsFoundThroughWrapping(t *testing.T) {
	reset := time.Date(2026, time.November, 23, 10, 0, 0, 0, time.UTC)
	err := fmt.Errorf("list checks: %w", &ForgeRateLimitedError{Reset: reset})
	limited, ok := errors.AsType[*ForgeRateLimitedError](err)
	if !ok {
		t.Fatalf("errors.AsType(%v) found no ForgeRateLimitedError", err)
	}
	if !limited.Reset.Equal(reset) {
		t.Fatalf("Reset = %v, want %v", limited.Reset, reset)
	}
}

func TestMergeRefusedErrorIsFoundThroughWrapping(t *testing.T) {
	err := fmt.Errorf("merge pull request: %w", &ForgeMergeRefusedError{State: MergeStateBlocked})
	refused, ok := errors.AsType[*ForgeMergeRefusedError](err)
	if !ok {
		t.Fatalf("errors.AsType(%v) found no ForgeMergeRefusedError", err)
	}
	if refused.State != MergeStateBlocked {
		t.Fatalf("State = %v, want MergeStateBlocked", refused.State)
	}
}

func TestMergeRefusedErrorNamesTheMergeState(t *testing.T) {
	err := &ForgeMergeRefusedError{State: MergeStateConflicting}
	want := "forge refused the merge, the pull request is conflicting"
	if err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestRateLimitErrorSaysWhenItsResetIsUnknown(t *testing.T) {
	err := &ForgeRateLimitedError{}
	want := "forge rate limit reached, reset time unknown"
	if err.Error() != want {
		t.Fatalf("Error() of a zero reset = %q, want %q", err.Error(), want)
	}
}

func TestRateLimitErrorNamesItsResetInUTC(t *testing.T) {
	reset := time.Date(2026, time.November, 23, 11, 0, 0, 0, time.FixedZone("CET", 3600))
	err := &ForgeRateLimitedError{Reset: reset}
	want := "forge rate limit reached, resets at 2026-11-23T10:00:00Z"
	if err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
}
