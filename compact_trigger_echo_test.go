package contracts

import (
	"errors"
	"testing"
)

func TestParseCompactTriggerRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseCompactTrigger(receivedSecret)
	if !errors.Is(err, ErrUnknownCompactTrigger) {
		t.Fatalf("error = %v, want ErrUnknownCompactTrigger", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
