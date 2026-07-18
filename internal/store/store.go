// Package store provides file storage abstractions for UpFast.
//
// The Store interface lets handlers work with files without worrying
// about whether they're stored on disk, in a database, or in the cloud.
// The FileSystemStore implementation saves files to a local directory
// and keeps a meta.json sidecar for fast listing without re-sniffing
// content types.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileType represents the broad category of a file for display purposes.
type FileType string

const (
	FileTypeImage FileType = "image"
	FileTypeVideo FileType = "video"
	FileTypeText  FileType = "text"
	FileTypeOther FileType = "other"
)

// FileInfo holds metadata about a stored file.
type FileInfo struct {
	Name       string   `json:"name"`
	FileType   FileType `json:"fileType"`
	Size       int64    `json:"size"`
	UploadedAt int64    `json:"uploadedAt"` // Unix timestamp (microseconds)
}

// Store defines the interface for file storage operations.
// Handlers depend on this interface, not the concrete implementation.
type Store interface {
	// Save stores a file with the given filename and returns its metadata.
	// If a file with the same name already exists, it returns ErrExists.
	Save(filename string, src io.Reader) (*FileInfo, error)

	// List returns all stored file metadata, newest first.
	List() ([]FileInfo, error)

	// ListPaginated returns a subset of stored file metadata for pagination.
	// offset is the starting index (0-based), limit is the max number to return.
	ListPaginated(offset, limit int) ([]FileInfo, error)

	// Delete removes a file by filename. Returns ErrNotExist if not found.
	Delete(filename string) error

	// Path returns the full filesystem path for a stored filename.
	Path(filename string) string

	// ReadContent reads a stored file's content up to maxSize bytes.
	// Returns the content as a string. If the file doesn't exist, returns ErrNotExist.
	ReadContent(filename string, maxSize int64) (string, error)
}

// Common sentinel errors.
var (
	ErrExists   = errors.New("file already exists")
	ErrNotExist = errors.New("file does not exist")
)

// FileSystemStore stores files in a local directory with a meta.json sidecar
// for fast metadata lookups without re-reading file headers.
type FileSystemStore struct {
	dir      string
	metaPath string
	mu       sync.RWMutex
}

// NewFileSystemStore creates a store rooted at dir, creating it if needed.
func NewFileSystemStore(dir string) (*FileSystemStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	s := &FileSystemStore{
		dir:      dir,
		metaPath: filepath.Join(dir, "meta.json"),
	}

	// Ensure a meta.json exists (empty array if not)
	if _, err := os.Stat(s.metaPath); errors.Is(err, fs.ErrNotExist) {
		if err := s.writeMeta([]FileInfo{}); err != nil {
			return nil, fmt.Errorf("init meta.json: %w", err)
		}
	}

	return s, nil
}

// Save stores a file and records its metadata.
// The content type is detected from the first 512 bytes (http.DetectContentType).
func (s *FileSystemStore) Save(filename string, src io.Reader) (*FileInfo, error) {
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		return nil, fmt.Errorf("invalid filename: %s", filename)
	}

	dstPath := filepath.Join(s.dir, filename)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicates using metadata
	meta, err := s.readMeta()
	if err != nil {
		return nil, fmt.Errorf("read meta: %w", err)
	}
	for _, f := range meta {
		if f.Name == filename {
			return nil, ErrExists
		}
	}

	// Write the file to disk
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer func() {
		if err := dst.Close(); err != nil {
			slog.Warn("closing file after write", "path", dstPath, "err", err)
		}
	}()

	written, err := io.Copy(dst, src)
	if err != nil {
		// Clean up partial write on error
		if rmErr := os.Remove(dstPath); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			slog.Warn("cleaning up partial write", "path", dstPath, "err", rmErr)
		}
		return nil, fmt.Errorf("write file: %w", err)
	}

	// Detect content type from the file on disk (read first 512 bytes)
	ft := detectFileType(dstPath)

	info := &FileInfo{
		Name:       filename,
		FileType:   ft,
		Size:       written,
		UploadedAt: time.Now().UnixMicro(),
	}

	meta = append(meta, *info)
	if err := s.writeMeta(meta); err != nil {
		// Metadata write failed — clean up the file
		if rmErr := os.Remove(dstPath); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			slog.Warn("cleaning up file after meta write failure", "path", dstPath, "err", rmErr)
		}
		return nil, fmt.Errorf("write meta: %w", err)
	}

	return info, nil
}

// List returns all files sorted newest-first.
func (s *FileSystemStore) List() ([]FileInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readMeta()
}

// ListPaginated returns a slice of files for offset/limit pagination.
// The full list is sorted newest-first, then we slice.
func (s *FileSystemStore) ListPaginated(offset, limit int) ([]FileInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meta, err := s.readMeta()
	if err != nil {
		return nil, err
	}

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}

	if offset >= len(meta) {
		return []FileInfo{}, nil
	}

	end := offset + limit
	if end > len(meta) {
		end = len(meta)
	}

	return meta[offset:end], nil
}

// Delete removes a file and its metadata entry.
func (s *FileSystemStore) Delete(filename string) error {
	dstPath := filepath.Join(s.dir, filename)

	s.mu.Lock()
	defer s.mu.Unlock()

	meta, err := s.readMeta()
	if err != nil {
		return fmt.Errorf("read meta: %w", err)
	}

	idx := -1
	for i, f := range meta {
		if f.Name == filename {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrNotExist
	}

	if err := os.Remove(dstPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// File missing from disk but in meta — clean meta and proceed
			slog.Warn("file missing from disk, cleaning metadata", "filename", filename)
		} else {
			return fmt.Errorf("remove file: %w", err)
		}
	}

	meta = append(meta[:idx], meta[idx+1:]...)
	return s.writeMeta(meta)
}

// Path returns the full filesystem path for a filename.
func (s *FileSystemStore) Path(filename string) string {
	return filepath.Join(s.dir, filename)
}

// ReadContent reads file content up to maxSize bytes. Returns ErrNotExist if file doesn't exist.
func (s *FileSystemStore) ReadContent(filename string, maxSize int64) (string, error) {
	path := filepath.Join(s.dir, filename)

	// Verify file exists in metadata
	s.mu.RLock()
	meta, err := s.readMeta()
	s.mu.RUnlock()
	if err != nil {
		return "", err
	}
	found := false
	for _, f := range meta {
		if f.Name == filename {
			found = true
			break
		}
	}
	if !found {
		return "", ErrNotExist
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", ErrNotExist
		}
		return "", fmt.Errorf("read file: %w", err)
	}

	if int64(len(data)) > maxSize {
		data = data[:maxSize]
	}

	return string(data), nil
}

// RebuildMeta scans the store directory and rebuilds meta.json from the
// actual files on disk. This is useful for migrating from the old flat
// storage format (where files existed without metadata).
//
// Files already in meta.json are preserved (their metadata is kept).
// Files on disk without a meta entry get their type detected and are added.
// Meta entries without a corresponding file on disk are removed.
func (s *FileSystemStore) RebuildMeta() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read existing meta (ignore errors — start fresh if corrupt)
	meta, _ := s.readMeta()
	if meta == nil {
		meta = []FileInfo{}
	}

	// Build a lookup of existing entries
	existing := make(map[string]bool)
	for _, f := range meta {
		existing[f.Name] = true
	}

	// Read all entries in the directory
	dirEntries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range dirEntries {
		name := entry.Name()

		// Skip meta.json and temp files
		if name == "meta.json" || name == "meta.json.tmp" || name == "meta.json.tmp.tmp" {
			continue
		}

		if entry.IsDir() {
			continue
		}

		if existing[name] {
			// Already in metadata — keep it
			continue
		}

		info, err := entry.Info()
		if err != nil {
			slog.Warn("reading file info", "file", name, "err", err)
			continue
		}

		path := filepath.Join(s.dir, name)
		ft := detectFileType(path)

		meta = append(meta, FileInfo{
			Name:       name,
			FileType:   ft,
			Size:       info.Size(),
			UploadedAt: info.ModTime().UnixMicro(),
		})

		slog.Info("migrated file", "name", name, "type", ft, "size", info.Size())
	}

	// Remove meta entries without corresponding files on disk
	diskFiles := make(map[string]bool)
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			diskFiles[entry.Name()] = true
		}
	}

	cleaned := make([]FileInfo, 0, len(meta))
	for _, f := range meta {
		if f.Name == "meta.json" || f.Name == "meta.json.tmp" || f.Name == "meta.json.tmp.tmp" || diskFiles[f.Name] {
			cleaned = append(cleaned, f)
		} else {
			slog.Info("removed stale meta entry (file missing)", "name", f.Name)
		}
	}

	return s.writeMeta(cleaned)
}

// readMeta reads and returns the full metadata list, sorted newest-first.
func (s *FileSystemStore) readMeta() ([]FileInfo, error) {
	data, err := os.ReadFile(s.metaPath)
	if err != nil {
		return nil, fmt.Errorf("read meta.json: %w", err)
	}

	var meta []FileInfo
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse meta.json: %w", err)
	}

	// Ensure consistent ordering: newest first
	sort.Slice(meta, func(i, j int) bool {
		return meta[i].UploadedAt > meta[j].UploadedAt
	})

	return meta, nil
}

// writeMeta atomically writes the metadata list to disk.
func (s *FileSystemStore) writeMeta(meta []FileInfo) error {
	// Sort newest-first before saving
	sort.Slice(meta, func(i, j int) bool {
		return meta[i].UploadedAt > meta[j].UploadedAt
	})

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}

	// Write to a temp file first, then rename for atomicity
	tmpPath := s.metaPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write meta.tmp: %w", err)
	}
	if err := os.Rename(tmpPath, s.metaPath); err != nil {
		if rmErr := os.Remove(tmpPath); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			slog.Warn("cleaning up temp meta file", "path", tmpPath, "err", rmErr)
		}
		return fmt.Errorf("rename meta: %w", err)
	}

	return nil
}

// detectFileType reads the first 512 bytes of a file and categorizes it.
func detectFileType(path string) FileType {
	f, err := os.Open(path)
	if err != nil {
		slog.Warn("could not open file for type detection", "path", path, "err", err)
		return FileTypeOther
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Warn("closing file after type detection", "path", path, "err", err)
		}
	}()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	buf = buf[:n]

	// http.DetectContentType uses the first bytes to sniff MIME type
	mimeType := httpDetectContentType(buf)

	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return FileTypeImage
	case strings.HasPrefix(mimeType, "video/"):
		return FileTypeVideo
	case strings.HasPrefix(mimeType, "text/"):
		return FileTypeText
	case mimeType == "application/json":
		return FileTypeText
	default:
		return FileTypeOther
	}
}

// httpDetectContentType is http.DetectContentType but we implement it inline
// to avoid importing net/http in the store package (keeps separation clean).
// This implements the same algorithm: https://mimesniff.spec.whatwg.org/
func httpDetectContentType(data []byte) string {
	if len(data) == 0 {
		return "application/octet-stream"
	}

	// Check for HTML
	if len(data) >= 2 && data[0] == '<' {
		if len(data) >= 4 && (data[1] == '!' || data[1] == '?' ||
			data[1] == 'h' || data[1] == 'H') {
			return "text/html; charset=utf-8"
		}
	}

	// Image magic bytes
	signatures := []struct {
		offset int
		bytes  []byte
		mime   string
	}{
		{0, []byte{0xFF, 0xD8, 0xFF}, "image/jpeg"},
		{0, []byte{0x89, 0x50, 0x4E, 0x47}, "image/png"},
		{0, []byte{0x47, 0x49, 0x46}, "image/gif"},
		{0, []byte{0x52, 0x49, 0x46, 0x46}, "image/webp"},
		{0, []byte{0x42, 0x4D}, "image/bmp"},
		{0, []byte{0x00, 0x00, 0x01, 0x00}, "image/x-icon"},
		{0, []byte{0x00, 0x00, 0x00, 0x0C, 0x6A, 0x50, 0x20, 0x20}, "image/jp2"},
		// Video
		{4, []byte{0x66, 0x74, 0x79, 0x70}, "video/mp4"},
		{0, []byte{0x1A, 0x45, 0xDF, 0xA3}, "video/webm"},
		{0, []byte{0x00, 0x00, 0x01, 0xBA}, "video/mpeg"},
		{0, []byte{0x00, 0x00, 0x01, 0xB3}, "video/mpeg"},
		{0, []byte{0x2E, 0x52, 0x4D, 0x46}, "video/x-msvideo"},
		// Audio
		{0, []byte{0x49, 0x44, 0x33}, "audio/mpeg"},
		{0, []byte{0xFF, 0xFB}, "audio/mpeg"},
		{0, []byte{0x4F, 0x67, 0x67, 0x53}, "audio/ogg"},
		{0, []byte{0x52, 0x49, 0x46, 0x46}, "audio/wav"},
		// Archive
		{0, []byte{0x50, 0x4B, 0x03, 0x04}, "application/zip"},
		{0, []byte{0x1F, 0x8B}, "application/gzip"},
		{0, []byte{0x42, 0x5A, 0x68}, "application/bzip2"},
		{257, []byte{0x75, 0x73, 0x74, 0x61, 0x72}, "application/x-tar"},
	}

	for _, sig := range signatures {
		if len(data) >= sig.offset+len(sig.bytes) {
			match := true
			for i, b := range sig.bytes {
				if data[sig.offset+i] != b {
					match = false
					break
				}
			}
			if match {
				return sig.mime
			}
		}
	}

	// Check if printable text (simple check)
	isText := true
	for _, b := range data {
		if b == 0 {
			isText = false
			break
		}
	}
	if isText {
		return "text/plain; charset=utf-8"
	}

	return "application/octet-stream"
}
