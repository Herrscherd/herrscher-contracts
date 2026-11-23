package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseCompactTriggerRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseCompactTrigger(receivedSecret)
	if !errors.Is(err, ErrUnknownCompactTrigger) {
		t.Fatalf("error = %v, want ErrUnknownCompactTrigger", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedSecret)
}
