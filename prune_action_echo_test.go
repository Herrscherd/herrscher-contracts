package contracts

import (
	"errors"
	"testing"
)

func TestParsePruneActionRefusesAnUnknownTextWithoutEcho(t *testing.T) {
	_, err := ParsePruneAction(receivedSecret)
	if !errors.Is(err, ErrUnknownPruneAction) {
		t.Fatalf("error = %v, want ErrUnknownPruneAction", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
