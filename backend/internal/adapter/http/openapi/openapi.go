// Package openapi embeds the API description and the Swagger UI page that renders it.
package openapi

import (
	"embed"
	"io/fs"
)

//go:embed openapi.yaml
var Spec []byte

//go:embed docs.html docs.js swagger-ui/swagger-ui-bundle.js swagger-ui/swagger-ui.css
var files embed.FS

// UI holds docs.html, docs.js and the vendored swagger-ui bundle, served under /api/docs.
var UI fs.FS = files
