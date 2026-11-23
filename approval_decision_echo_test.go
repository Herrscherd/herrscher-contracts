package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseApprovalDecisionRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseApprovalDecision(receivedSecret)
	if !errors.Is(err, ErrInvalidDecision) {
		t.Fatalf("error = %v, want ErrInvalidDecision", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
