# UpFast — minimal file upload and sharing tool
#
# Common commands:
#   make build     — build the binary
#   make run       — run the server (args pass via ARGS, e.g. ARGS="-p 8080")
#   make test      — run all tests
#   make lint      — run golangci-lint
#   make fmt       — format all Go files with gofumpt
#   make clean     — remove build artifacts
#   make all       — fmt -> vet -> lint -> test -> build

BINARY   ?= upfast
CMD_DIR  := ./cmd/upfast
OUT_DIR  := ./bin

# Allow overriding Go flags (e.g., make build GOFLAGS="-tags=netgo")
GO       ?= go
GOFLAGS  ?=
LDFLAGS  ?= -s -w

# Detect OS/ARCH for output naming
GOOS     ?= $(shell $(GO) env GOOS 2>/dev/null || echo linux)
GOARCH   ?= $(shell $(GO) env GOARCH 2>/dev/null || echo amd64)

.PHONY: all build run test test-race lint vet fmt tidy clean

all: fmt vet lint test build

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

build: $(OUT_DIR)/$(BINARY) $(OUT_DIR)/migrate

$(OUT_DIR)/$(BINARY): | $(OUT_DIR)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $@ $(CMD_DIR)

$(OUT_DIR)/migrate: | $(OUT_DIR)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/migrate

$(OUT_DIR):
	@mkdir -p $(OUT_DIR)

build-race:
	@mkdir -p $(OUT_DIR)
	$(GO) build $(GOFLAGS) -race -ldflags="$(LDFLAGS)" -o $(OUT_DIR)/$(BINARY)-race $(CMD_DIR)

# ---------------------------------------------------------------------------
# Run
# ---------------------------------------------------------------------------

run: build
	@echo "Starting $(BINARY)..."
	$(OUT_DIR)/$(BINARY) $(ARGS)

# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------

test:
	$(GO) test $(GOFLAGS) -count=1 ./...

test-race:
	$(GO) test $(GOFLAGS) -race -count=1 ./...

test-verbose:
	$(GO) test $(GOFLAGS) -v -count=1 ./...

test-cover:
	$(GO) test $(GOFLAGS) -cover -coverprofile=coverage.out -count=1 ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ---------------------------------------------------------------------------
# Lint & Vet
# ---------------------------------------------------------------------------

lint:
	golangci-lint run

vet:
	$(GO) vet $(GOFLAGS) ./...

# ---------------------------------------------------------------------------
# Format
# ---------------------------------------------------------------------------

fmt:
	gofumpt -l -w .

tidy:
	$(GO) mod tidy

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------

clean:
	rm -rf $(OUT_DIR) coverage.out coverage.html
