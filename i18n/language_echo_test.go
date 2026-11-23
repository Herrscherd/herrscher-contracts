package i18n_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

func TestParseLanguageRefusesAnUnsupportedTextWithoutEcho(t *testing.T) {
	const received = "s3cret-received-text"
	_, err := i18n.ParseLanguage(received + "_XX.UTF-8")
	if !errors.Is(err, i18n.ErrUnsupportedLanguage) {
		t.Fatalf("error = %v, want ErrUnsupportedLanguage", err)
	}
	if strings.Contains(err.Error(), received) {
		t.Fatalf("message %q echoes the received text", err)
	}
}
