package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseHistoryRoleRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseHistoryRole(receivedSecret)
	if !errors.Is(err, ErrUnknownHistoryRole) {
		t.Fatalf("error = %v, want ErrUnknownHistoryRole", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedSecret)
}
