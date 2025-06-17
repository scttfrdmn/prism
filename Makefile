# CloudWorkstation Makefile
# Builds both daemon and CLI client

VERSION := 0.1.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# Default target
.PHONY: all
all: build

# Build both binaries
.PHONY: build
build: build-daemon build-cli

# Build daemon binary
.PHONY: build-daemon
build-daemon:
	@echo "Building CloudWorkstation daemon..."
	@go build $(LDFLAGS) -o bin/cwsd ./cmd/cwsd

# Build CLI binary  
.PHONY: build-cli
build-cli:
	@echo "Building CloudWorkstation CLI..."
	@go build $(LDFLAGS) -o bin/cws ./cmd/cws

# Install binaries to system
.PHONY: install
install: build
	@echo "Installing CloudWorkstation..."
	@sudo cp bin/cwsd /usr/local/bin/
	@sudo cp bin/cws /usr/local/bin/
	@echo "✅ CloudWorkstation installed successfully"
	@echo "Start daemon with: cwsd"
	@echo "Use CLI with: cws --help"

# Uninstall binaries from system
.PHONY: uninstall
uninstall:
	@echo "Uninstalling CloudWorkstation..."
	@sudo rm -f /usr/local/bin/cwsd
	@sudo rm -f /usr/local/bin/cws
	@echo "✅ CloudWorkstation uninstalled"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test ./...

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run --issues-exit-code=0

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Update dependencies
.PHONY: deps
deps:
	@echo "Updating dependencies..."
	@go mod tidy
	@go mod download

# Development: build and run daemon
.PHONY: dev-daemon
dev-daemon: build-daemon
	@echo "Starting daemon in development mode..."
	@./bin/cwsd

# Development: quick CLI test
.PHONY: dev-cli
dev-cli: build-cli
	@echo "Testing CLI..."
	@./bin/cws --help

# Create release builds for multiple platforms
.PHONY: release
release: clean
	@echo "Building release binaries..."
	@mkdir -p bin/release
	
	# Linux amd64
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/release/linux-amd64-cwsd ./cmd/cwsd
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/release/linux-amd64-cws ./cmd/cws
	
	# Linux arm64
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/release/linux-arm64-cwsd ./cmd/cwsd
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/release/linux-arm64-cws ./cmd/cws
	
	# macOS amd64
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/release/darwin-amd64-cwsd ./cmd/cwsd
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/release/darwin-amd64-cws ./cmd/cws
	
	# macOS arm64 (Apple Silicon)
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/release/darwin-arm64-cwsd ./cmd/cwsd
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/release/darwin-arm64-cws ./cmd/cws
	
	# Windows amd64
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/release/windows-amd64-cwsd.exe ./cmd/cwsd
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/release/windows-amd64-cws.exe ./cmd/cws
	
	@echo "✅ Release binaries built in bin/release/"

# Create bin directory
bin:
	@mkdir -p bin

# Ensure bin directory exists before building
build-daemon: bin
build-cli: bin

# Docker builds (future)
.PHONY: docker
docker:
	@echo "Docker support not implemented yet"

# Show version info
.PHONY: version
version:
	@echo "CloudWorkstation v$(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git commit: $(GIT_COMMIT)"

# Show help
.PHONY: help
help:
	@echo "CloudWorkstation Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build        Build both daemon and CLI"
	@echo "  build-daemon Build daemon binary (cwsd)"
	@echo "  build-cli    Build CLI binary (cws)"
	@echo "  install      Install binaries to /usr/local/bin"
	@echo "  uninstall    Remove binaries from /usr/local/bin"
	@echo "  clean        Remove build artifacts"
	@echo "  test         Run tests"
	@echo "  lint         Run linter"
	@echo "  fmt          Format code"
	@echo "  deps         Update dependencies"
	@echo "  release      Build release binaries for all platforms"
	@echo "  dev-daemon   Build and run daemon for development"
	@echo "  dev-cli      Build and test CLI"
	@echo "  version      Show version information"
	@echo "  help         Show this help"