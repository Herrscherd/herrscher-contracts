package contracts

import (
	"errors"
	"testing"
)

func TestParseLimitWindowRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseLimitWindow(receivedSecret)
	if !errors.Is(err, ErrUnknownLimitWindow) {
		t.Fatalf("error = %v, want ErrUnknownLimitWindow", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
