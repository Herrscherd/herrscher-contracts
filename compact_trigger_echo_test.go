package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseCompactTriggerRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseCompactTrigger(receivedSecret)
	if !errors.Is(err, ErrUnknownCompactTrigger) {
		t.Fatalf("error = %v, want ErrUnknownCompactTrigger", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
