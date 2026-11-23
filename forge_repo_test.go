package contracts

import (
	"errors"
	"testing"
)

func TestParseRepoRefAcceptsEachForm(t *testing.T) {
	tests := []struct {
		text, host, owner, name string
	}{
		{"octo/hello", "github.com", "octo", "hello"},
		{"gitlab.example.com/team/app", "gitlab.example.com", "team", "app"},
		{"GitHub.com/octo/hello", "github.com", "octo", "hello"},
		{"https://github.com/octo/hello", "github.com", "octo", "hello"},
		{"https://github.com/octo/hello.git", "github.com", "octo", "hello"},
		{"https://github.com/octo/hello/", "github.com", "octo", "hello"},
		{"git@github.com:octo/hello.git", "github.com", "octo", "hello"},
		{"git@forge.example.com:team/app_v2", "forge.example.com", "team", "app_v2"},
	}
	for _, tt := range tests {
		ref, err := ParseRepoRef(tt.text)
		if err != nil {
			t.Errorf("ParseRepoRef(%q) error = %v, want none", tt.text, err)
			continue
		}
		if ref.Host() != tt.host || ref.Owner() != tt.owner || ref.Name() != tt.name {
			t.Errorf("ParseRepoRef(%q) = %s/%s/%s, want %s/%s/%s",
				tt.text, ref.Host(), ref.Owner(), ref.Name(), tt.host, tt.owner, tt.name)
		}
	}
}

func TestParseRepoRefRefusesOtherForms(t *testing.T) {
	texts := []string{
		"",
		"hello",
		"octo/",
		"/hello",
		"octo/hello/",
		"a/b/c/d",
		"octo/../hello",
		"octo/he llo",
		"octo/hello?tab=1",
		"http://github.com/octo/hello",
		"https://user@github.com/octo/hello",
		"https://github.com:8443/octo/hello",
		"https://github.com/octo",
		"ssh://git@github.com/octo/hello",
		"git@github.com/octo/hello",
		"git@github.com:octo",
		"bad_host/octo/hello",
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
