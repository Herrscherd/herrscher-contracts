package contracts

import (
	"errors"
	"testing"
)

func TestParseHistoryRoleRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseHistoryRole(receivedSecret)
	if !errors.Is(err, ErrUnknownHistoryRole) {
		t.Fatalf("error = %v, want ErrUnknownHistoryRole", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
