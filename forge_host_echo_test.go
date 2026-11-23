package contracts

import (
	"errors"
	"testing"
)

func TestParseForgeHostRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseForgeHost(receivedSecret)
	if !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("error = %v, want ErrInvalidForgeHost", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
