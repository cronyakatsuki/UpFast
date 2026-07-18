// Package assets holds embedded templates and static files.
// It lives at the project root because //go:embed paths are
// relative to the source file's directory and cannot use "..".
package assets

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
)

//go:embed public/views/*.html
var templateFiles embed.FS

//go:embed static/*
var staticFiles embed.FS

// ParseTemplates parses the embedded HTML templates.
func ParseTemplates() *template.Template {
	return template.Must(template.ParseFS(templateFiles, "public/views/*.html"))
}

// StaticFileSystem returns an http.FileSystem for embedded static assets.
func StaticFileSystem() http.FileSystem {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("failed to get static sub-fs: %v", err)
	}
	return http.FS(sub)
}
