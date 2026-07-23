package contracts

import "context"

// TurnReactor is an OPTIONAL Orchestrator capability: it reacts to the model's
// reply after a turn, handling in-band memory markers (<recall> to search memory,
// <remember> to store a durable fact) and returning the reply with those markers
// stripped so the human never sees them. Orchestrators that do not implement it
// are unaffected — the host type-asserts, exactly like SkillNative / ResumeAware.
type TurnReactor interface {
	// React handles any memory markers in reply and returns the cleaned reply. It
	// is best-effort: a memory failure never breaks the turn, it just yields the
	// reply unchanged (minus any markers it managed to strip).
	React(ctx context.Context, reply string) string
}
