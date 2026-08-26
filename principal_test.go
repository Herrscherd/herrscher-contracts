package contracts

import (
	"context"
	"testing"
)

func TestPrincipalRoundTrips(t *testing.T) {
	ctx := WithPrincipal(context.Background(), "chat:1234")
	if got := PrincipalFrom(ctx); got != "chat:1234" {
		t.Fatalf("PrincipalFrom = %q, want %q", got, "chat:1234")
	}
}

func TestPrincipalFromIsEmptyWhenNobodyNamedOne(t *testing.T) {
	if got := PrincipalFrom(context.Background()); got != "" {
		t.Fatalf("PrincipalFrom = %q, want empty", got)
	}
}

// A gateway that names nobody must not be distinguishable from one that never
// called WithPrincipal: both mean "this caller is unnamed", and the daemon
// answers that one way.
func TestPrincipalEmptyStringIsNotAName(t *testing.T) {
	ctx := WithPrincipal(context.Background(), "")
	if got := PrincipalFrom(ctx); got != "" {
		t.Fatalf("PrincipalFrom = %q, want empty", got)
	}
}
