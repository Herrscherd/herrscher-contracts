package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseHealthStateRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParseHealthState(receivedSecret)
	if !errors.Is(err, ErrUnknownHealthState) {
		t.Fatalf("error = %v, want ErrUnknownHealthState", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedSecret)
}
