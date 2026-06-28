package contracts

import (
	"context"
	"strings"
	"testing"
)

// recordingSink is a Gateway that also records routed/plain sink calls.
type recordingSink struct {
	plain  []Event
	routed []Conversation
}

func (r *recordingSink) Manifest() Manifest { return Manifest{Kind: "rec"} }
func (r *recordingSink) Post(context.Context, Conversation, string) (MessageID, error) {
	return "", nil
}
func (r *recordingSink) Reply(context.Context, Conversation, MessageID, string) (MessageID, error) {
	return "", nil
}
func (r *recordingSink) React(context.Context, Conversation, MessageID, string) error { return nil }
func (r *recordingSink) Menu(context.Context, Conversation, MessageID, string, []Choice) error {
	return nil
}
func (r *recordingSink) Emit(e Event)                   { r.plain = append(r.plain, e) }
func (r *recordingSink) EmitTo(c Conversation, _ Event) { r.routed = append(r.routed, c) }

// plainSink is a Gateway that only implements EventSink (not RoutedEventSink).
type plainSink struct {
	calls []Event
}

func (p *plainSink) Manifest() Manifest { return Manifest{Kind: "plain"} }
func (p *plainSink) Post(context.Context, Conversation, string) (MessageID, error) {
	return "", nil
}
func (p *plainSink) Reply(context.Context, Conversation, MessageID, string) (MessageID, error) {
	return "", nil
}
func (p *plainSink) React(context.Context, Conversation, MessageID, string) error { return nil }
func (p *plainSink) Menu(context.Context, Conversation, MessageID, string, []Choice) error {
	return nil
}
func (p *plainSink) Emit(e Event) { p.calls = append(p.calls, e) }

func full() Capabilities { return Capabilities{Reactions: true, SelectMenus: true, Replies: true} }

func TestDegradePassThroughWhenCapable(t *testing.T) {
	rec := &recGateway{manifest: Manifest{Capabilities: full()}}
	d := Degrade(rec)
	ctx, conv := context.Background(), Conversation{Gateway: "discord", ID: "c"}

	_, _ = d.Reply(ctx, conv, "m", "hello")
	_ = d.React(ctx, conv, "m", "👀")
	_ = d.Menu(ctx, conv, "m", "pick", []Choice{{Label: "A", Value: "a"}})

	if len(rec.replies) != 1 || len(rec.reacts) != 1 || len(rec.menus) != 1 {
		t.Fatalf("capable gateway should receive rich calls: %+v", rec)
	}
	if len(rec.posts) != 0 {
		t.Fatalf("no fallback Post expected when capable")
	}
}

func TestDegradeReplyToPost(t *testing.T) {
	rec := &recGateway{manifest: Manifest{Capabilities: Capabilities{Reactions: true, SelectMenus: true}}}
	d := Degrade(rec)
	_, _ = d.Reply(context.Background(), Conversation{}, "m", "hello")
	if len(rec.replies) != 0 || len(rec.posts) != 1 || rec.posts[0] != "hello" {
		t.Fatalf("reply should degrade to post: %+v", rec)
	}
}

func TestDegradeReactNoOp(t *testing.T) {
	rec := &recGateway{manifest: Manifest{Capabilities: Capabilities{SelectMenus: true, Replies: true}}}
	d := Degrade(rec)
	if err := d.React(context.Background(), Conversation{}, "m", "👀"); err != nil {
		t.Fatalf("degraded react should be a no-op, got %v", err)
	}
	if len(rec.reacts) != 0 {
		t.Fatalf("react must not reach a gateway without Reactions")
	}
}

func TestDegradeMenuToNumberedList(t *testing.T) {
	rec := &recGateway{manifest: Manifest{Capabilities: Capabilities{Reactions: true, Replies: true}}}
	d := Degrade(rec)
	_ = d.Menu(context.Background(), Conversation{}, "m", "Choose:", []Choice{
		{Label: "Alpha", Value: "a"}, {Label: "Beta", Value: "b"},
	})
	if len(rec.menus) != 0 || len(rec.posts) != 1 {
		t.Fatalf("menu should degrade to a post: %+v", rec)
	}
	got := rec.posts[0]
	if !strings.Contains(got, "Choose:") || !strings.Contains(got, "1. Alpha") || !strings.Contains(got, "2. Beta") {
		t.Fatalf("numbered list malformed: %q", got)
	}
}

func TestDegradeForwardsSinks(t *testing.T) {
	rec := &recordingSink{}
	d := Degrade(rec)

	es, ok := d.(EventSink)
	if !ok {
		t.Fatal("degraded gateway must satisfy EventSink")
	}
	es.Emit(Event{T: "chunk", Text: "x"})
	if len(rec.plain) != 1 {
		t.Fatalf("Emit not forwarded to inner: %+v", rec.plain)
	}

	rs, ok := d.(RoutedEventSink)
	if !ok {
		t.Fatal("degraded gateway must satisfy RoutedEventSink")
	}
	rs.EmitTo(Conversation{Gateway: "rec", ID: "c1"}, Event{T: "reply"})
	if len(rec.routed) != 1 || rec.routed[0].ID != "c1" {
		t.Fatalf("EmitTo not forwarded to inner: %+v", rec.routed)
	}
}

func TestDegradeEmitToFallsBackToEmit(t *testing.T) {
	plain := &plainSink{}
	d := Degrade(plain)

	rs, ok := d.(RoutedEventSink)
	if !ok {
		t.Fatal("degraded gateway must satisfy RoutedEventSink")
	}
	rs.EmitTo(Conversation{Gateway: "plain", ID: "c1"}, Event{T: "test", Text: "fallback"})
	if len(plain.calls) != 1 {
		t.Fatalf("EmitTo should fall back to Emit on plain sink: %+v", plain.calls)
	}
	if plain.calls[0].T != "test" || plain.calls[0].Text != "fallback" {
		t.Fatalf("fallback event malformed: %+v", plain.calls[0])
	}
}
