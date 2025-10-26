#!/bin/bash
# CloudWorkstation DEB Package Builder
# Professional Ubuntu/Debian Distribution

set -euo pipefail

# Colors and formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Configuration
PACKAGE_NAME="prism"
VERSION="${VERSION:-0.4.2}"
REVISION="${REVISION:-1}"
ARCH="${ARCH:-$(dpkg --print-architecture 2>/dev/null || echo "amd64")}"
BUILD_DIR="$(pwd)/packaging/deb"
DEBIAN_DIR="$BUILD_DIR/debian"
DIST_DIR="$(pwd)/dist/deb"
WORK_DIR="/tmp/prism-build-$$"

# Functions
print_header() {
    echo -e "${BOLD}${BLUE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${BLUE}║                     CloudWorkstation DEB Package Builder                     ║${NC}"
    echo -e "${BOLD}${BLUE}╠══════════════════════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${BOLD}${BLUE}║${NC} Building professional Ubuntu/Debian packages                             ${BOLD}${BLUE}║${NC}"
    echo -e "${BOLD}${BLUE}║${NC} Version: $VERSION-$REVISION                                                    ${BOLD}${BLUE}║${NC}"
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
    command -v dpkg-deb >/dev/null 2>&1 || missing_deps+=("dpkg-dev")
    command -v debuild >/dev/null 2>&1 || missing_deps+=("devscripts")
    command -v dh >/dev/null 2>&1 || missing_deps+=("debhelper")
    command -v go >/dev/null 2>&1 || missing_deps+=("golang-go")
    command -v make >/dev/null 2>&1 || missing_deps+=("make")
    command -v tar >/dev/null 2>&1 || missing_deps+=("tar")
    command -v gzip >/dev/null 2>&1 || missing_deps+=("gzip")
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        print_error "Missing required dependencies: ${missing_deps[*]}"
        echo "Please install the missing dependencies:"
        echo "  sudo apt-get update"
        echo "  sudo apt-get install ${missing_deps[*]}"
        exit 1
    fi
    
    # Check Go version
    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    local required_version="1.20"
    if ! printf '%s\n' "$required_version" "$go_version" | sort -V | head -n1 | grep -q "^$required_version\$"; then
        print_warning "Go version $go_version detected, but $required_version or higher is recommended"
    fi
    
    # Check for recommended tools
    local recommended=()
    command -v lintian >/dev/null 2>&1 || recommended+=("lintian")
    command -v dpkg-sig >/dev/null 2>&1 || recommended+=("dpkg-sig")
    
    if [[ ${#recommended[@]} -gt 0 ]]; then
        print_warning "Recommended tools not found: ${recommended[*]}"
        echo "Install with: sudo apt-get install ${recommended[*]}"
    fi
    
    print_success "All required build dependencies are available"
}

validate_environment() {
    print_step "Validating build environment..."
    
    # Check if we're in the right directory
    if [[ ! -f "go.mod" ]] || [[ ! -d "cmd/cws" ]] || [[ ! -d "cmd/cwsd" ]]; then
        print_error "Must run from CloudWorkstation project root directory"
        exit 1
    fi
    
    # Check debian directory exists
    if [[ ! -d "$DEBIAN_DIR" ]]; then
        print_error "Debian packaging directory not found: $DEBIAN_DIR"
        exit 1
    fi
    
    # Check required debian files
    local required_files=("control" "changelog" "copyright" "rules")
    for file in "${required_files[@]}"; do
        if [[ ! -f "$DEBIAN_DIR/$file" ]]; then
            print_error "Missing required debian file: $DEBIAN_DIR/$file"
            exit 1
        fi
    done
    
    # Validate architecture
    case "$ARCH" in
        amd64|x86_64)
            export GOARCH=amd64
            ARCH=amd64  # Normalize for DEB
            ;;
        arm64|aarch64)
            export GOARCH=arm64
            ARCH=arm64  # Normalize for DEB
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    print_success "Build environment validated"
}

prepare_build_directory() {
    print_step "Preparing build directory..."
    
    # Clean up any previous builds
    rm -rf "$WORK_DIR"
    mkdir -p "$WORK_DIR"
    
    # Create source directory structure
    local source_name="${PACKAGE_NAME}-${VERSION}"
    local source_dir="$WORK_DIR/$source_name"
    
    print_step "Copying source files..."
    
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
        --exclude='prism-*.tar.gz' \
        --exclude='packaging/deb/build/' \
        --exclude='packaging/rpm/BUILD/*' \
        --exclude='packaging/rpm/RPMS/*' \
        --exclude='packaging/rpm/SRPMS/*' \
        ./ "$source_dir/"
    
    # Copy debian directory
    cp -r "$DEBIAN_DIR" "$source_dir/"
    
    # Set working directory
    cd "$source_dir"
    
    print_success "Build directory prepared: $source_dir"
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
                     -X github.com/scttfrdmn/prism/pkg/version.GitCommit=deb-build \
                     -w -s\""
    
    # Clean and create build directory
    mkdir -p build
    
    # Build CLI
    print_step "Building CLI binary (cws)..."
    eval "go build $ldflags -o build/cws ./cmd/cws"
    
    # Build daemon
    print_step "Building daemon binary (cwsd)..."
    eval "go build $ldflags -o build/cwsd ./cmd/cwsd"
    
    # Verify binaries
    if [[ ! -x "build/cws" ]] || [[ ! -x "build/cwsd" ]]; then
        print_error "Failed to build binaries"
        exit 1
    fi
    
    # Show binary information
    print_success "Built binaries:"
    echo "  CLI:    $(file build/cws)"
    echo "  Daemon: $(file build/cwsd)"
    
    # Test binary execution
    print_step "Testing binary functionality..."
    if ./build/cws --version >/dev/null 2>&1 && ./build/cwsd --version >/dev/null 2>&1; then
        print_success "Binaries execute correctly"
    else
        print_error "Binary functionality test failed"
        exit 1
    fi
}

build_package() {
    print_step "Building DEB package..."
    
    # Set environment variables for debian/rules
    export DEB_HOST_ARCH="$ARCH"
    export PACKAGE_VERSION="$VERSION"
    
    # Use debuild to build the package
    # -us -uc: don't sign source and changes
    # -b: binary only build (no source package)
    if ! debuild -us -uc -b; then
        print_error "DEB package build failed"
        exit 1
    fi
    
    print_success "DEB package built successfully"
}

validate_package() {
    print_step "Validating DEB package..."
    
    # Find the built DEB package
    local deb_file
    deb_file=$(find "$WORK_DIR" -name "${PACKAGE_NAME}_${VERSION}-${REVISION}_${ARCH}.deb" | head -1)
    
    if [[ ! -f "$deb_file" ]]; then
        print_error "Could not find built DEB package"
        echo "Looking for: ${PACKAGE_NAME}_${VERSION}-${REVISION}_${ARCH}.deb"
        echo "Available files:"
        find "$WORK_DIR" -name "*.deb" || echo "  No .deb files found"
        exit 1
    fi
    
    print_step "Running DEB validation tests..."
    
    # Basic DEB validation
    echo "📋 Package information:"
    dpkg-deb -I "$deb_file"
    
    echo ""
    echo "📋 Package contents:"
    dpkg-deb -c "$deb_file"
    
    # Run lintian if available
    if command -v lintian >/dev/null 2>&1; then
        print_step "Running lintian validation..."
        lintian "$deb_file" || print_warning "lintian found some issues (may not be fatal)"
    else
        print_warning "lintian not available, skipping detailed package validation"
    fi
    
    # Test package installation (dry-run)
    print_step "Testing package installation (dry-run)..."
    if dpkg --dry-run -i "$deb_file" >/dev/null 2>&1; then
        print_success "Package installation test passed"
    else
        print_warning "Package installation test failed (may be due to dependencies)"
    fi
    
    print_success "DEB package validation completed"
    return 0
}

organize_artifacts() {
    print_step "Organizing build artifacts..."
    
    # Create distribution directory
    mkdir -p "$DIST_DIR"
    
    # Copy DEB files from work directory
    cp -v "$WORK_DIR"/*.deb "$DIST_DIR/" 2>/dev/null || true
    cp -v "$WORK_DIR"/*.dsc "$DIST_DIR/" 2>/dev/null || true
    cp -v "$WORK_DIR"/*.changes "$DIST_DIR/" 2>/dev/null || true
    cp -v "$WORK_DIR"/*.buildinfo "$DIST_DIR/" 2>/dev/null || true
    
    # Generate checksums
    (cd "$DIST_DIR" && sha256sum *.deb > SHA256SUMS 2>/dev/null || true)
    
    # Generate package list
    cat > "$DIST_DIR/PACKAGES.txt" << EOF
CloudWorkstation DEB Packages
Generated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')
Version: $VERSION-$REVISION
Architecture: $ARCH

Files:
EOF
    
    # List all DEB files with details
    for deb in "$DIST_DIR"/*.deb; do
        if [[ -f "$deb" ]]; then
            local filename size
            filename=$(basename "$deb")
            size=$(du -h "$deb" | cut -f1)
            echo "  $filename ($size)" >> "$DIST_DIR/PACKAGES.txt"
        fi
    done
    
    print_success "Build artifacts organized in: $DIST_DIR"
}

print_build_summary() {
    echo ""
    echo -e "${BOLD}${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${GREEN}║                           DEB Build Complete                                ║${NC}"
    echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BOLD}📦 Package Information:${NC}"
    echo "   Name:         $PACKAGE_NAME"
    echo "   Version:      $VERSION-$REVISION"
    echo "   Architecture: $ARCH"
    echo ""
    echo -e "${BOLD}📁 Artifacts Location:${NC}"
    echo "   Directory:    $DIST_DIR"
    echo "   Packages:     $(find "$DIST_DIR" -name "*.deb" 2>/dev/null | wc -l)"
    echo ""
    echo -e "${BOLD}🧪 Installation Test:${NC}"
    echo "   Ubuntu/Debian: sudo dpkg -i $DIST_DIR/${PACKAGE_NAME}_${VERSION}-${REVISION}_${ARCH}.deb"
    echo "   Fix deps:      sudo apt-get install -f"
    echo "   With apt:      sudo apt install $DIST_DIR/${PACKAGE_NAME}_${VERSION}-${REVISION}_${ARCH}.deb"
    echo ""
    echo -e "${BOLD}📚 Post-Installation:${NC}"
    echo "   1. Configure AWS credentials in /etc/prism/aws/"
    echo "   2. Copy templates: sudo cp /etc/prism/aws/*.template /etc/prism/aws/"
    echo "   3. Edit credentials: sudo nano /etc/prism/aws/credentials"
    echo "   4. Start service: sudo systemctl start prism"
    echo "   5. Enable auto-start: sudo systemctl enable prism"
    echo "   6. Test: cws --version && cws templates"
    echo ""
}

cleanup() {
    if [[ "${CLEANUP_ON_EXIT:-1}" == "1" ]]; then
        print_step "Cleaning up temporary files..."
        rm -rf "$WORK_DIR"
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
                echo "CloudWorkstation DEB Package Builder"
                echo ""
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --version VERSION    Set package version (default: $VERSION)"
                echo "  --revision REVISION  Set package revision (default: $REVISION)"
                echo "  --arch ARCH          Set target architecture (default: $ARCH)"
                echo "  --no-cleanup         Don't cleanup temporary files on exit"
                echo "  --help               Show this help message"
                echo ""
                echo "Environment Variables:"
                echo "  VERSION              Package version"
                echo "  REVISION             Package revision number"
                echo "  ARCH                 Target architecture"
                echo ""
                exit 0
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --revision)
                REVISION="$2"
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
    prepare_build_directory
    build_binaries
    build_package
    validate_package
    organize_artifacts
    print_build_summary
    
    print_success "DEB package build completed successfully!"
}

# Execute main function
main "$@"