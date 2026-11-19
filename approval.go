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

// ApprovalAnswer is the daemon's verdict on the request with the same ID. Reason is
// free text the daemon supplies for the human. The contracts never translate it.
type ApprovalAnswer struct {
	ID       string           `json:"ID"`
	Decision ApprovalDecision `json:"Decision"`
	Reason   string           `json:"Reason"`
}

// Allowed reports whether the answer lets the tool run. Only DecisionAllow does, so
// an empty, unknown or malformed decision fails closed.
func (a ApprovalAnswer) Allowed() bool {
	return a.Decision == DecisionAllow
}
