package contracts

// ApprovalDecision is the verdict on one tool approval request. The closed set is
// DecisionAllow and DecisionDeny. Any other value, empty included, is a denial.
type ApprovalDecision string

const (
	// DecisionAllow lets the tool run.
	DecisionAllow ApprovalDecision = "allow"
	// DecisionDeny refuses the tool.
	DecisionDeny ApprovalDecision = "deny"
)

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
