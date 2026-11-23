package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseRepoRefAcceptsEachForm(t *testing.T) {
	tests := []struct {
		text, host, namespace, name string
	}{
		{"github.com/octo/hello", "github.com", "octo", "hello"},
		{"gitlab.example.com/team/platform/app", "gitlab.example.com", "team/platform", "app"},
		{"GitHub.com/octo/hello", "github.com", "octo", "hello"},
		{"github.com/octo/hello/", "github.com", "octo", "hello"},
		{"https://github.com/octo/hello", "github.com", "octo", "hello"},
		{"https://github.com/octo/hello.git", "github.com", "octo", "hello"},
		{"https://github.com/octo/hello/", "github.com", "octo", "hello"},
		{"https://github.com:8443/octo/hello", "github.com:8443", "octo", "hello"},
		{"https://github.com:443/octo/hello", "github.com", "octo", "hello"},
		{"https://github.com:0443/octo/hello", "github.com", "octo", "hello"},
		{"https://github.com:80/octo/hello", "github.com:80", "octo", "hello"},
		{"git@github.com:octo/hello.git", "github.com", "octo", "hello"},
		{"git@forge.example.com:team/app_v2", "forge.example.com", "team", "app_v2"},
		{"ssh://git@github.com/octo/hello.git", "github.com", "octo", "hello"},
		{"ssh://git@github.com:22/octo/hello", "github.com", "octo", "hello"},
		{"ssh://git@forge.example.com:2222/team/app.git", "forge.example.com", "team", "app"},
	}
	for _, tt := range tests {
		ref, err := ParseRepoRef(tt.text)
		if err != nil {
			t.Errorf("ParseRepoRef(%q) error = %v, want none", tt.text, err)
			continue
		}
		if ref.Host().String() != tt.host || ref.Namespace() != tt.namespace || ref.Name() != tt.name {
			t.Errorf("ParseRepoRef(%q) = %s/%s/%s, want %s/%s/%s",
				tt.text, ref.Host(), ref.Namespace(), ref.Name(), tt.host, tt.namespace, tt.name)
		}
	}
}

func TestParseRepoRefKeepsEveryDepthItIsGiven(t *testing.T) {
	ref, err := ParseRepoRef("github.com/octo/hello/issues")
	if err != nil {
		t.Fatalf("ParseRepoRef() error = %v, want none: the depth is the plugin's to check", err)
	}
	if ref.Namespace() != "octo/hello" || ref.Name() != "issues" {
		t.Fatalf("ParseRepoRef() = %q/%q, want the deeper path split at its last segment", ref.Namespace(), ref.Name())
	}
}

func TestParseRepoRefRefusesOtherForms(t *testing.T) {
	texts := []string{
		"",
		"hello",
		"octo/",
		"/hello",
		"octo/hello",
		"octo/hello/",
		"Herrscherd/herrscher/issues",
		"a/b/c/d",
		"github.com/octo/../hello",
		"github.com/octo/./hello",
		"github.com//hello",
		"github.com/octo//",
		"github.com/octo/-hello",
		"github.com/octo/.hello",
		"github.com/octo/he llo",
		"github.com/octo/héllo",
		"github.com/octo/hello?tab=1",
		"github.com/octo/hello?",
		"github.com/octo/hello#",
		"github.com/octo/hello#frag",
		"github.com/octo/hel@lo",
		"github.com/octo/hello@1",
		"@github.com/octo/hello",
		"https://github.com/octo/hello?x=1",
		"https://github.com/octo/hello#frag",
		"https://github.com/octo/hello?",
		"https://github.com:443:1/octo/hello",
		"https://github.com:/octo/hello",
		"github.com:/octo/hello",
		"github.com:octo/hello",
		"forge.example.com:octo/hello",
		"github.com/%6Fcto/hello",
		"http://github.com/octo/hello",
		"https://user@github.com/octo/hello",
		"https://user:pass@github.com/octo/hello",
		"https://github.com/octo",
		"ssh://github.com/octo/hello",
		"ssh://github.com:22/octo/hello",
		"ssh://bob@github.com/octo/hello.git",
		"ssh://git@github.com:/octo/hello",
		"ssh://git@github.com:x/octo/hello",
		"ssh://git@github.com:0/octo/hello",
		"ssh://git@github.com:65536/octo/hello",
		"git@github.com/octo/hello",
		"git@github.com:octo",
		"git@github.com:octo/hello/",
		"git@github.com:octo/.git",
		"git@github.com:octo/hel@lo",
		"bad_host/octo/hello",
		"github/octo/hello",
		"githuK.com/octo/hello",
	}
	for _, text := range texts {
		ref, err := ParseRepoRef(text)
		if !errors.Is(err, ErrInvalidRepoRef) {
			t.Errorf("ParseRepoRef(%q) error = %v, want ErrInvalidRepoRef", text, err)
		}
		if ref != (ForgeRepoRef{}) {
			t.Errorf("ParseRepoRef(%q) = %v, want the zero value on error", text, ref)
		}
	}
}

func TestParseRepoRefPinsTheNameLimit(t *testing.T) {
	if _, err := ParseRepoRef("github.com/octo/" + strings.Repeat("n", 100)); err != nil {
		t.Fatalf("ParseRepoRef(name of 100 bytes) error = %v, want none", err)
	}
	if _, err := ParseRepoRef("github.com/octo/" + strings.Repeat("n", 101)); !errors.Is(err, ErrInvalidRepoRef) {
		t.Fatalf("ParseRepoRef(name of 101 bytes) error = %v, want ErrInvalidRepoRef", err)
	}
}

func TestParseRepoRefPinsTheInputLimit(t *testing.T) {
	const prefix, suffix = "github.com/", "/hello"
	atLimit := prefix + strings.Repeat("a", 512-len(prefix)-len(suffix)) + suffix
	if len(atLimit) != 512 {
		t.Fatalf("fixture is %d bytes, want 512", len(atLimit))
	}
	if _, err := ParseRepoRef(atLimit); err != nil {
		t.Fatalf("ParseRepoRef(512 bytes) error = %v, want none", err)
	}
	overLimit := prefix + strings.Repeat("a", 513-len(prefix)-len(suffix)) + suffix
	if len(overLimit) != 513 {
		t.Fatalf("fixture is %d bytes, want 513", len(overLimit))
	}
	if _, err := ParseRepoRef(overLimit); !errors.Is(err, ErrInvalidRepoRef) {
		t.Fatalf("ParseRepoRef(513 bytes) error = %v, want ErrInvalidRepoRef", err)
	}
}

func TestRepoRefStringRoundTripsThroughParse(t *testing.T) {
	ref, err := ParseRepoRef("https://github.com/octo/hello.git")
	if err != nil {
		t.Fatalf("ParseRepoRef() error = %v", err)
	}
	back, err := ParseRepoRef(ref.String())
	if err != nil || back != ref {
		t.Fatalf("ParseRepoRef(%q) = %v, %v, want %v", ref.String(), back, err, ref)
	}
}

func TestZeroRepoRefStringIsEmpty(t *testing.T) {
	if got := (ForgeRepoRef{}).String(); got != "" {
		t.Fatalf("zero ForgeRepoRef String() = %q, want empty", got)
	}
}

func TestRequireRepoHostRefusesAnotherHost(t *testing.T) {
	github := mustParseForgeHost(t, "github.com")
	gitlab := mustParseForgeHost(t, "gitlab.example.com")
	ref := mustOctoHelloRepo(t)

	if err := RequireRepoHost(github, ref); err != nil {
		t.Fatalf("RequireRepoHost(same host) error = %v, want none", err)
	}
	if err := RequireRepoHost(gitlab, ref); !errors.Is(err, ErrForgeWrongHost) {
		t.Fatalf("RequireRepoHost(other host) error = %v, want ErrForgeWrongHost", err)
	}
}

func TestRequireRepoHostRefusesTheZeroValues(t *testing.T) {
	github := mustParseForgeHost(t, "github.com")
	ref := mustOctoHelloRepo(t)

	if err := RequireRepoHost(ForgeHost{}, ref); !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("RequireRepoHost(zero host) error = %v, want ErrInvalidForgeHost", err)
	}
	if err := RequireRepoHost(github, ForgeRepoRef{}); !errors.Is(err, ErrInvalidRepoRef) {
		t.Fatalf("RequireRepoHost(zero repo) error = %v, want ErrInvalidRepoRef", err)
	}
	if err := RequireRepoHost(ForgeHost{}, ForgeRepoRef{}); !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("RequireRepoHost(zero host, zero repo) error = %v, want ErrInvalidForgeHost", err)
	}
}

func TestSameRepoIgnoresCaseOnNamespaceAndName(t *testing.T) {
	lower := mustParseRepo(t, "github.com/octo/hello")
	mixed := mustParseRepo(t, "GitHub.com/Octo/HELLO")
	if !lower.SameRepo(mixed) || !mixed.SameRepo(lower) {
		t.Fatalf("SameRepo(%v, %v) = false, want true: the host is lowercased and the rest compares without case", lower, mixed)
	}
}

func TestSameRepoRefusesAnotherHostOrRepository(t *testing.T) {
	base := mustParseRepo(t, "github.com/octo/hello")
	for _, other := range []string{"gitlab.example.com/octo/hello", "github.com/octo/world", "github.com/other/hello"} {
		if base.SameRepo(mustParseRepo(t, other)) {
			t.Errorf("SameRepo(%v, %s) = true, want false", base, other)
		}
	}
	if !(ForgeRepoRef{}).SameRepo(ForgeRepoRef{}) {
		t.Errorf("SameRepo of two zero references = false, want true")
	}
}

func TestEqualityOfRepoRefIsCaseSensitive(t *testing.T) {
	lower := mustParseRepo(t, "github.com/octo/hello")
	mixed := mustParseRepo(t, "github.com/Octo/hello")
	if lower == mixed {
		t.Fatalf("%v == %v, want false: == is case-sensitive on the namespace and name", lower, mixed)
	}
}

func mustOctoHelloRepo(t *testing.T) ForgeRepoRef {
	t.Helper()
	return mustParseRepo(t, "github.com/octo/hello")
}

func mustParseRepo(t *testing.T, text string) ForgeRepoRef {
	t.Helper()
	ref, err := ParseRepoRef(text)
	if err != nil {
		t.Fatalf("ParseRepoRef(%q) error = %v", text, err)
	}
	return ref
}

func TestParseRepoRefRefusalRepeatsNoPartOfTheText(t *testing.T) {
	texts := map[string]string{
		"too long":     receivedSecret + strings.Repeat("/x", 300),
		"query":        receivedSecret + "/octo/hello?",
		"no path":      receivedSecret,
		"refused host": receivedSecret + "!/octo/hello",
		"refused name": "github.com/octo/" + receivedSecret + "!",
	}
	for name, text := range texts {
		t.Run(name, func(t *testing.T) {
			_, err := ParseRepoRef(text)
			requireBareSentinel(t, err, ErrInvalidRepoRef)
		})
	}
}
