package contracts

import "context"

// Channel identifies a conversation a gateway can create or reuse.
type Channel struct {
	ID   string
	Name string
}

// Prober measures gateway round-trip reachability for liveness.
type Prober interface {
	Probe(ctx context.Context) (latencyMS int64, err error)
}

// ChannelReader is the neutral read/lifecycle side of a channel port: presence,
// the default conversation, channel bootstrap, history reads, reaction removal,
// and the single self-updating status message. A gateway that drives the bridge
// and the status loop implements it alongside Gateway. Optional: a gateway that
// only emits messages may omit it (the host degrades).
type ChannelReader interface {
	Enabled() bool
	DefaultChannel() string
	EnsureChannel(ctx context.Context, parentID, name string) (Channel, error)
	Read(ctx context.Context, channelID string, limit int, after string) ([]Message, error)
	Unreact(ctx context.Context, channelID, messageID, emoji string) error
	UpsertStatusMessage(ctx context.Context, channelID, messageID, content string) (string, error)
}

// MenuRouter is an optional channel capability: post an interactive menu whose
// picks are delivered back to a named neutral route (e.g. a session) rather than
// to the channel the menu lives in. The plugin owns how a pick is encoded and
// delivered back to that route — contracts never sees the wire encoding.
type MenuRouter interface {
	RouteMenu(ctx context.Context, channelID, replyTo, prompt, route string, opts []Choice) (MessageID, error)
}

// ChannelAdmin is the optional channel-management capability the manager needs to
// create/archive session channels and post into them.
type ChannelAdmin interface {
	Kind(ctx context.Context, id string) (string, error)
	CreateUnder(ctx context.Context, parentID, name string) (channelID string, err error)
	ForumPost(ctx context.Context, forumID, name, content string) (channelID string, err error)
	Archive(ctx context.Context, id string) error
	Send(ctx context.Context, channelID, content string) error
}
