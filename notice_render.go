package contracts

import (
	"embed"
	"fmt"
	"io/fs"
	"sync"

	"github.com/Herrscherd/herrscher-contracts/i18n"
)

//go:embed locales
var sharedLocales embed.FS

func sharedSources() (fs.FS, error) {
	sources, err := fs.Sub(sharedLocales, "locales")
	if err != nil {
		return nil, fmt.Errorf("shared locales: %w", err)
	}
	return sources, nil
}

func newSharedCatalog() (*i18n.Catalog, error) {
	sources, err := sharedSources()
	if err != nil {
		return nil, err
	}
	return i18n.NewCatalog(nil, sources)
}

// Renderer turns a Notice into text through the shared catalog. Its zero value
// is not usable: build it with NewRenderer. The catalog is built on the first
// render, and a failure to build it is returned by every render.
type Renderer struct {
	catalog func() (*i18n.Catalog, error)
}

// NewRenderer returns a renderer over the shared catalog of the module.
func NewRenderer() *Renderer {
	return &Renderer{catalog: sync.OnceValues(newSharedCatalog)}
}

// Render returns the text of notice in lang. It refuses a notice whose code is
// unknown, or whose arguments do not match the parameters of its code, with
// ErrUnknownCode or ErrNoticeArgs. It refuses a code whose key has no English
// message with ErrMissingMessage. A key missing in lang renders in English, as the
// catalog does.
func (r *Renderer) Render(lang i18n.Language, notice Notice) (string, error) {
	params, err := notice.params()
	if err != nil {
		return "", err
	}
	catalog, err := r.catalog()
	if err != nil {
		return "", err
	}
	key := notice.Code.Key()
	if !catalog.Has(key) {
		return "", fmt.Errorf("code %q: %w", key, ErrMissingMessage)
	}
	return catalog.Message(lang, key, params)
}
