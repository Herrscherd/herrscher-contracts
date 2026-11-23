package contracts

import (
	"errors"
	"testing"
)

func TestProfileHomeEnvRefusesAVendorWithoutEchoingIt(t *testing.T) {
	_, err := ProfileHomeEnv(receivedSecret)
	if !errors.Is(err, ErrNoProfileHome) {
		t.Fatalf("error = %v, want ErrNoProfileHome", err)
	}
	requireNoSecretFragment(t, err.Error(), receivedSecret)
}
