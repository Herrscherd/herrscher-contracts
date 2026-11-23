package contracts

import (
	"errors"
	"testing"
)

func TestParseApprovalRefusalRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseApprovalRefusal(receivedSecret)
	if !errors.Is(err, ErrInvalidRefusal) {
		t.Fatalf("error = %v, want ErrInvalidRefusal", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
