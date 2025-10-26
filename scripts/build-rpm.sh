#!/bin/bash
# CloudWorkstation RPM Package Builder
# Professional Enterprise Linux RPM Distribution

set -euo pipefail

# Colors and formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Configuration
PACKAGE_NAME="cloudworkstation"
VERSION="${VERSION:-0.5.1}"
RELEASE="${RELEASE:-1}"
ARCH="${ARCH:-$(uname -m)}"
BUILD_DIR="$(pwd)/packaging/rpm"
SPEC_FILE="$BUILD_DIR/cloudworkstation.spec"
SOURCE_DIR="$BUILD_DIR/sources"
DIST_DIR="$(pwd)/dist/rpm"

# Functions
print_header() {
    echo -e "${BOLD}${BLUE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${BLUE}║                     CloudWorkstation RPM Package Builder                     ║${NC}"
    echo -e "${BOLD}${BLUE}╠══════════════════════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${BOLD}${BLUE}║${NC} Building professional enterprise Linux RPM packages                      ${BOLD}${BLUE}║${NC}"
    echo -e "${BOLD}${BLUE}║${NC} Version: $VERSION-$RELEASE                                                    ${BOLD}${BLUE}║${NC}"
    echo -e "${BOLD}${BLUE}║${NC} Architecture: $ARCH                                                        ${BOLD}${BLUE}║${NC}"
    echo -e "${BOLD}${BLUE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    echo -e "${BOLD}${YELLOW}▶${NC} $1"
}

print_success() {
    echo -e "${GREEN}✅${NC} $1"
}

print_error() {
    echo -e "${RED}❌${NC} $1" >&2
}

print_warning() {
    echo -e "${YELLOW}⚠️${NC} $1"
}

# Validation functions
check_dependencies() {
    print_step "Checking build dependencies..."
    
    local missing_deps=()
    
    # Check for required tools
    command -v rpmbuild >/dev/null 2>&1 || missing_deps+=("rpm-build")
    command -v go >/dev/null 2>&1 || missing_deps+=("golang")
    command -v make >/dev/null 2>&1 || missing_deps+=("make")
    command -v tar >/dev/null 2>&1 || missing_deps+=("tar")
    command -v gzip >/dev/null 2>&1 || missing_deps+=("gzip")
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        print_error "Missing required dependencies: ${missing_deps[*]}"
        echo "Please install the missing dependencies:"
        echo "  RHEL/CentOS/Fedora: sudo dnf install ${missing_deps[*]}"
        echo "  Debian/Ubuntu: sudo apt-get install ${missing_deps[*]}"
        exit 1
    fi
    
    # Check Go version
    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    local required_version="1.20"
    if ! printf '%s\n' "$required_version" "$go_version" | sort -V | head -n1 | grep -q "^$required_version\$"; then
        print_warning "Go version $go_version detected, but $required_version or higher is recommended"
    fi
    
    print_success "All build dependencies are available"
}

validate_environment() {
    print_step "Validating build environment..."
    
    # Check if we're in the right directory
    if [[ ! -f "go.mod" ]] || [[ ! -d "cmd/cws" ]] || [[ ! -d "cmd/cwsd" ]]; then
        print_error "Must run from CloudWorkstation project root directory"
        exit 1
    fi
    
    # Check spec file exists
    if [[ ! -f "$SPEC_FILE" ]]; then
        print_error "RPM spec file not found: $SPEC_FILE"
        exit 1
    fi
    
    # Validate architecture
    case "$ARCH" in
        x86_64|amd64)
            export GOARCH=amd64
            ARCH=x86_64  # Normalize for RPM
            ;;
        aarch64|arm64)
            export GOARCH=arm64
            ARCH=aarch64  # Normalize for RPM
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    print_success "Build environment validated"
}

prepare_source_package() {
    print_step "Preparing source package..."
    
    # Clean and create directories
    rm -rf "$SOURCE_DIR"
    mkdir -p "$SOURCE_DIR"
    mkdir -p "$BUILD_DIR"/{BUILD,RPMS,SRPMS,tmp}
    
    # Create source archive excluding development files
    local temp_dir
    temp_dir=$(mktemp -d)
    local source_name="${PACKAGE_NAME}-${VERSION}"
    
    print_step "Creating source archive..."
    
    # Copy source files excluding development artifacts
    rsync -av \
        --exclude='.git*' \
        --exclude='bin/' \
        --exclude='dist/' \
        --exclude='*.log' \
        --exclude='.DS_Store' \
        --exclude='node_modules/' \
        --exclude='coverage.*' \
        --exclude='*.test' \
        --exclude='__pycache__/' \
        --exclude='*.pyc' \
        --exclude='.pytest_cache/' \
        --exclude='test_results/' \
        --exclude='volume/' \
        --exclude='cloudworkstation-*.tar.gz' \
        --exclude='packaging/rpm/BUILD/*' \
        --exclude='packaging/rpm/RPMS/*' \
        --exclude='packaging/rpm/SRPMS/*' \
        --exclude='packaging/rpm/sources/*' \
        --exclude='packaging/rpm/tmp/*' \
        ./ "$temp_dir/$source_name/"
    
    # Create tarball
    (cd "$temp_dir" && tar -czf "$SOURCE_DIR/${source_name}.tar.gz" "$source_name")
    
    # Cleanup
    rm -rf "$temp_dir"
    
    print_success "Source package created: $(basename "$SOURCE_DIR/${source_name}.tar.gz")"
}

build_binaries() {
    print_step "Building CloudWorkstation binaries for $ARCH..."
    
    # Set build environment
    export GOOS=linux
    export CGO_ENABLED=0
    
    # Build flags with version information
    local ldflags
    ldflags="-ldflags \"-X github.com/scttfrdmn/prism/pkg/version.Version=$VERSION \
                     -X github.com/scttfrdmn/prism/pkg/version.BuildDate=$(date -u '+%Y-%m-%d_%H:%M:%S') \
                     -X github.com/scttfrdmn/prism/pkg/version.GitCommit=rpm-build \
                     -w -s\""
    
    # Clean and create bin directory
    mkdir -p bin
    
    # Build CLI
    print_step "Building CLI binary (cws)..."
    eval "go build $ldflags -o bin/cws ./cmd/cws"
    
    # Build daemon
    print_step "Building daemon binary (cwsd)..."
    eval "go build $ldflags -o bin/prismd ./cmd/cwsd"
    
    # Verify binaries
    if [[ ! -x "bin/cws" ]] || [[ ! -x "bin/prismd" ]]; then
        print_error "Failed to build binaries"
        exit 1
    fi
    
    # Show binary information
    print_success "Built binaries:"
    echo "  CLI:    $(file bin/cws)"
    echo "  Daemon: $(file bin/prismd)"
    
    # Test binary execution
    print_step "Testing binary functionality..."
    if ./bin/prism --version >/dev/null 2>&1 && ./bin/prismd --version >/dev/null 2>&1; then
        print_success "Binaries execute correctly"
    else
        print_error "Binary functionality test failed"
        exit 1
    fi
}

build_rpm() {
    print_step "Building RPM package..."
    
    # Prepare RPM build environment
    export RPM_BUILD_DIR="$BUILD_DIR"
    
    # Build the RPM
    rpmbuild \
        --define "_topdir $BUILD_DIR" \
        --define "_builddir $BUILD_DIR/BUILD" \
        --define "_rpmdir $BUILD_DIR/RPMS" \
        --define "_sourcedir $SOURCE_DIR" \
        --define "_specdir $BUILD_DIR" \
        --define "_srcrpmdir $BUILD_DIR/SRPMS" \
        --define "_tmppath $BUILD_DIR/tmp" \
        --define "version $VERSION" \
        --define "release $RELEASE" \
        --target "$ARCH" \
        -ba "$SPEC_FILE"
    
    if [[ $? -ne 0 ]]; then
        print_error "RPM build failed"
        exit 1
    fi
    
    print_success "RPM package built successfully"
}

validate_rpm() {
    print_step "Validating RPM package..."
    
    # Find the built RPM
    local rpm_file
    rpm_file=$(find "$BUILD_DIR/RPMS" -name "${PACKAGE_NAME}-${VERSION}-${RELEASE}.*.rpm" | head -1)
    
    if [[ ! -f "$rpm_file" ]]; then
        print_error "Could not find built RPM package"
        exit 1
    fi
    
    print_step "Running RPM validation tests..."
    
    # Basic RPM validation
    echo "📋 Package information:"
    rpm -qip "$rpm_file"
    
    echo ""
    echo "📋 Package files:"
    rpm -qlp "$rpm_file"
    
    echo ""
    echo "📋 Package dependencies:"
    rpm -qRp "$rpm_file"
    
    echo ""
    echo "📋 Package provides:"
    rpm -q --provides -p "$rpm_file"
    
    # Run rpmlint if available
    if command -v rpmlint >/dev/null 2>&1; then
        print_step "Running rpmlint validation..."
        rpmlint "$rpm_file" || print_warning "rpmlint found some issues (non-fatal)"
    else
        print_warning "rpmlint not available, skipping detailed package validation"
    fi
    
    print_success "RPM package validation completed"
    return 0
}

organize_artifacts() {
    print_step "Organizing build artifacts..."
    
    # Create distribution directory
    mkdir -p "$DIST_DIR"
    
    # Copy RPM files
    cp -v "$BUILD_DIR/RPMS"/*/*.rpm "$DIST_DIR/" 2>/dev/null || true
    cp -v "$BUILD_DIR/SRPMS"/*.rpm "$DIST_DIR/" 2>/dev/null || true
    
    # Generate checksums
    (cd "$DIST_DIR" && sha256sum *.rpm > SHA256SUMS)
    
    # Generate package list
    cat > "$DIST_DIR/PACKAGES.txt" << EOF
CloudWorkstation RPM Packages
Generated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')
Version: $VERSION-$RELEASE
Architecture: $ARCH

Files:
EOF
    
    # List all RPM files with details
    for rpm in "$DIST_DIR"/*.rpm; do
        if [[ -f "$rpm" ]]; then
            local filename size
            filename=$(basename "$rpm")
            size=$(du -h "$rpm" | cut -f1)
            echo "  $filename ($size)" >> "$DIST_DIR/PACKAGES.txt"
        fi
    done
    
    print_success "Build artifacts organized in: $DIST_DIR"
}

print_build_summary() {
    echo ""
    echo -e "${BOLD}${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${GREEN}║                           RPM Build Complete                                ║${NC}"
    echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BOLD}📦 Package Information:${NC}"
    echo "   Name:         $PACKAGE_NAME"
    echo "   Version:      $VERSION-$RELEASE"
    echo "   Architecture: $ARCH"
    echo ""
    echo -e "${BOLD}📁 Artifacts Location:${NC}"
    echo "   Directory:    $DIST_DIR"
    echo "   Packages:     $(find "$DIST_DIR" -name "*.rpm" | wc -l)"
    echo ""
    echo -e "${BOLD}🧪 Installation Test:${NC}"
    echo "   RHEL/CentOS:  sudo dnf install $DIST_DIR/${PACKAGE_NAME}-${VERSION}-${RELEASE}.*.rpm"
    echo "   Fedora:       sudo dnf install $DIST_DIR/${PACKAGE_NAME}-${VERSION}-${RELEASE}.*.rpm"
    echo "   SUSE:         sudo zypper install $DIST_DIR/${PACKAGE_NAME}-${VERSION}-${RELEASE}.*.rpm"
    echo ""
    echo -e "${BOLD}📚 Post-Installation:${NC}"
    echo "   1. Configure AWS credentials in /etc/cloudworkstation/aws/"
    echo "   2. Start service: sudo systemctl start cloudworkstation"
    echo "   3. Enable auto-start: sudo systemctl enable cloudworkstation"
    echo "   4. Test: cws --version && cws templates"
    echo ""
}

cleanup() {
    if [[ "${CLEANUP_ON_EXIT:-1}" == "1" ]]; then
        print_step "Cleaning up temporary files..."
        rm -rf "$BUILD_DIR"/{BUILD,tmp}/*
    fi
}

# Main execution
main() {
    # Set up exit trap
    trap cleanup EXIT
    
    # Handle command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                echo "CloudWorkstation RPM Package Builder"
                echo ""
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --version VERSION    Set package version (default: $VERSION)"
                echo "  --release RELEASE    Set package release (default: $RELEASE)"
                echo "  --arch ARCH          Set target architecture (default: $ARCH)"
                echo "  --no-cleanup         Don't cleanup temporary files on exit"
                echo "  --help               Show this help message"
                echo ""
                echo "Environment Variables:"
                echo "  VERSION              Package version"
                echo "  RELEASE              Package release number"
                echo "  ARCH                 Target architecture"
                echo ""
                exit 0
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --release)
                RELEASE="$2"
                shift 2
                ;;
            --arch)
                ARCH="$2"
                shift 2
                ;;
            --no-cleanup)
                export CLEANUP_ON_EXIT=0
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
    
    # Execute build pipeline
    print_header
    check_dependencies
    validate_environment
    build_binaries
    prepare_source_package
    build_rpm
    validate_rpm
    organize_artifacts
    print_build_summary
    
    print_success "RPM package build completed successfully!"
}

# Execute main function
main "$@"