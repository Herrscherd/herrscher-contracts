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
	// Scrollback returns the last recorded transcript lines for a session (empty
	// when none), so a gateway can seed a reopened view with history before live
	// events arrive. Best-effort: a session with no transcript yields nil.
	Scrollback(name string) []ScrollbackLine
	// Resume revives an archived session: it unarchives it and brings it live
	// (backend resumed via its stored token). A live session is a no-op success.
	Resume(name string) error
}

// ScrollbackLine is one replayed transcript entry, carried across the seam so a
// gateway (the terminal TUI) can repaint history without reading the state dir
// (which lives behind the core's internal packages).
type ScrollbackLine struct {
	Role string // "user" | "assistant"
	Text string
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
	// Base is the git ref the new session's worktree branches off (empty = the
	// default fresh branch). A handoff sets it to the source's session/<A> so B
	// continues A's committed tip without a merge.
	Base string
	// Parent names the lead session that delegated this one (result-back P3).
	// Empty = no parent (default; any non-delegated session). The coordinator
	// uses it to deliver a worker's completion report back to its lead.
	Parent string
}

// SessionInfo is a read-only view of a hub session.
type SessionInfo struct {
	Name      string
	ChannelID string
	Type      string // "text" | "forum"
	Gateways  []string
	// Vendor is the agent backend vendor ("claude"/"codex"/"cursor"), shown as a
	// /resume picker column. Empty when unknown.
	Vendor string
	// Project is the workspace sub-dir the session started from, a picker column.
	Project string
	// Archived is true for a closed-but-kept session: a gateway skips it when
	// building live tabs and lists it in the /resume picker.
	Archived bool
	// Resumable is true when the session carries a backend resume token (⟲ column).
	Resumable bool
	// LastTs is the last transcript entry's timestamp (RFC3339), for sorting the
	// picker by recency. Empty when the session has no transcript.
	LastTs string
}

// SessionControlReceiver is the opt-in seam by which a gateway receives the hub's
// SessionControl. After the hub is built, the host calls BindSessionControl once
// on any Gateway that implements it. Implementing it is how a gateway drives the
// session lifecycle (e.g. from slash commands) without the host knowing the
// gateway's platform.
type SessionControlReceiver interface {
	BindSessionControl(SessionControl)
}
