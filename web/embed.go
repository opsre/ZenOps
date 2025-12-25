package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var distFS embed.FS

// GetFileSystem returns the embedded file system for the frontend dist directory
func GetFileSystem() (http.FileSystem, error) {
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(subFS), nil
}

// GetFS returns the raw embed.FS for more flexible usage
func GetFS() embed.FS {
	return distFS
}
