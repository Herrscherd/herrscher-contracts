package contracts_test

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	contracts "github.com/Herrscherd/herrscher-contracts"
	"github.com/Herrscherd/herrscher-contracts/i18n"
	"github.com/Herrscherd/herrscher-contracts/i18n/i18ntest"
)

type catalogPlugin struct {
	catalog fs.FS
}

func (p catalogPlugin) Catalog() fs.FS { return p.catalog }

var _ contracts.CatalogSource = catalogPlugin{}

func helloCatalog(french string) fs.FS {
	return fstest.MapFS{
		"en.json": &fstest.MapFile{Data: []byte(`[{"id": "gateway.hello", "other": "hello {{.Name}}"}]`)},
		"fr.json": &fstest.MapFile{Data: []byte(`[{"id": "gateway.hello", "other": "` + french + `"}]`)},
	}
}

func TestCatalogSourceFeedsTheCatalog(t *testing.T) {
	t.Parallel()
	var source contracts.CatalogSource = catalogPlugin{catalog: helloCatalog("bonjour {{.Name}}")}
	i18ntest.Check(t, source.Catalog())
	catalog, err := i18n.NewCatalog(nil, source.Catalog())
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	got, err := catalog.Message(i18n.French, "gateway.hello", map[string]any{"Name": "Shan"})
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	if got != "bonjour Shan" {
		t.Fatalf("Message = %q, want %q", got, "bonjour Shan")
	}
}

func TestCatalogSourceReturningNilIsIgnored(t *testing.T) {
	t.Parallel()
	var source contracts.CatalogSource = catalogPlugin{}
	catalog, err := i18n.NewCatalog(nil, source.Catalog(), helloCatalog("bonjour {{.Name}}"))
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	got, err := catalog.Message(i18n.French, "gateway.hello", map[string]any{"Name": "Shan"})
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	if got != "bonjour Shan" {
		t.Fatalf("Message = %q, want %q", got, "bonjour Shan")
	}
}

func TestCatalogSourcesDefiningTheSameKeyAreRefused(t *testing.T) {
	t.Parallel()
	first := catalogPlugin{catalog: helloCatalog("bonjour {{.Name}}")}
	second := catalogPlugin{catalog: helloCatalog("salut {{.Name}}")}
	_, err := i18n.NewCatalog(nil, first.Catalog(), second.Catalog())
	if !errors.Is(err, i18n.ErrDuplicateKey) {
		t.Fatalf("NewCatalog error = %v, want i18n.ErrDuplicateKey", err)
	}
}
