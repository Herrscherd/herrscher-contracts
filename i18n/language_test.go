package i18n_test

import (
	"errors"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

func TestParseLanguageAcceptsLocalesInAnyCase(t *testing.T) {
	cases := []struct {
		text string
		want i18n.Language
	}{
		{"fr", i18n.French},
		{"fr-FR", i18n.French},
		{"fr_FR.UTF-8", i18n.French},
		{"FR_fr", i18n.French},
		{"fr_FR@euro", i18n.French},
		{"en", i18n.English},
		{"en-GB", i18n.English},
		{"en_US.UTF-8", i18n.English},
		{"EN", i18n.English},
		{"C", i18n.English},
		{"C.UTF-8", i18n.English},
		{"POSIX", i18n.English},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			got, err := i18n.ParseLanguage(tc.text)
			if err != nil {
				t.Fatalf("ParseLanguage(%q) error = %v", tc.text, err)
			}
			if got != tc.want {
				t.Fatalf("ParseLanguage(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}

func TestParseLanguageRefusesUnsupportedLanguage(t *testing.T) {
	for _, text := range []string{"de", "de_DE.UTF-8", "es-ES", "."} {
		t.Run(text, func(t *testing.T) {
			_, err := i18n.ParseLanguage(text)
			if !errors.Is(err, i18n.ErrUnsupportedLanguage) {
				t.Fatalf("ParseLanguage(%q) error = %v, want ErrUnsupportedLanguage", text, err)
			}
		})
	}
}

func TestParseLanguageRefusesEmptyText(t *testing.T) {
	_, err := i18n.ParseLanguage("")
	if !errors.Is(err, i18n.ErrNoLanguage) {
		t.Fatalf("ParseLanguage(\"\") error = %v, want ErrNoLanguage", err)
	}
	if errors.Is(err, i18n.ErrUnsupportedLanguage) {
		t.Fatalf("ParseLanguage(\"\") error = %v, must not be ErrUnsupportedLanguage", err)
	}
}

func TestLanguageStringIsBCP47Tag(t *testing.T) {
	cases := map[i18n.Language]string{
		i18n.English:     "en",
		i18n.French:      "fr",
		i18n.Language(0): "und",
	}
	for lang, want := range cases {
		if got := lang.String(); got != want {
			t.Fatalf("Language(%d).String() = %q, want %q", uint8(lang), got, want)
		}
	}
}

func TestResolveTakesFirstSupportedCandidate(t *testing.T) {
	cases := []struct {
		name       string
		candidates []i18n.Language
		want       i18n.Language
	}{
		{"zero candidates are skipped", []i18n.Language{0, i18n.French, i18n.English}, i18n.French},
		{"first defined candidate wins", []i18n.Language{i18n.English, i18n.French}, i18n.English},
		{"unsupported value is skipped", []i18n.Language{i18n.Language(9), i18n.French}, i18n.French},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := i18n.Resolve(tc.candidates...); got != tc.want {
				t.Fatalf("Resolve(%v) = %v, want %v", tc.candidates, got, tc.want)
			}
		})
	}
}

func TestResolveFallsBackToEnglish(t *testing.T) {
	if got := i18n.Resolve(); got != i18n.English {
		t.Fatalf("Resolve() = %v, want English", got)
	}
	if got := i18n.Resolve(0, 0); got != i18n.English {
		t.Fatalf("Resolve(0, 0) = %v, want English", got)
	}
}

func TestFromEnvReadsLocaleVariablesInOrder(t *testing.T) {
	cases := []struct {
		name     string
		env      map[string]string
		want     i18n.Language
		wantFind bool
	}{
		{"LC_ALL wins", map[string]string{"LC_ALL": "fr_FR.UTF-8", "LANG": "en_US.UTF-8"}, i18n.French, true},
		{"LC_MESSAGES comes before LANG", map[string]string{"LC_MESSAGES": "en_US", "LANG": "fr_FR"}, i18n.English, true},
		{"empty variable is skipped", map[string]string{"LC_ALL": "", "LC_MESSAGES": "fr"}, i18n.French, true},
		{"unsupported variable is skipped", map[string]string{"LC_ALL": "de_DE", "LANG": "fr_FR"}, i18n.French, true},
		{"C locale is English", map[string]string{"LANG": "C"}, i18n.English, true},
		{"no supported variable", map[string]string{"LANG": "de_DE.UTF-8"}, 0, false},
		{"no variable at all", make(map[string]string), 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, found := i18n.FromEnv(func(name string) string { return tc.env[name] })
			if got != tc.want || found != tc.wantFind {
				t.Fatalf("FromEnv() = (%v, %v), want (%v, %v)", got, found, tc.want, tc.wantFind)
			}
		})
	}
}
