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
	// CodeSettingInvalid reports that the setting Setting holds the value Value,
	// which it does not accept.
	CodeSettingInvalid
	// CodeSettingMissing reports that the settings named in Settings must be set.
	// Settings is a text the caller composes, for instance the names joined with
	// ", ", and it is shown as it is.
	CodeSettingMissing
	// CodeNodeMerged reports that a node is folded into an umbrella and must be
	// detached before it can be restored.
	CodeNodeMerged
	// CodeCompactUnavailable reports that a context compaction cannot run. Reason
	// holds the technical reason, given as a text argument.
	CodeCompactUnavailable
	// CodeCompacted reports that the context of the session was compacted.
	CodeCompacted
	// CodeAddressPlaintextRefused reports that a plaintext connection to Address
	// was refused, because Address is not a loopback address and TLS is not set.
	CodeAddressPlaintextRefused
	// CodeCompactTimeoutInvalid reports that the compaction timeout is not a positive duration.
	CodeCompactTimeoutInvalid
	// CodeCompactInstructionRefused reports that the backend cannot take an instruction for a compaction.
	CodeCompactInstructionRefused
	// CodeCompactNoThread reports that no turn has run yet, so there is no context to compact.
	CodeCompactNoThread
	// CodeSessionLimit reports that no session can be created because Count sessions already run.
	CodeSessionLimit
	// CodeNoHome reports that a session cannot be created before a home channel is set.
	CodeNoHome
	// CodeNoHostRegistered reports that a remote host is requested but none is registered.
	CodeNoHostRegistered
	// CodeServiceControlUnavailable reports that the host cannot control its own service.
	CodeServiceControlUnavailable
	// CodeModelRequired reports that a session under the gateway-only route policy names no model.
	CodeModelRequired
	// CodeExplicitCmdRefused reports that a session names a cmd under the gateway-only route policy.
	CodeExplicitCmdRefused
	// CodeInvalidValue reports that the value Value is malformed or out of range for the field Field.
	CodeInvalidValue
	// CodeSharedAgent reports that the agent Agent cannot run in a session that shares the main checkout.
	CodeSharedAgent
	// CodeUnknownApprovalMode reports that the approval mode Mode is not one the host knows.
	CodeUnknownApprovalMode
	// CodeAgentExists reports that an agent named Name already exists.
	CodeAgentExists
	// CodeAgentInvalidName reports that Name is not a usable agent name.
	CodeAgentInvalidName
	// CodeAgentNotFound reports that no agent is named Name.
	CodeAgentNotFound
	// CodeUnsupportedHomeType reports that the home channel kind Type cannot host sessions.
	CodeUnsupportedHomeType
	// CodeEntityNotFound reports that no Kind named Name exists. Kind is shown as it is.
	CodeEntityNotFound
	// CodeEntityUnknown reports that Name is not a known Kind. Kind is shown as it is.
	CodeEntityUnknown
	// CodeScheduleNotFound reports that no schedule is named Name.
	CodeScheduleNotFound
	// CodeChannelKindUnder reports that the channel Channel has the kind Kind, which cannot hold an under placement.
	CodeChannelKindUnder
	// CodeChannelKindHome reports that the channel Channel has the kind Kind, which cannot be the home.
	CodeChannelKindHome
	// CodeInvalidValueHint reports that the value Value is malformed or out of range for the field Field,
	// and that Hint says why. Hint is a text the caller composes, and it is shown as it is.
	CodeInvalidValueHint
	// CodeSessionCloseDirty reports that the session Name was not closed, because its worktree holds uncommitted changes.
	CodeSessionCloseDirty
	// CodeHealthConnected reports that the account is logged in and answered the check.
	CodeHealthConnected
	// CodeHealthSignedInAs reports that the service answered as the identity Name, such as a bot name.
	CodeHealthSignedInAs
	// CodeHealthNotLoggedIn reports that the account is not logged in.
	CodeHealthNotLoggedIn
	// CodeHealthRateLimited reports that the vendor answered with a rate limit.
	CodeHealthRateLimited
	// CodeHealthCredentialsRefused reports that the service refused the credentials it was given.
	CodeHealthCredentialsRefused
	// CodeHealthUnreachable reports that the service could not be reached. Reason holds the technical reason, shown as it is.
	CodeHealthUnreachable
	// CodeHealthEnvMissing reports that the environment variable Name is not set.
	CodeHealthEnvMissing
	// CodeHealthRemedyRun tells the operator to run the command Command, shown as it is.
	CodeHealthRemedyRun
	// CodeHealthRemedyEnvRestart tells the operator to add the variable Name to the service environment and restart the service.
	CodeHealthRemedyEnvRestart
)

// Codes returns every code, in declaration order. Key and params switch over the
// same constants: a constant added to one of them must be added here too, and
// the test suite checks that no named constant is missing from this list.
func Codes() []Code {
	return []Code{
		CodeSessionNotFound, CodeSessionExists, CodeUnknownAgent, CodeSessionCreated, CodeSessionClosed, CodeUnexpected,
		CodeSettingInvalid, CodeSettingMissing, CodeNodeMerged, CodeCompactUnavailable, CodeCompacted,
		CodeAddressPlaintextRefused,
		CodeCompactTimeoutInvalid, CodeCompactInstructionRefused, CodeCompactNoThread,
		CodeSessionLimit, CodeNoHome, CodeNoHostRegistered, CodeServiceControlUnavailable,
		CodeModelRequired, CodeExplicitCmdRefused, CodeInvalidValue, CodeSharedAgent,
		CodeUnknownApprovalMode, CodeAgentExists, CodeAgentInvalidName, CodeAgentNotFound,
		CodeUnsupportedHomeType,
		CodeEntityNotFound, CodeEntityUnknown, CodeScheduleNotFound,
		CodeChannelKindUnder, CodeChannelKindHome, CodeInvalidValueHint, CodeSessionCloseDirty,
		CodeHealthConnected, CodeHealthSignedInAs, CodeHealthNotLoggedIn, CodeHealthRateLimited, CodeHealthCredentialsRefused,
		CodeHealthUnreachable, CodeHealthEnvMissing, CodeHealthRemedyRun, CodeHealthRemedyEnvRestart,
	}
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
	case CodeSettingInvalid:
		return "setting.invalid"
	case CodeSettingMissing:
		return "setting.missing"
	case CodeNodeMerged:
		return "node.merged"
	case CodeCompactUnavailable:
		return "compact.unavailable"
	case CodeCompacted:
		return "compaction.done"
	case CodeAddressPlaintextRefused:
		return "address.plaintext_refused"
	case CodeCompactTimeoutInvalid:
		return "compact.timeout_invalid"
	case CodeCompactInstructionRefused:
		return "compact.instruction_refused"
	case CodeCompactNoThread:
		return "compact.no_thread"
	case CodeSessionLimit:
		return "session.limit"
	case CodeNoHome:
		return "home.missing"
	case CodeNoHostRegistered:
		return "host.none_registered"
	case CodeServiceControlUnavailable:
		return "service.control_unavailable"
	case CodeModelRequired:
		return "model.required"
	case CodeExplicitCmdRefused:
		return "cmd.explicit_refused"
	case CodeInvalidValue:
		return "value.invalid"
	case CodeSharedAgent:
		return "agent.shared_refused"
	case CodeUnknownApprovalMode:
		return "approval.mode_unknown"
	case CodeAgentExists:
		return "agent.exists"
	case CodeAgentInvalidName:
		return "agent.invalid_name"
	case CodeAgentNotFound:
		return "agent.not_found"
	case CodeUnsupportedHomeType:
		return "home.type_unsupported"
	case CodeEntityNotFound:
		return "entity.not_found"
	case CodeEntityUnknown:
		return "entity.unknown"
	case CodeScheduleNotFound:
		return "schedule.not_found"
	case CodeChannelKindUnder:
		return "channel.kind_under"
	case CodeChannelKindHome:
		return "channel.kind_home"
	case CodeInvalidValueHint:
		return "value.invalid_hint"
	case CodeSessionCloseDirty:
		return "session.close_dirty"
	case CodeHealthConnected:
		return "health.connected"
	case CodeHealthSignedInAs:
		return "health.signed_in_as"
	case CodeHealthNotLoggedIn:
		return "health.not_logged_in"
	case CodeHealthRateLimited:
		return "health.rate_limited"
	case CodeHealthCredentialsRefused:
		return "health.credentials_refused"
	case CodeHealthUnreachable:
		return "health.unreachable"
	case CodeHealthEnvMissing:
		return "health.env_missing"
	case CodeHealthRemedyRun:
		return "health.remedy_run"
	case CodeHealthRemedyEnvRestart:
		return "health.remedy_env_restart"
	}
	return "code." + strconv.FormatUint(uint64(c), 10)
}

func (c Code) params() (map[string]ArgKind, bool) {
	switch c {
	case CodeSessionNotFound, CodeSessionExists, CodeUnknownAgent, CodeSessionCreated, CodeSessionClosed:
		return map[string]ArgKind{nameParam: ArgText}, true
	case CodeUnexpected:
		return map[string]ArgKind{"Detail": ArgText}, true
	case CodeSettingInvalid:
		return map[string]ArgKind{"Setting": ArgText, valueParam: ArgText}, true
	case CodeSettingMissing:
		return map[string]ArgKind{"Settings": ArgText}, true
	case CodeCompactUnavailable:
		return map[string]ArgKind{"Reason": ArgText}, true
	case CodeAddressPlaintextRefused:
		return map[string]ArgKind{"Address": ArgText}, true
	case CodeNodeMerged, CodeCompacted,
		CodeCompactTimeoutInvalid, CodeCompactInstructionRefused, CodeCompactNoThread:
		return nil, true
	case CodeSessionLimit:
		return map[string]ArgKind{countName: ArgCount}, true
	case CodeNoHome, CodeNoHostRegistered, CodeServiceControlUnavailable,
		CodeModelRequired, CodeExplicitCmdRefused:
		return nil, true
	case CodeInvalidValue:
		return map[string]ArgKind{"Field": ArgText, valueParam: ArgText}, true
	case CodeSharedAgent:
		return map[string]ArgKind{"Agent": ArgText}, true
	case CodeUnknownApprovalMode:
		return map[string]ArgKind{"Mode": ArgText}, true
	case CodeAgentExists, CodeAgentInvalidName, CodeAgentNotFound:
		return map[string]ArgKind{nameParam: ArgText}, true
	case CodeUnsupportedHomeType:
		return map[string]ArgKind{"Type": ArgText}, true
	case CodeEntityNotFound, CodeEntityUnknown:
		return map[string]ArgKind{"Kind": ArgText, nameParam: ArgText}, true
	case CodeScheduleNotFound:
		return map[string]ArgKind{nameParam: ArgText}, true
	case CodeChannelKindUnder, CodeChannelKindHome:
		return map[string]ArgKind{"Channel": ArgRef, "Kind": ArgText}, true
	case CodeInvalidValueHint:
		return map[string]ArgKind{"Field": ArgText, valueParam: ArgText, "Hint": ArgText}, true
	case CodeSessionCloseDirty:
		return map[string]ArgKind{nameParam: ArgText}, true
	case CodeHealthConnected, CodeHealthNotLoggedIn, CodeHealthRateLimited, CodeHealthCredentialsRefused:
		return nil, true
	case CodeHealthSignedInAs, CodeHealthEnvMissing, CodeHealthRemedyEnvRestart:
		return map[string]ArgKind{nameParam: ArgText}, true
	case CodeHealthUnreachable:
		return map[string]ArgKind{"Reason": ArgText}, true
	case CodeHealthRemedyRun:
		return map[string]ArgKind{"Command": ArgText}, true
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

const (
	nameParam  = "Name"
	valueParam = "Value"
)

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
	// its code declares: a malformed Arg, a name given twice, a missing or
	// unexpected name, or a kind other than the one the parameter declares.
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
		if want, expected := declared[arg.name]; !expected || want != arg.kind {
			return nil, fmt.Errorf("argument %q of kind %d does not match code %q: %w", arg.name, uint8(arg.kind), n.Code.Key(), ErrNoticeArgs)
		}
		if _, given := params[arg.name]; given {
			return nil, fmt.Errorf("argument %q given twice: %w", arg.name, ErrNoticeArgs)
		}
		params[arg.name] = value
	}
	if len(params) != len(declared) {
		return nil, fmt.Errorf("code %q takes %v, given %v: %w", n.Code.Key(), slices.Sorted(maps.Keys(declared)), slices.Sorted(maps.Keys(params)), ErrNoticeArgs)
	}
	return params, nil
}
