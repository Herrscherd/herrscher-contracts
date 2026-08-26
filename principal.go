package contracts

import "context"

// principalKey is unexported, so the only way to name a caller is WithPrincipal.
// A context value keyed by a string would let any package in the process write
// one, which is the opposite of what naming a caller is for.
type principalKey struct{}

// WithPrincipal marks a context as carrying who is asking. A gateway sets it
// from the identity its platform already gives it, prefixed with its own kind:
// "chat:1234", "web:alice". The daemon reads it to decide what that caller may
// run.
//
// It is additive on purpose. A gateway built before this existed names nobody,
// and the daemon's answer to an unnamed caller is to leave the decision where
// it already was, so an old plugin keeps working exactly as it did.
//
// An empty principal is not a name. It yields the same context back, so a
// gateway that has no identity to offer does not have to know whether calling
// this with "" would be worse than not calling it at all.
func WithPrincipal(ctx context.Context, principal string) context.Context {
	if principal == "" {
		return ctx
	}
	return context.WithValue(ctx, principalKey{}, principal)
}

// PrincipalFrom returns the principal a context carries, empty when none does.
func PrincipalFrom(ctx context.Context) string {
	p, _ := ctx.Value(principalKey{}).(string)
	return p
}
