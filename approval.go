package contracts

import (
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
// asker's verdict. The known codes are the constants below. [ParseApprovalRefusal]
// reads only those, and writes are strict for the same reason. Reading is tolerant:
// [ApprovalRefusal.UnmarshalText] keeps any non-empty code a newer version sends,
// and [ApprovalRefusal.Known] tells whether this version can translate it.
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

// ErrInvalidRefusal is returned for text that names no approval refusal: any text
// through [ParseApprovalRefusal] that is not a known code, and an empty refusal
// through the text forms. Match it with errors.Is.
var ErrInvalidRefusal = errors.New("unknown approval refusal")

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

// MarshalText writes the text form of the refusal. Any non-empty code is written as
// it is, so an answer relayed from a newer version keeps its code. The empty refusal
// returns an error matching [ErrInvalidRefusal]: an absent refusal is an omitted field.
func (r ApprovalRefusal) MarshalText() ([]byte, error) {
	if r == "" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidRefusal, string(r))
	}
	return []byte(r), nil
}

// UnmarshalText reads the text form of a refusal. Any non-empty text is kept as the
// refusal, known or not, and the empty text returns an error matching [ErrInvalidRefusal].
func (r *ApprovalRefusal) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return fmt.Errorf("%w: %q", ErrInvalidRefusal, "")
	}
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
	// Refusal is set only when the transport refused the approval itself, and then
	// Decision is always DecisionDeny. It is empty when an asker or the daemon decided.
	// A reader from before v0.35.0 ignores Refusal and decides on Decision alone.
	Refusal ApprovalRefusal `json:"Refusal,omitempty"`
}

// RefusedAnswer builds the answer to the request with id when the transport refused
// it: Decision is DecisionDeny and Refusal names the cause. Pass a known code, the
// constants or a result of [ParseApprovalRefusal].
func RefusedAnswer(id string, refusal ApprovalRefusal) ApprovalAnswer {
	return ApprovalAnswer{ID: id, Decision: DecisionDeny, Refusal: refusal}
}

// Allowed reports whether the answer lets the tool run. Only DecisionAllow with an
// empty Refusal does, so an empty, unknown, malformed or refused decision fails
// closed, and so does any answer that carries a refusal, known or not.
func (a ApprovalAnswer) Allowed() bool {
	return a.Decision == DecisionAllow && a.Refusal == ""
}
