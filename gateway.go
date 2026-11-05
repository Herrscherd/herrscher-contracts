package contracts

import (
	"context"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

// Gateway is the port a chat-platform plugin implements. Method-based: the
// manager calls the rich method and the degrading decorator falls back when a
// capability is missing.
type Gateway interface {
	Manifest() Manifest
	Post(ctx context.Context, conv Conversation, text string) (MessageID, error)
	Reply(ctx context.Context, conv Conversation, replyTo MessageID, text string) (MessageID, error)
	React(ctx context.Context, conv Conversation, msg MessageID, emoji string) error
	Menu(ctx context.Context, conv Conversation, replyTo MessageID, prompt string, opts []Choice) error
}

// GatewayCloser is the optional shutdown port of a gateway. Close stops the
// goroutines the plugin started and waits for them to return, bounded by ctx.
// If ctx expires before they have returned, Close returns ctx.Err().
type GatewayCloser interface {
	Close(ctx context.Context) error
}

// NoticeSink is the optional port of a gateway that renders a Notice itself, in
// the language it is given, with its own markup and emojis. A gateway without it
// receives the notice as plain text, rendered by Degrade, so it never shows a
// notice in a language it was not asked for.
type NoticeSink interface {
	Notify(ctx context.Context, conv Conversation, lang i18n.Language, notice Notice) (MessageID, error)
}

// Inbound is a message arriving from a gateway into the manager.
type Inbound struct {
	Conversation Conversation
	Author       string
	// AuthorID is the platform's stable id for the author, used for participant
	// journalling. Author is the display name shown to the agent.
	AuthorID    string
	Text        string
	Attachments []Attachment
	MessageID   MessageID
}
