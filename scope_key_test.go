package contracts

import (
	"context"
	"testing"
)

func TestScopeKeyHelpers(t *testing.T) {
	if got := ProjectKey("game"); got != "projects/game" {
		t.Fatalf("ProjectKey: got %q, want projects/game", got)
	}
	if got := AgentKey("scripter"); got != "agents/scripter" {
		t.Fatalf("AgentKey: got %q, want agents/scripter", got)
	}
}

// compile-time check that the interface shape is what callers depend on.
type stubProvisioner struct{}

func (stubProvisioner) EnsureProject(_ context.Context, _, _ string) error { return nil }
func (stubProvisioner) EnsureAgent(_ context.Context, _, _ string) error   { return nil }

var _ Provisioner = stubProvisioner{}
