package mocks

import (
	"embed"
	"io/fs"
)

//go:embed all:*
var content embed.FS

// FS returns the embedded filesystem for all mock files
var FS = content

// GetSubFS returns a sub-filesystem for a specific directory (e.g., "forms")
func GetSubFS(dir string) (fs.FS, error) {
	return fs.Sub(content, dir)
}
