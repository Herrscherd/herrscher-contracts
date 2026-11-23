package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseForgeHostAcceptsItsForms(t *testing.T) {
	tests := []struct{ text, want string }{
		{"github.com", "github.com"},
		{"GitHub.COM", "github.com"},
		{"forge.example.com:8443", "forge.example.com:8443"},
		{"forge.example.com:0443", "forge.example.com:443"},
		{"a-b.example.org:65535", "a-b.example.org:65535"},
		{"10.0.0.1", "10.0.0.1"},
	}
	for _, tt := range tests {
		host, err := ParseForgeHost(tt.text)
		if err != nil {
			t.Errorf("ParseForgeHost(%q) error = %v, want none", tt.text, err)
			continue
		}
		if host.String() != tt.want {
			t.Errorf("ParseForgeHost(%q) = %q, want %q", tt.text, host.String(), tt.want)
		}
	}
}

func TestParseForgeHostRefusesOtherText(t *testing.T) {
	texts := []string{
		"",
		"github",
		".github.com",
		"github.com.",
		"github..com",
		"..",
		"git-.example.com",
		"-git.example.com",
		"git..example.com:",
		"bad_host.example.com",
		"user@github.com",
		"github.com:",
		"github.com:/",
		"github.com:0",
		"github.com:65536",
		"github.com:+443",
		"github.com:-443",
		"github.com:4a3",
		"github.com:1:2",
		"github.com/octo",
		"héllo.example.com",
		"https://github.com",
		strings.Repeat("a", 64) + ".com",
		strings.Repeat("a.", 127) + "com",
	}
	for _, text := range texts {
		host, err := ParseForgeHost(text)
		if !errors.Is(err, ErrInvalidForgeHost) {
			t.Errorf("ParseForgeHost(%q) error = %v, want ErrInvalidForgeHost", text, err)
		}
		if host != (ForgeHost{}) {
			t.Errorf("ParseForgeHost(%q) = %q, want the zero value on error", text, host)
		}
	}
}

func TestParseForgeHostAcceptsTheLabelAndTotalLimits(t *testing.T) {
	longestLabel := strings.Repeat("a", maxDNSLabelBytes) + ".com"
	if _, err := ParseForgeHost(longestLabel); err != nil {
		t.Fatalf("ParseForgeHost(label of %d bytes) error = %v, want none", maxDNSLabelBytes, err)
	}
	longestTotal := strings.Repeat("a", 60) + "." + strings.Repeat("b", 60) + "." + strings.Repeat("c", 60) + "." + strings.Repeat("d", 60)
	if len(longestTotal) > maxForgeHostBytes {
		t.Fatalf("fixture is %d bytes, want at most %d", len(longestTotal), maxForgeHostBytes)
	}
	if _, err := ParseForgeHost(longestTotal); err != nil {
		t.Fatalf("ParseForgeHost(%d bytes) error = %v, want none", len(longestTotal), err)
	}
	overTotal := longestTotal + ".example.com"
	if _, err := ParseForgeHost(overTotal); !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("ParseForgeHost(%d bytes) error = %v, want ErrInvalidForgeHost", len(overTotal), err)
	}
}

func TestParseForgeHostPortRoundTrips(t *testing.T) {
	host := mustParseForgeHost(t, "forge.example.com:0443")
	back := mustParseForgeHost(t, host.String())
	if back != host || back.String() != "forge.example.com:443" {
		t.Fatalf("round trip = %q, want the canonical form forge.example.com:443", back)
	}
}

func TestZeroForgeHostStringIsEmpty(t *testing.T) {
	if got := (ForgeHost{}).String(); got != "" {
		t.Fatalf("zero ForgeHost String() = %q, want empty", got)
	}
}

func mustParseForgeHost(t *testing.T, text string) ForgeHost {
	t.Helper()
	host, err := ParseForgeHost(text)
	if err != nil {
		t.Fatalf("ParseForgeHost(%q) error = %v", text, err)
	}
	return host
}
