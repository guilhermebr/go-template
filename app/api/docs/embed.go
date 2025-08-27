package docs

import "embed"

// DocsFS contains all the documentation files
//go:embed *.html *.yaml *.json
var DocsFS embed.FS