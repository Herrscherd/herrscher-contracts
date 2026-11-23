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
		{"https://github.com/octo/hello", "github.com", "octo", "hello"},
		{"https://github.com/octo/hello.git", "github.com", "octo", "hello"},
		{"https://github.com/octo/hello/", "github.com", "octo", "hello"},
		{"https://github.com:8443/octo/hello", "github.com:8443", "octo", "hello"},
		{"git@github.com:octo/hello.git", "github.com", "octo", "hello"},
		{"git@forge.example.com:team/app_v2", "forge.example.com", "team", "app_v2"},
		{"ssh://git@github.com/octo/hello.git", "github.com", "octo", "hello"},
		{"ssh://git@forge.example.com:2222/team/app.git", "forge.example.com:2222", "team", "app"},
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
		"github.com/octo/-hello",
		"github.com/octo/.hello",
		"github.com/octo/he llo",
		"github.com/octo/héllo",
		"github.com/octo/hello?tab=1",
		"github.com/octo/hello?",
		"github.com/octo/hello#",
		"github.com/octo/hello#frag",
		"https://github.com/octo/hello?x=1",
		"https://github.com/octo/hello#frag",
		"https://github.com/octo/hello?",
		"github.com:/octo/hello",
		"https://github.com:/octo/hello",
		"github.com/%6Fcto/hello",
		"http://github.com/octo/hello",
		"https://user@github.com/octo/hello",
		"https://github.com/octo",
		"ssh://github.com/octo/hello",
		"ssh://bob@github.com/octo/hello.git",
		"git@github.com/octo/hello",
		"git@github.com:octo",
		"git@github.com:octo/hello/",
		"git@github.com:octo/.git",
		"bad_host/octo/hello",
		"github/octo/hello",
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

func TestParseRepoRefRefusesTheNameLimit(t *testing.T) {
	longest := "github.com/octo/" + strings.Repeat("n", maxRepoNameBytes)
	if _, err := ParseRepoRef(longest); err != nil {
		t.Fatalf("ParseRepoRef(name of %d bytes) error = %v, want none", maxRepoNameBytes, err)
	}
	tooLong := "github.com/octo/" + strings.Repeat("n", maxRepoNameBytes+1)
	if _, err := ParseRepoRef(tooLong); !errors.Is(err, ErrInvalidRepoRef) {
		t.Fatalf("ParseRepoRef(name of %d bytes) error = %v, want ErrInvalidRepoRef", maxRepoNameBytes+1, err)
	}
}

func TestParseRepoRefRefusesTheInputLimit(t *testing.T) {
	const prefix, suffix = "github.com/", "/hello"
	atLimit := prefix + strings.Repeat("a", maxRepoRefBytes-len(prefix)-len(suffix)) + suffix
	if _, err := ParseRepoRef(atLimit); err != nil {
		t.Fatalf("ParseRepoRef(%d bytes) error = %v, want none", len(atLimit), err)
	}
	overLimit := prefix + strings.Repeat("a", maxRepoRefBytes+1-len(prefix)-len(suffix)) + suffix
	if _, err := ParseRepoRef(overLimit); !errors.Is(err, ErrInvalidRepoRef) {
		t.Fatalf("ParseRepoRef(%d bytes) error = %v, want ErrInvalidRepoRef", len(overLimit), err)
	}
}

func TestParseRepoRefQuotesAtMostTheFirstSixtyFourBytes(t *testing.T) {
	long := "github.com/octo/" + strings.Repeat("x", 600) + "?"
	_, err := ParseRepoRef(long)
	if !errors.Is(err, ErrInvalidRepoRef) {
		t.Fatalf("ParseRepoRef(long) error = %v, want ErrInvalidRepoRef", err)
	}
	if strings.Contains(err.Error(), long[:maxQuotedBytes+1]) {
		t.Fatalf("error quotes more than %d bytes of the input: %d bytes", maxQuotedBytes, len(err.Error()))
	}
	if !strings.Contains(err.Error(), long[:maxQuotedBytes]) {
		t.Fatalf("error = %q, want the first %d bytes of the input", err.Error(), maxQuotedBytes)
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

func mustOctoHelloRepo(t *testing.T) ForgeRepoRef {
	t.Helper()
	ref, err := ParseRepoRef("github.com/octo/hello")
	if err != nil {
		t.Fatalf("ParseRepoRef(github.com/octo/hello) error = %v", err)
	}
	return ref
}
