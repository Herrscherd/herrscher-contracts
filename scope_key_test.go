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

// TestScopeKeyNormalizes pins case- and separator-folding so the same logical
// scope can never split into two vault files (the neublox/Neublox duplicate).
func TestScopeKeyNormalizes(t *testing.T) {
	cases := []struct{ in, want string }{
		{"neublox", "projects/neublox"},
		{"Neublox", "projects/neublox"},
		{"NEUBLOX", "projects/neublox"},
		{"  Neu Blox  ", "projects/neu-blox"},
		{"my/evil..name", "projects/my-evil-name"},
		{"Café", "projects/café"},
	}
	for _, c := range cases {
		if got := ProjectKey(c.in); got != c.want {
			t.Fatalf("ProjectKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := AgentKey("Scripter Bot"); got != "agents/scripter-bot" {
		t.Fatalf("AgentKey normalize: got %q, want agents/scripter-bot", got)
	}
}

// compile-time check that the interface shape is what callers depend on.
type stubProvisioner struct{}

func (stubProvisioner) EnsureProject(_ context.Context, _, _ string) error { return nil }
func (stubProvisioner) EnsureAgent(_ context.Context, _, _ string) error   { return nil }

var _ Provisioner = stubProvisioner{}
