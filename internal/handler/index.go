package handler

import "net/http"

// IndexData is passed to the index HTML template.
type IndexData struct {
	Host string
}

// Index handles GET / — renders the upload page.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	data := IndexData{Host: h.domain}
	h.renderTemplate(w, "index", data)
}
