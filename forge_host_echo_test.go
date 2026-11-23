package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseForgeHostRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseForgeHost(receivedSecret)
	if !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("error = %v, want ErrInvalidForgeHost", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
