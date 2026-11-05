package i18n

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Language is a supported human language. Its zero value is not a language: no
// parser returns it, it is not supported, and String renders it as the
// undetermined tag und.
type Language uint8

const (
	// English is the source language: every message key is defined in it.
	English Language = iota + 1
	// French is a supported language, translated from English.
	French
)

var (
	// ErrNoLanguage reports an empty text, which names no language.
	ErrNoLanguage = errors.New("no language named")
	// ErrUnsupportedLanguage reports a language that herrscher does not translate into.
	ErrUnsupportedLanguage = errors.New("unsupported language")
)

// Languages returns every supported language, English first.
func Languages() []Language {
	return []Language{English, French}
}

// String returns the BCP 47 tag of the language, or und for a value that is not a
// supported language.
func (l Language) String() string {
	switch l {
	case English:
		return "en"
	case French:
		return "fr"
	}
	return "und"
}

func (l Language) supported() bool {
	return slices.Contains(Languages(), l)
}

// ParseLanguage reads a language from a BCP 47 tag or from a POSIX locale such as
// fr_FR.UTF-8, in any letter case. The C and POSIX locales are English. It returns
// ErrNoLanguage for an empty text, and an error wrapping ErrUnsupportedLanguage
// for a language that is not supported.
func ParseLanguage(text string) (Language, error) {
	if text == "" {
		return 0, ErrNoLanguage
	}
	switch primarySubtag(text) {
	case "c", "posix", "en":
		return English, nil
	case "fr":
		return French, nil
	}
	return 0, fmt.Errorf("language %q: %w", text, ErrUnsupportedLanguage)
}

func primarySubtag(text string) string {
	withoutCodeset, _, _ := strings.Cut(text, ".")
	withoutModifier, _, _ := strings.Cut(withoutCodeset, "@")
	tag := strings.ReplaceAll(withoutModifier, "_", "-")
	primary, _, _ := strings.Cut(tag, "-")
	return strings.ToLower(primary)
}
