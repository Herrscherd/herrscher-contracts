// Package catalogfile reads message catalogues shared by the i18n runtime and its test helper.
package catalogfile

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

// ErrDuplicateKey is returned when a message key is defined twice across the sources.
var ErrDuplicateKey = errors.New("message key defined twice")

// Read parses every catalogue file of the sources and groups the messages by language,
// as languageOf names them. A key defined twice in one language is refused with ErrDuplicateKey.
func Read[L comparable](sources []fs.FS, languageOf func(name string) (L, error)) (map[L]map[string]*goi18n.Message, error) {
	byLanguage := make(map[L]map[string]*goi18n.Message)
	for _, source := range sources {
		if err := addSource(byLanguage, source, languageOf); err != nil {
			return nil, err
		}
	}
	return byLanguage, nil
}

func addSource[L comparable](byLanguage map[L]map[string]*goi18n.Message, source fs.FS, languageOf func(name string) (L, error)) error {
	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return fmt.Errorf("read catalog: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}
		if err := addFile(byLanguage, source, entry.Name(), languageOf); err != nil {
			return err
		}
	}
	return nil
}

func addFile[L comparable](byLanguage map[L]map[string]*goi18n.Message, source fs.FS, name string, languageOf func(name string) (L, error)) error {
	lang, err := languageOf(name)
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
			return fmt.Errorf("message %q in %v: %w", message.ID, lang, ErrDuplicateKey)
		}
		byLanguage[lang][message.ID] = message
	}
	return nil
}
