package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseRepoRefRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseRepoRef(receivedProbe)
	if !errors.Is(err, ErrInvalidRepoRef) {
		t.Fatalf("error = %v, want ErrInvalidRepoRef", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
}
