package i18n

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"path"
	"slices"
	"strings"
	"sync"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// ErrDuplicateKey reports a message key that one language defines twice across the
// sources of a catalog.
var ErrDuplicateKey = errors.New("message key defined twice")

const countParam = "Count"

// Messages reads every source and returns its messages by language and key. A
// source is a file system whose root holds one go-i18n JSON file per language,
// named by its BCP 47 tag (en.json, fr.json). It refuses a file named after an
// unsupported language, and a key that a language defines twice across the
// sources, with an error wrapping ErrDuplicateKey.
func Messages(sources ...fs.FS) (map[Language]map[string]*goi18n.Message, error) {
	byLanguage := make(map[Language]map[string]*goi18n.Message, len(Languages()))
	for _, source := range sources {
		if err := addSource(byLanguage, source); err != nil {
			return nil, err
		}
	}
	return byLanguage, nil
}

func addSource(byLanguage map[Language]map[string]*goi18n.Message, source fs.FS) error {
	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return fmt.Errorf("read catalog: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}
		if err := addFile(byLanguage, source, entry.Name()); err != nil {
			return err
		}
	}
	return nil
}

func addFile(byLanguage map[Language]map[string]*goi18n.Message, source fs.FS, name string) error {
	lang, err := ParseLanguage(strings.TrimSuffix(name, ".json"))
	if err != nil {
		return fmt.Errorf("catalog file %q: %w", name, err)
	}
	data, err := fs.ReadFile(source, name)
	if err != nil {
		return fmt.Errorf("read catalog file %q: %w", name, err)
	}
	file, err := goi18n.ParseMessageFileBytes(data, name, nil)
	if err != nil {
		return fmt.Errorf("parse catalog file %q: %w", name, err)
	}
	if byLanguage[lang] == nil {
		byLanguage[lang] = make(map[string]*goi18n.Message, len(file.Messages))
	}
	for _, message := range file.Messages {
		if _, defined := byLanguage[lang][message.ID]; defined {
			return fmt.Errorf("message %q in %s: %w", message.ID, lang, ErrDuplicateKey)
		}
		byLanguage[lang][message.ID] = message
	}
	return nil
}

// Catalog renders messages in the supported languages. Its zero value is not
// usable: build it with NewCatalog. Once built, it is safe for concurrent use.
type Catalog struct {
	localizers map[Language]*goi18n.Localizer
	logger     *slog.Logger
	warned     sync.Map
}

type warning struct {
	lang   Language
	key    string
	absent bool
}

// NewCatalog builds a catalog from the sources given to Messages. The logger
// receives one warning per key and language that falls back or is missing; a nil
// logger uses slog.Default.
func NewCatalog(logger *slog.Logger, sources ...fs.FS) (*Catalog, error) {
	byLanguage, err := Messages(sources...)
	if err != nil {
		return nil, err
	}
	bundle := goi18n.NewBundle(language.Und)
	localizers := make(map[Language]*goi18n.Localizer, len(Languages()))
	for _, lang := range Languages() {
		if err = bundle.AddMessages(lang.tag(), slices.Collect(maps.Values(byLanguage[lang]))...); err != nil {
			return nil, fmt.Errorf("add messages in %s: %w", lang, err)
		}
		localizers[lang] = goi18n.NewLocalizer(bundle, lang.String())
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Catalog{localizers: localizers, logger: logger}, nil
}

func (l Language) tag() language.Tag {
	return language.Make(l.String())
}

// Message renders the text of key in lang. Each params entry fills the {{.Name}}
// field of that name, and an entry named Count also selects the CLDR plural form.
// A key missing in lang renders in English and logs a warning once per key and
// language. A key missing in every language returns the key itself and logs a
// warning once. It returns an error for an unsupported language, or a message
// that cannot be rendered with params.
func (c *Catalog) Message(lang Language, key string, params map[string]any) (string, error) {
	text, err := c.render(lang, key, params)
	if err == nil {
		return text, nil
	}
	if !isMissing(err) {
		return "", err
	}
	if lang != English {
		c.noteFallback(lang, key)
		text, err = c.render(English, key, params)
		if err == nil {
			return text, nil
		}
		if !isMissing(err) {
			return "", err
		}
	}
	c.noteAbsent(lang, key)
	return key, nil
}

func (c *Catalog) render(lang Language, key string, params map[string]any) (string, error) {
	localizer, found := c.localizers[lang]
	if !found {
		return "", fmt.Errorf("language %d: %w", uint8(lang), ErrUnsupportedLanguage)
	}
	text, err := localizer.Localize(&goi18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: params,
		PluralCount:  params[countParam],
	})
	if err != nil {
		return "", fmt.Errorf("render %q: %w", key, err)
	}
	return text, nil
}

func isMissing(err error) bool {
	_, missing := errors.AsType[*goi18n.MessageNotFoundErr](err)
	return missing
}

func (c *Catalog) noteFallback(lang Language, key string) {
	if _, loaded := c.warned.LoadOrStore(warning{lang: lang, key: key}, struct{}{}); !loaded {
		c.logger.Warn("message missing in language, using English", "lang", lang.String(), "key", key)
	}
}

func (c *Catalog) noteAbsent(lang Language, key string) {
	if _, loaded := c.warned.LoadOrStore(warning{lang: lang, key: key, absent: true}, struct{}{}); !loaded {
		c.logger.Warn("message missing from the catalog, returning its key", "lang", lang.String(), "key", key)
	}
}
