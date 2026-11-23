package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseCommitSHARefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseCommitSHA(receivedSecret)
	if !errors.Is(err, ErrInvalidCommitSHA) {
		t.Fatalf("error = %v, want ErrInvalidCommitSHA", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedSecret)
}
