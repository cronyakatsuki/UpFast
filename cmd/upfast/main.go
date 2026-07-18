// Command upfast is a minimal file upload and sharing server.
//
// Usage:
//
//	upfast -p 1323 -a 127.0.0.1 -d https://upfast.example.com
package main

import (
	"flag"
	"log/slog"
	"os"
	"strconv"

	"cronyakatsuki/upfast"
	"cronyakatsuki/upfast/internal/server"
)

func main() {
	p := flag.Int("p", 1323, "port to listen on")
	a := flag.String("a", "127.0.0.1", "IP address to listen on")
	d := flag.String("d", "127.0.0.1", "public domain name (returned in upload URLs)")
	flag.Parse()

	// Parse HTML templates and get static FS from embedded assets
	tmpl := assets.ParseTemplates()
	staticFS := assets.StaticFileSystem()

	cfg := server.Config{
		Host:   *a,
		Port:   *p,
		Domain: *d,
	}

	srv, err := server.New(cfg, tmpl, staticFS)
	if err != nil {
		slog.Error("failed to create server", "err", err)
		os.Exit(1)
	}

	slog.Info("starting server",
		"addr", cfg.Host+":"+strconv.Itoa(cfg.Port),
		"domain", cfg.Domain,
	)

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
