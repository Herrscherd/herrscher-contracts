package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseRepoRefRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseRepoRef(receivedSecret)
	if !errors.Is(err, ErrInvalidRepoRef) {
		t.Fatalf("error = %v, want ErrInvalidRepoRef", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
