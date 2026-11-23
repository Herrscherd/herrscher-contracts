package contracts

import (
	"errors"
	"testing"
)

func TestParseProfileNameRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseProfileName(receivedSecret + "!")
	if !errors.Is(err, ErrProfileName) {
		t.Fatalf("error = %v, want ErrProfileName", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
