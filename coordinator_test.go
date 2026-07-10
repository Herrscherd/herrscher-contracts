package contracts

import (
	"context"
	"testing"
)

// fakeCoordinator locks the port signature at compile time and exercises it.
type fakeCoordinator struct{ got HandoffRequest }

func (f *fakeCoordinator) Handoff(_ context.Context, req HandoffRequest) (string, error) {
	f.got = req
	return req.ToAgent + "-session", nil
}

func TestCoordinatorPortRoundTrip(t *testing.T) {
	var c Coordinator = &fakeCoordinator{}
	name, err := c.Handoff(context.Background(), HandoffRequest{
		FromSession: "alpha", ToAgent: "scripter", Task: "finir le module",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "scripter-session" {
		t.Fatalf("got %q", name)
	}
}

func TestCreateSessionHasBase(t *testing.T) {
	// Base is the ref a new session's worktree branches off (empty = current behaviour).
	spec := CreateSession{Name: "b", Base: "session/alpha"}
	if spec.Base != "session/alpha" {
		t.Fatalf("Base not set: %q", spec.Base)
	}
}
