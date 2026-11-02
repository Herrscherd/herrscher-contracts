package contracts

import (
	"context"
	"fmt"
	"strings"
)

// Degrade wraps a Gateway and rabats actions the plugin does not announce. The
// manager always calls the rich method; degradation lives here, never in the
// domain.
func Degrade(gateway Gateway) Gateway {
	wrapped := degrading{gateway}
	if isEventSink(gateway) {
		return degradingSink{wrapped}
	}
	return wrapped
}

func isEventSink(gateway Gateway) bool {
	if _, ok := gateway.(RoutedEventSink); ok {
		return true
	}
	_, ok := gateway.(EventSink)
	return ok
}

type degrading struct{ g Gateway }

// Manifest returns the manifest of the wrapped gateway, unchanged.
func (d degrading) Manifest() Manifest { return d.g.Manifest() }

// Post sends text through the wrapped gateway as it is.
func (d degrading) Post(ctx context.Context, conv Conversation, text string) (MessageID, error) {
	return d.g.Post(ctx, conv, text)
}

// Reply replies to replyTo through the wrapped gateway. When the gateway does not
// announce replies, it posts text as a plain message instead.
func (d degrading) Reply(ctx context.Context, conv Conversation, replyTo MessageID, text string) (MessageID, error) {
	if !d.g.Manifest().Capabilities.Replies {
		return d.g.Post(ctx, conv, text)
	}
	return d.g.Reply(ctx, conv, replyTo, text)
}

// React reacts to msg through the wrapped gateway. When the gateway does not announce
// reactions, it does nothing.
func (d degrading) React(ctx context.Context, conv Conversation, msg MessageID, emoji string) error {
	if !d.g.Manifest().Capabilities.Reactions {
		return nil
	}
	return d.g.React(ctx, conv, msg, emoji)
}

// Menu offers opts through the wrapped gateway. When the gateway does not announce
// select menus, it posts the prompt followed by the options as a numbered list.
func (d degrading) Menu(ctx context.Context, conv Conversation, replyTo MessageID, prompt string, opts []Choice) error {
	if !d.g.Manifest().Capabilities.SelectMenus {
		var body strings.Builder
		_, _ = body.WriteString(prompt)
		for i, o := range opts {
			fmt.Fprintf(&body, "\n%d. %s", i+1, o.Label)
		}
		_, err := d.g.Post(ctx, conv, body.String())
		return err
	}
	return d.g.Menu(ctx, conv, replyTo, prompt, opts)
}

// BindSessionControl hands c to the wrapped gateway when it can receive session
// control, and does nothing otherwise.
func (d degrading) BindSessionControl(control SessionControl) {
	if r, ok := d.g.(SessionControlReceiver); ok {
		r.BindSessionControl(control)
	}
}

type degradingSink struct{ degrading }

// Emit forwards e to the wrapped gateway when it is an event sink, and drops it otherwise.
func (d degradingSink) Emit(event Event) {
	if s, ok := d.g.(EventSink); ok {
		s.Emit(event)
	}
}

// EmitTo forwards e for conv to the wrapped gateway: to its routed sink when it has one,
// otherwise to its plain sink, if it has one.
func (d degradingSink) EmitTo(conv Conversation, event Event) {
	if s, ok := d.g.(RoutedEventSink); ok {
		s.EmitTo(conv, event)
		return
	}
	if s, ok := d.g.(EventSink); ok {
		s.Emit(event)
	}
}
