package contracts

import "io/fs"

// CatalogSource is an optional capability of a live plugin instance: the
// translated texts it contributes to the host's catalog. The file system it
// returns holds one go-i18n JSON file per language at its root, named by BCP 47
// tag (en.json, fr.json), in the format i18n.NewCatalog reads. A plugin without
// translated texts does not implement it.
type CatalogSource interface {
	Catalog() fs.FS
}
