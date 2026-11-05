package contracts

import (
	"testing"

	"github.com/Herrscherd/herrscher-contracts/i18n/i18ntest"
)

func declaredParams() map[string][]string {
	declared := make(map[string][]string, len(Codes()))
	for _, code := range Codes() {
		names, _ := code.params()
		declared[code.Key()] = names
	}
	return declared
}

func TestSharedCatalogIsCompleteAndMatchesCodes(t *testing.T) {
	sources, err := sharedSources()
	if err != nil {
		t.Fatal(err)
	}
	i18ntest.Check(t, sources)
	i18ntest.CheckDeclared(t, declaredParams(), sources)
}
