package contracts

import "context"

// MemoryScope expresses the P1 sharing policy *over the existing graph* — not a
// new port. A game's durable memory hangs under the shared Project node (every
// agent of the game recalls it); an agent's learned skills hang under its own
// Agent node (private to that agent). A scope names both roots for one agent
// working on one project. An empty Agent means "no private scope": private
// writes fall back to shared and scoped recall returns only the shared view.
type MemoryScope struct {
	Project string // Key of the KindProject node — the shared root
	Agent   string // Key of the KindAgent node — this agent's private root ("" = none)
}

// Relations the policy uses when hanging facts/skills off a root.
const (
	RelContains  = "contains"   // structural: a root contains a fact or skill
	RelAppliesTo = "applies-to" // a skill applies to a project/system
)

// RecordShared upserts n and links it under the project root, making it visible
// to every agent of the game. Use for project memory (decisions, conventions,
// the Studio tree map, …).
func RecordShared(ctx context.Context, m Memory, s MemoryScope, n Node) error {
	if err := m.Record(ctx, n); err != nil {
		return err
	}
	return m.Links(ctx, s.Project, n.Key, RelContains)
}

// RecordPrivate upserts n and links it under the agent root, keeping it private
// to this agent. Use for learned skills. With no Agent in scope it falls back to
// shared so a fact is never dropped on the floor.
func RecordPrivate(ctx context.Context, m Memory, s MemoryScope, n Node) error {
	if err := m.Record(ctx, n); err != nil {
		return err
	}
	root := s.Agent
	if root == "" {
		root = s.Project
	}
	return m.Links(ctx, root, n.Key, RelContains)
}

// RecallScoped returns what an agent should see at the start of a turn: the
// shared project subgraph merged with the agent's private subgraph, de-duplicated
// by Key. depth bounds link-following from each root. With no Agent in scope it
// returns the shared subgraph alone.
func RecallScoped(ctx context.Context, m Memory, s MemoryScope, depth int) (Subgraph, error) {
	shared, err := m.Recall(ctx, s.Project, depth)
	if err != nil {
		return Subgraph{}, err
	}
	if s.Agent == "" {
		return shared, nil
	}
	private, err := m.Recall(ctx, s.Agent, depth)
	if err != nil {
		return Subgraph{}, err
	}
	return mergeSubgraphs(shared, private), nil
}

// mergeSubgraphs unions two subgraphs, keeping the shared root and dropping
// duplicate nodes by Key (the shared view wins on collisions).
func mergeSubgraphs(shared, private Subgraph) Subgraph {
	out := Subgraph{Root: shared.Root}
	seen := map[string]bool{}
	add := func(nodes ...Node) {
		for _, n := range nodes {
			if n.Key == "" || seen[n.Key] {
				continue
			}
			seen[n.Key] = true
			out.Nodes = append(out.Nodes, n)
		}
	}
	add(shared.Root, private.Root)
	add(shared.Nodes...)
	add(private.Nodes...)
	// Edges are deduplicated by value too: an identical (To, Rel) link present in
	// both subgraphs is redundant, and would otherwise double up in the merge.
	seenEdge := map[Link]bool{}
	for _, e := range append(append([]Link{}, shared.Edges...), private.Edges...) {
		if seenEdge[e] {
			continue
		}
		seenEdge[e] = true
		out.Edges = append(out.Edges, e)
	}
	return out
}
