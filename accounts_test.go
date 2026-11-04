package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseProfileName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "plain", input: "perso"},
		{name: "digits and separators", input: "work-2_b"},
		{name: "single digit", input: "7"},
		{name: "longest allowed", input: strings.Repeat("a", 32)},
		{name: "empty", input: "", wantErr: true},
		{name: "too long", input: strings.Repeat("a", 33), wantErr: true},
		{name: "reserved native", input: NativeProfile, wantErr: true},
		{name: "leading dash", input: "-work", wantErr: true},
		{name: "leading underscore", input: "_work", wantErr: true},
		{name: "upper case", input: "Work", wantErr: true},
		{name: "path separator", input: "../work", wantErr: true},
		{name: "space", input: "my work", wantErr: true},
		{name: "non ascii", input: "trabajé", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseProfileName(tt.input)
			if tt.wantErr {
				if !errors.Is(err, ErrProfileName) {
					t.Fatalf("ParseProfileName(%q) error = %v, want ErrProfileName", tt.input, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseProfileName(%q): %v", tt.input, err)
			}
			if string(got) != tt.input {
				t.Fatalf("ParseProfileName(%q) = %q", tt.input, got)
			}
		})
	}
}

func TestProfileHomeEnvPerVendor(t *testing.T) {
	tests := []struct {
		vendor  string
		want    string
		wantErr bool
	}{
		{vendor: VendorClaude, want: "CLAUDE_CONFIG_DIR"},
		{vendor: VendorCodex, want: "CODEX_HOME"},
		{vendor: VendorCursor, wantErr: true},
		{vendor: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.vendor, func(t *testing.T) {
			got, err := ProfileHomeEnv(tt.vendor)
			if tt.wantErr {
				if !errors.Is(err, ErrNoProfileHome) {
					t.Fatalf("ProfileHomeEnv(%q) error = %v, want ErrNoProfileHome", tt.vendor, err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("ProfileHomeEnv(%q) = %q, %v; want %q", tt.vendor, got, err, tt.want)
			}
		})
	}
}

func TestProfileEnvSetsOnlyTheVendorVariable(t *testing.T) {
	env, err := ProfileEnv(VendorCodex, "/home/shan/.config/herrscher/accounts/codex/work")
	if err != nil {
		t.Fatalf("ProfileEnv: %v", err)
	}
	want := map[string]string{"CODEX_HOME": "/home/shan/.config/herrscher/accounts/codex/work"}
	if len(env) != len(want) || env["CODEX_HOME"] != want["CODEX_HOME"] {
		t.Fatalf("ProfileEnv = %v, want %v", env, want)
	}
}

func TestProfileEnvRefusesBlankHomeAndNoHomeVendor(t *testing.T) {
	if _, err := ProfileEnv(VendorClaude, ""); !errors.Is(err, ErrProfileHomeEmpty) {
		t.Fatalf("blank home error = %v, want ErrProfileHomeEmpty", err)
	}
	if _, err := ProfileEnv(VendorCursor, "/tmp/x"); !errors.Is(err, ErrNoProfileHome) {
		t.Fatalf("cursor error = %v, want ErrNoProfileHome", err)
	}
}
