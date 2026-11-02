package contracts

// AgentInfo is the delegation-relevant projection of a roster agent: what the
// model needs to name a delegate and reason about what it is good for.
type AgentInfo struct {
	Name    string
	Backend string
	Summary string
	Tags    []string
}

// RosterProvider lists the agents a session may delegate to. It is an OPTIONAL
// capability the host supplies to the bridge: a nil provider (or an empty roster)
// yields no delegation affordance, so a deployment with no agents is unchanged.
type RosterProvider interface {
	Agents() []AgentInfo
}
