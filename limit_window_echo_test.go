package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseLimitWindowRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseLimitWindow(receivedSecret)
	if !errors.Is(err, ErrUnknownLimitWindow) {
		t.Fatalf("error = %v, want ErrUnknownLimitWindow", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
