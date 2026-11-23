package contracts

import (
	"errors"
	"strings"
	"testing"
)

const echoFragmentBytes = 4

func requireRefusalWithoutEcho(tb testing.TB, err, sentinel error, secret string) {
	tb.Helper()
	if !errors.Is(err, sentinel) {
		tb.Fatalf("error = %v, want %v", err, sentinel)
	}
	requireNoSecretFragment(tb, err.Error(), secret)
}

func requireBareSentinel(tb testing.TB, err, sentinel error) {
	tb.Helper()
	if err == nil || !errors.Is(err, sentinel) || err.Error() != sentinel.Error() {
		tb.Fatalf("error = %v, want the bare sentinel %v", err, sentinel)
	}
}

func requireNoSecretFragment(tb testing.TB, msg, secret string) {
	tb.Helper()
	for start := 0; start+echoFragmentBytes <= len(secret); start++ {
		fragment := secret[start : start+echoFragmentBytes]
		if strings.Contains(msg, fragment) {
			tb.Fatalf("message %q echoes %q from the received input", msg, fragment)
		}
	}
}
