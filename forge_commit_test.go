package contracts

import (
	"errors"
	"strings"
	"testing"
)

func TestParseCommitSHAAcceptsSHA1AndSHA256(t *testing.T) {
	for _, text := range []string{strings.Repeat("a1", 20), strings.Repeat("0f", 32)} {
		sha, err := ParseCommitSHA(text)
		if err != nil {
			t.Errorf("ParseCommitSHA(%d hex digits) error = %v, want none", len(text), err)
			continue
		}
		if sha.String() != text {
			t.Errorf("ParseCommitSHA(%q).String() = %q, want the text back", text, sha.String())
		}
	}
}

func TestParseCommitSHARefusesOtherText(t *testing.T) {
	texts := []string{
		"",
		strings.Repeat("a", 39),
		strings.Repeat("a", 41),
		strings.Repeat("a", 63),
		strings.Repeat("a", 65),
		strings.Repeat("A", 40),
		strings.Repeat("g", 40),
		strings.Repeat("a", 39) + " ",
		"refs/heads/" + strings.Repeat("a", 29),
	}
	for _, text := range texts {
		sha, err := ParseCommitSHA(text)
		if !errors.Is(err, ErrInvalidCommitSHA) {
			t.Errorf("ParseCommitSHA(%q) error = %v, want ErrInvalidCommitSHA", text, err)
		}
		if sha != (CommitSHA{}) {
			t.Errorf("ParseCommitSHA(%q) = %q, want the zero value on error", text, sha)
		}
	}
}

func TestZeroCommitSHAStringIsEmpty(t *testing.T) {
	if got := (CommitSHA{}).String(); got != "" {
		t.Fatalf("zero CommitSHA String() = %q, want empty", got)
	}
}
