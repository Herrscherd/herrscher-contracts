package contracts

import (
	"context"
	"fmt"
	"strings"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

// Degrade wraps a Gateway and falls back to a simpler action for each action the
// plugin does not announce. The manager always calls the rich method; degradation
// lives here, never in the domain.
func Degrade(gateway Gateway) Gateway {
	wrapped := degrading{g: gateway, renderer: NewRenderer()}
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

type degrading struct {
	g        Gateway
	renderer *Renderer
}

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

func (d degrading) Notify(ctx context.Context, conv Conversation, lang i18n.Language, notice Notice) (MessageID, error) {
	if s, ok := d.g.(NoticeSink); ok {
		return s.Notify(ctx, conv, lang, notice)
	}
	text, err := d.renderer.Render(lang, notice)
	if err != nil {
		return "", err
	}
	return d.g.Post(ctx, conv, text)
}

func (d degrading) BindSessionControl(control SessionControl) {
	if r, ok := d.g.(SessionControlReceiver); ok {
		r.BindSessionControl(control)
	}
}

type degradingSink struct{ degrading }

func (d degradingSink) Emit(event Event) {
	if s, ok := d.g.(EventSink); ok {
		s.Emit(event)
	}
}

func (d degradingSink) EmitTo(conv Conversation, event Event) {
	if s, ok := d.g.(RoutedEventSink); ok {
		s.EmitTo(conv, event)
		return
	}
	if s, ok := d.g.(EventSink); ok {
		s.Emit(event)
	}
}
