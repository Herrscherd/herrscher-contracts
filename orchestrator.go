package contracts

import "context"

// Orchestrator is the conversation-policy port: it decides how a turn is run.
// The host drives it around each turn — Context primes the backend before a
// turn, Observe records the turn after — and it owns whatever curation strategy
// sits over Memory (a rolling transcript, summarisation, multi-agent routing,
// …). It is the edge where conversation behaviour is swapped without touching
// the gateway, backend, or memory ports.
//
// An Orchestrator is session-scoped: the host builds one per session and passes
// the session name in the factory's PluginConfig (key "session"). It composes
// the Memory port the factory receives — a nil Memory means recall/record are
// no-ops, so an orchestrator still answers, just without continuity.
type Orchestrator interface {
	// Context returns background to prepend to the next prompt ("" = none, e.g.
	// the first turn). It must never return a turn-breaking error: on any failure
	// it yields "".
	Context(ctx context.Context) string
	// Observe records a completed turn (the inbound prompt and the reply). A
	// best-effort error is returned for logging; the host never breaks the loop on
	// it.
	Observe(ctx context.Context, p Prompt, reply string) error
	// CurationHook (Consolidate) is the proactive-curation seam: summarise/prune
	// out of band. The default orchestrator keeps a bounded transcript inline and
	// no-ops it.
	CurationHook
	// Close releases any resources the orchestrator holds.
	Close() error
}
