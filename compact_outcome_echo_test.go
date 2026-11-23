package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseCompactOutcomeRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseCompactOutcome(receivedSecret)
	if !errors.Is(err, ErrUnknownCompactOutcome) {
		t.Fatalf("error = %v, want ErrUnknownCompactOutcome", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
