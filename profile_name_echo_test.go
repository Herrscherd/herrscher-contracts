package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseProfileNameRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseProfileName(receivedProbe + "!")
	if !errors.Is(err, ErrProfileName) {
		t.Fatalf("error = %v, want ErrProfileName", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
}
