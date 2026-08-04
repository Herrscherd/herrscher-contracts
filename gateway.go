package contracts

import "context"

// Gateway is the port a chat-platform plugin implements. Method-based: the
// manager calls the rich method and the degrading decorator rabats when a
// capability is missing.
type Gateway interface {
	Manifest() Manifest
	Post(ctx context.Context, conv Conversation, text string) (MessageID, error)
	Reply(ctx context.Context, conv Conversation, replyTo MessageID, text string) (MessageID, error)
	React(ctx context.Context, conv Conversation, msg MessageID, emoji string) error
	Menu(ctx context.Context, conv Conversation, replyTo MessageID, prompt string, opts []Choice) error
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
