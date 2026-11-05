package i18n_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

func newCatalog(t *testing.T, logger *slog.Logger, dirs ...string) *i18n.Catalog {
	t.Helper()
	sources := make([]fs.FS, 0, len(dirs))
	for _, dir := range dirs {
		sources = append(sources, os.DirFS(filepath.Join("testdata", dir)))
	}
	catalog, err := i18n.NewCatalog(logger, sources...)
	if err != nil {
		t.Fatalf("NewCatalog(%v) error = %v", dirs, err)
	}
	return catalog
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestMessageRendersEachLanguage(t *testing.T) {
	catalog := newCatalog(t, discardLogger(), "complete")
	cases := []struct {
		lang i18n.Language
		key  string
		want string
	}{
		{i18n.English, "session.archived", "Session archived"},
		{i18n.French, "session.archived", "Session archivée"},
		{i18n.English, "session.name", "Session Atlas"},
		{i18n.French, "session.name", "Session Atlas"},
	}
	for _, tc := range cases {
		got, err := catalog.Message(tc.lang, tc.key, map[string]any{"Name": "Atlas"})
		if err != nil {
			t.Fatalf("Message(%v, %q) error = %v", tc.lang, tc.key, err)
		}
		if got != tc.want {
			t.Fatalf("Message(%v, %q) = %q, want %q", tc.lang, tc.key, got, tc.want)
		}
	}
}

func TestMessagePluralFollowsCount(t *testing.T) {
	catalog := newCatalog(t, discardLogger(), "complete")
	cases := []struct {
		lang  i18n.Language
		count int
		want  string
	}{
		{i18n.English, 0, "0 sessions"},
		{i18n.English, 1, "1 session"},
		{i18n.English, 2, "2 sessions"},
		{i18n.French, 1, "1 session"},
		{i18n.French, 2, "2 sessions"},
	}
	for _, tc := range cases {
		got, err := catalog.Message(tc.lang, "session.count", map[string]any{"Count": tc.count})
		if err != nil {
			t.Fatalf("Message(%v, count %d) error = %v", tc.lang, tc.count, err)
		}
		if got != tc.want {
			t.Fatalf("Message(%v, count %d) = %q, want %q", tc.lang, tc.count, got, tc.want)
		}
	}
}

func TestMessageFallsBackToEnglishForMissingKey(t *testing.T) {
	catalog := newCatalog(t, discardLogger(), "fallback")
	got, err := catalog.Message(i18n.French, "only.english", nil)
	if err != nil {
		t.Fatalf("Message(French, only.english) error = %v", err)
	}
	if got != "Only in English" {
		t.Fatalf("Message(French, only.english) = %q, want the English text", got)
	}
	got, err = catalog.Message(i18n.French, "greeting", nil)
	if err != nil || got != "Bonjour" {
		t.Fatalf("Message(French, greeting) = %q, %v, want Bonjour", got, err)
	}
}

func TestMessageWarnsOncePerKeyAndLanguageUnderConcurrency(t *testing.T) {
	var logs bytes.Buffer
	catalog := newCatalog(t, slog.New(slog.NewTextHandler(&logs, nil)), "fallback")
	var group sync.WaitGroup
	for range 32 {
		group.Go(func() {
			if _, err := catalog.Message(i18n.French, "only.english", nil); err != nil {
				t.Errorf("Message(French, only.english) error = %v", err)
			}
		})
	}
	group.Wait()
	if got := strings.Count(logs.String(), "key=only.english"); got != 1 {
		t.Fatalf("warnings for only.english = %d, want 1; log:\n%s", got, logs.String())
	}
}

func TestMessageReturnsKeyWhenMissingEverywhere(t *testing.T) {
	var logs bytes.Buffer
	catalog := newCatalog(t, slog.New(slog.NewTextHandler(&logs, nil)), "complete")
	for range 3 {
		got, err := catalog.Message(i18n.English, "nowhere.key", nil)
		if err != nil {
			t.Fatalf("Message(English, nowhere.key) error = %v", err)
		}
		if got != "nowhere.key" {
			t.Fatalf("Message(English, nowhere.key) = %q, want the key", got)
		}
	}
	if got := strings.Count(logs.String(), "key=nowhere.key"); got != 1 {
		t.Fatalf("warnings for nowhere.key = %d, want 1; log:\n%s", got, logs.String())
	}
}

func TestMessageRefusesUnsupportedLanguage(t *testing.T) {
	catalog := newCatalog(t, discardLogger(), "complete")
	_, err := catalog.Message(i18n.Language(9), "session.archived", nil)
	if !errors.Is(err, i18n.ErrUnsupportedLanguage) {
		t.Fatalf("Message(9, session.archived) error = %v, want ErrUnsupportedLanguage", err)
	}
}

func TestMessageReturnsErrorForParamsThatCannotRender(t *testing.T) {
	catalog := newCatalog(t, discardLogger(), "complete")
	if _, err := catalog.Message(i18n.English, "session.count", map[string]any{"Count": "many"}); err == nil {
		t.Fatal("Message with a non numeric Count returned no error")
	}
}

func TestNewCatalogComposesSources(t *testing.T) {
	catalog := newCatalog(t, discardLogger(), "complete", "extra")
	got, err := catalog.Message(i18n.French, "extra.hello", nil)
	if err != nil || got != "Bonjour depuis le catalogue additionnel" {
		t.Fatalf("Message(French, extra.hello) = %q, %v", got, err)
	}
	got, err = catalog.Message(i18n.French, "session.archived", nil)
	if err != nil || got != "Session archivée" {
		t.Fatalf("Message(French, session.archived) = %q, %v", got, err)
	}
}

func TestNewCatalogRefusesKeyDefinedTwice(t *testing.T) {
	sources := []fs.FS{os.DirFS("testdata/complete"), os.DirFS("testdata/duplicate")}
	_, err := i18n.NewCatalog(discardLogger(), sources...)
	if !errors.Is(err, i18n.ErrDuplicateKey) {
		t.Fatalf("NewCatalog() error = %v, want ErrDuplicateKey", err)
	}
}

func TestNewCatalogRefusesUnsupportedLanguageFile(t *testing.T) {
	_, err := i18n.NewCatalog(discardLogger(), os.DirFS("testdata/unsupported"))
	if !errors.Is(err, i18n.ErrUnsupportedLanguage) {
		t.Fatalf("NewCatalog() error = %v, want ErrUnsupportedLanguage", err)
	}
}

func assertMessage(t *testing.T, catalog *i18n.Catalog, lang i18n.Language, key string, params map[string]any, want string) {
	t.Helper()
	got, err := catalog.Message(lang, key, params)
	if err != nil {
		t.Fatalf("Message(%v, %q) error = %v", lang, key, err)
	}
	if got != want {
		t.Fatalf("Message(%v, %q) = %q, want %q", lang, key, got, want)
	}
}

func TestMessagePluralForms(t *testing.T) {
	catalog := newCatalog(t, discardLogger(), "complete")
	cases := []struct {
		name  string
		lang  i18n.Language
		count any
		want  string
	}{
		{"French zero takes the one form", i18n.French, 0, "0 session"},
		{"negative count uses its magnitude", i18n.English, -1, "-1 session"},
		{"negative plural count", i18n.English, -2, "-2 sessions"},
		{"decimal string in English is plural", i18n.English, "1.5", "1.5 sessions"},
		{"decimal string in French is singular", i18n.French, "1.5", "1.5 session"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertMessage(t, catalog, tc.lang, "session.count", map[string]any{"Count": tc.count}, tc.want)
		})
	}
}

func TestMessageRefusesUnsupportedCountType(t *testing.T) {
	catalog := newCatalog(t, discardLogger(), "complete")
	for _, count := range []any{true, 1.5} {
		if _, err := catalog.Message(i18n.English, "session.count", map[string]any{"Count": count}); err == nil {
			t.Fatalf("Message with Count %v (%T) returned no error", count, count)
		}
	}
}

func TestMessageRefusesMissingParam(t *testing.T) {
	catalog := newCatalog(t, discardLogger(), "complete")
	if _, err := catalog.Message(i18n.English, "session.name", nil); err == nil {
		t.Fatal("Message without the Name param returned no error")
	}
	if _, err := catalog.Message(i18n.French, "session.name", map[string]any{"Title": "Atlas"}); err == nil {
		t.Fatal("Message with Title but no Name returned no error")
	}
}

func TestMessageAcceptsNilParamValue(t *testing.T) {
	catalog := newCatalog(t, discardLogger(), "complete")
	if _, err := catalog.Message(i18n.English, "session.name", map[string]any{"Name": nil}); err != nil {
		t.Fatalf("Message with a nil Name returned error %v", err)
	}
}

func TestMessageWarningsAreBounded(t *testing.T) {
	var logs bytes.Buffer
	catalog := newCatalog(t, slog.New(slog.NewTextHandler(&logs, nil)), "complete")
	missing := func(index int) {
		if _, err := catalog.Message(i18n.English, fmt.Sprintf("nowhere.%d", index), nil); err != nil {
			t.Fatalf("Message(English, nowhere.%d) error = %v", index, err)
		}
	}
	for index := range 1025 {
		missing(index)
	}
	missing(0)
	missing(1024)
	if got := strings.Count(logs.String(), "message missing from the catalog"); got != 1026 {
		t.Fatalf("warnings = %d, want 1026 (1024 remembered, then 1024 and 1024 again logged); log:\n%s", got, logs.String())
	}
}

func TestNewCatalogRefusesUnparsableTemplate(t *testing.T) {
	if _, err := i18n.NewCatalog(discardLogger(), os.DirFS("testdata/badtemplate")); err == nil {
		t.Fatal("NewCatalog with an unclosed action returned no error")
	}
}

func TestNewCatalogRefusesTemplateFunction(t *testing.T) {
	if _, err := i18n.NewCatalog(discardLogger(), os.DirFS("testdata/functions")); err == nil {
		t.Fatal("NewCatalog with a template function returned no error")
	}
}
