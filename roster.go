package contracts

// AgentInfo is the delegation-relevant projection of a roster agent: what the
// model needs to name a delegate and reason about what it is good for.
type AgentInfo struct {
	Name    string   // name used in a ⟢ delegate: <name> marker
	Backend string   // backend vendor it runs on (claude, codex, …); "" = host default
	Summary string   // one-line description for the menu (may be empty)
	Tags    []string // capability tags, also what ⟢ route: matches on
}

// RosterProvider lists the agents a session may delegate to. It is an OPTIONAL
// capability the host supplies to the bridge: a nil provider (or an empty roster)
// yields no delegation affordance, so a deployment with no agents is unchanged.
type RosterProvider interface {
	Agents() []AgentInfo
}
