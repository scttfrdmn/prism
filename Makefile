# CloudWorkstation Makefile
# Builds both daemon and CLI client

VERSION := 0.1.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=$(VERSION) -X github.com/scttfrdmn/cloudworkstation/pkg/version.BuildDate=$(BUILD_TIME) -X github.com/scttfrdmn/cloudworkstation/pkg/version.GitCommit=$(GIT_COMMIT)"

# Default target
.PHONY: all
all: build

# Build all binaries
.PHONY: build
build: build-daemon build-cli build-gui

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

# Build GUI binary
.PHONY: build-gui
build-gui:
	@echo "Building CloudWorkstation GUI..."
	@go build $(LDFLAGS) -o bin/cws-gui ./cmd/cws-gui

# Install binaries to system
.PHONY: install
install: build
	@echo "Installing CloudWorkstation..."
	@sudo cp bin/cwsd /usr/local/bin/
	@sudo cp bin/cws /usr/local/bin/
	@sudo cp bin/cws-gui /usr/local/bin/
	@echo "✅ CloudWorkstation installed successfully"
	@echo "Start daemon with: cwsd"
	@echo "Use CLI with: cws --help"
	@echo "Use GUI with: cws-gui"

# Uninstall binaries from system
.PHONY: uninstall
uninstall:
	@echo "Uninstalling CloudWorkstation..."
	@sudo rm -f /usr/local/bin/cwsd
	@sudo rm -f /usr/local/bin/cws
	@sudo rm -f /usr/local/bin/cws-gui
	@echo "✅ CloudWorkstation uninstalled"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

# Test targets
.PHONY: test test-unit test-integration test-e2e test-coverage test-all

# Run all unit tests
test-unit:
	@echo "🧪 Running unit tests..."
	@go test -race -short ./... -coverprofile=unit-coverage.out

# Run integration tests with LocalStack
test-integration:
	@echo "🔗 Running integration tests..."
	@docker-compose -f docker-compose.test.yml up -d localstack
	@echo "⏳ Waiting for LocalStack to be ready..."
	@sleep 10
	@INTEGRATION_TESTS=1 go test -tags=integration ./pkg/aws -v -coverprofile=integration-coverage.out
	@docker-compose -f docker-compose.test.yml down

# Run end-to-end tests
test-e2e: build
	@echo "🎯 Running end-to-end tests..."
	@E2E_TESTS=1 go test -tags=e2e ./e2e -v -timeout=30m

# Generate comprehensive coverage report
test-coverage:
	@echo "📊 Generating coverage report..."
	@go test ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total
	@echo "📋 Coverage report generated: coverage.html"

# Run all tests (unit + integration + e2e)
test-all: test-unit test-integration test-e2e test-coverage

# Legacy test target for backwards compatibility
test: test-unit

# Validate entire build and test pipeline
.PHONY: validate
validate:
	@echo "🔧 Running CloudWorkstation validation pipeline..."
	@./scripts/validate.sh

# Quality gates
.PHONY: quality-check vet security check-docs

# Run all quality checks
quality-check: fmt vet lint security check-docs test-coverage
	@echo "✅ All quality checks passed!"

# Check documentation standards
check-docs:
	@echo "📚 Checking documentation standards..."
	@./scripts/check-docs.sh

# Enhanced linting
.PHONY: lint
lint:
	@echo "🔍 Running linter..."
	@golangci-lint run --issues-exit-code=1 --timeout=5m

# Vet code
vet:
	@echo "🔎 Running go vet..."
	@go vet ./...

# Security scan
security:
	@echo "🔒 Running security scan..."
	@gosec -quiet ./...

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

# Pre-commit simulation
.PHONY: pre-commit
pre-commit: quality-check test-unit
	@echo "🚀 Pre-commit checks complete!"

# CI/CD targets
.PHONY: ci-test ci-coverage ci-build

# Full CI test suite
ci-test:
	@echo "🤖 Running CI test suite..."
	@make quality-check
	@make test-unit
	@make test-integration
	@make build

# CI coverage enforcement
ci-coverage:
	@echo "📊 Checking CI coverage requirements..."
	@./scripts/check-coverage.sh

# CI build verification
ci-build:
	@echo "🏗️ Verifying CI build..."
	@make clean
	@make build
	@make test-unit

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

# Version bumping targets following SemVer
.PHONY: bump-major bump-minor bump-patch

# Bump major version (e.g., 1.2.3 -> 2.0.0)
bump-major:
	@echo "Bumping major version..."
	$(eval MAJOR := $(shell echo $(VERSION) | cut -d. -f1))
	$(eval NEW_VERSION := $$(( $(MAJOR) + 1 )).0.0)
	@sed -i.bak "s/VERSION := $(VERSION)/VERSION := $(NEW_VERSION)/" Makefile
	@sed -i.bak "s/Version = \"$(VERSION)\"/Version = \"$(NEW_VERSION)\"/" pkg/version/version.go
	@echo "✅ Version bumped from $(VERSION) to $(NEW_VERSION)"
	@echo "Don't forget to update the CHANGELOG.md!"
	@rm -f Makefile.bak pkg/version/version.go.bak

# Bump minor version (e.g., 1.2.3 -> 1.3.0)
bump-minor:
	@echo "Bumping minor version..."
	$(eval MAJOR := $(shell echo $(VERSION) | cut -d. -f1))
	$(eval MINOR := $(shell echo $(VERSION) | cut -d. -f2))
	$(eval NEW_VERSION := $(MAJOR).$$(( $(MINOR) + 1 )).0)
	@sed -i.bak "s/VERSION := $(VERSION)/VERSION := $(NEW_VERSION)/" Makefile
	@sed -i.bak "s/Version = \"$(VERSION)\"/Version = \"$(NEW_VERSION)\"/" pkg/version/version.go
	@echo "✅ Version bumped from $(VERSION) to $(NEW_VERSION)"
	@echo "Don't forget to update the CHANGELOG.md!"
	@rm -f Makefile.bak pkg/version/version.go.bak

# Bump patch version (e.g., 1.2.3 -> 1.2.4)
bump-patch:
	@echo "Bumping patch version..."
	$(eval MAJOR := $(shell echo $(VERSION) | cut -d. -f1))
	$(eval MINOR := $(shell echo $(VERSION) | cut -d. -f2))
	$(eval PATCH := $(shell echo $(VERSION) | cut -d. -f3))
	$(eval NEW_VERSION := $(MAJOR).$(MINOR).$$(( $(PATCH) + 1 )))
	@sed -i.bak "s/VERSION := $(VERSION)/VERSION := $(NEW_VERSION)/" Makefile
	@sed -i.bak "s/Version = \"$(VERSION)\"/Version = \"$(NEW_VERSION)\"/" pkg/version/version.go
	@echo "✅ Version bumped from $(VERSION) to $(NEW_VERSION)"
	@echo "Don't forget to update the CHANGELOG.md!"
	@rm -f Makefile.bak pkg/version/version.go.bak

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
	@echo "  test         Run unit tests (legacy)"
	@echo "  test-unit    Run unit tests"
	@echo "  test-integration Run integration tests with LocalStack"
	@echo "  test-e2e     Run end-to-end tests"
	@echo "  test-coverage Generate coverage report"
	@echo "  test-all     Run all tests"
	@echo "  validate     Validate entire build and test pipeline"
	@echo "  quality-check Run all quality checks"
	@echo "  lint         Run linter"
	@echo "  vet          Run go vet"
	@echo "  security     Run security scan"
	@echo "  pre-commit   Simulate pre-commit checks"
	@echo "  fmt          Format code"
	@echo "  deps         Update dependencies"
	@echo "  release      Build release binaries for all platforms"
	@echo "  dev-daemon   Build and run daemon for development"
	@echo "  dev-cli      Build and test CLI"
	@echo "  version      Show version information"
	@echo "  bump-major   Bump major version (X.y.z)"
	@echo "  bump-minor   Bump minor version (x.Y.z)"
	@echo "  bump-patch   Bump patch version (x.y.Z)"
	@echo "  help         Show this help"