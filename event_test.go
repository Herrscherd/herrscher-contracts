package contracts

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestEventJSONLegacyPayloadIsUnchanged(t *testing.T) {
	e := Event{
		T:    "reply",
		Who:  "alice",
		Text: "done",
		Done: true,
		Cost: 0.0042,
	}

	got, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal legacy event: %v", err)
	}
	const want = `{"t":"reply","who":"alice","text":"done","done":true,"cost":0.0042}`
	if string(got) != want {
		t.Fatalf("legacy JSON changed:\n got %s\nwant %s", got, want)
	}
}

func TestEventJSONPreservesTurnIdentity(t *testing.T) {
	want := Event{
		T:                  "reply",
		Text:               "done",
		Done:               true,
		SessionIncarnation: "session-42-v3",
		TurnID:             "turn-8f31",
		Agent:              "reviewer",
	}

	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal enriched event: %v", err)
	}
	const wantJSON = `{"t":"reply","text":"done","done":true,"session_incarnation":"session-42-v3","turn_id":"turn-8f31","agent":"reviewer"}`
	if string(encoded) != wantJSON {
		t.Fatalf("enriched JSON mismatch:\n got %s\nwant %s", encoded, wantJSON)
	}

	var got Event
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal enriched event: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("turn identity was not preserved:\n got %+v\nwant %+v", got, want)
	}
}

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
