# UpFast

Minimal file upload and sharing tool. Influenced by [0x0.st](https://0x0.st).

Upload files via curl or browser, get a shareable URL back. Files are listed
with previews for images, videos, and text.

```bash
curl -F "file=@/path/to/image.jpg" https://upfast.example.com
# → https://upfast.example.com/files/image.jpg
```

## Features

- **Single binary** — zero runtime dependencies, pure Go
- **Upload** via curl (`-F "file=@..."`) or browser form
- **File listing** with inline previews (images, videos, text)
- **Infinite scroll** on the file listing page (loads 20 files at a time)
- **curl-aware** — returns plain text for curl, HTML for browsers
- **Delete** files via `curl -X DELETE`
- **JSON API** — `GET /api/files?offset=0&limit=20` for pagination
- **Metadata sidecar** — file types detected once at upload, saved in `meta.json`
- **No external dependencies** — uses only Go standard library (Go 1.23+)

## Quick start

```bash
# Build
go build -o upfast ./cmd/upfast

# Run (default: http://127.0.0.1:1323)
./upfast

# Or with options
./upfast -p 8080 -a 0.0.0.0 -d https://upfast.example.com
```

Or using the Makefile:

```bash
make build   # -> bin/upfast
make run     # -> runs on :1323
make run ARGS="-p 8080 -a 0.0.0.0 -d https://upfast.example.com"
```

## Usage

### Upload a file

```bash
curl -F "file=@/path/to/image.jpg" http://127.0.0.1:1323
# → http://127.0.0.1:1323/files/image.jpg
```

### List all files (plain text)

```bash
curl http://127.0.0.1:1323/files
# → image.jpg
# → notes.txt
```

### List via browser

Open `http://127.0.0.1:1323/files` in a browser. The page loads the first 20
files and automatically loads more as you scroll.

### Delete a file

```bash
curl -X DELETE http://127.0.0.1:1323/files/image.jpg
# → Deleted file from server: image.jpg
```

### JSON API (for custom clients)

```bash
curl 'http://127.0.0.1:1323/api/files?offset=0&limit=10'
# → {"files":[{"name":"image.jpg","fileType":"image","url":"...","size":12345}],"hasMore":false,"offset":0,"limit":10}
```

## Development

### Prerequisites

- Go 1.23 or later
- Optionally: [Nix](https://nixos.org/) for the dev shell

### Commands

```bash
make all        # fmt → vet → lint → test → build
make test       # run all tests
make test-race  # run tests with race detector
make test-cover # run tests with coverage report
make lint       # run golangci-lint
make fmt        # format code with gofumpt
make clean      # remove build artifacts
```

### Nix shell

A `flake.nix` and `shell.nix` are provided for a reproducible development
environment:

```bash
nix develop          # enter dev shell (Go 1.26, golangci-lint, gopls, etc.)
nix build            # build the package
nix flake check      # run all checks (build + lint)
```

### Project structure

```
├── assets.go              ← Embedded templates and static files
├── cmd/
│   └── upfast/
│       └── main.go        ← Entry point
├── internal/
│   ├── handler/           ← HTTP handlers (one file per endpoint)
│   │   ├── handler.go     ← Shared Handler struct + helpers
│   │   ├── index.go       ← GET /
│   │   ├── upload.go      ← POST /
│   │   ├── files.go       ← GET /files/ + GET /api/files
│   │   ├── delete.go      ← DELETE /files/{file}
│   │   └── handler_test.go
│   ├── server/
│   │   └── server.go      ← Route setup, middleware, Config
│   └── store/
│       ├── store.go       ← Store interface + FileSystemStore
│       └── store_test.go
├── public/views/           ← HTML templates (index, files)
├── static/css/             ← Stylesheets
├── go.mod                  ← Go 1.23, zero dependencies
├── Makefile
├── flake.nix / shell.nix   ← Nix development environment
└── README.md
```

## Deployment

### Option A: Direct binary

```bash
# Build on the target machine
go build -o /usr/local/bin/upfast ./cmd/upfast

# Or cross-compile from your dev machine
GOOS=linux GOARCH=amd64 go build -o upfast-linux-amd64 ./cmd/upfast
```

### Option B: Nix

```bash
nix build           # produces result/bin/upfast
cp result/bin/upfast /usr/local/bin/
```

### Production hardening

Create a dedicated user and limit storage with a virtual filesystem:

```bash
# Create upfast user
useradd --shell /usr/sbin/nologin --system -M upfast
usermod -L upfast

# Limit storage with a 20GB loop filesystem
dd if=/dev/zero of=/usr/local/upfast/storage.img bs=1M count=20480
mkfs.ext4 /usr/local/upfast/storage.img
mount -o loop,rw /usr/local/upfast/storage.img /usr/local/upfast/data

# Add to /etc/fstab:
# /usr/local/upfast/storage.img /usr/local/upfast/data ext4 loop,rw,usrquota,grpquota 0 0

# Set ownership
chown upfast:upfast -R /usr/local/upfast
chmod 700 /usr/local/upfast
```

### Systemd service

Create `/etc/systemd/system/upfast.service`:

```ini
[Unit]
Description=UpFast file sharing service
Documentation=https://github.com/cronyakatsuki/upfast
After=network.target

[Service]
User=upfast
Group=upfast
WorkingDirectory=/usr/local/upfast/data
ExecStart=/usr/local/upfast/upfast -p 8000 -a 127.0.0.1 -d https://upfast.example.com
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable --now upfast
```

### Nginx reverse proxy

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name upfast.example.com;

    client_max_body_size 0;  # no upload size limit

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## How it works

Files are stored in a `files/` directory. Metadata (filename, type, size,
upload timestamp) is kept in a `files/meta.json` sidecar for fast listing
without re-reading every file's header.

When a file is uploaded, its content type is detected from the first 512
bytes using magic byte signatures (JPEG, PNG, GIF, MP4, etc.). This
happens once at upload time, not on every listing request.

The server detects curl via the User-Agent header and returns plain text
responses instead of HTML.

## License

MIT
