package contracts

import (
	"encoding/json"
	"testing"
)

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

func TestEventCarriesTokenBreakdown(t *testing.T) {
	e := Event{T: "reply", Done: true, Cost: 0.004, Tokens: 55, TokensIn: 30, CacheRead: 12, CacheCreate: 3}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var got Event
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.TokensIn != 30 || got.CacheRead != 12 || got.CacheCreate != 3 {
		t.Fatalf("token breakdown lost: %+v", got)
	}
}
