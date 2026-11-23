package contracts

import (
	"errors"
	"testing"
)

func TestParseCompactOutcomeRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseCompactOutcome(receivedSecret)
	if !errors.Is(err, ErrUnknownCompactOutcome) {
		t.Fatalf("error = %v, want ErrUnknownCompactOutcome", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
