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
	// Nodes stays root-excluded: the shared root "proj" is sg.Root, not a node.
	// The private root "ag" is a genuine non-root node, so it does appear.
	if keys["proj"] != 0 {
		t.Fatalf("shared root should not be listed in Nodes: %+v", sg.Nodes)
	}
	for _, want := range []string{"ag", "fact", "skill", "dup"} {
		if keys[want] == 0 {
			t.Fatalf("merged subgraph missing %q: %+v", want, sg.Nodes)
		}
	}
	if keys["dup"] != 1 {
		t.Fatalf("dup node not de-duplicated: appears %d times", keys["dup"])
	}
}

func TestRecallScopedDedupsEdges(t *testing.T) {
	edge := Link{To: "dup", Rel: RelContains}
	m := &scopeMem{graphs: map[string]Subgraph{
		"proj": {Root: Node{Key: "proj"}, Edges: []Link{edge}},
		"ag":   {Root: Node{Key: "ag"}, Edges: []Link{edge}},
	}}
	sg, err := RecallScoped(context.Background(), m, MemoryScope{Project: "proj", Agent: "ag"}, 1)
	if err != nil {
		t.Fatalf("RecallScoped: %v", err)
	}
	n := 0
	for _, e := range sg.Edges {
		if e == edge {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("identical edge not de-duplicated: appears %d times", n)
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

func keysOf(ns []Node) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = n.Key
	}
	return out
}

func TestRecallRelevant_TopKByScoreAcrossScopes(t *testing.T) {
	m := &scopeMem{graphs: map[string]Subgraph{
		"proj": {
			Root:  Node{Key: "proj", Kind: KindProject, Title: "root", Links: []Link{{To: "a", Rel: RelContains}, {To: "b", Rel: RelContains}}},
			Nodes: []Node{{Key: "a", Kind: KindDecision, Title: "use nats transport"}, {Key: "b", Kind: KindSession, Title: "nats note"}},
		},
		"agent": {
			Root:  Node{Key: "agent", Kind: KindAgent, Title: "me", Links: []Link{{To: "c", Rel: RelContains}}},
			Nodes: []Node{{Key: "c", Kind: KindSession, Title: "redis cache"}}, // no match
		},
	}}
	s := MemoryScope{Project: "proj", Agent: "agent"}

	got, err := RecallRelevant(context.Background(), m, s, "nats transport", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want top-2, got %d (%v)", len(got), keysOf(got))
	}
	if got[0].Key != "a" {
		t.Fatalf("highest-score node should be 'a', got %q", got[0].Key)
	}
	for _, n := range got {
		if n.Key == "c" {
			t.Fatal("zero-text-match node 'c' must never appear")
		}
	}
}

func TestRecallRelevant_ProximityBreaksTies(t *testing.T) {
	// near and far have identical text/kind; near is a direct child of the root.
	m := &scopeMem{graphs: map[string]Subgraph{
		"proj": {
			Root: Node{Key: "proj", Kind: KindProject, Title: "root", Links: []Link{{To: "near", Rel: RelContains}, {To: "mid", Rel: RelContains}}},
			Nodes: []Node{
				{Key: "near", Kind: KindDecision, Title: "nats"},
				{Key: "mid", Kind: KindDecision, Title: "hop", Links: []Link{{To: "far", Rel: RelContains}}},
				{Key: "far", Kind: KindDecision, Title: "nats"},
			},
		},
	}}
	s := MemoryScope{Project: "proj"}

	got, err := RecallRelevant(context.Background(), m, s, "nats", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Key != "near" {
		t.Fatalf("nearer node should rank first on tie; got %v", keysOf(got))
	}
}

func TestRecallRelevant_ExcludesNonMatchesAndRespectsK(t *testing.T) {
	m := &scopeMem{graphs: map[string]Subgraph{
		"proj": {
			Root: Node{Key: "proj", Kind: KindProject, Title: "root", Links: []Link{{To: "a", Rel: RelContains}, {To: "b", Rel: RelContains}, {To: "c", Rel: RelContains}}},
			Nodes: []Node{
				{Key: "a", Kind: KindDecision, Title: "nats one"},
				{Key: "b", Kind: KindDecision, Title: "nats two nats"},
				{Key: "c", Kind: KindDecision, Title: "unrelated"},
			},
		},
	}}
	s := MemoryScope{Project: "proj"}

	got, err := RecallRelevant(context.Background(), m, s, "nats", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("k=1 should cap to a single result, got %v", keysOf(got))
	}
	if got[0].Key != "b" {
		t.Fatalf("higher term-frequency node 'b' should win, got %q", got[0].Key)
	}
}
