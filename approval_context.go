package contracts

import "context"

type approvalAskerKey struct{}

// ApprovalAsker asks the daemon for one tool approval and waits for the answer. It
// returns when the answer arrives or ctx ends. An asker that cannot reach the
// daemon returns a denial, never an allowance.
type ApprovalAsker func(ctx context.Context, req ApprovalRequest) ApprovalAnswer

// WithApprovalAsker returns a context that carries asker for the turn it is passed
// to. A backend that gates tools reads it back with ApprovalAskerFrom.
func WithApprovalAsker(ctx context.Context, asker ApprovalAsker) context.Context {
	return context.WithValue(ctx, approvalAskerKey{}, asker)
}

// ApprovalAskerFrom returns the asker a context carries. It returns false when none
// does, or when the one it carries is nil.
func ApprovalAskerFrom(ctx context.Context) (ApprovalAsker, bool) {
	asker, isAsker := ctx.Value(approvalAskerKey{}).(ApprovalAsker)
	return asker, isAsker && asker != nil
}
