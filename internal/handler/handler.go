// Package handler implements HTTP handlers for UpFast.
//
// Each handler is a method on Handler, which holds dependencies
// (Store, templates, config). This makes handlers testable — you
// can pass a mock store and verify behavior without a real filesystem.
package handler

import (
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"cronyakatsuki/upfast/internal/store"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	store  store.Store
	domain string
	tmpl   *template.Template
}

// New creates a new Handler with the given dependencies.
func New(s store.Store, domain string, tmpl *template.Template) *Handler {
	return &Handler{
		store:  s,
		domain: domain,
		tmpl:   tmpl,
	}
}

// renderTemplate executes a named template and writes it to the response.
func (h *Handler) renderTemplate(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("template execution failed", "template", name, "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

// renderText writes a plain text response.
func renderText(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	io.WriteString(w, msg)
}

// isCurl returns true if the request appears to come from curl.
func isCurl(r *http.Request) bool {
	return strings.HasPrefix(r.UserAgent(), "curl/")
}
