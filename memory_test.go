package contracts

import (
	"context"
	"testing"
)

// recMemory is a recording stub proving the Memory interface is implementable.
type recMemory struct {
	recorded []Node
	closed   bool
}

func (m *recMemory) Recall(_ context.Context, key string, _ int) (Subgraph, error) {
	return Subgraph{Root: Node{Key: key}}, nil
}
func (m *recMemory) Record(_ context.Context, n Node) error {
	m.recorded = append(m.recorded, n)
	return nil
}
func (m *recMemory) Search(_ context.Context, _ Query) ([]Node, error) { return nil, nil }
func (m *recMemory) Links(_ context.Context, _, _, _ string) error     { return nil }
func (m *recMemory) Close() error                                      { m.closed = true; return nil }

func TestMemoryInterfaceIsImplementable(t *testing.T) {
	var mem Memory = &recMemory{}
	if err := mem.Record(context.Background(), Node{Key: "k", Kind: KindProject}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	sg, err := mem.Recall(context.Background(), "k", 1)
	if err != nil || sg.Root.Key != "k" {
		t.Fatalf("Recall returned %+v, %v", sg, err)
	}
	if err := mem.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNodeKindConstants(t *testing.T) {
	kinds := []NodeKind{
		KindOrganization, KindProject, KindRepo, KindServer,
		KindArchitecture, KindProduction, KindSession, KindDecision, KindUser,
		KindAgent,
	}
	if len(kinds) != 10 {
		t.Fatalf("expected 10 node kinds, got %d", len(kinds))
	}
}

// curStub proves the CurationHook seam is implementable (no production impl ships here).
type curStub struct{ called bool }

func (c *curStub) Consolidate(_ context.Context) error { c.called = true; return nil }

func TestCurationHookIsImplementable(t *testing.T) {
	var h CurationHook = &curStub{}
	if err := h.Consolidate(context.Background()); err != nil {
		t.Fatalf("Consolidate: %v", err)
	}
}

func TestKindDomainConstant(t *testing.T) {
	if KindDomain != "domain" {
		t.Fatalf("KindDomain = %q, want %q", KindDomain, "domain")
	}
}
