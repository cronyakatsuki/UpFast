package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"cronyakatsuki/upfast/internal/store"
)

// Max text preview size — files larger than this won't have inline content.
const maxTextPreview = 100 * 1024 // 100 KB

// FilesData is passed to the files HTML template.
type FilesData struct {
	Files     []FileItem
	HasMore   bool
	NextBatch int
}

// FileItem represents a file for template rendering.
type FileItem struct {
	Name     string
	FileType string
	Content  string
}

// apiFileItem is the JSON representation of a file.
type apiFileItem struct {
	Name     string `json:"name"`
	FileType string `json:"fileType"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
}

const defaultPageSize = 20

// Files handles GET /files/ — lists stored files.
// For curl: returns plain text list.
// For browsers: returns HTML page with first batch + infinite scroll JS.
func (h *Handler) Files(w http.ResponseWriter, r *http.Request) {
	if isCurl(r) {
		h.filesPlain(w, r)
		return
	}
	h.filesHTML(w, r)
}

// filesPlain returns a plain text file listing for curl users.
func (h *Handler) filesPlain(w http.ResponseWriter, _ *http.Request) {
	fileList, err := h.store.List()
	if err != nil {
		http.Error(w, "Error listing files\n", http.StatusInternalServerError)
		return
	}

	var out strings.Builder
	for _, f := range fileList {
		out.WriteString(f.Name)
		out.WriteString("\n")
	}
	renderText(w, http.StatusOK, out.String())
}

// filesHTML renders the HTML file listing page with infinite scroll.
func (h *Handler) filesHTML(w http.ResponseWriter, _ *http.Request) {
	files, err := h.store.ListPaginated(0, defaultPageSize)
	if err != nil {
		http.Error(w, "Error listing files\n", http.StatusInternalServerError)
		return
	}

	items := make([]FileItem, 0, len(files))
	for _, f := range files {
		items = append(items, h.fileInfoToItem(f))
	}

	data := FilesData{
		Files:     items,
		HasMore:   len(files) >= defaultPageSize,
		NextBatch: defaultPageSize,
	}

	h.renderTemplate(w, "files", data)
}

// APIFiles handles GET /api/files?offset=0&limit=20 — JSON paginated list.
func (h *Handler) APIFiles(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = defaultPageSize
	}
	if offset < 0 {
		offset = 0
	}

	files, err := h.store.ListPaginated(offset, limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list files"})
		return
	}

	items := make([]apiFileItem, 0, len(files))
	for _, f := range files {
		items = append(items, apiFileItem{
			Name:     f.Name,
			FileType: string(f.FileType),
			URL:      h.domain + "/files/" + f.Name,
			Size:     f.Size,
		})
	}

	resp := map[string]any{
		"files":   items,
		"offset":  offset,
		"limit":   limit,
		"hasMore": len(files) >= limit,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(resp)
}

// fileInfoToItem converts a store.FileInfo to a template-friendly FileItem.
// For text files under maxTextPreview, it reads and embeds the content.
func (h *Handler) fileInfoToItem(f store.FileInfo) FileItem {
	item := FileItem{
		Name:     f.Name,
		FileType: string(f.FileType),
	}

	// Read text file content for inline preview (only for initial page render)
	if f.FileType == store.FileTypeText && f.Size <= maxTextPreview {
		content, err := h.store.ReadContent(f.Name, maxTextPreview)
		if err == nil {
			item.Content = content
		}
	}

	return item
}
