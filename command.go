package contracts

import "context"

// Command is a platform-agnostic slash-command invocation routed to the manager.
type Command struct {
	Invoker string
	Data    CommandData
}

// CommandData carries the invoked command name and its option tree. For a
// component interaction the command fields are empty and CustomID/Values carry
// the clicked component's id and selected value(s).
type CommandData struct {
	Name     string
	Options  []Option
	CustomID string
	Values   []string
}

// Option is one command option; subcommands and groups nest via Options.
type Option struct {
	Name    string
	Type    OptionType
	Value   any
	Focused bool
	Options []Option
}

type OptionType int

const (
	OptSubcommand      OptionType = 1
	OptSubcommandGroup OptionType = 2
)

func (d CommandData) Opt(name string) (string, bool) { return findOpt(d.Options, name) }

func findOpt(opts []Option, name string) (string, bool) {
	for _, o := range opts {
		if o.Name == name {
			if s, ok := o.Value.(string); ok {
				return s, true
			}
		}
		if v, ok := findOpt(o.Options, name); ok {
			return v, true
		}
	}
	return "", false
}

func (d CommandData) OptBool(name string) bool {
	b, _ := findBool(d.Options, name)
	return b
}

func findBool(opts []Option, name string) (bool, bool) {
	for _, o := range opts {
		if o.Name == name {
			if b, ok := o.Value.(bool); ok {
				return b, true
			}
		}
		if b, ok := findBool(o.Options, name); ok {
			return b, true
		}
	}
	return false, false
}

func (d CommandData) Focused() (name, value string, ok bool) { return findFocused(d.Options) }

func findFocused(opts []Option) (string, string, bool) {
	for _, o := range opts {
		if o.Focused {
			s, _ := o.Value.(string)
			return o.Name, s, true
		}
		if n, v, ok := findFocused(o.Options); ok {
			return n, v, ok
		}
	}
	return "", "", false
}

func (d CommandData) Subcommand() (string, []Option) {
	for _, o := range d.Options {
		if o.Type == OptSubcommand {
			return o.Name, o.Options
		}
	}
	return "", nil
}

// CommandResponse is the reply a handler produces. Private restricts visibility
// to the invoker when the gateway supports it.
type CommandResponse struct {
	Content string
	Private bool
}

// CommandKind classifies an inbound command a ChannelSource delivers.
//   - KindCommand: a regular command invocation the host handles and answers.
//   - KindChoicePick: the user picked an option from a routed menu; CustomID
//     carries the resolved route (e.g. a session name), Values the pick(s).
//   - KindSuggest: a partial-input suggestion request (autocomplete); the host
//     answers with Responder.Suggest.
type CommandKind int

const (
	KindCommand CommandKind = iota
	KindChoicePick
	KindSuggest
)

// Responder answers one inbound command. It is the neutral reply intent: the
// host declares WHAT to say and whether the work is slow; the plugin decides HOW
// (e.g. Discord's ack-then-edit defer dance, ephemeral flags, menu rendering)
// entirely inside its own implementation. The host never sees a response token
// or any platform mechanic.
type Responder interface {
	// Respond answers the command. When slow is true, produce runs after the
	// plugin has acknowledged within the platform's callback deadline (the plugin
	// owns that ack-then-edit detail); otherwise produce runs inline.
	Respond(ctx context.Context, slow bool, produce func(context.Context) CommandResponse) error
	// Suggest answers a KindSuggest request with completion choices.
	Suggest(ctx context.Context, choices []Choice) error
	// AckPick acknowledges a KindChoicePick with a short confirmation line.
	AckPick(ctx context.Context, content string) error
}

// InboundCommand is one command a ChannelSource delivers to the host dispatch
// loop, carrying the per-command Responder used to answer it.
type InboundCommand struct {
	Kind      CommandKind
	Command   Command
	Responder Responder
}
