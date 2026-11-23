package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseHistoryRoleRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseHistoryRole(receivedSecret)
	if !errors.Is(err, ErrUnknownHistoryRole) {
		t.Fatalf("error = %v, want ErrUnknownHistoryRole", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
