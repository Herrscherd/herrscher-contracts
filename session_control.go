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
	// Create starts a session from a typed spec, the structured counterpart to a
	// "session create" Dispatch. Programmatic callers (a gateway bootstrapping its
	// default session, a UI close/create button) use this instead of assembling
	// flag argv, so a renamed flag is a compile error rather than a silent no-op.
	// It brings the new session live, like Dispatch.
	Create(ctx context.Context, spec CreateSession) (string, error)
	// Close tears a session down by name, the typed counterpart to a
	// "session close" Dispatch. force discards an uncommitted worktree.
	Close(ctx context.Context, name string, force bool) (string, error)
	// Sessions returns a snapshot of the hub's sessions, for a gateway that needs
	// to enumerate them (e.g. autocompleting a session name).
	Sessions() []SessionInfo
}

// CreateSession is the typed spec for SessionControl.Create — the structured
// equivalent of the "session create" flags. The zero value of each optional
// field means "flag omitted", so a caller sets only what it needs. Name is
// required. Gateways selects the bound gateways; an empty slice with
// TerminalOnly true binds the terminal only.
type CreateSession struct {
	Name             string
	Project          string
	Clone            string
	Cmd              string
	Backend          string
	Gateways         []string
	TerminalOnly     bool
	Shared           bool
	Agent            string
	Extractor        string
	Journal          string
	ConsolidateEvery int
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
