package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseProfileNameRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseProfileName(receivedSecret + "!")
	if !errors.Is(err, ErrProfileName) {
		t.Fatalf("error = %v, want ErrProfileName", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
