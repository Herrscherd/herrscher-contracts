package contracts

import (
	"context"
	"fmt"
	"strings"
)

// Degrade wraps a Gateway and rabats actions the plugin does not announce. The
// manager always calls the rich method; degradation lives here, never in the
// domain.
func Degrade(g Gateway) Gateway { return degrading{g} }

type degrading struct{ g Gateway }

func (d degrading) Manifest() Manifest { return d.g.Manifest() }

func (d degrading) Post(ctx context.Context, conv Conversation, text string) (MessageID, error) {
	return d.g.Post(ctx, conv, text)
}

func (d degrading) Reply(ctx context.Context, conv Conversation, replyTo MessageID, text string) (MessageID, error) {
	if !d.g.Manifest().Capabilities.Replies {
		return d.g.Post(ctx, conv, text)
	}
	return d.g.Reply(ctx, conv, replyTo, text)
}

func (d degrading) React(ctx context.Context, conv Conversation, msg MessageID, emoji string) error {
	if !d.g.Manifest().Capabilities.Reactions {
		return nil
	}
	return d.g.React(ctx, conv, msg, emoji)
}

func (d degrading) Menu(ctx context.Context, conv Conversation, replyTo MessageID, prompt string, opts []Choice) error {
	if !d.g.Manifest().Capabilities.SelectMenus {
		var b strings.Builder
		b.WriteString(prompt)
		for i, o := range opts {
			fmt.Fprintf(&b, "\n%d. %s", i+1, o.Label)
		}
		_, err := d.g.Post(ctx, conv, b.String())
		return err
	}
	return d.g.Menu(ctx, conv, replyTo, prompt, opts)
}

// BindSessionControl forwards the runtime session controller to the wrapped
// gateway when it drives the session lifecycle itself (e.g. slash commands).
// Degradation never hides this capability, so the host's SessionControlReceiver
// assertion succeeds even on a wrapped gateway; gateways that don't want it
// simply aren't SessionControlReceivers underneath and the call is a no-op.
func (d degrading) BindSessionControl(c SessionControl) {
	if r, ok := d.g.(SessionControlReceiver); ok {
		r.BindSessionControl(c)
	}
}

func (d degrading) Emit(e Event) {
	if s, ok := d.g.(EventSink); ok {
		s.Emit(e)
	}
}

func (d degrading) EmitTo(conv Conversation, e Event) {
	if s, ok := d.g.(RoutedEventSink); ok {
		s.EmitTo(conv, e)
		return
	}
	if s, ok := d.g.(EventSink); ok {
		s.Emit(e) // inner renders unrouted; acceptable for single-conversation gateways
	}
}
