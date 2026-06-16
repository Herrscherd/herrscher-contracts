package contracts

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

// CommandKind classifies an inbound command delivered by a CommandSource.
type CommandKind int

const (
	KindSlash CommandKind = iota
	KindComponent
	KindAutocomplete
)

// ResponseToken is an opaque, gateway-private handle correlating an inbound
// command to its eventual response. The host shuttles it back to the responder
// without inspecting it (the Discord adapter packs the interaction id+token).
type ResponseToken any

// InboundCommand is one command a CommandSource delivers to the host dispatch
// loop. For KindComponent the Command's CustomID carries the resolved target
// (e.g. a session name) and Values the picked value(s).
type InboundCommand struct {
	Kind    CommandKind
	Command Command
	Token   ResponseToken
}

// AutocompleteChoice is one suggestion for an autocomplete request.
type AutocompleteChoice struct {
	Label string
	Value string
}
