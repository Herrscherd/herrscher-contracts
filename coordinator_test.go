package contracts

import (
	"context"
	"testing"
)

// fakeCoordinator locks the port signature at compile time and exercises it.
type fakeCoordinator struct {
	got      HandoffRequest
	gotDel   DelegateRequest
	gotRep   ReportRequest
	gotMrg   MergeRequest
	gotSeal  SealRequest
	gotFan   FanOutRequest
	gotRoute RouteRequest
}

func (f *fakeCoordinator) Handoff(_ context.Context, req HandoffRequest) (string, error) {
	f.got = req
	return req.ToAgent + "-session", nil
}
func (f *fakeCoordinator) Delegate(_ context.Context, req DelegateRequest) (string, error) {
	f.gotDel = req
	return req.ToAgent + "-worker", nil
}
func (f *fakeCoordinator) Report(_ context.Context, req ReportRequest) (string, error) {
	f.gotRep = req
	return "lead", nil
}
func (f *fakeCoordinator) Merge(_ context.Context, req MergeRequest) (string, error) {
	f.gotMrg = req
	return req.FromSession, nil
}
func (f *fakeCoordinator) Seal(_ context.Context, req SealRequest) (string, error) {
	f.gotSeal = req
	return req.FromSession, nil
}

func (f *fakeCoordinator) FanOut(_ context.Context, req FanOutRequest) ([]string, error) {
	f.gotFan = req
	spawned := make([]string, len(req.Tasks))
	for i := range req.Tasks {
		spawned[i] = req.ToAgent + "-worker"
	}
	return spawned, nil
}

func (f *fakeCoordinator) Route(_ context.Context, req RouteRequest) (string, string, error) {
	f.gotRoute = req
	return "picked", req.FromSession + "-routed", nil
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

	worker, err := c.Delegate(context.Background(), DelegateRequest{
		FromSession: "lead", ToAgent: "scripter", Task: "écris le module",
	})
	if err != nil || worker != "scripter-worker" {
		t.Fatalf("delegate round-trip: %q %v", worker, err)
	}

	parent, err := c.Report(context.Background(), ReportRequest{
		FromSession: "worker", Summary: "fini",
	})
	if err != nil || parent != "lead" {
		t.Fatalf("report round-trip: %q %v", parent, err)
	}
}

func TestCreateSessionHasBase(t *testing.T) {
	// Base is the ref a new session's worktree branches off (empty = current behaviour).
	spec := CreateSession{Name: "b", Base: "session/alpha"}
	if spec.Base != "session/alpha" {
		t.Fatalf("Base not set: %q", spec.Base)
	}
}

func TestCreateSessionHasParent(t *testing.T) {
	// Parent names the lead that delegated this session (empty = no parent).
	spec := CreateSession{Name: "w", Parent: "lead"}
	if spec.Parent != "lead" {
		t.Fatalf("Parent not set: %q", spec.Parent)
	}
}

// mergeStub is the smallest type carrying the full Coordinator surface, to
// assert the port shape (including Merge) at compile time.
type mergeStub struct{}

func (mergeStub) Handoff(context.Context, HandoffRequest) (string, error)     { return "", nil }
func (mergeStub) Delegate(context.Context, DelegateRequest) (string, error)   { return "", nil }
func (mergeStub) Report(context.Context, ReportRequest) (string, error)       { return "", nil }
func (mergeStub) Merge(context.Context, MergeRequest) (string, error)         { return "", nil }
func (mergeStub) Seal(context.Context, SealRequest) (string, error)           { return "", nil }
func (mergeStub) FanOut(context.Context, FanOutRequest) ([]string, error)     { return nil, nil }
func (mergeStub) Route(context.Context, RouteRequest) (string, string, error) { return "", "", nil }

func TestCoordinatorPortIncludesMerge(t *testing.T) {
	var _ Coordinator = mergeStub{}
	req := MergeRequest{FromSession: "lead", Worker: "w"}
	if req.FromSession != "lead" || req.Worker != "w" {
		t.Fatalf("MergeRequest fields not wired: %+v", req)
	}
}

// sealStub carries the full Coordinator surface (incl. Seal) to assert the port
// shape at compile time.
type sealStub struct{}

func (sealStub) Handoff(context.Context, HandoffRequest) (string, error)     { return "", nil }
func (sealStub) Delegate(context.Context, DelegateRequest) (string, error)   { return "", nil }
func (sealStub) Report(context.Context, ReportRequest) (string, error)       { return "", nil }
func (sealStub) Merge(context.Context, MergeRequest) (string, error)         { return "", nil }
func (sealStub) Seal(context.Context, SealRequest) (string, error)           { return "", nil }
func (sealStub) FanOut(context.Context, FanOutRequest) ([]string, error)     { return nil, nil }
func (sealStub) Route(context.Context, RouteRequest) (string, string, error) { return "", "", nil }

func TestCoordinatorPortIncludesSeal(t *testing.T) {
	var _ Coordinator = sealStub{}
	req := SealRequest{FromSession: "lead", Expected: 5}
	if req.FromSession != "lead" || req.Expected != 5 {
		t.Fatalf("SealRequest fields not wired: %+v", req)
	}
}

// fanoutStub carries the full Coordinator surface (incl. FanOut) to assert the
// port shape at compile time.
type fanoutStub struct{}

func (fanoutStub) Handoff(context.Context, HandoffRequest) (string, error)     { return "", nil }
func (fanoutStub) Delegate(context.Context, DelegateRequest) (string, error)   { return "", nil }
func (fanoutStub) Report(context.Context, ReportRequest) (string, error)       { return "", nil }
func (fanoutStub) Merge(context.Context, MergeRequest) (string, error)         { return "", nil }
func (fanoutStub) Seal(context.Context, SealRequest) (string, error)           { return "", nil }
func (fanoutStub) FanOut(context.Context, FanOutRequest) ([]string, error)     { return nil, nil }
func (fanoutStub) Route(context.Context, RouteRequest) (string, string, error) { return "", "", nil }

func TestCoordinatorPortIncludesFanOut(t *testing.T) {
	var _ Coordinator = fanoutStub{}
	req := FanOutRequest{FromSession: "lead", ToAgent: "scripter", Tasks: []string{"a", "b"}}
	if req.FromSession != "lead" || req.ToAgent != "scripter" || len(req.Tasks) != 2 {
		t.Fatalf("FanOutRequest fields not wired: %+v", req)
	}
}

// routeStub carries the full Coordinator surface (incl. Route) to assert the
// port shape at compile time.
type routeStub struct{}

func (routeStub) Handoff(context.Context, HandoffRequest) (string, error)   { return "", nil }
func (routeStub) Delegate(context.Context, DelegateRequest) (string, error) { return "", nil }
func (routeStub) Report(context.Context, ReportRequest) (string, error)     { return "", nil }
func (routeStub) Merge(context.Context, MergeRequest) (string, error)       { return "", nil }
func (routeStub) Seal(context.Context, SealRequest) (string, error)         { return "", nil }
func (routeStub) FanOut(context.Context, FanOutRequest) ([]string, error)   { return nil, nil }
func (routeStub) Route(context.Context, RouteRequest) (string, string, error) {
	return "", "", nil
}

func TestCoordinatorPortIncludesRoute(t *testing.T) {
	var _ Coordinator = routeStub{}
	req := RouteRequest{FromSession: "lead", Task: "écris le module réseau"}
	if req.FromSession != "lead" || req.Task == "" {
		t.Fatalf("RouteRequest fields not wired: %+v", req)
	}
}
