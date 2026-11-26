package contracts

// ContextPart names a block the host writes into the context of every turn,
// beside the prompt of the vendor it drives. A reader that meets a part it does
// not know shows it under its wire name: the list grows with the host, and a
// size estimate is never a reason to refuse an event.
type ContextPart string

// The blocks the host injects into a turn.
const (
	ContextMemory       ContextPart = "memory"
	ContextSkills       ContextPart = "skills"
	ContextDelegation   ContextPart = "delegation"
	ContextCapabilities ContextPart = "capabilities"
	ContextIdentity     ContextPart = "identity"
	// ContextSelfTools is the declarations of the herrscher MCP server a session gets, which the vendor sends with every turn.
	ContextSelfTools ContextPart = "self_tools"
)

// InjectedPart is the estimated size of one block the host injected into a
// turn. Tokens is an estimate made by the host from the text it wrote, not a
// count reported by the vendor.
type InjectedPart struct {
	Part   ContextPart `json:"part"`
	Tokens int         `json:"tokens"`
}
