package contracts

// Conversation is an opaque address into a chat platform. Comparable so it can
// key a map (Conversation -> SessionID).
type Conversation struct {
	Gateway string
	ID      string
}

type (
	// SessionID identifies the agent session that serves one conversation.
	SessionID string
	// MessageID identifies a message on the chat platform, so that a reply or a reaction can point at it.
	MessageID string
)

// Choice is one option of a select menu. Label is what the user reads, and Value is
// what the plugin receives back when the option is picked.
type Choice struct {
	Label string
	Value string
}

// Attachment is a file sent with a message, located by its URL on the platform.
// Size is the size the platform reports.
type Attachment struct {
	Filename    string
	URL         string
	ContentType string
	Size        int
}
