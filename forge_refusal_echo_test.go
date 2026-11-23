package contracts

import (
	"errors"
	"strings"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/echotest"
)

func TestParseForgeHostRefusalsDoNotEchoTheirInput(t *testing.T) {
	label := strings.Repeat("a", 60)
	tests := []struct {
		name string
		text string
	}{
		{"nonASCII", receivedProbe + "é.example.com"},
		{"port", "forge.example.com:" + receivedProbe},
		{"length", strings.Join([]string{receivedProbe, label, label, label, label, "com"}, ".")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseForgeHost(tt.text)
			if !errors.Is(err, ErrInvalidForgeHost) {
				t.Fatalf("error = %v, want ErrInvalidForgeHost", err)
			}
			echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
		})
	}
}

func TestParseRepoRefRefusalsDoNotEchoTheirInput(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"reservedQuery", "github.com/octo/" + receivedProbe + "?x"},
		{"reservedLength", "github.com/octo/" + receivedProbe + strings.Repeat("a", maxRepoRefBytes)},
		{"hostError", receivedProbe + "/octo/hello"},
		{"path", "github.com/" + receivedProbe},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRepoRef(tt.text)
			if !errors.Is(err, ErrInvalidRepoRef) {
				t.Fatalf("error = %v, want ErrInvalidRepoRef", err)
			}
			echotest.RequireNoSecretFragment(t, err.Error(), receivedProbe)
		})
	}
}
