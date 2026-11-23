package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestTurnOutcomeTextFormIsPinned(t *testing.T) {
	tests := []struct {
		outcome TurnOutcome
		text    string
	}{
		{outcome: TurnFailed, text: "failed"},
		{outcome: TurnInterrupted, text: "interrupted"},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			marshaled, err := tt.outcome.MarshalText()
			if err != nil || string(marshaled) != tt.text {
				t.Fatalf("MarshalText() = %q, %v, want %q", marshaled, err, tt.text)
			}
			var parsed TurnOutcome
			if err := parsed.UnmarshalText([]byte(tt.text)); err != nil || parsed != tt.outcome {
				t.Fatalf("UnmarshalText(%q) = %q, %v, want %q", tt.text, parsed, err, tt.outcome)
			}
		})
	}
}

func TestTurnOutcomeEmptyIsACompletedTurn(t *testing.T) {
	outcome, err := ParseTurnOutcome("")
	if err != nil || outcome != "" {
		t.Fatalf("ParseTurnOutcome(\"\") = %q, %v, want the completed turn", outcome, err)
	}
	encoded, err := json.Marshal(Event{T: "reply", Done: true})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), "outcome") {
		t.Fatalf("a completed turn writes an outcome: %s", encoded)
	}
}

func TestTurnOutcomeRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	var outcome TurnOutcome
	err := outcome.UnmarshalText([]byte(receivedSecret))
	if !errors.Is(err, ErrUnknownTurnOutcome) {
		t.Fatalf("error = %v, want ErrUnknownTurnOutcome", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}

func TestTurnOutcomeMarshalRefusesAnUnknownValue(t *testing.T) {
	_, err := TurnOutcome("exploded").MarshalText()
	if !errors.Is(err, ErrUnknownTurnOutcome) {
		t.Fatalf("MarshalText() error = %v, want ErrUnknownTurnOutcome", err)
	}
}

func TestDecodeEventRefusesAnUnknownOutcomeByItsWireName(t *testing.T) {
	input := fmt.Sprintf(`{"t":"reply","done":true,"outcome":%q}`, receivedSecret)
	_, err := DecodeEvent([]byte(input))
	if !errors.Is(err, ErrInvalidEvent) || !errors.Is(err, ErrUnknownTurnOutcome) {
		t.Fatalf("error = %v, want ErrInvalidEvent and ErrUnknownTurnOutcome", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "outcome") {
		t.Fatalf("message %q does not name outcome", msg)
	}
	if strings.Contains(msg, receivedSecret) {
		t.Fatalf("message %q echoes the received text", msg)
	}
}
