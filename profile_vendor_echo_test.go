package contracts

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestProfileHomeEnvRefusesAVendorWithoutEchoingIt(t *testing.T) {
	_, err := ProfileHomeEnv(receivedSecret)
	if !errors.Is(err, ErrNoProfileHome) {
		t.Fatalf("error = %v, want ErrNoProfileHome", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), receivedSecret)
}
