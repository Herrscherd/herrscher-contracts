package contracts

// Event is one message on the session bus. The bridge (a pure backend runner)
// emits turn events for the hub to fan out; the hub injects input/pick down to
// the bridge. One Event encodes to exactly one JSON line on the wire.
//
// chunk carries assistant prose; status carries a tool/progress line.
//
//	{"t":"human","who":"alice","text":"refactor the env loader"}
//	{"t":"status","text":"reading envfile.go"}
//	{"t":"chunk","text":"proposing 3 changes"}
//	{"t":"reply","text":"done","done":true,"cost":0.0042}
//	{"t":"input","who":"terminal","text":"apply them"}
//	{"t":"pick","value":"2"}
//	{"t":"reset"}  // discard the in-progress turn (backend was reset mid-turn)
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
}

// EventSink is an optional gateway capability: a gateway that renders the live
// turn event stream itself (the terminal TUI does) implements it. The hub fans
// every turn event to each bound gateway implementing EventSink; a gateway
// without it is driven by the host's default path, which posts only the final
// reply (chunked) through the Gateway port — no platform-specific rendering.
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
