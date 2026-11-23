package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseHealthStateRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseHealthState(receivedSecret)
	if !errors.Is(err, ErrUnknownHealthState) {
		t.Fatalf("error = %v, want ErrUnknownHealthState", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
