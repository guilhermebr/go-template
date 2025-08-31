package docs

import (
	"embed"
	"io/fs"
)

// DocsFS exposes the documentation assets to be consumed by apps.
//
//go:embed *.html *.yaml *.json
var embeddedDocs embed.FS

// FS returns a filesystem rooted at the docs directory.
// This abstracts the underlying embed so apps can import and serve it.
func FS() fs.FS {
	return embeddedDocs
}
