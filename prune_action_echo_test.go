package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParsePruneActionRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParsePruneAction(receivedSecret)
	if !errors.Is(err, ErrUnknownPruneAction) {
		t.Fatalf("error = %v, want ErrUnknownPruneAction", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
