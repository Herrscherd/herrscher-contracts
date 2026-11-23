package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseCommitSHARefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseCommitSHA(receivedSecret)
	if !errors.Is(err, ErrInvalidCommitSHA) {
		t.Fatalf("error = %v, want ErrInvalidCommitSHA", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
