package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

func FS() fs.FS {
	sub, _ := fs.Sub(distFS, "dist")
	return sub
}
