package contracts

import (
	"maps"
	"slices"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/i18n/i18ntest"
)

func declaredParams(t *testing.T) map[string][]string {
	t.Helper()
	declared := make(map[string][]string, len(Codes()))
	for _, code := range Codes() {
		kinds, known := code.params()
		if !known {
			t.Fatalf("code %q has no declared params", code.Key())
		}
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
	i18ntest.CheckDeclared(t, declaredParams(t), sources)
}
