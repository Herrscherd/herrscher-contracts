package contracts

import "time"

// Node lifecycle states, stored in Node.Meta[MetaState]. An absent value is
// treated as StateActive.
const (
	StateActive   = "active"
	StateStale    = "stale"
	StateArchived = "archived"
)

// Reserved Meta keys for the staleness machine.
const (
	MetaState    = "state"
	MetaLastSeen = "lastSeen"
)

// NextState derives a node's lifecycle state purely from how long ago it was
// last seen. age = now.Sub(lastSeen). A duration <= 0 disables that step:
// staleAfter <= 0 means nodes never become stale; archiveAfter <= 0 means they
// never become archived. When both are set, archiveAfter should exceed
// staleAfter; if archiveAfter <= staleAfter, archival still wins once its
// threshold is crossed. The current state is intentionally not an input:
// transitions (including reactivation) depend only on age, so the function is
// total and hysteresis-free.
func NextState(lastSeen, now time.Time, staleAfter, archiveAfter time.Duration) string {
	age := now.Sub(lastSeen)
	if archiveAfter > 0 && age >= archiveAfter {
		return StateArchived
	}
	if staleAfter > 0 && age >= staleAfter {
		return StateStale
	}
	return StateActive
}
