package contracts

import (
	"errors"
	"testing"
)

func TestParseApprovalDecisionRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseApprovalDecision(receivedSecret)
	if !errors.Is(err, ErrInvalidDecision) {
		t.Fatalf("error = %v, want ErrInvalidDecision", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
