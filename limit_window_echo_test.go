package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseLimitWindowRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseLimitWindow(receivedSecret)
	if !errors.Is(err, ErrUnknownLimitWindow) {
		t.Fatalf("error = %v, want ErrUnknownLimitWindow", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedSecret)
}
