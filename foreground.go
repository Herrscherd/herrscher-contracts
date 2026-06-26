package contracts

import "context"

// Foreground is an optional gateway capability for a gateway that must own the
// process's main thread — typically a TUI that takes over stdin/stdout (the
// in-tree terminal gateway does). The composition root runs at most one
// foreground gateway, and only on an interactive TTY; every other bound gateway
// runs headless inside the hub. Quitting the foreground gateway must cancel the
// daemon, so RunForeground is handed the cancel func and is expected to block
// until the user exits or ctx is cancelled.
//
// A gateway without this capability is driven entirely by the hub (it polls the
// gateway's reader and fans turn events to it); nothing special runs in the
// foreground for it.
type Foreground interface {
	RunForeground(ctx context.Context, cancel context.CancelFunc) error
}
