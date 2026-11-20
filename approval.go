package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ApprovalDecision is the verdict on one tool approval request. The closed set is
// DecisionAllow and DecisionDeny. [ParseApprovalDecision] refuses any other text,
// so an unknown verdict never crosses the boundary, and [ApprovalAnswer.Allowed]
// still denies it if one is built in memory.
type ApprovalDecision string

const (
	// DecisionAllow lets the tool run.
	DecisionAllow ApprovalDecision = "allow"
	// DecisionDeny refuses the tool.
	DecisionDeny ApprovalDecision = "deny"
)

// ErrInvalidDecision is returned for text, or a value, that names no approval
// decision. Match it with errors.Is.
var ErrInvalidDecision = errors.New("unknown approval decision")

// ParseApprovalDecision reads a decision from its text form. Any other text, the
// empty string included, returns an error matching [ErrInvalidDecision].
func ParseApprovalDecision(text string) (ApprovalDecision, error) {
	switch decision := ApprovalDecision(text); decision {
	case DecisionAllow, DecisionDeny:
		return decision, nil
	}
	return "", fmt.Errorf("%w: %q", ErrInvalidDecision, text)
}

// MarshalText writes the text form of the decision. A value that names no decision
// returns an error matching [ErrInvalidDecision].
func (d ApprovalDecision) MarshalText() ([]byte, error) {
	if _, err := ParseApprovalDecision(string(d)); err != nil {
		return nil, err
	}
	return []byte(d), nil
}

// UnmarshalText reads the text form of a decision, as [ParseApprovalDecision] does.
func (d *ApprovalDecision) UnmarshalText(text []byte) error {
	parsed, err := ParseApprovalDecision(string(text))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// ApprovalRefusal names why the transport refused an approval itself, without an
// asker's verdict. The known codes are the constants below, and [ParseApprovalRefusal]
// reads only those. [RefusedAnswer] builds no answer with any other code. Reading is
// tolerant: [ApprovalRefusal.UnmarshalText] keeps any code a newer version sends, and
// [ApprovalRefusal.Known] tells whether this version can translate it.
type ApprovalRefusal string

const (
	// RefusalInvalid means the request or its answer was malformed.
	RefusalInvalid ApprovalRefusal = "invalid"
	// RefusalTooLarge means a message exceeded its size bound.
	RefusalTooLarge ApprovalRefusal = "too_large"
	// RefusalTooMany means the transport already had its maximum of approvals in flight.
	RefusalTooMany ApprovalRefusal = "too_many"
	// RefusalNoAsker means no asker was available to answer.
	RefusalNoAsker ApprovalRefusal = "no_asker"
	// RefusalCancelled means the request was cancelled before an answer.
	RefusalCancelled ApprovalRefusal = "cancelled"
	// RefusalDeadline means the request ran out of time before an answer.
	RefusalDeadline ApprovalRefusal = "deadline"
	// RefusalStreamClosed means the stream closed before an answer.
	RefusalStreamClosed ApprovalRefusal = "stream_closed"
	// RefusalSendFailed means the answer could not be sent back.
	RefusalSendFailed ApprovalRefusal = "send_failed"
	// RefusalUnclassified means the transport refused for a cause it cannot name.
	RefusalUnclassified ApprovalRefusal = "unclassified"
)

// ErrInvalidRefusal is returned for text that names no approval refusal: through
// [ParseApprovalRefusal] any text but a known code, and through [RefusedAnswer] an
// empty or unknown code. Match it with errors.Is.
var ErrInvalidRefusal = errors.New("unknown approval refusal")

// ErrRefusedAllow is returned when an answer carries a Refusal but its decision is
// DecisionAllow. Match it with errors.Is.
var ErrRefusedAllow = errors.New("approval answer allows a refused tool")

// ParseApprovalRefusal reads a known refusal code from its text form. Any other text,
// the empty string included, returns an error matching [ErrInvalidRefusal].
func ParseApprovalRefusal(text string) (ApprovalRefusal, error) {
	switch refusal := ApprovalRefusal(text); refusal {
	case RefusalInvalid, RefusalTooLarge, RefusalTooMany, RefusalNoAsker,
		RefusalCancelled, RefusalDeadline, RefusalStreamClosed, RefusalSendFailed,
		RefusalUnclassified:
		return refusal, nil
	}
	return "", fmt.Errorf("%w: %q", ErrInvalidRefusal, text)
}

// Known reports whether the refusal is one of the codes this version defines. A
// reader that finds an unknown code still treats the answer as refused.
func (r ApprovalRefusal) Known() bool {
	_, err := ParseApprovalRefusal(string(r))
	return err == nil
}

// MarshalText writes the refusal as its code, known or not, so an answer relayed from
// a newer version keeps its code.
func (r ApprovalRefusal) MarshalText() ([]byte, error) {
	return []byte(r), nil
}

// UnmarshalText reads the refusal as its code, known or not. The empty text is no
// refusal.
func (r *ApprovalRefusal) UnmarshalText(text []byte) error {
	*r = ApprovalRefusal(text)
	return nil
}

// ApprovalRequest asks the daemon whether a tool may run during a turn. Tool names
// the tool and Subject what it would act on, as the backend reports them. ID
// correlates the request with its answer within one stream. It is not a secret.
// The request carries no session and no token: the daemon knows its own sessions,
// and no credential leaves the machine that runs it.
type ApprovalRequest struct {
	ID      string `json:"ID"`
	Tool    string `json:"Tool"`
	Subject string `json:"Subject"`
}

// ApprovalAnswer is the verdict on the request with the same ID.
type ApprovalAnswer struct {
	ID       string           `json:"ID"`
	Decision ApprovalDecision `json:"Decision"`
	// Reason is free text the daemon supplies for the human. The contracts never translate it.
	Reason string `json:"Reason"`
	// Refusal is set only when the transport refused the approval itself. It is empty
	// when an asker or the daemon decided. A non-empty Refusal must come with
	// DecisionDeny: [ApprovalAnswer.MarshalJSON] refuses to write any other decision.
	Refusal ApprovalRefusal `json:"Refusal,omitempty"`
}

// RefusedAnswer builds the answer to the request with requestID when the transport refused
// it: Decision is DecisionDeny and Refusal names the cause. It returns an error
// matching [ErrInvalidRefusal] when refusal is empty or not a known code.
func RefusedAnswer(requestID string, refusal ApprovalRefusal) (ApprovalAnswer, error) {
	if _, err := ParseApprovalRefusal(string(refusal)); err != nil {
		return ApprovalAnswer{}, err
	}
	return ApprovalAnswer{ID: requestID, Decision: DecisionDeny, Refusal: refusal}, nil
}

// MarshalJSON writes the answer. A Refusal must come with DecisionDeny: an allow with a
// Refusal returns ErrRefusedAllow, and an empty or unknown decision with a Refusal returns
// ErrInvalidDecision. A reader from before v0.35.0 ignores Refusal and decides on Decision
// alone, so it reads no allowance from an answer this module writes.
func (a ApprovalAnswer) MarshalJSON() ([]byte, error) {
	type wire ApprovalAnswer
	out := wire(a)
	if out.Refusal != "" && out.Decision == DecisionAllow {
		return nil, ErrRefusedAllow
	}
	return json.Marshal(out)
}

// UnmarshalJSON reads the answer. A null leaves it unchanged. A Refusal is read as
// DecisionDeny, so an allow, or a Decision missing from the JSON, becomes a denial. A
// Decision that is an empty string or unknown fails to decode; a Decision of null
// reads as missing. It does not inherit the caller's json.Decoder settings, so
// DisallowUnknownFields does not reject unknown keys.
func (a *ApprovalAnswer) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	type wire ApprovalAnswer
	var incoming wire
	if err := json.Unmarshal(data, &incoming); err != nil {
		return err
	}
	if incoming.Refusal != "" {
		incoming.Decision = DecisionDeny
	}
	*a = ApprovalAnswer(incoming)
	return nil
}

// Allowed reports whether the answer lets the tool run. Only DecisionAllow with an
// empty Refusal does, so an empty, unknown, malformed or refused decision fails
// closed, and so does any answer that carries a refusal, known or not.
func (a ApprovalAnswer) Allowed() bool {
	return a.Decision == DecisionAllow && a.Refusal == ""
}
