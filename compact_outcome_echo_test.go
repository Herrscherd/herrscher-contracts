package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseCompactOutcomeRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseCompactOutcome(receivedProbe)
	if !errors.Is(err, ErrUnknownCompactOutcome) {
		t.Fatalf("error = %v, want ErrUnknownCompactOutcome", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
}
