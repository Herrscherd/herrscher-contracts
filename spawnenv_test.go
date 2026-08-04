package contracts

import (
	"strings"
	"testing"
)

// TestGatewayEnvKeyNames pins the wire names themselves. The constants exist
// so the host and the backends stop spelling four magic strings independently,
// but a constant is only a contract if its VALUE is pinned: the vendor CLIs
// are third-party binaries that recognise these exact names, so changing one
// here would not break compilation anywhere while silently disabling the
// gateway route at run time.
func TestGatewayEnvKeyNames(t *testing.T) {
	for _, tc := range []struct{ got, want string }{
		{EnvAnthropicBaseURL, "ANTHROPIC_BASE_URL"},
		{EnvAnthropicAuthToken, "ANTHROPIC_AUTH_TOKEN"},
		{EnvOpenAIBaseURL, "OPENAI_BASE_URL"},
		{EnvNeubloxToken, "NEUBLOX_TOKEN"},
	} {
		if tc.got != tc.want {
			t.Errorf("gateway env key = %q, want %q (the vendor CLI reads this exact name)", tc.got, tc.want)
		}
	}
}

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
	// An injected variable must win over the one inherited from the daemon:
	// otherwise a stale ANTHROPIC_BASE_URL lingering in the daemon's
	// environment would silently hijack a session. And exec.Cmd has
	// platform-dependent behavior when a key appears twice.
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
	// Non-regression for the native route: with no injection, the produced
	// environment is exactly the one received.
	base := []string{"PATH=/bin", "HOME=/home/x"}
	got := MergeEnv(base, nil)
	if len(got) != 2 || got[0] != "PATH=/bin" || got[1] != "HOME=/home/x" {
		t.Fatalf("MergeEnv with no injection changed the environment: %v", got)
	}
}

// This is the shape the native route actually produces: ParseEnvSetting("")
// returns a non-nil empty map, not nil. Identity must hold for it too.
func TestMergeEnvEmptyMapIsIdentity(t *testing.T) {
	base := []string{"PATH=/bin", "HOME=/home/x"}
	got := MergeEnv(base, map[string]string{})
	if len(got) != 2 || got[0] != "PATH=/bin" || got[1] != "HOME=/home/x" {
		t.Fatalf("MergeEnv with an empty map changed the environment: %v", got)
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
	// Tokens are often base64-encoded and contain '='. Only the first one
	// separates the key from the value.
	if got := ParseEnvSetting("T=a=b=c"); got["T"] != "a=b=c" {
		t.Fatalf("ParseEnvSetting mangled a value containing '=': %q", got["T"])
	}
}

func TestEncodeEnvSettingIsDeterministic(t *testing.T) {
	// Keys are sorted: without that the value changes on every call, so it is
	// not testable and the transported setting differs between two identical
	// spawns.
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
	// THE test that justifies these two functions living in the same repo:
	// the host encodes, the backend decodes, and nothing else ties the two
	// together.
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
