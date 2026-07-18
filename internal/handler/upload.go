package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"cronyakatsuki/upfast/internal/store"
)

// Upload handles POST / — accepts a multipart file upload.
// It stores the file, detects its type, and returns the URL.
// If the request comes from curl, it returns plain text; otherwise HTML.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	// Limit upload to 10MB in memory; larger files go to temp disk.
	// Actual size limit will be enforced in the DB phase.
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		slog.Warn("failed to parse multipart form", "err", err)
		renderText(w, http.StatusBadRequest, "Failed to parse upload: "+err.Error()+"\n")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		slog.Warn("no file in upload request", "err", err)
		renderText(w, http.StatusBadRequest, "No file provided. Use -F \"file=@/path/to/file\"\n")
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Warn("closing uploaded file", "filename", header.Filename, "err", err)
		}
	}()

	info, err := h.store.Save(header.Filename, file)
	if err != nil {
		if err == store.ErrExists {
			renderText(w, http.StatusConflict, "A file with the same name already exists on the server.\n")
			return
		}
		slog.Error("failed to save file", "filename", header.Filename, "err", err)
		renderText(w, http.StatusInternalServerError, "Failed to save file.\n")
		return
	}

	fileURL := h.domain + "/files/" + info.Name + "\n"
	slog.Info("file uploaded", "name", info.Name, "type", info.FileType, "size", info.Size)

	if isCurl(r) {
		renderText(w, http.StatusOK, fileURL)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintf(w, "File uploaded at url: <strong>%s</strong>", fileURL); err != nil {
		slog.Warn("writing upload response", "err", err)
	}
}
