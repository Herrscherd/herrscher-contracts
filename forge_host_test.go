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
		{"forge.example.com:1", "forge.example.com:1"},
		{"forge.example.com:65535", "forge.example.com:65535"},
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
		"githuK.com",
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

func TestParseForgeHostRefusesANonASCIICharacterThatLowercasesIntoASCII(t *testing.T) {
	const kelvinSign = "K"
	if !strings.EqualFold(kelvinSign, "k") {
		t.Fatalf("fixture: the Kelvin sign no longer lowercases to k")
	}
	if _, err := ParseForgeHost("githu" + kelvinSign + ".com"); !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("ParseForgeHost(Kelvin sign) error = %v, want ErrInvalidForgeHost", err)
	}
}

func TestParseForgeHostPinsTheLabelLimit(t *testing.T) {
	if _, err := ParseForgeHost(strings.Repeat("a", 63) + ".com"); err != nil {
		t.Fatalf("ParseForgeHost(label of 63 bytes) error = %v, want none", err)
	}
	if _, err := ParseForgeHost(strings.Repeat("a", 64) + ".com"); !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("ParseForgeHost(label of 64 bytes) error = %v, want ErrInvalidForgeHost", err)
	}
}

func TestParseForgeHostPinsTheTotalLimitWithAndWithoutThePort(t *testing.T) {
	atLimit := hostOfLength(253)
	if len(atLimit) != 253 {
		t.Fatalf("fixture is %d bytes, want 253", len(atLimit))
	}
	if _, err := ParseForgeHost(atLimit); err != nil {
		t.Fatalf("ParseForgeHost(253 bytes) error = %v, want none", err)
	}
	if _, err := ParseForgeHost(hostOfLength(254)); !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("ParseForgeHost(254 bytes) error = %v, want ErrInvalidForgeHost", err)
	}
	if _, err := ParseForgeHost(hostOfLength(247) + ":65535"); err != nil {
		t.Fatalf("ParseForgeHost(253 bytes with port) error = %v, want none", err)
	}
	if _, err := ParseForgeHost(hostOfLength(248) + ":65535"); !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("ParseForgeHost(254 bytes with port) error = %v, want ErrInvalidForgeHost", err)
	}
}

func TestParseForgeHostPinsThePortRange(t *testing.T) {
	if _, err := ParseForgeHost("forge.example.com:65535"); err != nil {
		t.Fatalf("ParseForgeHost(port 65535) error = %v, want none", err)
	}
	if _, err := ParseForgeHost("forge.example.com:65536"); !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("ParseForgeHost(port 65536) error = %v, want ErrInvalidForgeHost", err)
	}
}

func TestParseForgeHostQuotesAtMostTheFirstSixtyFourBytes(t *testing.T) {
	long := strings.Repeat("x", 200) + "!"
	_, err := ParseForgeHost(long)
	if !errors.Is(err, ErrInvalidForgeHost) {
		t.Fatalf("ParseForgeHost(long) error = %v, want ErrInvalidForgeHost", err)
	}
	if strings.Contains(err.Error(), long[:65]) {
		t.Fatalf("error quotes more than 64 bytes of the input: %d bytes", len(err.Error()))
	}
	if !strings.Contains(err.Error(), long[:64]) {
		t.Fatalf("error = %q, want the first 64 bytes of the input", err.Error())
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

func TestSSHPortIsDroppedAndTheWebPortKept(t *testing.T) {
	ssh, err := ParseRepoRef("ssh://git@github.com:22/octo/hello")
	if err != nil {
		t.Fatalf("ParseRepoRef(ssh with port) error = %v, want none", err)
	}
	if ssh.Host() != mustParseForgeHost(t, "github.com") {
		t.Fatalf("ssh host = %q, want github.com: the port is the SSH service's", ssh.Host())
	}
	web, err := ParseRepoRef("https://github.com:443/octo/hello")
	if err != nil {
		t.Fatalf("ParseRepoRef(https with default port) error = %v, want none", err)
	}
	if web.Host() != mustParseForgeHost(t, "github.com") {
		t.Fatalf("https host = %q, want github.com: 443 is the default port", web.Host())
	}
}

func hostOfLength(length int) string {
	const label, dots = 63, 3
	last := length - 3*label - dots
	return strings.Repeat("a", label) + "." + strings.Repeat("b", label) + "." +
		strings.Repeat("c", label) + "." + strings.Repeat("d", last)
}

func mustParseForgeHost(t *testing.T, text string) ForgeHost {
	t.Helper()
	host, err := ParseForgeHost(text)
	if err != nil {
		t.Fatalf("ParseForgeHost(%q) error = %v", text, err)
	}
	return host
}
