package contracts

import "context"

// Backend is the model edge: it turns one inbound prompt into a reply, optionally
// emitting intermediate progress events. The host neither knows nor cares which
// model answers (Claude, Codex, …) — every backend is one implementation of this
// port. Close releases any persistent process the backend holds.
type Backend interface {
	Respond(ctx context.Context, p Prompt, onEvent func(BackendEvent)) (string, error)
	Close() error
}

// ResumeAware is implemented by backends that can be resumed later. The host
// reads the opaque token after a turn, persists it, and feeds it back at
// construction via cfg.Settings["resume"]. "" means "no resumable id yet".
type ResumeAware interface {
	ResumeToken() string
}

// Prompt is the platform-neutral input a backend answers: the message text, who
// sent it, its identity, and local filesystem paths to any attachments already
// downloaded for the backend to reference. Context carries memory-recalled
// background (empty when no Memory plugin is wired) — data the backend fences
// into the turn for continuity, never the user's instructions.
type Prompt struct {
	Content     string
	Context     string
	Author      string
	MessageID   string
	ChannelID   string
	Attachments []string
}

// BackendEvent is one intermediate occurrence a backend surfaces mid-turn for a
// progress consumer: a tool invocation, streamed assistant text, or the terminal
// result carrying cost.
type BackendEvent struct {
	Kind      string  // "tool" | "text" | "thinking" | "usage" | "result" | "reset"
	Tool      string  // tool name (Kind == "tool")
	Detail    string  // tool: salient input field; text/thinking: the assistant text
	Cost      float64 // Kind == "result": total cost in USD
	IsError   bool    // Kind == "result"
	InTokens  int     // Kind == "usage" (live) and "result" (final): input tokens
	OutTokens int     // Kind == "usage" (live) and "result" (final): output tokens
}

// PendingChoice is an interactive selection a backend is waiting on after a turn
// (e.g. a tool-permission prompt), surfaced so the host can render a select menu.
type PendingChoice struct {
	Question string
	Options  []Choice
}

// ChoiceAware is implemented by backends that can pause on an interactive choice;
// after Respond the host asks PendingChoice whether to attach a select menu.
type ChoiceAware interface {
	PendingChoice() (PendingChoice, bool)
}

// ChoiceInjector is implemented by backends that can answer a pending choice
// out-of-band (a routed select-menu click), serialized with normal turns.
type ChoiceInjector interface {
	InjectChoice(ctx context.Context, value string) (string, error)
}
