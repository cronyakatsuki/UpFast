package handler

import (
	"bytes"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"cronyakatsuki/upfast/internal/store"
)

// mustSave saves a file to the mock store and fails the test on error.
func mustSave(t *testing.T, ms *mockStore, name, content string) {
	t.Helper()
	if _, err := ms.Save(name, strings.NewReader(content)); err != nil {
		t.Fatalf("mockStore.Save(%q): %v", name, err)
	}
}

// ---------------------------------------------------------------------------
// Mock store — in-memory implementation of store.Store for tests
// ---------------------------------------------------------------------------

type mockStore struct {
	mu      sync.RWMutex
	files   map[string]*store.FileInfo
	counter int64 // monotonic counter for ordering
}

func newMockStore() *mockStore {
	return &mockStore{files: make(map[string]*store.FileInfo)}
}

func (m *mockStore) Save(filename string, src io.Reader) (*store.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.files[filename]; exists {
		return nil, store.ErrExists
	}

	data, _ := io.ReadAll(src)
	ft := store.FileTypeText
	if strings.HasPrefix(filename, "img") {
		ft = store.FileTypeImage
	} else if strings.HasPrefix(filename, "vid") {
		ft = store.FileTypeVideo
	} else if strings.HasPrefix(filename, "bin") {
		ft = store.FileTypeOther
	}

	m.counter++
	info := &store.FileInfo{
		Name:       filename,
		FileType:   ft,
		Size:       int64(len(data)),
		UploadedAt: m.counter, // monotonic: guaranteed ordering
	}
	m.files[filename] = info
	return info, nil
}

func (m *mockStore) List() ([]store.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]store.FileInfo, 0, len(m.files))
	for _, f := range m.files {
		result = append(result, *f)
	}
	// Newest first
	sort.Slice(result, func(i, j int) bool {
		return result[i].UploadedAt > result[j].UploadedAt
	})
	return result, nil
}

func (m *mockStore) ListPaginated(offset, limit int) ([]store.FileInfo, error) {
	all, _ := m.List()
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if offset >= len(all) {
		return []store.FileInfo{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (m *mockStore) Delete(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.files[filename]; !exists {
		return store.ErrNotExist
	}
	delete(m.files, filename)
	return nil
}

func (m *mockStore) Path(filename string) string {
	return "/tmp/" + filename
}

func (m *mockStore) ReadContent(filename string, maxSize int64) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, exists := m.files[filename]
	if !exists {
		return "", store.ErrNotExist
	}
	if f.Size > maxSize {
		return "", nil
	}
	return "mock content for " + filename, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// testHandler creates a Handler with a mock store and a minimal template
// that covers all the rendering paths used by the actual templates.
func testHandler(t *testing.T, ms *mockStore) *Handler {
	t.Helper()

	tmpl := template.Must(template.New("").Parse(`
{{- define "index"}}<html><body>Index {{.Host}}</body></html>{{end}}
{{- define "files"}}<html><body>{{range .Files}}<div class="file">{{.Name}}<pre>{{.Content}}</pre></div>{{end}}</body></html>{{end}}
`))

	return New(ms, "http://test.local", tmpl)
}

func TestIndex(t *testing.T) {
	h := testHandler(t, newMockStore())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	h.Index(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Index http://test.local") {
		t.Errorf("body = %q, want to contain %q", string(body), "Index http://test.local")
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/html; charset=utf-8")
	}
}

// ---------------------------------------------------------------------------
// Upload
// ---------------------------------------------------------------------------

func TestUpload(t *testing.T) {
	h := testHandler(t, newMockStore())

	body := &bytes.Buffer{}
	writer := NewMultipartWriter(body, "file", "test.txt", "hello world")
	contentType := writer.FormDataContentType()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", body)
	r.Header.Set("Content-Type", contentType)
	h.Upload(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	respBody, _ := io.ReadAll(resp.Body)
	expected := "File uploaded at url: <strong>http://test.local/files/test.txt\n</strong>"
	if string(respBody) != expected {
		t.Errorf("body = %q, want %q", string(respBody), expected)
	}
}

func TestUpload_Curl(t *testing.T) {
	h := testHandler(t, newMockStore())

	body := &bytes.Buffer{}
	writer := NewMultipartWriter(body, "file", "test.txt", "curl content")
	contentType := writer.FormDataContentType()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", body)
	r.Header.Set("Content-Type", contentType)
	r.Header.Set("User-Agent", "curl/8.0.0")
	h.Upload(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	respBody, _ := io.ReadAll(resp.Body)
	expected := "http://test.local/files/test.txt\n"
	if string(respBody) != expected {
		t.Errorf("body = %q, want %q", string(respBody), expected)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/plain; charset=utf-8")
	}
}

func TestUpload_NoFile(t *testing.T) {
	h := testHandler(t, newMockStore())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	h.Upload(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestUpload_Duplicate(t *testing.T) {
	ms := newMockStore()
	h := testHandler(t, ms)

	// First upload
	body1 := &bytes.Buffer{}
	w1 := NewMultipartWriter(body1, "file", "dup.txt", "first")
	r1 := httptest.NewRequest("POST", "/", body1)
	r1.Header.Set("Content-Type", w1.FormDataContentType())
	h.Upload(httptest.NewRecorder(), r1)

	// Second upload (same name)
	body2 := &bytes.Buffer{}
	w2 := NewMultipartWriter(body2, "file", "dup.txt", "second")
	r2 := httptest.NewRequest("POST", "/", body2)
	r2.Header.Set("Content-Type", w2.FormDataContentType())

	w := httptest.NewRecorder()
	h.Upload(w, r2)

	resp := w.Result()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

// ---------------------------------------------------------------------------
// Files (listing)
// ---------------------------------------------------------------------------

func TestFiles_Curl(t *testing.T) {
	ms := newMockStore()
	h := testHandler(t, ms)
	mustSave(t, ms, "a.txt", "aaa")
	mustSave(t, ms, "b.txt", "bbb")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/files", nil)
	r.Header.Set("User-Agent", "curl/8.0.0")
	h.Files(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	// Should be plain text list (b.txt first since newest)
	if string(body) != "b.txt\na.txt\n" {
		t.Errorf("body = %q, want %q", string(body), "b.txt\na.txt\n")
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/plain; charset=utf-8")
	}
}

func TestFiles_HTML(t *testing.T) {
	ms := newMockStore()
	h := testHandler(t, ms)
	mustSave(t, ms, "a.txt", "aaa")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/files", nil)
	r.Header.Set("User-Agent", "Mozilla/5.0")
	h.Files(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.HasPrefix(string(body), "<html>") {
		t.Errorf("expected HTML, got %q", string(body))
	}
	if !strings.Contains(string(body), "a.txt") {
		t.Errorf("body should contain filename, got %q", string(body))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/html; charset=utf-8")
	}
}

// ---------------------------------------------------------------------------
// API Files (JSON paginated)
// ---------------------------------------------------------------------------

func TestAPIFiles(t *testing.T) {
	ms := newMockStore()
	h := testHandler(t, ms)

	// Save 3 files
	mustSave(t, ms, "a.txt", "aaa")
	mustSave(t, ms, "b.txt", "bbb")
	mustSave(t, ms, "c.txt", "ccc")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/files?offset=0&limit=2", nil)
	h.APIFiles(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}

	// Verify it's valid JSON with expected structure
	if !strings.Contains(string(body), `"name":"c.txt"`) {
		t.Errorf("JSON body missing c.txt: %s", string(body))
	}
	if !strings.Contains(string(body), `"hasMore":true`) {
		t.Errorf("JSON body should have hasMore true: %s", string(body))
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDelete(t *testing.T) {
	ms := newMockStore()
	h := testHandler(t, ms)
	mustSave(t, ms, "delete-me.txt", "bye")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/files/delete-me.txt", nil)
	r.SetPathValue("file", "delete-me.txt")
	h.Delete(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Deleted file from server: delete-me.txt") {
		t.Errorf("body = %q", string(body))
	}

	// Verify it's actually deleted from store
	list, _ := ms.List()
	if len(list) != 0 {
		t.Errorf("expected 0 files after delete, got %d", len(list))
	}
}

func TestDelete_NotExist(t *testing.T) {
	h := testHandler(t, newMockStore())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/files/nonexistent.txt", nil)
	r.SetPathValue("file", "nonexistent.txt")
	h.Delete(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestDelete_EmptyFilename(t *testing.T) {
	h := testHandler(t, newMockStore())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/files/", nil)
	r.SetPathValue("file", "")
	h.Delete(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// ---------------------------------------------------------------------------
// Helper: multipart form writer (stdlib, no external deps)
// ---------------------------------------------------------------------------

// NewMultipartWriter creates a multipart form body with a single file field.
// Returns the buffer (writable) and you can get the Content-Type via
// .FormDataContentType().
func NewMultipartWriter(body *bytes.Buffer, fieldName, fileName, content string) *multipartWriter {
	w := &multipartWriter{
		body: body,
	}
	w.boundary = "--testBoundary" + randomSuffix()
	w.write(body, fieldName, fileName, content)
	return w
}

type multipartWriter struct {
	body     *bytes.Buffer
	boundary string
}

func (mw *multipartWriter) FormDataContentType() string {
	return "multipart/form-data; boundary=" + mw.boundary
}

func (mw *multipartWriter) write(body *bytes.Buffer, field, filename, content string) {
	body.WriteString("--" + mw.boundary + "\r\n")
	body.WriteString(`Content-Disposition: form-data; name="` + field + `"; filename="` + filename + `"` + "\r\n")
	body.WriteString("Content-Type: text/plain\r\n")
	body.WriteString("\r\n")
	body.WriteString(content)
	body.WriteString("\r\n")
	body.WriteString("--" + mw.boundary + "--\r\n")
}

func randomSuffix() string {
	// Just a fixed suffix for deterministic tests
	return "000001"
}
