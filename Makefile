# CloudWorkstation Makefile
# Builds both daemon and CLI client

VERSION := 0.4.4
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=$(VERSION) -X github.com/scttfrdmn/cloudworkstation/pkg/version.BuildDate=$(BUILD_TIME) -X github.com/scttfrdmn/cloudworkstation/pkg/version.GitCommit=$(GIT_COMMIT)"

# Default target
.PHONY: all
all: build

# Build all binaries (CLI, daemon, GUI, TUI integrated)
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

# Build GUI binary
.PHONY: build-gui
build-gui:
	@echo "Building CloudWorkstation GUI (Wails 3.x)..."
	@if ! command -v wails >/dev/null 2>&1; then \
		echo "❌ Wails CLI not found. Install with: go install github.com/wailsapp/wails/v3/cmd/wails@latest"; \
		exit 1; \
	fi
	@cd cmd/cws-gui && wails build

# Force GUI build (for development/testing only)
.PHONY: build-gui-force
build-gui-force:
	@echo "⚠️  Force building CloudWorkstation GUI (may fail)..."
	@CGO_LDFLAGS="-Wl,-no_warn_duplicate_libraries" go build $(LDFLAGS) -o bin/cws-gui ./cmd/cws-gui

# Install binaries to system
.PHONY: install
install: build
	@echo "Installing CloudWorkstation..."
	@sudo cp bin/cwsd /usr/local/bin/
	@sudo cp bin/cws /usr/local/bin/
	@echo "✅ CloudWorkstation core binaries installed successfully"
	@echo "Start daemon with: cwsd"
	@echo "Use CLI with: cws --help"
	@echo ""
	@echo "🔧 Service Management:"
	@echo "  Install service: make service-install"
	@echo "  Service status:  make service-status"
	@echo "  Service help:    make service-help"

# Uninstall binaries from system
.PHONY: uninstall
uninstall:
	@echo "Uninstalling CloudWorkstation..."
	@./scripts/service-manager.sh uninstall 2>/dev/null || echo "Service not installed or already removed"
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
.PHONY: test test-unit test-integration test-e2e test-coverage test-all test-aws test-aws-quick test-aws-setup

# Run all unit tests (excluding GUI/TUI packages planned for Phase 2)
# Uses development mode to avoid keychain password prompts
test-unit:
	@echo "🧪 Running unit tests..."
	@CLOUDWORKSTATION_DEV=true GO_ENV=test go test -race -short $$(go list ./... | grep -v -E "(cmd/cws-gui|internal/tui)") -coverprofile=unit-coverage.out

# Run integration tests with LocalStack
test-integration:
	@echo "🔗 Running integration tests..."
	@docker-compose -f docker-compose.test.yml up -d localstack
	@echo "⏳ Waiting for LocalStack to be ready..."
	@sleep 10
	@CLOUDWORKSTATION_DEV=true INTEGRATION_TESTS=1 go test -tags=integration ./pkg/aws -v -coverprofile=aws-integration-coverage.out
	@CLOUDWORKSTATION_DEV=true INTEGRATION_TESTS=1 go test -tags=integration ./pkg/ami -v -coverprofile=ami-integration-coverage.out
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

# Run AWS integration tests against real AWS account
test-aws: build
	@echo "☁️  Running AWS integration tests..."
	@echo "⚠️  This will create real AWS resources and may incur costs!"
	@echo "📋 Ensure you have:"
	@echo "  - AWS profile 'aws' configured"
	@echo "  - CloudWorkstation daemon running (./bin/cwsd)"
	@echo "  - Appropriate AWS permissions"
	@echo ""
	@read -p "Continue? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	@echo ""
	@RUN_AWS_TESTS=true AWS_PROFILE=aws go test -v -tags=aws_integration ./internal/cli/ -run TestAWS -timeout=20m

# Quick AWS integration tests (subset for faster feedback)
test-aws-quick: build
	@echo "⚡ Running quick AWS integration tests..."
	@echo "📋 Testing: templates, daemon connectivity, basic operations"
	@RUN_AWS_TESTS=true AWS_PROFILE=aws go test -v -tags=aws_integration ./internal/cli/ -run "TestAWSTemplate|TestAWSDaemon" -timeout=5m

# Setup and validate AWS integration test environment
test-aws-setup:
	@echo "🔧 Validating AWS integration test setup..."
	@./scripts/validate-aws-setup.sh

# Run all tests (unit + integration + e2e + AWS if configured)
test-all: test-unit test-integration test-e2e test-coverage
	@if [ "$$RUN_AWS_TESTS" = "true" ]; then \
		echo "☁️  Including AWS integration tests..."; \
		$(MAKE) test-aws; \
	fi

# Legacy test target for backwards compatibility
test: test-unit

# Validate entire build and test pipeline
.PHONY: validate
validate:
	@echo "🔧 Running CloudWorkstation validation pipeline..."
	@./scripts/validate.sh

# Quality gates
.PHONY: quality-check vet security check-docs validate-templates

# Run all quality checks
quality-check: fmt vet lint security check-docs validate-templates test-coverage
	@echo "✅ All quality checks passed!"

# Check documentation standards
check-docs:
	@echo "📚 Checking documentation standards..."
	@./scripts/check-docs.sh

# Validate templates
validate-templates: build-cli
	@echo "🔍 Validating all templates..."
	@./bin/cws templates validate

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
	
	# Linux amd64 (GUI excluded due to cross-compile OpenGL issues)
	@mkdir -p bin/release/linux-amd64
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -tags crosscompile -o bin/release/linux-amd64/cwsd ./cmd/cwsd
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -tags crosscompile -o bin/release/linux-amd64/cws ./cmd/cws
	
	# Linux arm64 (GUI excluded due to cross-compile OpenGL issues)
	@mkdir -p bin/release/linux-arm64
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -tags crosscompile -o bin/release/linux-arm64/cwsd ./cmd/cwsd
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -tags crosscompile -o bin/release/linux-arm64/cws ./cmd/cws
	
	# macOS amd64
	@mkdir -p bin/release/darwin-amd64
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -tags crosscompile -o bin/release/darwin-amd64/cwsd ./cmd/cwsd
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -tags crosscompile -o bin/release/darwin-amd64/cws ./cmd/cws
	
	# macOS arm64 (Apple Silicon)
	@mkdir -p bin/release/darwin-arm64
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -tags crosscompile -o bin/release/darwin-arm64/cwsd ./cmd/cwsd
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -tags crosscompile -o bin/release/darwin-arm64/cws ./cmd/cws
	
	# Windows amd64 (GUI excluded due to cross-compile OpenGL issues)
	@mkdir -p bin/release/windows-amd64
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -tags crosscompile -o bin/release/windows-amd64/cwsd.exe ./cmd/cwsd
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -tags crosscompile -o bin/release/windows-amd64/cws.exe ./cmd/cws
	
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

# Service management targets
.PHONY: service-install service-uninstall service-start service-stop service-restart service-status service-logs service-follow service-validate service-help service-info

# Install system service (auto-start on boot)
service-install: build
	@echo "🔧 Installing CloudWorkstation system service..."
	@./scripts/service-manager.sh install

# Uninstall system service
service-uninstall:
	@echo "🔧 Uninstalling CloudWorkstation system service..."
	@./scripts/service-manager.sh uninstall

# Start service
service-start:
	@echo "▶️  Starting CloudWorkstation service..."
	@./scripts/service-manager.sh start

# Stop service
service-stop:
	@echo "⏹️  Stopping CloudWorkstation service..."
	@./scripts/service-manager.sh stop

# Restart service
service-restart:
	@echo "🔄 Restarting CloudWorkstation service..."
	@./scripts/service-manager.sh restart

# Show service status
service-status:
	@./scripts/service-manager.sh status

# Show service logs
service-logs:
	@./scripts/service-manager.sh logs

# Follow service logs in real-time
service-follow:
	@./scripts/service-manager.sh follow

# Validate service configuration
service-validate:
	@./scripts/service-manager.sh validate

# Show service help
service-help:
	@./scripts/service-manager.sh help

# Show system information for service management
service-info:
	@./scripts/service-manager.sh info

# Complete installation with service setup
.PHONY: install-complete
install-complete: install service-install
	@echo ""
	@echo "🎉 CloudWorkstation installation complete!"
	@echo ""
	@echo "📋 What's installed:"
	@echo "  ✅ Core binaries (cws, cwsd) in /usr/local/bin/"
	@echo "  ✅ System service configured for auto-start"
	@echo ""
	@echo "🚀 Quick start:"
	@echo "  cws --help                    # CLI help"
	@echo "  make service-status           # Check service status"
	@echo "  make service-logs             # View service logs"
	@echo ""

# Windows installer targets
.PHONY: windows-installer windows-service windows-sign-msi windows-build-custom-actions windows-installer-dev

# Build Windows MSI installer
windows-installer:
	@echo "🪟 Building Windows MSI installer..."
	@if [ "$(shell uname)" = "Darwin" ] || [ "$(shell uname | grep -i linux)" ]; then \
		echo "⚠️  Windows installer must be built on Windows with WiX Toolset"; \
		echo "   Available as: scripts/build-msi.ps1 or scripts/build-msi.bat"; \
		echo "   On Windows: powershell -ExecutionPolicy Bypass -File scripts/build-msi.ps1"; \
		exit 1; \
	fi
	@powershell -ExecutionPolicy Bypass -File scripts/build-msi.ps1 -Version $(VERSION)

# Build Windows service wrapper only
windows-service: bin
	@echo "🪟 Building Windows service wrapper..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/cwsd-service.exe ./cmd/cwsd-service
	@echo "✅ Windows service wrapper built: bin/cwsd-service.exe"

# Sign Windows MSI (requires certificate)
windows-sign-msi:
	@echo "🪟 Signing Windows MSI installer..."
	@if [ "$(shell uname)" = "Darwin" ] || [ "$(shell uname | grep -i linux)" ]; then \
		echo "⚠️  MSI signing must be done on Windows with SignTool"; \
		echo "   Available as: scripts/sign-msi.ps1"; \
		echo "   On Windows: powershell -ExecutionPolicy Bypass -File scripts/sign-msi.ps1"; \
		exit 1; \
	fi
	@powershell -ExecutionPolicy Bypass -File scripts/sign-msi.ps1

# Build custom actions DLL (requires Visual Studio/MSBuild)
windows-build-custom-actions:
	@echo "🪟 Building Windows installer custom actions..."
	@if [ "$(shell uname)" = "Darwin" ] || [ "$(shell uname | grep -i linux)" ]; then \
		echo "⚠️  Custom actions DLL must be built on Windows with MSBuild"; \
		echo "   Project: packaging/windows/SetupCustomActions/SetupCustomActions.csproj"; \
		echo "   On Windows: msbuild packaging/windows/SetupCustomActions/SetupCustomActions.csproj /p:Configuration=Release"; \
		exit 1; \
	fi
	@msbuild packaging/windows/SetupCustomActions/SetupCustomActions.csproj /p:Configuration=Release /p:Platform=x64

# Development Windows installer (faster build, minimal features)
windows-installer-dev:
	@echo "🪟 Building development Windows MSI installer..."
	@if [ "$(shell uname)" = "Darwin" ] || [ "$(shell uname | grep -i linux)" ]; then \
		echo "⚠️  Windows installer must be built on Windows with WiX Toolset"; \
		echo "   Available as: scripts/build-msi.ps1"; \
		echo "   On Windows: powershell -ExecutionPolicy Bypass -File scripts/build-msi.ps1 -SkipCustomActions"; \
		exit 1; \
	fi
	@powershell -ExecutionPolicy Bypass -File scripts/build-msi.ps1 -Version $(VERSION) -SkipCustomActions

# Show help
# Package manager distribution targets
.PHONY: package package-homebrew package-chocolatey package-conda package-linux package-rpm package-deb package-test

# Create all package distribution files
package: package-homebrew package-chocolatey package-conda package-linux

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
	@cp packaging/chocolatey/cloudworkstation.nuspec dist/chocolatey/
	@cp packaging/chocolatey/tools/chocolateyinstall.ps1 dist/chocolatey/tools/
	@cp packaging/chocolatey/tools/chocolateyuninstall.ps1 dist/chocolatey/tools/
	@cd bin/release && zip -j ../../dist/chocolatey/tools/cloudworkstation-windows-amd64.zip darwin-amd64-*
	@cp packaging/conda/meta.yaml dist/conda/
	
	# Generate test checksums
	@openssl sha256 dist/homebrew/cloudworkstation-darwin-amd64.tar.gz > dist/homebrew/darwin-amd64.sha256
	@openssl sha256 dist/chocolatey/tools/cloudworkstation-windows-amd64.zip > dist/chocolatey/windows-amd64.sha256
	
	@echo "✅ Test packages created in dist/ directory"

# Create Homebrew formula
package-homebrew: release
	@echo "Creating Homebrew package..."
	@mkdir -p dist/homebrew
	@cp packaging/homebrew/cloudworkstation.rb dist/homebrew/
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
	@cp packaging/chocolatey/cloudworkstation.nuspec dist/chocolatey/
	@cp packaging/chocolatey/tools/chocolateyinstall.ps1 dist/chocolatey/tools/
	@cp packaging/chocolatey/tools/chocolateyuninstall.ps1 dist/chocolatey/tools/
	@cd bin/release && zip -j ../../dist/chocolatey/tools/cloudworkstation-windows-amd64.zip windows-amd64-*
	@openssl sha256 dist/chocolatey/tools/cloudworkstation-windows-amd64.zip | awk '{print $$2}' > dist/chocolatey/windows-amd64.sha256
	@echo "✅ Chocolatey package created in dist/chocolatey"

# Create Conda package
package-conda: release
	@echo "Creating Conda package..."
	@mkdir -p dist/conda
	@cp packaging/conda/meta.yaml dist/conda/
	@cp packaging/conda/build.sh dist/conda/
	@cp packaging/conda/bld.bat dist/conda/
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

# Linux packaging targets (RPM and DEB)
.PHONY: package-linux package-rpm package-deb package-rpm-test package-deb-test linux-packages-clean

# Create all Linux packages (RPM and DEB)
package-linux: package-rpm package-deb

# Create RPM package for RHEL/CentOS/Fedora/SUSE
package-rpm: build
	@echo "🐧 Building RPM package for enterprise Linux distributions..."
	@./scripts/build-rpm.sh
	@echo "✅ RPM package created in dist/rpm/"

# Create DEB package for Ubuntu/Debian/Mint  
package-deb: build
	@echo "🐧 Building DEB package for Ubuntu/Debian distributions..."
	@./scripts/build-deb.sh
	@echo "✅ DEB package created in dist/deb/"

# Build RPM package with specific version and architecture
package-rpm-custom:
	@echo "🐧 Building custom RPM package..."
	@if [ -z "$(VERSION)" ] || [ -z "$(ARCH)" ]; then \
		echo "❌ Usage: make package-rpm-custom VERSION=x.y.z ARCH=x86_64|aarch64"; \
		exit 1; \
	fi
	@./scripts/build-rpm.sh --version $(VERSION) --arch $(ARCH)
	@echo "✅ Custom RPM package created in dist/rpm/"

# Build DEB package with specific version and architecture
package-deb-custom:
	@echo "🐧 Building custom DEB package..."
	@if [ -z "$(VERSION)" ] || [ -z "$(ARCH)" ]; then \
		echo "❌ Usage: make package-deb-custom VERSION=x.y.z ARCH=amd64|arm64"; \
		exit 1; \
	fi
	@./scripts/build-deb.sh --version $(VERSION) --arch $(ARCH)
	@echo "✅ Custom DEB package created in dist/deb/"

# Test RPM package installation (requires Docker)
package-rpm-test:
	@echo "🧪 Testing RPM package installation..."
	@./scripts/test-linux-packages.sh --rpm
	@echo "✅ RPM package installation test completed"

# Test DEB package installation (requires Docker)
package-deb-test:
	@echo "🧪 Testing DEB package installation..."
	@./scripts/test-linux-packages.sh --deb
	@echo "✅ DEB package installation test completed"

# Test both RPM and DEB packages
package-linux-test: package-rpm-test package-deb-test

# Build packages for all Linux architectures
package-linux-all:
	@echo "🐧 Building Linux packages for all architectures..."
	@$(MAKE) package-rpm-custom VERSION=$(VERSION) ARCH=x86_64
	@$(MAKE) package-rpm-custom VERSION=$(VERSION) ARCH=aarch64
	@$(MAKE) package-deb-custom VERSION=$(VERSION) ARCH=amd64
	@$(MAKE) package-deb-custom VERSION=$(VERSION) ARCH=arm64
	@echo "✅ All Linux packages created"

# Clean Linux packaging artifacts
linux-packages-clean:
	@echo "🧹 Cleaning Linux packaging artifacts..."
	@rm -rf dist/rpm/ dist/deb/
	@rm -rf packaging/rpm/{BUILD,RPMS,SRPMS,sources,tmp}/*
	@echo "✅ Linux packaging artifacts cleaned"

# Validate Linux packages with linting tools
package-linux-validate:
	@echo "🔍 Validating Linux packages..."
	@if [ -f "dist/rpm/cloudworkstation-$(VERSION)-1.x86_64.rpm" ]; then \
		echo "📋 Validating RPM package..."; \
		rpm -qip "dist/rpm/cloudworkstation-$(VERSION)-1.x86_64.rpm"; \
		command -v rpmlint >/dev/null 2>&1 && rpmlint "dist/rpm/cloudworkstation-$(VERSION)-1.x86_64.rpm" || echo "⚠️ rpmlint not available"; \
	fi
	@if [ -f "dist/deb/cloudworkstation_$(VERSION)-1_amd64.deb" ]; then \
		echo "📋 Validating DEB package..."; \
		dpkg-deb -I "dist/deb/cloudworkstation_$(VERSION)-1_amd64.deb"; \
		command -v lintian >/dev/null 2>&1 && lintian "dist/deb/cloudworkstation_$(VERSION)-1_amd64.deb" || echo "⚠️ lintian not available"; \
	fi
	@echo "✅ Linux package validation completed"

# Create signed Linux packages (requires GPG setup)
package-linux-signed: package-linux
	@echo "🔐 Signing Linux packages..."
	@if command -v rpm >/dev/null 2>&1 && command -v gpg >/dev/null 2>&1; then \
		echo "📝 Signing RPM packages..."; \
		for rpm in dist/rpm/*.rpm; do \
			if [ -f "$$rpm" ]; then \
				rpm --addsign "$$rpm" || echo "⚠️ Failed to sign $$rpm"; \
			fi; \
		done; \
	fi
	@if command -v dpkg-sig >/dev/null 2>&1 && command -v gpg >/dev/null 2>&1; then \
		echo "📝 Signing DEB packages..."; \
		for deb in dist/deb/*.deb; do \
			if [ -f "$$deb" ]; then \
				dpkg-sig --sign builder "$$deb" || echo "⚠️ Failed to sign $$deb"; \
			fi; \
		done; \
	fi
	@echo "✅ Linux package signing completed"

# macOS DMG distribution targets
.PHONY: dmg dmg-dev dmg-universal dmg-signed dmg-notarized dmg-clean dmg-all

# Create standard DMG (current architecture)
dmg: build
	@echo "🍎 Creating macOS DMG package..."
	@./scripts/build-dmg.sh
	@echo "✅ DMG created in dist/dmg/"

# Create development DMG (without GUI, faster build)
dmg-dev: build-cli build-daemon
	@echo "🍎 Creating development DMG package..."
	@./scripts/build-dmg.sh --dev
	@echo "✅ Development DMG created in dist/dmg/"

# Create universal DMG (Intel + Apple Silicon)
dmg-universal: clean
	@echo "🍎 Creating universal macOS DMG package..."
	@./scripts/build-dmg.sh --universal
	@echo "✅ Universal DMG created in dist/dmg/"

# Sign DMG with Developer ID
dmg-signed: dmg
	@echo "🔐 Signing DMG package..."
	@./scripts/sign-dmg.sh
	@echo "✅ DMG signed successfully"

# Create signed universal DMG
dmg-universal-signed: dmg-universal
	@echo "🔐 Signing universal DMG package..."
	@./scripts/sign-dmg.sh
	@echo "✅ Universal DMG signed successfully"

# Notarize signed DMG with Apple
dmg-notarized:
	@echo "🍎 Notarizing DMG with Apple..."
	@./scripts/notarize-dmg.sh
	@echo "✅ DMG notarized and ready for distribution"

# Create complete notarized DMG (build → sign → notarize)
dmg-all: dmg-universal-signed dmg-notarized
	@echo "🎉 Complete DMG build process finished!"
	@echo "Final DMG ready for distribution in dist/dmg/"

# Clean DMG build artifacts
dmg-clean:
	@echo "🧹 Cleaning DMG build artifacts..."
	@rm -rf dist/dmg/
	@echo "✅ DMG build artifacts cleaned"

# Test DMG integrity
dmg-test:
	@echo "🧪 Testing DMG integrity..."
	@if [ -f "dist/dmg/CloudWorkstation-v$(VERSION).dmg" ]; then \
		hdiutil verify "dist/dmg/CloudWorkstation-v$(VERSION).dmg"; \
		echo "✅ DMG integrity test passed"; \
	else \
		echo "❌ No DMG file found to test"; \
		exit 1; \
	fi

# Install DMG creation prerequisites
dmg-setup:
	@echo "🔧 Setting up DMG creation prerequisites..."
	@if [ "$(shell uname)" != "Darwin" ]; then \
		echo "❌ DMG creation requires macOS"; \
		exit 1; \
	fi
	@command -v iconutil >/dev/null 2>&1 || { echo "❌ iconutil not found. Install Xcode command line tools."; exit 1; }
	@command -v hdiutil >/dev/null 2>&1 || { echo "❌ hdiutil not found."; exit 1; }
	@command -v SetFile >/dev/null 2>&1 || { echo "❌ SetFile not found. Install Xcode command line tools."; exit 1; }
	@if command -v python3 >/dev/null 2>&1; then \
		python3 -c "import PIL" 2>/dev/null || { \
			echo "Installing Pillow for icon generation..."; \
			pip3 install Pillow || echo "⚠️ Failed to install Pillow. Icon generation may use fallback method."; \
		}; \
	fi
	@chmod +x scripts/build-dmg.sh scripts/sign-dmg.sh scripts/notarize-dmg.sh
	@echo "✅ DMG creation prerequisites ready"

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
	@echo "  install-complete Install binaries and setup system service"
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
	@echo "  package-linux Create all Linux packages (RPM and DEB)"
	@echo "  package-rpm  Create RPM package for RHEL/CentOS/Fedora/SUSE"
	@echo "  package-deb  Create DEB package for Ubuntu/Debian/Mint"
	@echo ""
	@echo "Linux Enterprise Distribution:"
	@echo "  package-rpm-custom Create custom RPM (VERSION=x.y.z ARCH=x86_64|aarch64)"
	@echo "  package-deb-custom Create custom DEB (VERSION=x.y.z ARCH=amd64|arm64)"
	@echo "  package-linux-all  Create packages for all Linux architectures"
	@echo "  package-linux-test Test RPM and DEB package installation"
	@echo "  package-linux-validate Validate packages with lintian/rpmlint"
	@echo "  package-linux-signed Create signed packages (requires GPG)"
	@echo "  linux-packages-clean Clean Linux packaging artifacts"
	@echo ""
	@echo "Windows MSI Distribution:"
	@echo "  windows-installer Create Windows MSI installer (requires Windows + WiX Toolset)"
	@echo "  windows-installer-dev Create development MSI installer (minimal features)"
	@echo "  windows-service Build Windows service wrapper only"
	@echo "  windows-sign-msi Sign MSI with digital certificate (requires Windows + SignTool)"
	@echo "  windows-build-custom-actions Build custom actions DLL (requires MSBuild)"
	@echo ""
	@echo "macOS DMG Distribution:"
	@echo "  dmg          Create macOS DMG package"
	@echo "  dmg-dev      Create development DMG (CLI/daemon only)"
	@echo "  dmg-universal Create universal DMG (Intel + Apple Silicon)"
	@echo "  dmg-signed   Create and sign DMG with Developer ID"
	@echo "  dmg-universal-signed Create and sign universal DMG"
	@echo "  dmg-notarized Notarize signed DMG with Apple"
	@echo "  dmg-all      Complete DMG build process (build → sign → notarize)"
	@echo "  dmg-setup    Install DMG creation prerequisites"
	@echo "  dmg-test     Test DMG integrity"
	@echo "  dmg-clean    Clean DMG build artifacts"
	@echo "  dev-daemon   Build and run daemon for development"
	@echo "  dev-cli      Build and test CLI"
	@echo "  dev-gui      Build and test GUI"
	@echo "  version      Show version information"
	@echo "  bump-major   Bump major version (X.y.z)"
	@echo "  bump-minor   Bump minor version (x.Y.z)"
	@echo "  bump-patch   Bump patch version (x.y.Z)"
	@echo ""
	@echo "Service Management:"
	@echo "  service-install   Install system service (auto-start on boot)"
	@echo "  service-uninstall Uninstall system service"
	@echo "  service-start     Start the service"
	@echo "  service-stop      Stop the service"
	@echo "  service-restart   Restart the service"
	@echo "  service-status    Show service status"
	@echo "  service-logs      Show service logs"
	@echo "  service-follow    Follow service logs in real-time"
	@echo "  service-validate  Validate service configuration"
	@echo "  service-help      Show service management help"
	@echo "  service-info      Show system information for service management"
	@echo "  help         Show this help"