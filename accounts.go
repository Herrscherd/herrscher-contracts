package contracts

import (
	"errors"
	"fmt"
)

// Vendor names the vendor CLI a backend runs. It equals the Manifest.Kind of the backend plugin.
const (
	VendorClaude = "claude"
	VendorCodex  = "codex"
	VendorCursor = "cursor"
)

const (
	// EnvClaudeConfigDir points the claude CLI at the directory holding its login and conversations.
	EnvClaudeConfigDir = "CLAUDE_CONFIG_DIR"
	// EnvCodexHome points the codex CLI at the directory holding its login and config.toml.
	EnvCodexHome = "CODEX_HOME"
)

// NativeProfile names the implicit profile: the login already present on the machine.
const NativeProfile = "native"

const maxProfileNameLength = 32

var (
	// ErrProfileName reports a profile name outside the allowed shape or equal to NativeProfile.
	ErrProfileName = errors.New("account profile name must be 1 to 32 characters of a-z, 0-9, '-' or '_', start with a letter or a digit, and not be \"native\"")
	// ErrNoProfileHome reports a vendor whose CLI has no reliable variable to relocate its login.
	ErrNoProfileHome = errors.New("vendor has no reliable home variable, so it has no account profiles")
	// ErrProfileHomeEmpty reports a profile whose directory is blank.
	ErrProfileHomeEmpty = errors.New("account profile directory is empty")
)

// ProfileName is a validated account profile name. Build one with ParseProfileName.
type ProfileName string

// ParseProfileName validates a profile name, wrapping [ErrProfileName] when it is refused.
func ParseProfileName(name string) (ProfileName, error) {
	if !validProfileName(name) {
		return "", fmt.Errorf("%q: %w", name, ErrProfileName)
	}
	return ProfileName(name), nil
}

func validProfileName(name string) bool {
	if name == NativeProfile || name == "" || len(name) > maxProfileNameLength {
		return false
	}
	for index := range len(name) {
		if !profileNameByte(name[index], index == 0) {
			return false
		}
	}
	return true
}

func profileNameByte(character byte, first bool) bool {
	lowerOrDigit := (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9')
	if first {
		return lowerOrDigit
	}
	return lowerOrDigit || character == '-' || character == '_'
}

// ProfileHomeEnv returns the variable a vendor CLI reads to find its login directory.
func ProfileHomeEnv(vendor string) (string, error) {
	switch vendor {
	case VendorClaude:
		return EnvClaudeConfigDir, nil
	case VendorCodex:
		return EnvCodexHome, nil
	}
	return "", fmt.Errorf("vendor %q: %w", vendor, ErrNoProfileHome)
}

// ProfileEnv returns the single variable that points a vendor CLI at the profile directory home.
func ProfileEnv(vendor, home string) (map[string]string, error) {
	name, err := ProfileHomeEnv(vendor)
	if err != nil {
		return nil, err
	}
	if home == "" {
		return nil, ErrProfileHomeEmpty
	}
	return map[string]string{name: home}, nil
}
