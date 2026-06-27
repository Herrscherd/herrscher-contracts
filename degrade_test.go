package contracts

import (
	"context"
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
