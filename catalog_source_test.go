package contracts_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	contracts "github.com/Herrscherd/herrscher-contracts"
	"github.com/Herrscherd/herrscher-contracts/i18n"
	"github.com/Herrscherd/herrscher-contracts/i18n/i18ntest"
)

type catalogPlugin struct{}

func (catalogPlugin) Catalog() fs.FS {
	return fstest.MapFS{
		"en.json": &fstest.MapFile{Data: []byte(`[{"id": "gateway.hello", "other": "hello {{.Name}}"}]`)},
		"fr.json": &fstest.MapFile{Data: []byte(`[{"id": "gateway.hello", "other": "bonjour {{.Name}}"}]`)},
	}
}

var _ contracts.CatalogSource = catalogPlugin{}

func TestCatalogSourceFeedsTheCatalog(t *testing.T) {
	t.Parallel()
	var source contracts.CatalogSource = catalogPlugin{}
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
