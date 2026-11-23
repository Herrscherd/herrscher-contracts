package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseApprovalRefusalRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseApprovalRefusal(receivedSecret)
	if !errors.Is(err, ErrInvalidRefusal) {
		t.Fatalf("error = %v, want ErrInvalidRefusal", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedSecret)
}
