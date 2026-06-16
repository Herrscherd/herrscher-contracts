package contracts

// Message is an inbound chat message a gateway surfaces to the bridge. It is the
// platform-neutral shape of "someone said something in a conversation".
type Message struct {
	ID          string
	ChannelID   string
	Content     string
	AuthorID    string
	AuthorName  string
	AuthorBot   bool
	Attachments []Attachment
}
