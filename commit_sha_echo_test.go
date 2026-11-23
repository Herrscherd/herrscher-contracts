package contracts

import (
	"errors"
	"testing"
)

func TestParseCommitSHARefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseCommitSHA(receivedSecret)
	if !errors.Is(err, ErrInvalidCommitSHA) {
		t.Fatalf("error = %v, want ErrInvalidCommitSHA", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
