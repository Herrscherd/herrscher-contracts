package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseApprovalRefusalRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseApprovalRefusal(receivedSecret)
	if !errors.Is(err, ErrInvalidRefusal) {
		t.Fatalf("error = %v, want ErrInvalidRefusal", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
