package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// newTestStore creates a FileSystemStore in a temporary directory.
func newTestStore(t *testing.T) *FileSystemStore {
	t.Helper()
	s, err := NewFileSystemStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileSystemStore: %v", err)
	}
	return s
}

// mustSave saves content as filename and fails the test on error.
func mustSave(t *testing.T, s *FileSystemStore, filename, content string) *FileInfo {
	t.Helper()
	info, err := s.Save(filename, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save(%q): %v", filename, err)
	}
	return info
}

// ---------------------------------------------------------------------------
// NewFileSystemStore
// ---------------------------------------------------------------------------

func TestNewFileSystemStore_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir", "nested")

	s, err := NewFileSystemStore(sub)
	if err != nil {
		t.Fatalf("NewFileSystemStore: %v", err)
	}

	// Directory should exist
	if _, err := os.Stat(sub); os.IsNotExist(err) {
		t.Fatal("expected directory to be created")
	}

	// meta.json should exist
	if _, err := os.Stat(s.metaPath); os.IsNotExist(err) {
		t.Fatal("expected meta.json to be created")
	}
}

func TestNewFileSystemStore_ExistingDir(t *testing.T) {
	dir := t.TempDir()
	// Pre-create meta.json with existing entry
	metaContent := `[{"name":"pre-existing","fileType":"text","size":4,"uploadedAt":100}]`
	metaPath := filepath.Join(dir, "meta.json")
	if err := os.WriteFile(metaPath, []byte(metaContent), 0644); err != nil {
		t.Fatal(err)
	}

	s, err := NewFileSystemStore(dir)
	if err != nil {
		t.Fatalf("NewFileSystemStore: %v", err)
	}

	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "pre-existing" {
		t.Fatalf("expected 1 pre-existing file, got %+v", list)
	}
}

// ---------------------------------------------------------------------------
// Save
// ---------------------------------------------------------------------------

func TestSave_Basic(t *testing.T) {
	s := newTestStore(t)
	info := mustSave(t, s, "hello.txt", "Hello, World!")

	if info.Name != "hello.txt" {
		t.Errorf("Name = %q, want %q", info.Name, "hello.txt")
	}
	if info.FileType != FileTypeText {
		t.Errorf("FileType = %q, want %q", info.FileType, FileTypeText)
	}
	if info.Size != 13 {
		t.Errorf("Size = %d, want %d", info.Size, 13)
	}
	if info.UploadedAt == 0 {
		t.Error("UploadedAt should be non-zero")
	}

	// File should exist on disk
	content, err := os.ReadFile(s.Path("hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "Hello, World!" {
		t.Errorf("file content = %q, want %q", string(content), "Hello, World!")
	}
}

func TestSave_Duplicate(t *testing.T) {
	s := newTestStore(t)
	mustSave(t, s, "dup.txt", "first")

	_, err := s.Save("dup.txt", strings.NewReader("second"))
	if err != ErrExists {
		t.Errorf("expected ErrExists, got %v", err)
	}

	// Content should still be the first version
	content, _ := os.ReadFile(s.Path("dup.txt"))
	if string(content) != "first" {
		t.Errorf("content = %q, want %q", string(content), "first")
	}
}

func TestSave_InvalidFilename(t *testing.T) {
	s := newTestStore(t)

	tests := []string{
		"../etc/passwd",
		"foo/bar.txt",
		"a/../b",
	}
	for _, name := range tests {
		_, err := s.Save(name, strings.NewReader("content"))
		if err == nil {
			t.Errorf("expected error for filename %q", name)
		}
	}
}

func TestSave_TypeDetection(t *testing.T) {
	s := newTestStore(t)

	tests := []struct {
		name     string
		content  []byte
		wantType FileType
	}{
		{"plain.txt", []byte("hello"), FileTypeText},
		{"image.jpg", []byte{0xFF, 0xD8, 0xFF, 0x00}, FileTypeImage},
		{"image.png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, FileTypeImage},
		{"data.json", []byte(`{"a":1}`), FileTypeText},
		{"video.mp4", append([]byte{0x00, 0x00, 0x00, 0x00}, []byte("ftypmp4")...), FileTypeVideo},
		{"binary.bin", []byte{0x00, 0x01, 0x02, 0x03}, FileTypeOther},
	}

	for _, tt := range tests {
		info, err := s.Save(tt.name, strings.NewReader(string(tt.content)))
		if err != nil {
			t.Fatalf("Save(%q): %v", tt.name, err)
		}
		if info.FileType != tt.wantType {
			t.Errorf("Save(%q).FileType = %q, want %q", tt.name, info.FileType, tt.wantType)
		}
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList_Empty(t *testing.T) {
	s := newTestStore(t)
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("len = %d, want 0", len(list))
	}
}

func TestList_Order(t *testing.T) {
	s := newTestStore(t)

	// Save files. Since all saves happen in the current second,
	// they'll share the same UploadedAt timestamp, and sort order
	// within the same timestamp is stable (insertion order preserved
	// by sort.Slice). The most recently inserted appears last in
	// meta, so after sort (newest first), it'll be first.
	mustSave(t, s, "a.txt", "aaa")
	mustSave(t, s, "b.txt", "bbb")
	mustSave(t, s, "c.txt", "ccc")

	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 3 {
		t.Fatalf("len = %d, want 3", len(list))
	}

	// All timestamps are equal, so the sort is stable.
	// Last inserted (c) should be first after sorting newest-first.
	if list[0].Name != "c.txt" {
		t.Errorf("list[0].Name = %q, want %q", list[0].Name, "c.txt")
	}
	if list[1].Name != "b.txt" {
		t.Errorf("list[1].Name = %q, want %q", list[1].Name, "b.txt")
	}
	if list[2].Name != "a.txt" {
		t.Errorf("list[2].Name = %q, want %q", list[2].Name, "a.txt")
	}
}

// ---------------------------------------------------------------------------
// ListPaginated
// ---------------------------------------------------------------------------

func TestListPaginated(t *testing.T) {
	s := newTestStore(t)

	// Save 25 files
	for i := range 25 {
		mustSave(t, s, fmt.Sprintf("file-%d.txt", i), "content")
	}

	// First page
	page1, err := s.ListPaginated(0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 20 {
		t.Errorf("page1 len = %d, want 20", len(page1))
	}

	// Second page
	page2, err := s.ListPaginated(20, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 5 {
		t.Errorf("page2 len = %d, want 5", len(page2))
	}

	// Pages should not overlap
	names := make(map[string]bool)
	for _, f := range page1 {
		names[f.Name] = true
	}
	for _, f := range page2 {
		if names[f.Name] {
			t.Errorf("file %q appears in both pages", f.Name)
		}
	}
}

func TestListPaginated_EdgeCases(t *testing.T) {
	s := newTestStore(t)
	mustSave(t, s, "one.txt", "one")
	mustSave(t, s, "two.txt", "two")

	// Offset beyond end
	page, err := s.ListPaginated(100, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 0 {
		t.Errorf("expected empty page, got %d items", len(page))
	}

	// Negative offset
	page, err = s.ListPaginated(-1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 {
		t.Errorf("expected 2 items, got %d", len(page))
	}

	// Zero limit (should use default 20)
	page, err = s.ListPaginated(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 {
		t.Errorf("expected 2 items, got %d", len(page))
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDelete(t *testing.T) {
	s := newTestStore(t)
	mustSave(t, s, "delete-me.txt", "to be deleted")

	if err := s.Delete("delete-me.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Should not appear in listing
	list, _ := s.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}

	// File should be gone from disk
	if _, err := os.Stat(s.Path("delete-me.txt")); !os.IsNotExist(err) {
		t.Error("file should not exist on disk")
	}
}

func TestDelete_NotExist(t *testing.T) {
	s := newTestStore(t)
	err := s.Delete("nonexistent.txt")
	if err != ErrNotExist {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}

func TestDelete_CleansMissingFile(t *testing.T) {
	s := newTestStore(t)
	mustSave(t, s, "ghost.txt", "i'll be removed from disk manually")

	// Manually remove the file from disk (simulating a rogue delete)
	if err := os.Remove(s.Path("ghost.txt")); err != nil {
		t.Fatal(err)
	}

	// Delete should still succeed and clean up meta
	if err := s.Delete("ghost.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	list, _ := s.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

// ---------------------------------------------------------------------------
// ReadContent
// ---------------------------------------------------------------------------

func TestReadContent(t *testing.T) {
	s := newTestStore(t)
	mustSave(t, s, "readme.txt", "readable content")

	content, err := s.ReadContent("readme.txt", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if content != "readable content" {
		t.Errorf("content = %q, want %q", content, "readable content")
	}
}

func TestReadContent_MaxSize(t *testing.T) {
	s := newTestStore(t)
	mustSave(t, s, "long.txt", "hello world, this is longer than 10 bytes")

	content, err := s.ReadContent("long.txt", 10)
	if err != nil {
		t.Fatal(err)
	}
	if content != "hello worl" {
		t.Errorf("content = %q, want %q", content, "hello worl")
	}
}

func TestReadContent_NotExist(t *testing.T) {
	s := newTestStore(t)
	_, err := s.ReadContent("nope.txt", 100)
	if err != ErrNotExist {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Path
// ---------------------------------------------------------------------------

func TestPath(t *testing.T) {
	dir := t.TempDir()
	s, err := NewFileSystemStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	got := s.Path("myfile.txt")
	want := filepath.Join(dir, "myfile.txt")
	if got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// detectFileType
// ---------------------------------------------------------------------------

func TestDetectFileType(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want FileType
	}{
		{"text", []byte("plain text"), FileTypeText},
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, FileTypeImage},
		{"png", []byte{0x89, 0x50, 0x4E, 0x47}, FileTypeImage},
		{"gif", []byte{0x47, 0x49, 0x46, 0x38}, FileTypeImage},
		{"webp", []byte{0x52, 0x49, 0x46, 0x46}, FileTypeImage},
		{"mp4", append([]byte{0x00, 0x00, 0x00, 0x00}, []byte("ftypmp4")...), FileTypeVideo},
		{"json", []byte(`{"key":"value"}`), FileTypeText},
		{"html", []byte("<!DOCTYPE html>"), FileTypeText},
		{"binary", []byte{0x00, 0x01, 0x02}, FileTypeOther},
		{"empty", []byte{}, FileTypeOther},
	}

	for _, tt := range tests {
		got := httpDetectContentType(tt.data)
		gotType := categorizeMIME(got)
		if gotType != tt.want {
			t.Errorf("%s: httpDetectContentType = %q, categorized as %q, want %q",
				tt.name, got, gotType, tt.want)
		}
	}
}

// categorizeMIME maps a MIME type string to a FileType (duplicates the logic
// in detectFileType so we can test httpDetectContentType directly).
func categorizeMIME(mime string) FileType {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return FileTypeImage
	case strings.HasPrefix(mime, "video/"):
		return FileTypeVideo
	case strings.HasPrefix(mime, "text/"), mime == "application/json":
		return FileTypeText
	default:
		return FileTypeOther
	}
}

// ---------------------------------------------------------------------------
// Concurrency (no data races)
// ---------------------------------------------------------------------------

func TestConcurrentAccess(t *testing.T) {
	s := newTestStore(t)
	var wg sync.WaitGroup

	// Concurrent saves
	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := fmt.Sprintf("concurrent-%d.txt", n)
			_, err := s.Save(name, strings.NewReader(name))
			if err != nil {
				t.Errorf("concurrent save %d: %v", n, err)
			}
		}(i)
	}
	wg.Wait()

	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 10 {
		t.Errorf("expected 10 files, got %d", len(list))
	}
}
