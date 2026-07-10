package contracts

import "context"

// HandoffRequest is the typed intent for a relay A→B: FromSession finishes and B
// (a ToAgent profile) continues the same committed work. Task seeds B's opening
// turn. It is the shared source of truth for every coordination policy built on
// top of Coordinator (handoff now, supervisor→workers and fan-out later).
type HandoffRequest struct {
	FromSession string // the source session handing off (its branch is B's base)
	ToAgent     string // durable agent profile B is provisioned from
	Task        string // seeds B's opening turn
}

// Coordinator is the inter-session coordination port. It lives at the layer that
// sees every session and drives the hub (the host), NOT the per-session
// Orchestrator plugin (which only sees its own turns and holds no hub handle).
// The agent only signals intent; the host validates and the Coordinator executes.
type Coordinator interface {
	// Handoff creates B continuing FromSession's committed work and returns B's
	// session name. It errors on unknown agent, a missing/dirty source worktree,
	// or a failed create — leaving nothing partial.
	Handoff(ctx context.Context, req HandoffRequest) (session string, err error)
}
