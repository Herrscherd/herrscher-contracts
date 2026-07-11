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

// MergeRequest is the typed intent for a merge L←W: the lead L aggregates
// worker W's committed branch (session/<W>) into L's own worktree via a real
// git merge. Lead-initiated: L decides when and which worker to pull. W stays
// alive; the merge has no effect on the delivery-tracking join state.
type MergeRequest struct {
	FromSession string // the lead triggering the merge (the merge target)
	Worker      string // the worker whose session/<Worker> branch is aggregated
}

// SealRequest is a lead declaring how many workers its cohort expects, turning
// the best-effort join count into a deterministic barrier.
type SealRequest struct {
	FromSession string // the lead that declares
	Expected    int    // N expected (> 0)
}

// FanOutRequest is a lead spawning a whole cohort in one signal: one worker per
// task, all children of FromSession off its committed tip, all provisioned from
// the single agent ToAgent. It is the batch counterpart of DelegateRequest.
type FanOutRequest struct {
	FromSession string   // the lead spawning the cohort (each worker's base and parent)
	ToAgent     string   // durable agent profile shared by every worker
	Tasks       []string // one task per worker (≥ 1); each seeds its worker's opening turn
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
	// Merge aggregates worker W's committed branch into the lead's worktree and
	// returns the lead's name. It errors on an unknown lead or worker, a worker
	// that is not a child of this lead, or a dirty lead/worker worktree. A merge
	// conflict is not an error: the merge is aborted (lead left clean) and the
	// lead is seeded a diagnostic. W stays alive; join state is untouched.
	Merge(ctx context.Context, req MergeRequest) (lead string, err error)
	// Seal records the number of workers FromSession's cohort expects, so the
	// join can report "cohort complete" deterministically instead of best-effort.
	Seal(ctx context.Context, req SealRequest) (lead string, err error)
	// FanOut spawns one worker per task (all children of FromSession off its
	// committed tip, all from ToAgent) and seals the cohort to its real size,
	// returning the spawned worker names. It is the batch counterpart of Delegate.
	// Same per-worker guards as Delegate (unknown agent, missing/dirty lead, failed
	// create); a spawn failure mid-batch is not rolled back — the workers already
	// created are returned alongside the error, and the cohort is sealed to what was
	// actually spawned.
	FanOut(ctx context.Context, req FanOutRequest) (spawned []string, err error)
}
