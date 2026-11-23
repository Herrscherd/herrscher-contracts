package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParsePruneActionRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParsePruneAction(receivedProbe)
	if !errors.Is(err, ErrUnknownPruneAction) {
		t.Fatalf("error = %v, want ErrUnknownPruneAction", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
}
