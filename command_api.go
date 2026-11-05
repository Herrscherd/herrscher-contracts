package contracts

import "context"

// Cmd is the one neutral command concept the platform exposes. A command is
// declared once. A namespaced Path, its Params, and the Run handler. And a
// format (the CLI today, a gateway binding later) resolves an invocation to it.
// The handler is opaque: whatever Run closes over (a Discord client, a backend),
// the registry that holds the Cmd stays agnostic of it. Run returns a Result, and
// Help is the catalog key of the command's help text, not the text itself.
type Cmd struct {
	Path   []string
	Params []Param
	Help   string
	Run    func(ctx context.Context, in Input) (Result, error)
}

// Param is one declared input. Help is the catalog key of its help text. Required
// params missing at dispatch are an error. ValueRequired distinguishes an optional
// valued parameter from a valueless boolean flag: when true, supplying the parameter
// without a value is an error.
type Param struct {
	Name          string
	Help          string
	Required      bool
	ValueRequired bool
}

// Input is the parsed, format-agnostic invocation handed to a handler. A CLI
// format fills it from argv; a future gateway fills it from an interaction.
type Input struct {
	Args map[string]string
	Rest []string
	// JSON is the reserved global --json flag: the dispatcher strips it from
	// argv before per-command parsing and sets this. A verb's Run consults it
	// to emit machine JSON instead of human text. It is never a declared Param.
	JSON bool
}

// Lookup returns a param value and whether it was supplied.
func (in Input) Lookup(name string) (string, bool) {
	v, ok := in.Args[name]
	return v, ok
}

// Get returns a param value, empty if absent.
func (in Input) Get(name string) string { return in.Args[name] }

// Bool reports whether a param was supplied as the literal "true".
func (in Input) Bool(name string) bool { return in.Args[name] == "true" }

// Builder fluently declares a Cmd.
type Builder struct{ c Cmd }

// New starts a command declaration under the given namespace path.
func New(path ...string) *Builder { return &Builder{c: Cmd{Path: path}} }

// Help sets the catalog key of the command's help text and returns the builder
// for chaining. The module that declares the command renders the key.
func (b *Builder) Help(key string) *Builder { b.c.Help = key; return b }

// Param declares a parameter the command accepts, with the catalog key of its help
// text and whether the caller must supply it. It returns the builder for chaining.
func (b *Builder) Param(name, helpKey string, required bool) *Builder {
	b.c.Params = append(b.c.Params, Param{
		Name: name, Help: helpKey, Required: required, ValueRequired: required,
	})
	return b
}

// ValueParam declares a parameter that requires a value whenever it is
// supplied. required=false makes the parameter optional without turning a
// valueless occurrence into the boolean string "true".
func (b *Builder) ValueParam(name, helpKey string, required bool) *Builder {
	b.c.Params = append(b.c.Params, Param{
		Name: name, Help: helpKey, Required: required, ValueRequired: true,
	})
	return b
}

// Do sets the handler and returns the finished Cmd.
func (b *Builder) Do(fn func(ctx context.Context, in Input) (Result, error)) Cmd {
	b.c.Run = fn
	return b.c
}
