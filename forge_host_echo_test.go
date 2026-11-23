package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseForgeHostRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseForgeHost(receivedProbe)
	if !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("error = %v, want ErrInvalidForgeHost", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
}
