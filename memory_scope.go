package contracts

import (
	"context"
	"sort"
	"time"
)

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

// ProjectKey / AgentKey are the single source of truth for scope-root Keys, so
// the orchestrator (which derives a MemoryScope) and the provisioners (which
// create the root nodes) can never drift apart. Scheme: flat, English, no /index.
func ProjectKey(name string) string { return "projects/" + name }
func AgentKey(name string) string   { return "agents/" + name }

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

// relevantDepth bounds how far RecallRelevant walks from each scope root before
// ranking. Deep enough to reach a project's facts and an agent's skills, shallow
// enough to keep the candidate set (and cost) small.
const relevantDepth = 3

// RecallRelevant returns the top-k nodes from the scoped subgraph ranked by
// relevance to text, instead of the full merged subgraph — bounding how much is
// primed into a prompt. Nodes with no textual match are excluded; among matches,
// term frequency, title hits, recency, kind, and graph proximity to a scope root
// order the result (highest first). Fewer than k are returned when fewer match;
// k <= 0 returns all matches.
func RecallRelevant(ctx context.Context, m Memory, s MemoryScope, text string, k int) ([]Node, error) {
	sg, err := RecallScoped(ctx, m, s, relevantDepth)
	if err != nil {
		return nil, err
	}
	depth := scopeDepths(sg, s)
	r := newRanker(text, time.Now().UTC())

	type scored struct {
		n     Node
		score float64
	}
	// Dedup by Key: RecallScoped's merge lists both roots inside Nodes, so the
	// root can appear twice. Iterate {Root}+Nodes once, skipping repeats.
	seen := map[string]bool{}
	var hits []scored
	for _, n := range append([]Node{sg.Root}, sg.Nodes...) {
		if n.Key == "" || seen[n.Key] {
			continue
		}
		seen[n.Key] = true
		base, ok := r.score(n)
		if !ok {
			continue // no textual match → excluded
		}
		d, reached := depth[n.Key]
		if !reached {
			d = relevantDepth // unreached from either root: treat as farthest
		}
		hits = append(hits, scored{n: n, score: base + weightProx*proximityBoost(d)})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if k > 0 && len(hits) > k {
		hits = hits[:k]
	}
	out := make([]Node, len(hits))
	for i, h := range hits {
		out[i] = h.n
	}
	return out, nil
}

// scopeDepths BFS-labels every node in sg with its shortest distance (in edges)
// from a scope root (Project or Agent), following node Links (Link records no
// source, so Edges alone can't drive the walk). Roots are depth 0; nodes
// unreachable from either root are absent from the map.
func scopeDepths(sg Subgraph, s MemoryScope) map[string]int {
	byKey := map[string]Node{sg.Root.Key: sg.Root}
	for _, n := range sg.Nodes {
		byKey[n.Key] = n
	}
	depth := map[string]int{}
	var frontier []string
	for _, root := range []string{s.Project, s.Agent} {
		if root == "" {
			continue
		}
		if _, known := byKey[root]; !known {
			continue
		}
		if _, dup := depth[root]; !dup {
			depth[root] = 0
			frontier = append(frontier, root)
		}
	}
	for len(frontier) > 0 {
		var next []string
		for _, cur := range frontier {
			for _, l := range byKey[cur].Links {
				if _, ok := depth[l.To]; ok {
					continue
				}
				if _, exists := byKey[l.To]; !exists {
					continue
				}
				depth[l.To] = depth[cur] + 1
				next = append(next, l.To)
			}
		}
		frontier = next
	}
	return depth
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
