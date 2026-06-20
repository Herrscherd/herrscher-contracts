package contracts

import (
	"context"
	"testing"
)

// scopeMem is a stub Memory that records writes and serves canned subgraphs by key.
type scopeMem struct {
	recorded []Node
	links    [][3]string // {from, to, rel}
	graphs   map[string]Subgraph
}

func (m *scopeMem) Recall(_ context.Context, key string, _ int) (Subgraph, error) {
	if sg, ok := m.graphs[key]; ok {
		return sg, nil
	}
	return Subgraph{Root: Node{Key: key}}, nil
}
func (m *scopeMem) Record(_ context.Context, n Node) error {
	m.recorded = append(m.recorded, n)
	return nil
}
func (m *scopeMem) Search(_ context.Context, _ Query) ([]Node, error) { return nil, nil }
func (m *scopeMem) Links(_ context.Context, from, to, rel string) error {
	m.links = append(m.links, [3]string{from, to, rel})
	return nil
}
func (m *scopeMem) Close() error { return nil }

func hasLink(m *scopeMem, from, to, rel string) bool {
	for _, l := range m.links {
		if l == [3]string{from, to, rel} {
			return true
		}
	}
	return false
}

func TestRecordSharedLinksUnderProject(t *testing.T) {
	m := &scopeMem{}
	s := MemoryScope{Project: "proj", Agent: "ag"}
	if err := RecordShared(context.Background(), m, s, Node{Key: "fact", Kind: KindDecision}); err != nil {
		t.Fatalf("RecordShared: %v", err)
	}
	if len(m.recorded) != 1 || m.recorded[0].Key != "fact" {
		t.Fatalf("node not recorded: %+v", m.recorded)
	}
	if !hasLink(m, "proj", "fact", RelContains) {
		t.Fatalf("shared fact not linked under project: %+v", m.links)
	}
}

func TestRecordPrivateLinksUnderAgent(t *testing.T) {
	m := &scopeMem{}
	s := MemoryScope{Project: "proj", Agent: "ag"}
	if err := RecordPrivate(context.Background(), m, s, Node{Key: "skill", Kind: KindDecision}); err != nil {
		t.Fatalf("RecordPrivate: %v", err)
	}
	if !hasLink(m, "ag", "skill", RelContains) {
		t.Fatalf("private skill not linked under agent: %+v", m.links)
	}
	if hasLink(m, "proj", "skill", RelContains) {
		t.Fatalf("private skill leaked under project: %+v", m.links)
	}
}

func TestRecordPrivateFallsBackToProjectWithoutAgent(t *testing.T) {
	m := &scopeMem{}
	s := MemoryScope{Project: "proj"} // no Agent
	if err := RecordPrivate(context.Background(), m, s, Node{Key: "skill"}); err != nil {
		t.Fatalf("RecordPrivate: %v", err)
	}
	if !hasLink(m, "proj", "skill", RelContains) {
		t.Fatalf("expected fallback under project: %+v", m.links)
	}
}

func TestRecallScopedMergesAndDedups(t *testing.T) {
	m := &scopeMem{graphs: map[string]Subgraph{
		"proj": {Root: Node{Key: "proj", Kind: KindProject}, Nodes: []Node{{Key: "fact"}, {Key: "dup"}}},
		"ag":   {Root: Node{Key: "ag", Kind: KindAgent}, Nodes: []Node{{Key: "skill"}, {Key: "dup"}}},
	}}
	sg, err := RecallScoped(context.Background(), m, MemoryScope{Project: "proj", Agent: "ag"}, 1)
	if err != nil {
		t.Fatalf("RecallScoped: %v", err)
	}
	if sg.Root.Key != "proj" {
		t.Fatalf("root should be the shared project, got %q", sg.Root.Key)
	}
	keys := map[string]int{}
	for _, n := range sg.Nodes {
		keys[n.Key]++
	}
	for _, want := range []string{"proj", "ag", "fact", "skill", "dup"} {
		if keys[want] == 0 {
			t.Fatalf("merged subgraph missing %q: %+v", want, sg.Nodes)
		}
	}
	if keys["dup"] != 1 {
		t.Fatalf("dup node not de-duplicated: appears %d times", keys["dup"])
	}
}

func TestRecallScopedSharedOnlyWithoutAgent(t *testing.T) {
	m := &scopeMem{graphs: map[string]Subgraph{
		"proj": {Root: Node{Key: "proj"}, Nodes: []Node{{Key: "fact"}}},
	}}
	sg, err := RecallScoped(context.Background(), m, MemoryScope{Project: "proj"}, 1)
	if err != nil {
		t.Fatalf("RecallScoped: %v", err)
	}
	if sg.Root.Key != "proj" || len(sg.Nodes) != 1 || sg.Nodes[0].Key != "fact" {
		t.Fatalf("expected shared-only view, got %+v", sg)
	}
}
