package contracts

import "testing"

func TestNormalizeScopeAgreesWithProjectKey(t *testing.T) {
	for _, name := range []string{"Neublox", "herrscher docs", "  A/B  ", "é"} {
		if got, want := "projects/"+NormalizeScope(name), ProjectKey(name); got != want {
			t.Fatalf("NormalizeScope(%q) → %q, ProjectKey → %q", name, got, want)
		}
	}
}
