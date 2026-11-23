package contracts

import (
	"errors"
	"testing"
)

func TestParseRepoRefRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseRepoRef(receivedSecret)
	if !errors.Is(err, ErrInvalidRepoRef) {
		t.Fatalf("error = %v, want ErrInvalidRepoRef", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
