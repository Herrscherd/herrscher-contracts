package contracts

import "testing"

func TestEventSinkIsOptionalCapability(t *testing.T) {
	// A type implementing EventSink must satisfy the interface; the compiler
	// proves the method set. This test documents the contract shape.
	var _ EventSink = sinkStub{}
	e := Event{T: "chunk", Text: "hi"}
	if e.T != "chunk" || e.Text != "hi" {
		t.Fatalf("Event fields not wired: %+v", e)
	}
}

type sinkStub struct{}

func (sinkStub) Emit(Event) {}
