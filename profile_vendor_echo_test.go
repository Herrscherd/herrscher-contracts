package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestProfileHomeEnvRefusesAVendorWithoutEchoingIt(t *testing.T) {
	_, err := ProfileHomeEnv(receivedSecret)
	if !errors.Is(err, ErrNoProfileHome) {
		t.Fatalf("error = %v, want ErrNoProfileHome", err)
	}
	if strings.Contains(err.Error(), receivedSecret) {
		t.Fatalf("message %q echoes the received vendor", err)
	}
}
