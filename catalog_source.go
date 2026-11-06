package contracts

import "io/fs"

// CatalogSource is an optional capability of a live plugin instance: Catalog returns the translated texts it contributes, one go-i18n JSON file per language (en.json, fr.json) at the root of the file system.
type CatalogSource interface {
	Catalog() fs.FS
}
