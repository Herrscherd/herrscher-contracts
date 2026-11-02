package contracts

import (
	"context"
	"sort"
	"strings"
	"time"
	"unicode"
)

// MemoryScope expresses the P1 sharing policy *over the existing graph*. Not a
// new port. A game's durable memory hangs under the shared Project node (every
// agent of the game recalls it); an agent's learned skills hang under its own
// Agent node (private to that agent). A scope names both roots for one agent
// working on one project. An empty Agent means "no private scope": private
// writes fall back to shared and scoped recall returns only the shared view.
type MemoryScope struct {
	Project string
	Agent   string
}

// Relations the policy uses when hanging facts/skills off a root.
const (
	RelContains  = "contains"
	RelAppliesTo = "applies-to"
)

// ProjectKey / AgentKey are the single source of truth for scope-root Keys, so
// the orchestrator (which derives a MemoryScope) and the provisioners (which
// create the root nodes) can never drift apart. Scheme: flat, English, no /index.
func ProjectKey(name string) string { return "projects/" + normalizeScopeName(name) }

// AgentKey returns the scope root key of the agent called name, under the agents/
// prefix, normalized the same way as ProjectKey.
func AgentKey(name string) string { return "agents/" + normalizeScopeName(name) }

// NormalizeScope returns the name segment that ProjectKey and AgentKey put in
// their keys: the words of name in lower case, joined by hyphens, or "scope" when
// no word remains. A host that derives a scope name from elsewhere, such as a
// directory or a prompt, uses it so that its name matches the keys exactly, and
// both sides always agree on whether two spellings name the same project.
func NormalizeScope(name string) string { return normalizeScopeName(name) }

func normalizeScopeName(name string) string {
	var normalized strings.Builder
	pendingSep := false
	for _, char := range strings.TrimSpace(name) {
		switch {
		case isWordRune(char):
			if pendingSep && normalized.Len() > 0 {
				_ = normalized.WriteByte('-')
			}
			pendingSep = false
			_, _ = normalized.WriteRune(unicode.ToLower(char))
		default:
			pendingSep = true
		}
	}
	if normalized.Len() == 0 {
		return "scope"
	}
	return normalized.String()
}

// RecordShared upserts n and links it under the project root, making it visible
// to every agent of the game. Use for project memory (decisions, conventions,
// the Studio tree map, …).
func RecordShared(ctx context.Context, memory Memory, scope MemoryScope, node Node) error {
	if err := memory.Record(ctx, node); err != nil {
		return err
	}
	return memory.Links(ctx, scope.Project, node.Key, RelContains)
}

// RecordPrivate upserts n and links it under the agent root, keeping it private
// to this agent. Use for learned skills. With no Agent in scope it falls back to
// shared so a fact is never dropped on the floor.
func RecordPrivate(ctx context.Context, memory Memory, scope MemoryScope, node Node) error {
	if err := memory.Record(ctx, node); err != nil {
		return err
	}
	root := scope.Agent
	if root == "" {
		root = scope.Project
	}
	return memory.Links(ctx, root, node.Key, RelContains)
}

// RecallScoped returns what an agent should see at the start of a turn: the
// shared project subgraph merged with the agent's private subgraph, de-duplicated
// by Key. depth bounds link-following from each root. With no Agent in scope it
// returns the shared subgraph alone.
func RecallScoped(ctx context.Context, memory Memory, scope MemoryScope, depth int) (Subgraph, error) {
	shared, err := memory.Recall(ctx, scope.Project, depth)
	if err != nil {
		return Subgraph{}, err
	}
	if scope.Agent == "" {
		return shared, nil
	}
	private, err := memory.Recall(ctx, scope.Agent, depth)
	if err != nil {
		return Subgraph{}, err
	}
	return mergeSubgraphs(shared, private), nil
}

const relevantDepth = 3

// RecallRelevant returns up to limit nodes of the scoped subgraph, ranked by
// relevance to text, instead of the whole merged subgraph, so that only a bounded
// number of nodes is primed into a prompt. Nodes with no textual match are
// excluded. Among the matches, term frequency, title hits, recency measured at
// now, kind and graph proximity to a scope root order the result, highest first.
// Fewer than limit nodes are returned when fewer match; a limit of zero or less
// returns every match.
func RecallRelevant(ctx context.Context, memory Memory, scope MemoryScope, text string, limit int, now time.Time) ([]Node, error) {
	subgraph, err := RecallScoped(ctx, memory, scope, relevantDepth)
	if err != nil {
		return nil, err
	}
	depth := scopeDepths(&subgraph, scope)
	ranker := newRanker(text, now)

	type scored struct {
		n     Node
		score float64
	}
	seen := make(map[string]bool, len(subgraph.Nodes)+1)
	hits := make([]scored, 0, len(subgraph.Nodes)+1)
	consider := func(node Node) {
		if node.Key == "" || seen[node.Key] {
			return
		}
		seen[node.Key] = true
		base, ok := ranker.score(node)
		if !ok {
			return
		}
		distance, reached := depth[node.Key]
		if !reached {
			distance = relevantDepth
		}
		hits = append(hits, scored{n: node, score: base + weightProx*proximityBoost(distance)})
	}
	consider(subgraph.Root)
	for _, n := range subgraph.Nodes {
		consider(n)
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]Node, len(hits))
	for i, h := range hits {
		out[i] = h.n
	}
	return out, nil
}

func scopeDepths(sg *Subgraph, scope MemoryScope) map[string]int {
	byKey := indexNodes(sg)
	depth := make(map[string]int, len(byKey))
	frontier := seedScopeDepths(byKey, scope, depth)
	for len(frontier) > 0 {
		frontier = expandScopeDepths(byKey, depth, frontier)
	}
	return depth
}

func indexNodes(subgraph *Subgraph) map[string]Node {
	byKey := make(map[string]Node, len(subgraph.Nodes)+1)
	byKey[subgraph.Root.Key] = subgraph.Root
	for _, n := range subgraph.Nodes {
		byKey[n.Key] = n
	}
	return byKey
}

func seedScopeDepths(byKey map[string]Node, scope MemoryScope, depth map[string]int) []string {
	var frontier []string
	for _, root := range []string{scope.Project, scope.Agent} {
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
	return frontier
}

func expandScopeDepths(byKey map[string]Node, depth map[string]int, frontier []string) []string {
	var next []string
	for _, cur := range frontier {
		for _, link := range byKey[cur].Links {
			if _, ok := depth[link.To]; ok {
				continue
			}
			if _, exists := byKey[link.To]; !exists {
				continue
			}
			depth[link.To] = depth[cur] + 1
			next = append(next, link.To)
		}
	}
	return next
}

func mergeSubgraphs(shared, private Subgraph) Subgraph {
	out := Subgraph{Root: shared.Root}
	seen := make(map[string]bool)
	if shared.Root.Key != "" {
		seen[shared.Root.Key] = true
	}
	add := func(nodes ...Node) {
		for _, node := range nodes {
			if node.Key == "" || seen[node.Key] {
				continue
			}
			seen[node.Key] = true
			out.Nodes = append(out.Nodes, node)
		}
	}
	add(private.Root)
	add(shared.Nodes...)
	add(private.Nodes...)
	seenEdge := make(map[Link]bool)
	addEdges := func(edges []Link) {
		for _, edge := range edges {
			if seenEdge[edge] {
				continue
			}
			seenEdge[edge] = true
			out.Edges = append(out.Edges, edge)
		}
	}
	addEdges(shared.Edges)
	addEdges(private.Edges)
	return out
}
