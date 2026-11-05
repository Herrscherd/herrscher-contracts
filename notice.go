package contracts

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
)

// Code names one message a human reads: the host emits the code and its
// arguments, and each gateway or front end renders the text in the language it
// serves. The zero value is not a code, and a value outside the list is refused
// by Renderer.Render.
type Code uint16

const (
	// CodeSessionNotFound reports that no session carries the name Name.
	CodeSessionNotFound Code = iota + 1
	// CodeSessionExists reports that a session named Name already exists.
	CodeSessionExists
	// CodeUnknownAgent reports that no agent is named Name.
	CodeUnknownAgent
	// CodeSessionCreated reports that the session Name was created and is live.
	CodeSessionCreated
	// CodeSessionClosed reports that the session Name was closed.
	CodeSessionClosed
	// CodeUnexpected reports a failure that has no code of its own. Detail holds
	// its technical text, which is shown as is and never translated.
	CodeUnexpected
)

// Codes returns every code, in declaration order. Key and params switch over the
// same constants: a constant added to one of them must be added here too, and
// the test suite checks that no named constant is missing from this list.
func Codes() []Code {
	return []Code{CodeSessionNotFound, CodeSessionExists, CodeUnknownAgent, CodeSessionCreated, CodeSessionClosed, CodeUnexpected}
}

// Key returns the catalog key of the code. An out-of-set value returns its raw
// number, which no catalog defines, so the mistake is visible.
func (c Code) Key() string {
	switch c {
	case CodeSessionNotFound:
		return "session.not_found"
	case CodeSessionExists:
		return "session.exists"
	case CodeUnknownAgent:
		return "agent.unknown"
	case CodeSessionCreated:
		return "session.created"
	case CodeSessionClosed:
		return "session.closed"
	case CodeUnexpected:
		return "internal.unexpected"
	}
	return "code." + strconv.FormatUint(uint64(c), 10)
}

func (c Code) params() ([]string, bool) {
	switch c {
	case CodeSessionNotFound, CodeSessionExists, CodeUnknownAgent, CodeSessionCreated, CodeSessionClosed:
		return []string{"Name"}, true
	case CodeUnexpected:
		return []string{"Detail"}, true
	}
	return nil, false
}

// Notice is a message for a human: a code and the arguments its text needs. An
// Arg is built only by Text, Count or Ref, so a notice cannot carry a value its
// text cannot show.
type Notice struct {
	Code Code
	Args []Arg
}

// ArgKind is the closed set of argument kinds. The zero value is not a kind.
type ArgKind uint8

const (
	// ArgText is a string shown as it is.
	ArgText ArgKind = iota + 1
	// ArgCount is an integer that also selects the plural form of the text.
	ArgCount
	// ArgRef is the identifier of a platform object, such as a channel. A gateway
	// renders it as a mention; the plain text form shows the identifier.
	ArgRef
)

const countName = "Count"

// Arg is one named argument of a Notice. Its zero value is malformed, and a
// Notice that carries it is refused by Renderer.Render.
type Arg struct {
	name  string
	kind  ArgKind
	value any
}

// Text returns a string argument named name.
func Text(name, value string) Arg {
	return Arg{name: name, kind: ArgText, value: value}
}

// Count returns the count argument, named Count, that selects the plural form.
func Count(n int) Arg {
	return Arg{name: countName, kind: ArgCount, value: n}
}

// Ref returns the identifier argument named name of a platform object.
func Ref(name, id string) Arg {
	return Arg{name: name, kind: ArgRef, value: id}
}

// Name returns the name the text refers to the argument by.
func (a Arg) Name() string { return a.name }

// Kind returns the kind the argument was built with.
func (a Arg) Kind() ArgKind { return a.kind }

// Value returns the string of a text or ref argument, or the int of a count.
func (a Arg) Value() any { return a.value }

func (a Arg) checkKind() error {
	if (a.name == countName) != (a.kind == ArgCount) {
		return fmt.Errorf("argument %q of kind %d: %w", a.name, uint8(a.kind), ErrNoticeArgs)
	}
	return nil
}

func (a Arg) param() (any, bool) {
	switch a.kind {
	case ArgText, ArgRef, ArgCount:
		return a.value, true
	}
	return nil, false
}

// NoticeError is an error that reaches a human. It gives the notice that stands
// for it, and Error keeps its English text for logs and debugging.
type NoticeError interface {
	error
	Notice() Notice
}

// NoticeOf returns the notice an error stands for: the notice of the first
// NoticeError in its chain, or CodeUnexpected carrying the error text when there
// is none. The error must not be nil.
func NoticeOf(err error) Notice {
	if noticed, ok := errors.AsType[NoticeError](err); ok {
		return noticed.Notice()
	}
	return Notice{Code: CodeUnexpected, Args: []Arg{Text("Detail", err.Error())}}
}

var (
	// ErrUnknownCode reports a notice whose code is outside the list of Codes.
	ErrUnknownCode = errors.New("unknown notice code")
	// ErrNoticeArgs reports a notice whose arguments do not match the parameters
	// its code declares: a malformed Arg, a name given twice, or a missing or
	// unexpected name.
	ErrNoticeArgs = errors.New("notice arguments do not match the code")
	// ErrMissingMessage reports a code whose key the shared catalog does not define
	// in English, the language every message falls back to, so no text could be
	// shown to a human in any language.
	ErrMissingMessage = errors.New("no English message for the code")
)

func (n Notice) params() (map[string]any, error) {
	declared, known := n.Code.params()
	if !known {
		return nil, fmt.Errorf("code %d: %w", uint16(n.Code), ErrUnknownCode)
	}
	params := make(map[string]any, len(n.Args))
	for _, arg := range n.Args {
		value, valid := arg.param()
		if !valid {
			return nil, fmt.Errorf("argument %q of kind %d: %w", arg.name, uint8(arg.kind), ErrNoticeArgs)
		}
		if err := arg.checkKind(); err != nil {
			return nil, err
		}
		if _, given := params[arg.name]; given {
			return nil, fmt.Errorf("argument %q given twice: %w", arg.name, ErrNoticeArgs)
		}
		params[arg.name] = value
	}
	if !slices.Equal(slices.Sorted(maps.Keys(params)), slices.Sorted(slices.Values(declared))) {
		return nil, fmt.Errorf("code %q takes %v, given %v: %w", n.Code.Key(), declared, slices.Sorted(maps.Keys(params)), ErrNoticeArgs)
	}
	return params, nil
}
