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

// DelegateRequest is the typed intent for a delegation L→W: the lead L hands a
// task to a worker W branched off L's committed tip. Unlike a handoff, L stays
// alive and W records L as its parent for the result-back channel (Report).
type DelegateRequest struct {
	FromSession string // the lead delegating (its branch is W's base and W's parent)
	ToAgent     string // durable agent profile W is provisioned from
	Task        string // seeds W's opening turn
}

// ReportRequest is a worker W's completion signal to its lead: W reports a
// summary and the coordinator delivers {session/<W> branch ref + summary} to
// W's parent. W stays alive (pure delivery — no teardown, no merge).
type ReportRequest struct {
	FromSession string // the worker reporting (its parent is the delivery target)
	Summary     string // free-text completion summary delivered to the lead
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
	// Delegate creates a worker W off FromSession's committed tip, records
	// FromSession as W's parent, and returns W's name. The lead stays alive.
	// Same guards as Handoff (unknown agent, missing/dirty source, failed create).
	Delegate(ctx context.Context, req DelegateRequest) (worker string, err error)
	// Report delivers the worker's completion (branch ref + summary) to its
	// parent and returns the parent's name. It errors on an unknown worker, a
	// worker with no parent, or a parent no longer present. W stays alive.
	Report(ctx context.Context, req ReportRequest) (parent string, err error)
}
