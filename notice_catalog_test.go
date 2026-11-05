package contracts

import (
	"maps"
	"slices"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/i18n/i18ntest"
)

func declaredParams() map[string][]string {
	declared := make(map[string][]string, len(Codes()))
	for _, code := range Codes() {
		kinds, _ := code.params()
		declared[code.Key()] = slices.Sorted(maps.Keys(kinds))
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
