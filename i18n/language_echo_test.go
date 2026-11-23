package i18n_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

const echoFragmentBytes = 4

func TestParseLanguageRefusesAnUnsupportedTextWithoutEcho(t *testing.T) {
	const received = "zq8Vk2wXr7Lm"
	_, err := i18n.ParseLanguage(received + "_XX.UTF-8")
	if !errors.Is(err, i18n.ErrUnsupportedLanguage) {
		t.Fatalf("error = %v, want ErrUnsupportedLanguage", err)
	}
	requireNoSecretFragment(t, err.Error(), received)
}

func requireNoSecretFragment(tb testing.TB, msg, secret string) {
	tb.Helper()
	for start := 0; start+echoFragmentBytes <= len(secret); start++ {
		fragment := secret[start : start+echoFragmentBytes]
		if strings.Contains(msg, fragment) {
			tb.Fatalf("message %q echoes %q from the received text", msg, fragment)
		}
	}
}
