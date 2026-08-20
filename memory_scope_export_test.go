package contracts

import "testing"

// NormalizeScope exists so a host deriving a scope name from a directory or a
// prompt produces exactly the segment ProjectKey would, and never splits one
// scope into two vault files.
func TestNormalizeScopeAgreesWithProjectKey(t *testing.T) {
	for _, name := range []string{"Neublox", "herrscher docs", "  A/B  ", "é"} {
		if got, want := "projects/"+NormalizeScope(name), ProjectKey(name); got != want {
			t.Fatalf("NormalizeScope(%q) → %q, ProjectKey → %q", name, got, want)
		}
	}
}
