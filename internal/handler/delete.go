package handler

import (
	"log/slog"
	"net/http"

	"cronyakatsuki/upfast/internal/store"
)

// Delete handles DELETE /files/{file} — removes a stored file.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("file")
	if filename == "" {
		renderText(w, http.StatusBadRequest, "No filename specified.\n")
		return
	}

	if err := h.store.Delete(filename); err != nil {
		if err == store.ErrNotExist {
			renderText(w, http.StatusNotFound, "That file doesn't exist on the server.\n")
			return
		}
		slog.Error("failed to delete file", "filename", filename, "err", err)
		renderText(w, http.StatusInternalServerError, "Error while deleting "+filename+"\n")
		return
	}

	slog.Info("file deleted", "name", filename)
	renderText(w, http.StatusOK, "Deleted file from server: "+filename+"\n")
}
