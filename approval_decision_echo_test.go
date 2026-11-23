package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseApprovalDecisionRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseApprovalDecision(receivedSecret)
	if !errors.Is(err, ErrInvalidDecision) {
		t.Fatalf("error = %v, want ErrInvalidDecision", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedSecret)
}
