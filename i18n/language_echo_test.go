package i18n_test

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
	"github.com/Herrscherd/herrscher-contracts/i18n"
)

func TestParseLanguageRefusesAnUnsupportedTextWithoutEcho(t *testing.T) {
	_, err := i18n.ParseLanguage(echotest.Probe + "_XX.UTF-8")
	if !errors.Is(err, i18n.ErrUnsupportedLanguage) {
		t.Fatalf("error = %v, want ErrUnsupportedLanguage", err)
	}
	echotest.RequireNoSecretFragment(t, err.Error(), echotest.Probe)
}
