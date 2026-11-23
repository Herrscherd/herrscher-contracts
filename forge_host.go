package contracts

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxForgeHostBytes = 253
	maxDNSLabelBytes  = 63
	maxForgePort      = 65535
	maxQuotedBytes    = 64
	defaultHTTPSPort  = 443
)

// ForgeHost names the host a forge lives on, such as "github.com" or
// "forge.example.com:8443": a lowercase DNS name of at least two labels, with an
// optional port. The port is the one of the forge's web and API service, not of its
// SSH service, so [ParseRepoRef] drops the port of an ssh:// reference. Build one
// with [ParseForgeHost]; the zero value names no host.
type ForgeHost struct{ name string }

// ParseForgeHost reads a forge host and keeps it lowercase, with its port in
// canonical digits and the default https port 443 dropped, so "github.com:443"
// and "github.com" name the same host. It returns an error matching [ErrInvalidForgeHost] when the text
// holds a byte above 127, which is checked before lowercasing so that no character
// lowercases into ASCII; when a label is empty, longer than 63 bytes, starts or ends
// with a hyphen, or holds a byte other than a letter, a digit or a hyphen; when the
// name has no dot; when the port is empty, not made of digits, or outside 1 to
// 65535; or when the whole host, port included, is longer than 253 bytes. The error
// quotes at most the first 64 bytes of the text.
func ParseForgeHost(text string) (ForgeHost, error) {
	if !allBytes(text, isASCIIByte) {
		return ForgeHost{}, invalidForgeHost(text)
	}
	name, port, hasPort := strings.Cut(strings.ToLower(text), ":")
	if !validDNSName(name) {
		return ForgeHost{}, invalidForgeHost(text)
	}
	canonical := name
	if hasPort {
		number, ok := parsePort(port)
		if !ok {
			return ForgeHost{}, invalidForgeHost(text)
		}
		if number != defaultHTTPSPort {
			canonical += ":" + strconv.Itoa(number)
		}
	}
	if len(canonical) > maxForgeHostBytes {
		return ForgeHost{}, invalidForgeHost(text)
	}
	return ForgeHost{name: canonical}, nil
}

// String renders the host as ParseForgeHost reads it back. The zero value renders
// as the empty string.
func (h ForgeHost) String() string { return h.name }

func invalidForgeHost(text string) error {
	return fmt.Errorf("%q: %w", truncatedForError(text), ErrInvalidForgeHost)
}

func truncatedForError(text string) string {
	return text[:min(len(text), maxQuotedBytes)]
}

func validDNSName(name string) bool {
	if !strings.Contains(name, ".") {
		return false
	}
	for _, label := range strings.Split(name, ".") {
		if !validDNSLabel(label) {
			return false
		}
	}
	return true
}

func validDNSLabel(label string) bool {
	return label != "" && len(label) <= maxDNSLabelBytes &&
		label[0] != '-' && label[len(label)-1] != '-' && allBytes(label, isLabelByte)
}

func parsePort(text string) (int, bool) {
	if text == "" || !allBytes(text, isDigit) {
		return 0, false
	}
	number, err := strconv.Atoi(text)
	if err != nil || number < 1 || number > maxForgePort {
		return 0, false
	}
	return number, true
}

func isPort(text string) bool {
	_, ok := parsePort(text)
	return ok
}

func allBytes(text string, allowed func(byte) bool) bool {
	for index := range len(text) {
		if !allowed(text[index]) {
			return false
		}
	}
	return true
}

func isASCIIByte(character byte) bool {
	return character < utf8.RuneSelf
}

func isLabelByte(character byte) bool {
	return isLetterOrDigit(character) || character == '-'
}

func isDigit(character byte) bool {
	return character >= '0' && character <= '9'
}

func isLetterOrDigit(character byte) bool {
	return (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || isDigit(character)
}

func isLowerHexByte(character byte) bool {
	return isDigit(character) || (character >= 'a' && character <= 'f')
}
