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

func TestRateLimitErrorNamesItsResetInUTC(t *testing.T) {
	reset := time.Date(2026, time.November, 23, 11, 0, 0, 0, time.FixedZone("CET", 3600))
	err := &ForgeRateLimitedError{Reset: reset}
	want := "forge rate limit reached, resets at 2026-11-23T10:00:00Z"
	if err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
}
