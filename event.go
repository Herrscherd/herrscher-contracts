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
//	{"t":"reply","text":"done","done":true}
//	{"t":"input","who":"terminal","text":"apply them"}
//	{"t":"pick","value":"2"}
//	{"t":"reset"}  // discard the in-progress turn (backend was reset mid-turn)
type Event struct {
	T     string `json:"t"`
	Who   string `json:"who,omitempty"`
	Text  string `json:"text,omitempty"`
	Value string `json:"value,omitempty"`
	Done  bool   `json:"done,omitempty"`
}

// EventSink is an optional gateway capability: a gateway that renders the live
// turn event stream itself (the terminal TUI does) implements it. The hub fans
// every turn event to each bound gateway implementing EventSink; a gateway
// without it is driven by the host's default renderer, which posts the final
// reply (and a progress view) through the Gateway/ChannelReader ports.
type EventSink interface {
	Emit(Event)
}
