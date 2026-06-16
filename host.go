package contracts

import "context"

// Channel identifies a conversation a gateway can create or reuse.
type Channel struct {
	ID   string
	Name string
}

// CommandSource is a gateway's stream of inbound commands feeding the host
// dispatch loop. Run connects and pushes onto Commands until ctx is cancelled or
// the connection drops (returning an error so the host can reconnect).
type CommandSource interface {
	Run(ctx context.Context) error
	Commands() <-chan InboundCommand
}

// CommandResponder answers an inbound command, addressed by its ResponseToken.
// Slow returns whether a command needs the ack-then-edit (deferred) path.
type CommandResponder interface {
	Defer(ctx context.Context, tok ResponseToken, private bool) error
	Respond(ctx context.Context, tok ResponseToken, resp CommandResponse) error
	Edit(ctx context.Context, tok ResponseToken, resp CommandResponse) error
	Autocomplete(ctx context.Context, tok ResponseToken, choices []AutocompleteChoice) error
	AckComponent(ctx context.Context, tok ResponseToken, content string) error
}

// CommandRegistrar publishes the host's command surface to the gateway.
type CommandRegistrar interface {
	Register(ctx context.Context) error
}

// Prober measures gateway round-trip reachability for liveness.
type Prober interface {
	Probe(ctx context.Context) (latencyMS int64, err error)
}

// StatusReporter maintains a single self-updating status message, returning the
// (possibly new) message id to persist.
type StatusReporter interface {
	Upsert(ctx context.Context, prevID, content string) (string, error)
}

// Platform is what the bridge loop needs from a chat platform beyond outbound
// messaging (Gateway): reading a conversation, channel bootstrap, reaction
// removal, the self-updating progress message, and a routed select menu whose
// clicks return to the named session.
type Platform interface {
	Enabled() bool
	DefaultChannel() string
	EnsureChannel(ctx context.Context, parentID, name string) (Channel, error)
	Read(ctx context.Context, channelID string, limit int, after string) ([]Message, error)
	Unreact(ctx context.Context, channelID, messageID, emoji string) error
	UpsertStatusMessage(ctx context.Context, channelID, messageID, content string) (string, error)
	SendSelectMenu(ctx context.Context, channelID, replyTo, content, session string, opts []Choice) (MessageID, error)
}
