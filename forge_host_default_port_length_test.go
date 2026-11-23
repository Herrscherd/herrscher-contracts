package contracts

import (
	"testing"
)

func TestParseForgeHostMeasuresTheLimitOnTheCanonicalForm(t *testing.T) {
	tests := []struct {
		name, text, want string
	}{
		{"default port dropped before the limit", hostOfLength(253) + ":443", hostOfLength(253)},
		{"zero-padded port shortened before the limit", hostOfLength(247) + ":0443", hostOfLength(247)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, err := ParseForgeHost(tt.text)
			if err != nil {
				t.Fatalf("ParseForgeHost(%d bytes) error = %v, want none", len(tt.text), err)
			}
			if host.String() != tt.want {
				t.Fatalf("ParseForgeHost = %q, want %q", host.String(), tt.want)
			}
		})
	}
}
