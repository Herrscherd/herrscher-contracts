package contracts

import (
	"context"
	"errors"

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
// the language it is given, with its own markup and emojis. Notify receives lang
// as the recipient's language: a message the shared catalog lacks in lang renders
// in English, as the catalog does. A gateway without NoticeSink receives the
// notice as plain text, rendered by Degrade in lang with the same English fallback.
// A CodeUnexpected notice carries its Detail, the technical text of a failure: a
// gateway renders it in a code block, never translated, and Degrade renders it as
// plain text.
//
// When the gateway cannot render the notice, it may post a plain-text fallback in
// its place and return an error matching ErrNoticeFallbackPosted. The fallback has
// reached the conversation: the caller must not retry Notify, nor post the notice
// again through Degrade, and must treat the notice as delivered.
type NoticeSink interface {
	Notify(ctx context.Context, conv Conversation, lang i18n.Language, notice Notice) (MessageID, error)
}

// ErrNoticeFallbackPosted reports that a NoticeSink could not render a notice and
// posted a plain-text fallback instead. Match it with errors.Is: the fallback is
// already in the conversation, so the notice must not be sent again.
var ErrNoticeFallbackPosted = errors.New("notice posted as a fallback text")

// FileSender is the optional port of a gateway that posts files: SendFiles posts text
// in reply to replyTo, with files attached. The host checks each file with
// [ValidateOutboundFile] first. A gateway without FileSender receives a text that
// lists the paths of the files, so no file is dropped.
type FileSender interface {
	SendFiles(ctx context.Context, conv Conversation, replyTo MessageID, text string, files []OutboundFile) (MessageID, error)
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
