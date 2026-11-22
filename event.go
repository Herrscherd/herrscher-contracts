package contracts

import (
	"encoding/json"
	"errors"
)

// CoordinationEvent reports a handoff between sessions: one session delegates work to
// another, or reports its result back to its parent. The session fields name the
// sessions involved, Agent names the agent that acted, and Summary is the outcome in
// human words.
type CoordinationEvent struct {
	Kind          string `json:"kind"`
	SourceSession string `json:"source_session"`
	ParentSession string `json:"parent_session,omitempty"`
	TargetSession string `json:"target_session,omitempty"`
	Agent         string `json:"agent,omitempty"`
	Summary       string `json:"summary,omitempty"`
}

// Event is one message on the session bus. The bridge (a pure backend runner)
// emits turn events for the hub to fan out; the hub injects input/pick down to
// the bridge. One Event encodes to exactly one JSON line on the wire.
//
// chunk carries assistant prose; status carries a tool/progress line.
//
// Decoding an Event refuses a field of the wrong JSON type with
// [ErrInvalidEvent], whose message names the wire field and never the Go type
// or the value.
//
//	{"t":"human","who":"alice","text":"refactor the env loader"}
//	{"t":"status","text":"reading envfile.go"}
//	{"t":"chunk","text":"proposing 3 changes"}
//	{"t":"reply","text":"done","done":true,"cost":0.0042}
//	{"t":"input","who":"terminal","text":"apply them"}
//	{"t":"pick","value":"2"}
//	{"t":"interrupt"}  // cancel the in-flight turn (esc-to-interrupt from a gateway)
//	{"t":"reset"}  // the backend started a new process mid-turn and the same prompt runs again
type Event struct {
	T     string `json:"t"`
	Who   string `json:"who,omitempty"`
	Text  string `json:"text,omitempty"`
	Value string `json:"value,omitempty"`
	Done  bool   `json:"done,omitempty"`
	// Cost is the turn's total cost in USD, carried on the terminal reply
	// (Done) so the hub can render it in the progress summary. Zero when the
	// backend reports no cost.
	Cost float64 `json:"cost,omitempty"`
	// Tokens is the turn's cumulative output-token count, carried live on
	// status/chunk events so a gateway can render a growing counter, and finally
	// on the terminal reply (Done). The same piggyback path as Cost. Zero when
	// the backend reports no usage.
	Tokens int `json:"tokens,omitempty"`
	// TokensIn is the turn's input-token count; CacheRead/CacheCreate the
	// prompt-cache read/creation tokens. Carried on the terminal reply next to
	// Tokens (output) so a consumer can render the full usage breakdown. Zero
	// when the backend reports no usage.
	TokensIn    int `json:"tokens_in,omitempty"`
	CacheRead   int `json:"cache_read,omitempty"`
	CacheCreate int `json:"cache_create,omitempty"`
	// Models splits the usage of the turn by model, carried on the terminal reply
	// (Done) like Cost. Empty when the backend gives no breakdown: the totals
	// above then stand alone.
	Models []ModelUsage `json:"models,omitempty"`
	// Resume carries the backend's opaque resume token, piggybacked on the
	// terminal reply{done} so the daemon can persist it for cross-restart
	// --resume. Empty when the backend is not ResumeAware or has no id yet.
	Resume string `json:"resume,omitempty"`
	// Project carries the memory project this session settled on, piggybacked on
	// the terminal reply{done} so the daemon can persist it. The same path Resume
	// takes. Empty when the turn settled nothing, which is every turn of every
	// session whose project a human already chose.
	Project string `json:"project,omitempty"`
	// Attachments carries local filesystem paths to any images resolved for an
	// input frame. The host downloads/passes-through a message's attachments
	// before enqueueing the turn, and the bridge folds these into Prompt so the
	// backend can reference them as image blocks. Only set on "input" frames.
	Attachments []string `json:"attachments,omitempty"`
	// SessionIncarnation is the stable identity of the session object that
	// emitted this event.
	SessionIncarnation string `json:"session_incarnation,omitempty"`
	// TurnID uniquely identifies the turn this event belongs to.
	TurnID string `json:"turn_id,omitempty"`
	// Agent is the authoritative agent or role at the time of this turn.
	Agent        string             `json:"agent,omitempty"`
	Coordination *CoordinationEvent `json:"coordination,omitempty"`
	Todos        []TodoItem         `json:"todos,omitempty"`
	Subagent     *Subagent          `json:"subagent,omitempty"`
	ToolCall     *ToolCall          `json:"tool_call,omitempty"`
}

// ErrInvalidEvent is returned when a field of an event's JSON has the wrong
// JSON type. The message names the wire field, such as t or coordination, and
// never the Go type or the value. A field nested in coordination, todos, models
// or subagent is named by its top-level field. A tool_call refused inside the
// event keeps [ErrInvalidToolCall], and a document that is not valid JSON is
// returned as its *json.SyntaxError, which does not match this error. Match it
// with errors.Is.
var ErrInvalidEvent = errors.New("invalid event")

type eventWire Event

// UnmarshalJSON decodes an event. A null leaves it unchanged, as encoding/json
// does for a struct. A field of the wrong JSON type is refused with
// [ErrInvalidEvent], and a tool_call refused inside it with
// [ErrInvalidToolCall]. A document that is not valid JSON returns its
// *json.SyntaxError unchanged.
func (e *Event) UnmarshalJSON(data []byte) error {
	if string(data) == jsonNull {
		return nil
	}
	var decoded eventWire
	if err := json.Unmarshal(data, &decoded); err != nil {
		if typeErr, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
			return wrongJSONTypeError(ErrInvalidEvent, "event", eventWireFields(), typeErr)
		}
		return err
	}
	*e = Event(decoded)
	return nil
}

func eventWireFields() []string {
	return []string{
		"t", "who", "text", "value", "done", "cost", "tokens", "tokens_in",
		"cache_read", "cache_create", "models", "resume", "project",
		"attachments", "session_incarnation", "turn_id", "agent",
		"coordination", "todos", "subagent", "tool_call",
	}
}

// EventSink is an optional gateway capability: a gateway that renders the live
// turn event stream itself (the terminal TUI does) implements it. The hub fans
// every turn event to each bound gateway implementing EventSink; a gateway
// without it is driven by the host's default path, which posts only the final
// reply (chunked) through the Gateway port. No platform-specific rendering.
type EventSink interface {
	Emit(Event)
}

// RoutedEventSink is an optional gateway capability for a gateway that renders
// more than one conversation's live stream itself (the multi-session terminal
// TUI). When a gateway implements it the hub prefers it over EventSink and tags
// each event with the destination Conversation, so the gateway can demultiplex
// the streams of every session bound to it. A gateway that implements only
// EventSink (or neither) is unaffected.
type RoutedEventSink interface {
	EmitTo(conv Conversation, e Event)
}
