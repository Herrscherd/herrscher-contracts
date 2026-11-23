package contracts

import (
	"errors"
	"testing"
)

func TestParseHealthStateRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseHealthState(receivedSecret)
	if !errors.Is(err, ErrUnknownHealthState) {
		t.Fatalf("error = %v, want ErrUnknownHealthState", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
