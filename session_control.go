package contracts

import "context"

// SessionControl is the neutral seam by which a gateway drives the running hub's
// session lifecycle. The hub implements it; a gateway receives it via
// SessionControlReceiver and calls it from its own command handlers. The gateway
// formats its platform input (e.g. a Discord slash interaction) into a neutral
// argv and dispatches it, so the core never learns the gateway's command surface
// and stays platform-agnostic.
type SessionControl interface {
	// Dispatch runs one operator command (session/service/set …) given its argv —
	// the same vocabulary the operator CLI uses, e.g.
	// {"session","create","--name","foo"}. It applies the command and, for
	// lifecycle changes, brings the affected sessions live (or tears them down) in
	// the running hub. It returns a human-readable result or an error.
	Dispatch(ctx context.Context, args []string) (string, error)
	// Sessions returns a snapshot of the hub's sessions, for a gateway that needs
	// to enumerate them (e.g. autocompleting a session name).
	Sessions() []SessionInfo
}

// SessionInfo is a read-only view of a hub session.
type SessionInfo struct {
	Name      string
	ChannelID string
	Type      string // "text" | "forum"
	Gateways  []string
}

// SessionControlReceiver is the opt-in seam by which a gateway receives the hub's
// SessionControl. After the hub is built, the host calls BindSessionControl once
// on any Gateway that implements it. Implementing it is how a gateway drives the
// session lifecycle (e.g. from slash commands) without the host knowing the
// gateway's platform.
type SessionControlReceiver interface {
	BindSessionControl(SessionControl)
}
