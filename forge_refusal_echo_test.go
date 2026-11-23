package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseForgeHostRefusalsDoNotEchoTheirInput(t *testing.T) {
	label := strings.Repeat("a", 60)
	tests := []struct {
		name string
		text string
	}{
		{"nonASCII", receivedSecret + "é.example.com"},
		{"port", "forge.example.com:" + receivedSecret},
		{"length", strings.Join([]string{receivedSecret, label, label, label, label, "com"}, ".")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseForgeHost(tt.text)
			if !errors.Is(err, ErrInvalidForgeHost) {
				t.Fatalf("error = %v, want ErrInvalidForgeHost", err)
			}
			requireNoSecretFragment(t, err.Error(), receivedSecret)
		})
	}
}

func TestParseRepoRefRefusalsDoNotEchoTheirInput(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"reservedQuery", "github.com/octo/" + receivedSecret + "?x"},
		{"reservedLength", "github.com/octo/" + receivedSecret + strings.Repeat("a", maxRepoRefBytes)},
		{"hostError", receivedSecret + "/octo/hello"},
		{"path", "github.com/" + receivedSecret},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRepoRef(tt.text)
			if !errors.Is(err, ErrInvalidRepoRef) {
				t.Fatalf("error = %v, want ErrInvalidRepoRef", err)
			}
			requireNoSecretFragment(t, err.Error(), receivedSecret)
		})
	}
}
