package web

import (
	"embed"
	"io/fs"
)

// Static contains the embedded frontend files.
//go:embed dist
var Static embed.FS

// FS returns the frontend files as a http.FileSystem.
func FS() fs.FS {
	f, err := fs.Sub(Static, "dist")
	if err != nil {
		panic(err)
	}
	return f
}
