package i18n

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"text/template/parse"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	goi18ntemplate "github.com/nicksnyder/go-i18n/v2/i18n/template"
	"golang.org/x/text/language"

	"github.com/Herrscherd/herrscher-contracts/i18n/internal/catalogfile"
)

// ErrDuplicateKey is returned by NewCatalog when a key is defined twice across the sources.
var ErrDuplicateKey = catalogfile.ErrDuplicateKey

const (
	countParam    = "Count"
	maxRemembered = 1024
)

// Catalog renders messages in the supported languages. Its zero value is not
// usable: build it with NewCatalog. Once built, it is safe for concurrent use.
type Catalog struct {
	localizers map[Language]*goi18n.Localizer
	logger     *slog.Logger
	mu         sync.Mutex
	warned     map[warning]struct{}
}

type warning struct {
	lang   Language
	key    string
	absent bool
}

// NewCatalog builds a catalog from the sources of a catalog, each a file system
// whose root holds one go-i18n JSON file per language, named by its BCP 47 tag
// (en.json, fr.json). It refuses a file named after an unsupported language, a key
// that a language defines twice across the sources (ErrDuplicateKey), and a
// message whose template does not parse, which includes any call to a template
// function. The logger receives one warning per key and language that falls back
// or is missing, for at most 1024 of them; past that, each further missing key is
// logged every time it is asked for. A nil logger uses slog.Default.
func NewCatalog(logger *slog.Logger, sources ...fs.FS) (*Catalog, error) {
	byLanguage, err := catalogfile.Read(sources, languageOfFile)
	if err != nil {
		return nil, err
	}
	err = checkTemplates(byLanguage)
	if err != nil {
		return nil, err
	}
	bundle := goi18n.NewBundle(language.Und)
	localizers := make(map[Language]*goi18n.Localizer, len(Languages()))
	for _, lang := range Languages() {
		if err = bundle.AddMessages(language.Make(lang.bcp47()), slices.Collect(maps.Values(byLanguage[lang]))...); err != nil {
			return nil, fmt.Errorf("add messages in %s: %w", lang, err)
		}
		localizers[lang] = goi18n.NewLocalizer(bundle, lang.bcp47())
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Catalog{localizers: localizers, logger: logger, warned: make(map[warning]struct{})}, nil
}

func languageOfFile(name string) (Language, error) {
	return ParseLanguage(strings.TrimSuffix(name, ".json"))
}

func checkTemplates(byLanguage map[Language]map[string]*goi18n.Message) error {
	for _, lang := range Languages() {
		for _, key := range slices.Sorted(maps.Keys(byLanguage[lang])) {
			if err := parseMessage(byLanguage[lang][key]); err != nil {
				return fmt.Errorf("message %q in %s: %w", key, lang, err)
			}
		}
	}
	return nil
}

func parseMessage(message *goi18n.Message) error {
	for _, text := range []string{message.Zero, message.One, message.Two, message.Few, message.Many, message.Other} {
		if _, err := parse.Parse(message.ID, text, message.LeftDelim, message.RightDelim); err != nil {
			return err
		}
	}
	return nil
}

// Message renders the text of key in lang. Each params entry fills the {{.Name}}
// field of that name, and an entry named Count also selects the CLDR plural form.
// A field with no entry in params is an error; an entry whose value is nil is not.
// A key missing in lang renders in English and logs a warning once per key and
// language. A key missing in every language returns the key itself and logs a
// warning once. It returns an error for an unsupported language, or for a message
// that cannot be rendered with params. Rendering does not check that a message
// holds only plain fields: Check does, in the test suite.
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
		MessageID:      key,
		TemplateData:   params,
		PluralCount:    params[countParam],
		TemplateParser: &goi18ntemplate.TextParser{Option: "missingkey=error"},
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
	if c.firstWarning(warning{lang: lang, key: key}) {
		c.logger.Warn("message missing in language, using English", "lang", lang.String(), "key", key)
	}
}

func (c *Catalog) noteAbsent(lang Language, key string) {
	if c.firstWarning(warning{lang: lang, key: key, absent: true}) {
		c.logger.Warn("message missing from the catalog, returning its key", "lang", lang.String(), "key", key)
	}
}

func (c *Catalog) firstWarning(entry warning) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, seen := c.warned[entry]; seen {
		return false
	}
	if len(c.warned) < maxRemembered {
		c.warned[entry] = struct{}{}
	}
	return true
}
