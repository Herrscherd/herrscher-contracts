package contracts

import (
	"encoding/json"
	"testing"
)

func TestEventInjectedPartsKeepTheirWire(t *testing.T) {
	want := Event{T: "reply", Done: true, Injected: []InjectedPart{
		{Part: ContextMemory, Tokens: 1200},
		{Part: ContextSkills, Tokens: 340},
	}}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	const wire = `{"t":"reply","done":true,"injected":[{"part":"memory","tokens":1200},{"part":"skills","tokens":340}]}`
	if string(raw) != wire {
		t.Fatalf("wire JSON:\n got %s\nwant %s", raw, wire)
	}
	got, err := DecodeEvent(raw)
	if err != nil {
		t.Fatalf("DecodeEvent() error = %v", err)
	}
	if len(got.Injected) != 2 || got.Injected[0] != want.Injected[0] || got.Injected[1] != want.Injected[1] {
		t.Fatalf("decoded injected = %+v, want %+v", got.Injected, want.Injected)
	}
}

func TestDecodeEventKeepsAnInjectedPartItDoesNotKnow(t *testing.T) {
	got, err := DecodeEvent([]byte(`{"t":"reply","done":true,"injected":[{"part":"routines","tokens":80}]}`))
	if err != nil {
		t.Fatalf("DecodeEvent() error = %v", err)
	}
	if len(got.Injected) != 1 || got.Injected[0] != (InjectedPart{Part: "routines", Tokens: 80}) {
		t.Fatalf("decoded injected = %+v, want the unknown part kept", got.Injected)
	}
}

func TestDecodeEventNamesAWrongInjectedTypeByItsWireName(t *testing.T) {
	_, err := DecodeEvent([]byte(`{"t":"reply","injected":[{"part":"memory","tokens":"many"}]}`))
	requireNamedWireType(t, err, ErrInvalidEvent, "injected")
}
