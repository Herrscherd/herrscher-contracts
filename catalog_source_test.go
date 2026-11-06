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

func helloCatalog(french string) fs.FS {
	return fstest.MapFS{
		"en.json": &fstest.MapFile{Data: []byte(`[{"id": "gateway.hello", "other": "hello {{.Name}}"}]`)},
		"fr.json": &fstest.MapFile{Data: []byte(`[{"id": "gateway.hello", "other": "` + french + `"}]`)},
	}
}

func catalogsOf(t *testing.T, plugins ...contracts.Plugin) []fs.FS {
	t.Helper()
	var registry contracts.Registry
	for idx := range plugins {
		registry.Register(plugins[idx])
	}
	var catalogs []fs.FS
	registered := registry.Plugins()
	for idx := range registered {
		catalogs = append(catalogs, registered[idx].Catalog)
	}
	return catalogs
}

func TestPluginCatalogIsReadWithoutAnInstance(t *testing.T) {
	t.Parallel()
	catalogs := catalogsOf(t, contracts.Plugin{Catalog: helloCatalog("bonjour {{.Name}}")})
	i18ntest.Check(t, catalogs[0])
	catalog, err := i18n.NewCatalog(nil, catalogs...)
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

func TestPluginWithoutCatalogIsIgnored(t *testing.T) {
	t.Parallel()
	catalogs := catalogsOf(t, contracts.Plugin{}, contracts.Plugin{Catalog: helloCatalog("bonjour {{.Name}}")})
	catalog, err := i18n.NewCatalog(nil, catalogs...)
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

func TestPluginCatalogsDefiningTheSameKeyAreRefused(t *testing.T) {
	t.Parallel()
	catalogs := catalogsOf(t,
		contracts.Plugin{Catalog: helloCatalog("bonjour {{.Name}}")},
		contracts.Plugin{Catalog: helloCatalog("salut {{.Name}}")},
	)
	_, err := i18n.NewCatalog(nil, catalogs...)
	if !errors.Is(err, i18n.ErrDuplicateKey) {
		t.Fatalf("NewCatalog error = %v, want i18n.ErrDuplicateKey", err)
	}
}
