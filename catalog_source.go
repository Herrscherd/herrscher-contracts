package contracts

import "io/fs"

// CatalogSource is the instance-side form of Plugin.Catalog.
//
// Deprecated: set Plugin.Catalog instead. The host reads only the descriptor,
// because the texts must be known before any instance is built.
type CatalogSource interface {
	Catalog() fs.FS
}
