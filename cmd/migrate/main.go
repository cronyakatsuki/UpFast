// Command migrate rebuilds meta.json for an existing files/ directory.
//
// It scans the directory, detects file types, and writes a fresh meta.json
// that the UpFast server can read.
//
// Usage:
//
//	migrate -dir ./files
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"cronyakatsuki/upfast/internal/store"
)

func main() {
	dir := flag.String("dir", "files", "path to the files directory")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	s, err := store.NewFileSystemStore(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open store: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Scanning %s for files to migrate...\n", *dir)

	if err := s.RebuildMeta(); err != nil {
		fmt.Fprintf(os.Stderr, "error: rebuild failed: %v\n", err)
		os.Exit(1)
	}

	list, err := s.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: listing after rebuild: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Done. %d files registered in metadata.\n", len(list))
}
