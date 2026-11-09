package contracts

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

// Backend is the model edge: it turns one inbound prompt into a reply, optionally
// emitting intermediate progress events. The host neither knows nor cares which
// model answers (Claude, Codex, …). Every backend is one implementation of this
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
// background (empty when no Memory plugin is wired). Data the backend fences
// into the turn for continuity, never the user's instructions. Language is the
// language of the session the turn belongs to, and the zero value when the host
// does not know it: the backend then keeps its own default.
type Prompt struct {
	Content     string        `json:"Content"`
	Context     string        `json:"Context"`
	Author      string        `json:"Author"`
	MessageID   string        `json:"MessageID"`
	ChannelID   string        `json:"ChannelID"`
	Attachments []string      `json:"Attachments"`
	Language    i18n.Language `json:"Language"`
}

// BackendEvent is one intermediate occurrence a backend surfaces mid-turn for a
// progress consumer: a tool invocation, streamed assistant text, or the terminal
// result carrying cost. Models splits the usage of the terminal result by model,
// empty when the backend gives no breakdown.
type BackendEvent struct {
	Kind        string
	Tool        string
	Detail      string
	Cost        float64
	IsError     bool
	InTokens    int
	OutTokens   int
	CacheRead   int
	CacheCreate int
	Models      []ModelUsage
	Todos       []TodoItem
	Subagent    *Subagent
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

// OutboundFile is a file a backend rendered to disk while it answered. Path is its
// absolute local path, already written, Name is the name a gateway shows the human,
// and ContentType is its MIME type.
type OutboundFile struct {
	Path        string `json:"Path"`
	Name        string `json:"Name"`
	ContentType string `json:"ContentType"`
}

// ErrInvalidOutboundFile is returned by [ValidateOutboundFile] for a file whose path
// is not absolute, or whose name is empty or holds a path separator. Match it with
// errors.Is.
var ErrInvalidOutboundFile = errors.New("invalid outbound file")

// ValidateOutboundFile checks a file before the host hands it to a gateway: Path is
// absolute, and Name is not empty and names no directory. Every failure matches
// [ErrInvalidOutboundFile].
func ValidateOutboundFile(file OutboundFile) error {
	switch {
	case !filepath.IsAbs(file.Path):
		return fmt.Errorf("%w: path %q is not absolute", ErrInvalidOutboundFile, file.Path)
	case file.Name == "":
		return fmt.Errorf("%w: file %q has no name", ErrInvalidOutboundFile, file.Path)
	case strings.ContainsAny(file.Name, `/\`):
		return fmt.Errorf("%w: name %q holds a path separator", ErrInvalidOutboundFile, file.Name)
	}
	return nil
}

// FileProducer is an OPTIONAL capability of a backend: it hands over the files it
// rendered during the last Respond. The host calls TakeFiles once after each Respond,
// and a file taken is not returned by a later call.
type FileProducer interface {
	TakeFiles() []OutboundFile
}
