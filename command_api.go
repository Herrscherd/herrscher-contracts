package contracts

import "context"

// Cmd is the one neutral command concept the platform exposes. A command is
// declared once — a namespaced Path, its Params, and the Run handler — and a
// format (the CLI today, a gateway binding later) resolves an invocation to it.
// The handler is opaque: whatever Run closes over (a Discord client, a backend),
// the registry that holds the Cmd stays agnostic of it.
type Cmd struct {
	Path   []string
	Params []Param
	Help   string
	Run    func(ctx context.Context, in Input) (string, error)
}

// Param is one declared input. Required params missing at dispatch are an error.
type Param struct {
	Name     string
	Help     string
	Required bool
}

// Input is the parsed, format-agnostic invocation handed to a handler. A CLI
// format fills it from argv; a future gateway fills it from an interaction.
type Input struct {
	Args map[string]string
	Rest []string
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

func (b *Builder) Help(text string) *Builder { b.c.Help = text; return b }

func (b *Builder) Param(name, help string, required bool) *Builder {
	b.c.Params = append(b.c.Params, Param{Name: name, Help: help, Required: required})
	return b
}

// Do sets the handler and returns the finished Cmd.
func (b *Builder) Do(fn func(ctx context.Context, in Input) (string, error)) Cmd {
	b.c.Run = fn
	return b.c
}
