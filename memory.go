package contracts

import "context"

// NodeKind classifies a memory node. The structural spine
// (Organization → Project → Repo/Server) plus the documentary kinds, and the
// durable Agent node that anchors per-agent private memory (see MemoryScope).
type NodeKind string

const (
	KindOrganization NodeKind = "organization"
	KindProject      NodeKind = "project"
	KindRepo         NodeKind = "repo"
	KindServer       NodeKind = "server"
	KindArchitecture NodeKind = "architecture"
	KindProduction   NodeKind = "production"
	KindSession      NodeKind = "session"
	KindDecision     NodeKind = "decision"
	KindUser         NodeKind = "user"
	// KindAgent is a durable companion. A game's shared memory hangs under its
	// Project node; an agent's learned skills hang under its own Agent node, so
	// the same graph carries both shared and private memory (see MemoryScope).
	KindAgent NodeKind = "agent"
	// KindDomain is a transverse area-of-concern root (dev, research, …) grouping
	// projects and facts topically, above the ownership spine. A project links to
	// its domain with "in-domain"; a fact carries Meta["domain"] for filtering.
	KindDomain NodeKind = "domain"
)

// Link is a directed, typed edge to another node, identified by its Key.
type Link struct {
	To  string // target node Key
	Rel string // semantic relation: "depends-on", "decided-in", "applies-to", "contains", …
}

// Node is one unit of memory: a stable Key, a Kind, human Title/Body (markdown),
// outbound Links, and flat Meta (dates, status, tags). It is storage-neutral —
// the Obsidian plugin maps Meta to frontmatter and Links to [[wikilinks]], but
// the contract says nothing about files.
type Node struct {
	Key   string
	Kind  NodeKind
	Title string
	Body  string
	Links []Link
	Meta  map[string]string
}

// Query selects nodes without knowing their Key. An empty Query matches nothing
// useful; callers set at least Text or Kinds.
type Query struct {
	Text  string
	Kinds []NodeKind
	Tags  []string
	Limit int // 0 = no limit
	// Ranked, when true, asks the Memory to return results score-sorted by
	// relevance to Text (highest first) instead of storage/walk order. The zero
	// value (false) preserves the historical unranked behaviour, so existing
	// callers are unaffected.
	Ranked bool
}

// Subgraph is a Recall result: the root node plus every node reachable within the
// requested depth and the edges connecting them.
type Subgraph struct {
	Root  Node
	Nodes []Node
	Edges []Link
}

// Memory is the persistent-recall port. Implementations store a knowledge graph
// (the Obsidian plugin uses a markdown vault). The host/orchestrator drives only
// these passive verbs; the curation behaviour lives above the port (see
// CurationHook).
type Memory interface {
	// Recall fetches the node at key and follows its links up to depth (0 = root
	// only), returning the reachable subgraph.
	Recall(ctx context.Context, key string, depth int) (Subgraph, error)
	// Record upserts a node by Key — re-recording an existing Key updates it in
	// place rather than creating a duplicate.
	Record(ctx context.Context, n Node) error
	// Search finds nodes matching the query (keyword/kind/tag).
	Search(ctx context.Context, q Query) ([]Node, error)
	// Links creates a typed edge from one node to another.
	Links(ctx context.Context, from, to, rel string) error
	// Close releases any resources the implementation holds.
	Close() error
}

// CurationHook is the seam for proactive curation; the Orchestrator implements it
// and drives Memory.Record — it is deliberately not implemented by the Memory port,
// which stays passive verbs only.
type CurationHook interface {
	Consolidate(ctx context.Context) error
}

// Provisioner is an optional Memory capability: ensuring the scope-root nodes a
// MemoryScope points at exist before any Record/Recall runs against them. It is
// deliberately NOT part of the Memory port — node-creating implementations (the
// obsidian vault) satisfy it, and callers type-assert, so a remote memory proxy
// that cannot create roots is simply skipped.
type Provisioner interface {
	// EnsureProject ensures the shared KindProject root at key exists (idempotent).
	EnsureProject(ctx context.Context, key, title string) error
	// EnsureAgent ensures the private KindAgent root at key exists (idempotent).
	EnsureAgent(ctx context.Context, key, title string) error
}
