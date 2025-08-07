# CloudWorkstation Makefile
# Builds both daemon and CLI client

VERSION := 0.4.1
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=$(VERSION) -X github.com/scttfrdmn/cloudworkstation/pkg/version.BuildDate=$(BUILD_TIME) -X github.com/scttfrdmn/cloudworkstation/pkg/version.GitCommit=$(GIT_COMMIT)"

# Default target
.PHONY: all
all: build

# Build all binaries (CLI, daemon, GUI, TUI integrated)
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

# Force GUI build (for development/testing only)
.PHONY: build-gui-force
build-gui-force:
	@echo "⚠️  Force building CloudWorkstation GUI (may fail)..."
	@go build $(LDFLAGS) -o bin/cws-gui ./cmd/cws-gui

# Install binaries to system
.PHONY: install
install: build
	@echo "Installing CloudWorkstation..."
	@sudo cp bin/cwsd /usr/local/bin/
	@sudo cp bin/cws /usr/local/bin/
	@echo "✅ CloudWorkStation core binaries installed successfully"
	@echo "Start daemon with: cwsd"
	@echo "Use CLI with: cws --help"
	@echo "Note: GUI installation available in Phase 2"

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

# Run all unit tests (excluding GUI/TUI packages planned for Phase 2)
test-unit:
	@echo "🧪 Running unit tests..."
	@go test -race -short $$(go list ./... | grep -v -E "(cmd/cws-gui|internal/gui|internal/tui)") -coverprofile=unit-coverage.out

# Run integration tests with LocalStack
test-integration:
	@echo "🔗 Running integration tests..."
	@docker-compose -f docker-compose.test.yml up -d localstack
	@echo "⏳ Waiting for LocalStack to be ready..."
	@sleep 10
	@INTEGRATION_TESTS=1 go test -tags=integration ./pkg/aws -v -coverprofile=aws-integration-coverage.out
	@INTEGRATION_TESTS=1 go test -tags=integration ./pkg/ami -v -coverprofile=ami-integration-coverage.out
	@docker-compose -f docker-compose.test.yml down
	@echo "📊 Integration test coverage:"
	@go tool cover -func=aws-integration-coverage.out | grep "total"
	@go tool cover -func=ami-integration-coverage.out | grep "total"

# Run AMI builder integration tests specifically
test-ami-builder:
	@echo "🧪 Running AMI builder integration tests..."
	@./scripts/test-ami-builder.sh

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
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/release/linux-amd64-cws-gui ./cmd/cws-gui
	
	# Linux arm64
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/release/linux-arm64-cwsd ./cmd/cwsd
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/release/linux-arm64-cws ./cmd/cws
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/release/linux-arm64-cws-gui ./cmd/cws-gui
	
	# macOS amd64
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/release/darwin-amd64-cwsd ./cmd/cwsd
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/release/darwin-amd64-cws ./cmd/cws
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/release/darwin-amd64-cws-gui ./cmd/cws-gui
	
	# macOS arm64 (Apple Silicon)
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/release/darwin-arm64-cwsd ./cmd/cwsd
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/release/darwin-arm64-cws ./cmd/cws
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/release/darwin-arm64-cws-gui ./cmd/cws-gui
	
	# Windows amd64
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/release/windows-amd64-cwsd.exe ./cmd/cwsd
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/release/windows-amd64-cws.exe ./cmd/cws
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/release/windows-amd64-cws-gui.exe ./cmd/cws-gui
	
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
build-gui: bin

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
# Package manager distribution targets
.PHONY: package package-homebrew package-chocolatey package-conda package-test

# Create all package distribution files
package: package-homebrew package-chocolatey package-conda

# Create test packages with dummy files for testing distribution
package-test:
	@echo "Creating test packages for distribution testing..."
	@mkdir -p bin/release
	@mkdir -p dist/{homebrew,chocolatey/tools,conda}
	
	# Create dummy binaries
	@echo "#!/bin/sh\necho \"CloudWorkstation CLI v0.4.1\"" > bin/release/darwin-amd64-cws
	@echo "#!/bin/sh\necho \"CloudWorkstation Daemon v0.4.1\"" > bin/release/darwin-amd64-cwsd
	@echo "#!/bin/sh\necho \"CloudWorkstation GUI v0.4.1\"" > bin/release/darwin-amd64-cws-gui
	@chmod +x bin/release/darwin-amd64-cws bin/release/darwin-amd64-cwsd bin/release/darwin-amd64-cws-gui
	
	# Create test archives
	@cd bin/release && tar -czf ../../dist/homebrew/cloudworkstation-darwin-amd64.tar.gz darwin-amd64-*
	@cp scripts/chocolatey/cloudworkstation.nuspec dist/chocolatey/
	@cp scripts/chocolatey/tools/chocolateyinstall.ps1 dist/chocolatey/tools/
	@cp scripts/chocolatey/tools/chocolateyuninstall.ps1 dist/chocolatey/tools/
	@cd bin/release && zip -j ../../dist/chocolatey/tools/cloudworkstation-windows-amd64.zip darwin-amd64-*
	@cp scripts/conda/meta.yaml dist/conda/
	
	# Generate test checksums
	@openssl sha256 dist/homebrew/cloudworkstation-darwin-amd64.tar.gz > dist/homebrew/darwin-amd64.sha256
	@openssl sha256 dist/chocolatey/tools/cloudworkstation-windows-amd64.zip > dist/chocolatey/windows-amd64.sha256
	
	@echo "✅ Test packages created in dist/ directory"

# Create Homebrew formula
package-homebrew: release
	@echo "Creating Homebrew package..."
	@mkdir -p dist/homebrew
	@cp scripts/homebrew/cloudworkstation.rb dist/homebrew/
	@cd bin/release && tar -czf ../../dist/homebrew/cloudworkstation-darwin-amd64.tar.gz darwin-amd64-*
	@cd bin/release && tar -czf ../../dist/homebrew/cloudworkstation-darwin-arm64.tar.gz darwin-arm64-*
	@cd bin/release && tar -czf ../../dist/homebrew/cloudworkstation-linux-amd64.tar.gz linux-amd64-*
	@cd bin/release && tar -czf ../../dist/homebrew/cloudworkstation-linux-arm64.tar.gz linux-arm64-*
	@openssl sha256 dist/homebrew/cloudworkstation-darwin-amd64.tar.gz | awk '{print $$2}' > dist/homebrew/darwin-amd64.sha256
	@openssl sha256 dist/homebrew/cloudworkstation-darwin-arm64.tar.gz | awk '{print $$2}' > dist/homebrew/darwin-arm64.sha256
	@openssl sha256 dist/homebrew/cloudworkstation-linux-amd64.tar.gz | awk '{print $$2}' > dist/homebrew/linux-amd64.sha256
	@openssl sha256 dist/homebrew/cloudworkstation-linux-arm64.tar.gz | awk '{print $$2}' > dist/homebrew/linux-arm64.sha256
	@echo "✅ Homebrew package created in dist/homebrew"

# Create Chocolatey package
package-chocolatey: release
	@echo "Creating Chocolatey package..."
	@mkdir -p dist/chocolatey/tools
	@cp scripts/chocolatey/cloudworkstation.nuspec dist/chocolatey/
	@cp scripts/chocolatey/tools/chocolateyinstall.ps1 dist/chocolatey/tools/
	@cp scripts/chocolatey/tools/chocolateyuninstall.ps1 dist/chocolatey/tools/
	@cd bin/release && zip -j ../../dist/chocolatey/tools/cloudworkstation-windows-amd64.zip windows-amd64-*
	@openssl sha256 dist/chocolatey/tools/cloudworkstation-windows-amd64.zip | awk '{print $$2}' > dist/chocolatey/windows-amd64.sha256
	@echo "✅ Chocolatey package created in dist/chocolatey"

# Create Conda package
package-conda: release
	@echo "Creating Conda package..."
	@mkdir -p dist/conda
	@cp scripts/conda/meta.yaml dist/conda/
	@cp scripts/conda/build.sh dist/conda/
	@cp scripts/conda/bld.bat dist/conda/
	@cd bin/release && tar -czf ../../dist/conda/cloudworkstation-linux-amd64.tar.gz linux-amd64-*
	@cd bin/release && tar -czf ../../dist/conda/cloudworkstation-linux-arm64.tar.gz linux-arm64-*
	@cd bin/release && tar -czf ../../dist/conda/cloudworkstation-darwin-amd64.tar.gz darwin-amd64-*
	@cd bin/release && tar -czf ../../dist/conda/cloudworkstation-darwin-arm64.tar.gz darwin-arm64-*
	@cd bin/release && zip -j ../../dist/conda/cloudworkstation-windows-amd64.zip windows-amd64-*
	@openssl sha256 dist/conda/cloudworkstation-linux-amd64.tar.gz | awk '{print $$2}' > dist/conda/linux-amd64.sha256
	@openssl sha256 dist/conda/cloudworkstation-linux-arm64.tar.gz | awk '{print $$2}' > dist/conda/linux-arm64.sha256
	@openssl sha256 dist/conda/cloudworkstation-darwin-amd64.tar.gz | awk '{print $$2}' > dist/conda/darwin-amd64.sha256
	@openssl sha256 dist/conda/cloudworkstation-darwin-arm64.tar.gz | awk '{print $$2}' > dist/conda/darwin-arm64.sha256
	@openssl sha256 dist/conda/cloudworkstation-windows-amd64.zip | awk '{print $$2}' > dist/conda/windows-amd64.sha256
	@echo "✅ Conda package created in dist/conda"

.PHONY: help
help:
	@echo "CloudWorkstation Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build        Build daemon, CLI, and GUI"
	@echo "  build-daemon Build daemon binary (cwsd)"
	@echo "  build-cli    Build CLI binary (cws)"
	@echo "  build-gui    Build GUI binary (cws-gui)"
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
	@echo "  package      Create all package manager distribution files"
	@echo "  package-homebrew Create Homebrew formula and packages"
	@echo "  package-chocolatey Create Chocolatey package"
	@echo "  package-conda Create Conda package"
	@echo "  dev-daemon   Build and run daemon for development"
	@echo "  dev-cli      Build and test CLI"
	@echo "  dev-gui      Build and test GUI"
	@echo "  version      Show version information"
	@echo "  bump-major   Bump major version (X.y.z)"
	@echo "  bump-minor   Bump minor version (x.Y.z)"
	@echo "  bump-patch   Bump patch version (x.y.Z)"
	@echo "  help         Show this help"