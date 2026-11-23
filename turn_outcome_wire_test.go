package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDecodeEventReadsANullOutcomeAsACompletedTurn(t *testing.T) {
	event, err := DecodeEvent([]byte(`{"t":"reply","done":true,"outcome":null}`))
	if err != nil {
		t.Fatalf("DecodeEvent() error = %v, want nil", err)
	}
	if event.Outcome != "" {
		t.Fatalf("Outcome = %q, want the completed turn", event.Outcome)
	}
}

func TestPlainUnmarshalReturnsTheBareUnknownOutcome(t *testing.T) {
	var event Event
	err := json.Unmarshal([]byte(fmt.Sprintf(`{"t":"reply","outcome":%q}`, receivedProbe)), &event)
	if !errors.Is(err, ErrUnknownTurnOutcome) || err.Error() != ErrUnknownTurnOutcome.Error() {
		t.Fatalf("error = %v, want the bare ErrUnknownTurnOutcome", err)
	}
}

func TestMarshalOfAnUnknownOutcomeNamesTheGoTypeAsADocumentedException(t *testing.T) {
	_, err := json.Marshal(Event{T: "reply", Outcome: "exploded"})
	if !errors.Is(err, ErrUnknownTurnOutcome) {
		t.Fatalf("error = %v, want it to match ErrUnknownTurnOutcome", err)
	}
	if !strings.Contains(err.Error(), "contracts.TurnOutcome") {
		t.Fatalf("message %q does not name contracts.TurnOutcome", err)
	}
}
