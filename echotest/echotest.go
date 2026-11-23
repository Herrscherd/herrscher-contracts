// Package echotest holds the checks that a refusal does not echo the input it
// refuses. It is test support: import it from _test.go files only.
package echotest

import (
	"errors"
	"strings"
	"testing"
)

// Secret is the input a test feeds to a parser, and the value its refusal must
// not echo.
const Secret = "zq8Vk2wXr7Lm"

const fragmentBytes = 4

// RequireRefusalWithoutEcho fails the test unless err matches sentinel and its
// message contains no fragment of secret.
func RequireRefusalWithoutEcho(tb testing.TB, err, sentinel error, secret string) {
	tb.Helper()
	if !errors.Is(err, sentinel) {
		tb.Fatalf("error = %v, want %v", err, sentinel)
	}
	RequireNoSecretFragment(tb, err.Error(), secret)
}

// RequireBareSentinel fails the test unless err is sentinel with no text added
// to its message.
func RequireBareSentinel(tb testing.TB, err, sentinel error) {
	tb.Helper()
	if err == nil || !errors.Is(err, sentinel) || err.Error() != sentinel.Error() {
		tb.Fatalf("error = %v, want the bare sentinel %v", err, sentinel)
	}
}

// RequireNoSecretFragment fails the test if msg contains any run of
// four consecutive bytes of secret.
func RequireNoSecretFragment(tb testing.TB, msg, secret string) {
	tb.Helper()
	for start := 0; start+fragmentBytes <= len(secret); start++ {
		fragment := secret[start : start+fragmentBytes]
		if strings.Contains(msg, fragment) {
			tb.Fatalf("message %q echoes %q from the received input", msg, fragment)
		}
	}
}
