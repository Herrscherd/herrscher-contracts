package contracts

import (
	"path/filepath"
	"slices"
	"strings"
)

// ModelReadable reports whether the host hands att to a model: an image, a text
// file, a PDF, JSON, XML, TOML or YAML. It reads ContentType, without its
// parameters, and falls back to the extension of Filename when no type is given.
// A gateway that stages files itself calls it before downloading, so it never
// fetches a file the host would drop.
func ModelReadable(att Attachment) bool {
	kind, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(att.ContentType)), ";")
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return slices.Contains(readableExtensions(), strings.ToLower(filepath.Ext(att.Filename)))
	}
	return strings.HasPrefix(kind, "image/") || strings.HasPrefix(kind, "text/") || slices.Contains(readableTypes(), kind)
}

func readableTypes() []string {
	return []string{
		"application/pdf", "application/json", "application/xml", "application/toml",
		"application/x-yaml", "application/yaml",
	}
}

func readableExtensions() []string {
	return []string{
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".pdf", ".txt", ".md", ".markdown",
		".csv", ".tsv", ".log", ".json", ".yaml", ".yml", ".toml", ".xml",
	}
}
