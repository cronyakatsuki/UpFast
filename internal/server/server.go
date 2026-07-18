// Package server wires up the HTTP server, routes, and middleware.
//
// It's deliberately thin — just route registration and middleware setup.
// All handler logic lives in the handler package.
package server

import (
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"cronyakatsuki/upfast/internal/handler"
	"cronyakatsuki/upfast/internal/store"
)

// Config holds server configuration parameters.
type Config struct {
	Host   string // IP to listen on (e.g., "127.0.0.1")
	Port   int    // Port to listen on (e.g., 1323)
	Domain string // Public domain/URL prefix (e.g., "https://upfast.example.com")
}

// New creates and returns an *http.Server fully wired with routes.
// It takes a pre-parsed template and a static file system (both embedded from main.go).
func New(cfg Config, tmpl *template.Template, staticFS http.FileSystem) (*http.Server, error) {
	// Initialize the file store
	filesDir := "files"
	st, err := store.NewFileSystemStore(filesDir)
	if err != nil {
		return nil, err
	}

	// Initialize handlers with the pre-parsed template
	h := handler.New(st, cfg.Domain, tmpl)

	mux := http.NewServeMux()

	// Serve embedded static assets (CSS, JS)
	mux.Handle("GET /css/", http.FileServer(staticFS))
	mux.Handle("GET /js/", http.FileServer(staticFS))

	// Serve uploaded files directly from the files directory
	mux.Handle("GET /files/", http.StripPrefix("/files/",
		http.FileServer(http.Dir(filesDir)),
	))

	// API endpoints
	mux.HandleFunc("GET /api/files", h.APIFiles)

	// Main routes
	mux.HandleFunc("GET /", h.Index)
	mux.HandleFunc("POST /", h.Upload)
	mux.HandleFunc("GET /files", h.Files)
	mux.HandleFunc("DELETE /files/{file}", h.Delete)

	// Logging middleware
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	wrappedMux := withLogging(logger, mux)

	addr := cfg.Host + ":" + itoa(cfg.Port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      wrappedMux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return srv, nil
}

// withLogging wraps an http.Handler with request logging via slog.
func withLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start),
			"ua", r.UserAgent(),
		)
	})
}

// statusWriter captures the HTTP status code for logging.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// itoa is a small helper to avoid importing strconv just for this.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
