package contracts

import (
	"strings"
	"testing"
)

func TestMergeEnvAppendsPairs(t *testing.T) {
	got := MergeEnv([]string{"PATH=/bin", "HOME=/home/x"}, map[string]string{"ANTHROPIC_BASE_URL": "https://gw"})
	if !hasEntry(got, "ANTHROPIC_BASE_URL=https://gw") {
		t.Fatalf("MergeEnv dropped the injected pair: %v", got)
	}
	if !hasEntry(got, "PATH=/bin") {
		t.Fatalf("MergeEnv dropped an inherited pair: %v", got)
	}
}

func TestMergeEnvOverridesInherited(t *testing.T) {
	// Une variable injectée doit gagner sur celle héritée du daemon : sinon un
	// ANTHROPIC_BASE_URL traînant dans l'environnement du daemon détournerait
	// silencieusement une session. Et exec.Cmd a un comportement dépendant de la
	// plateforme quand une clé apparaît deux fois.
	got := MergeEnv([]string{"ANTHROPIC_BASE_URL=https://stale"}, map[string]string{"ANTHROPIC_BASE_URL": "https://gw"})
	var n int
	for _, e := range got {
		if strings.HasPrefix(e, "ANTHROPIC_BASE_URL=") {
			n++
			if e != "ANTHROPIC_BASE_URL=https://gw" {
				t.Fatalf("inherited value won: %q", e)
			}
		}
	}
	if n != 1 {
		t.Fatalf("ANTHROPIC_BASE_URL appears %d times, want exactly 1: %v", n, got)
	}
}

func TestMergeEnvNilIsIdentity(t *testing.T) {
	// Non-régression de la route native : sans injection, l'environnement produit
	// est exactement celui reçu.
	base := []string{"PATH=/bin", "HOME=/home/x"}
	got := MergeEnv(base, nil)
	if len(got) != 2 || got[0] != "PATH=/bin" || got[1] != "HOME=/home/x" {
		t.Fatalf("MergeEnv with no injection changed the environment: %v", got)
	}
}

func TestParseEnvSetting(t *testing.T) {
	got := ParseEnvSetting("A=1\nB=two\n")
	if len(got) != 2 || got["A"] != "1" || got["B"] != "two" {
		t.Fatalf("ParseEnvSetting = %+v", got)
	}
	if got := ParseEnvSetting(""); len(got) != 0 {
		t.Fatalf("ParseEnvSetting(\"\") = %+v, want empty", got)
	}
}

func TestParseEnvSettingKeepsEqualsInValue(t *testing.T) {
	// Les jetons sont souvent en base64 et contiennent des '='. Seul le premier
	// sépare la clé de la valeur.
	if got := ParseEnvSetting("T=a=b=c"); got["T"] != "a=b=c" {
		t.Fatalf("ParseEnvSetting mangled a value containing '=': %q", got["T"])
	}
}

func TestEncodeEnvSettingIsDeterministic(t *testing.T) {
	// Les clés sont triées : sans ça la valeur change à chaque appel, donc elle
	// n'est pas testable et le réglage transporté diffère entre deux spawns
	// identiques.
	env := map[string]string{"B": "2", "A": "1", "C": "3"}
	first := EncodeEnvSetting(env)
	for i := 0; i < 20; i++ {
		if EncodeEnvSetting(env) != first {
			t.Fatalf("EncodeEnvSetting is not deterministic")
		}
	}
	if first != "A=1\nB=2\nC=3\n" {
		t.Fatalf("EncodeEnvSetting = %q", first)
	}
}

func TestEncodeEnvSettingEmptyIsEmpty(t *testing.T) {
	if EncodeEnvSetting(nil) != "" || EncodeEnvSetting(map[string]string{}) != "" {
		t.Fatal("EncodeEnvSetting should render nothing for an empty map")
	}
}

func TestEncodeParseRoundTrip(t *testing.T) {
	// LE test qui justifie que ces deux fonctions soient dans le même dépôt :
	// le host encode, le backend décode, et rien d'autre ne relie les deux.
	want := map[string]string{
		"ANTHROPIC_BASE_URL":   "https://gw.example/v1",
		"ANTHROPIC_AUTH_TOKEN": "sk-abc=def==",
		"ANTHROPIC_MODEL":      "glm-4.7",
	}
	got := ParseEnvSetting(EncodeEnvSetting(want))
	if len(got) != len(want) {
		t.Fatalf("round-trip lost entries: %+v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("round-trip changed %q: %q != %q", k, got[k], v)
		}
	}
}

func hasEntry(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
