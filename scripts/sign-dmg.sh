#!/bin/bash

# CloudWorkstation DMG Code Signing Script
# Signs application bundle and DMG for macOS distribution
# Usage: ./scripts/sign-dmg.sh [DMG_PATH] [--dev-id IDENTITY]

set -euo pipefail

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Default signing identity (can be overridden)
SIGNING_IDENTITY="Developer ID Application"
DMG_PATH=""
VERIFY_ONLY=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dev-id)
            SIGNING_IDENTITY="$2"
            shift 2
            ;;
        --verify-only)
            VERIFY_ONLY=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [DMG_PATH] [--dev-id IDENTITY] [--verify-only]"
            echo ""
            echo "Options:"
            echo "  DMG_PATH           Path to DMG file to sign"
            echo "  --dev-id IDENTITY  Signing identity (default: 'Developer ID Application')"
            echo "  --verify-only      Only verify existing signatures"
            echo "  --help             Show this help"
            echo ""
            echo "Examples:"
            echo "  $0 dist/dmg/CloudWorkstation-v0.4.2.dmg"
            echo "  $0 --dev-id 'Developer ID Application: Your Name (TEAM123)'"
            echo "  $0 --verify-only dist/dmg/CloudWorkstation-v0.4.2.dmg"
            exit 0
            ;;
        *)
            if [[ -z "$DMG_PATH" ]]; then
                DMG_PATH="$1"
            else
                echo "Unknown option: $1"
                exit 1
            fi
            shift
            ;;
    esac
done

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking signing prerequisites..."
    
    # Check if running on macOS
    if [[ "$(uname)" != "Darwin" ]]; then
        log_error "Code signing requires macOS"
        exit 1
    fi
    
    # Check for required tools
    local required_tools=("codesign" "spctl" "hdiutil")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' not found"
            exit 1
        fi
    done
    
    # Check for Xcode tools
    if ! xcode-select -p &> /dev/null; then
        log_error "Xcode command line tools not installed. Run: xcode-select --install"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Find available signing identities
find_signing_identities() {
    log_info "Available signing identities:"
    
    # List available Developer ID Application certificates
    security find-identity -v -p codesigning | grep "Developer ID Application" || {
        log_warning "No 'Developer ID Application' certificates found"
        log_info "Available certificates:"
        security find-identity -v -p codesigning || log_warning "No code signing certificates found"
        return 1
    }
}

# Verify signing identity exists
verify_signing_identity() {
    local identity="$1"
    
    log_info "Verifying signing identity: '$identity'"
    
    if security find-identity -v -p codesigning | grep -q "$identity"; then
        log_success "Signing identity found"
        return 0
    else
        log_error "Signing identity '$identity' not found"
        find_signing_identities
        return 1
    fi
}

# Create entitlements file
create_entitlements() {
    local entitlements_path="$1"
    
    log_info "Creating entitlements file..."
    
    cat > "$entitlements_path" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <!-- Hardened Runtime entitlements -->
    <key>com.apple.security.cs.allow-jit</key>
    <false/>
    
    <key>com.apple.security.cs.allow-unsigned-executable-memory</key>
    <false/>
    
    <key>com.apple.security.cs.allow-dyld-environment-variables</key>
    <false/>
    
    <key>com.apple.security.cs.disable-library-validation</key>
    <false/>
    
    <!-- Network access for daemon -->
    <key>com.apple.security.network.client</key>
    <true/>
    
    <key>com.apple.security.network.server</key>
    <true/>
    
    <!-- Apple Events for GUI interactions -->
    <key>com.apple.security.automation.apple-events</key>
    <true/>
    
    <!-- File access -->
    <key>com.apple.security.files.user-selected.read-write</key>
    <true/>
    
    <key>com.apple.security.files.downloads.read-write</key>
    <true/>
</dict>
</plist>
EOF
    
    log_success "Entitlements file created"
}

# Sign individual binary
sign_binary() {
    local binary_path="$1"
    local entitlements_path="$2"
    local binary_name
    binary_name=$(basename "$binary_path")
    
    log_info "Signing binary: $binary_name"
    
    # Sign with Hardened Runtime
    codesign \
        --sign "$SIGNING_IDENTITY" \
        --entitlements "$entitlements_path" \
        --options runtime \
        --timestamp \
        --verbose \
        "$binary_path"
    
    # Verify signature
    if codesign --verify --verbose "$binary_path" &> /dev/null; then
        log_success "Binary '$binary_name' signed successfully"
    else
        log_error "Failed to verify signature for '$binary_name'"
        return 1
    fi
}

# Sign application bundle
sign_app_bundle() {
    local app_path="$1"
    local entitlements_path="$2"
    
    log_info "Signing application bundle: $(basename "$app_path")"
    
    # Find and sign all binaries in the bundle
    local macos_dir="$app_path/Contents/MacOS"
    if [[ -d "$macos_dir" ]]; then
        log_info "Signing binaries in MacOS directory..."
        for binary in "$macos_dir"/*; do
            if [[ -f "$binary" ]] && [[ -x "$binary" ]]; then
                sign_binary "$binary" "$entitlements_path"
            fi
        done
    fi
    
    # Sign frameworks if present
    local frameworks_dir="$app_path/Contents/Frameworks"
    if [[ -d "$frameworks_dir" ]]; then
        log_info "Signing frameworks..."
        for framework in "$frameworks_dir"/*.framework; do
            if [[ -d "$framework" ]]; then
                codesign \
                    --sign "$SIGNING_IDENTITY" \
                    --entitlements "$entitlements_path" \
                    --options runtime \
                    --timestamp \
                    --verbose \
                    "$framework"
            fi
        done
    fi
    
    # Sign the main application bundle
    log_info "Signing main application bundle..."
    codesign \
        --sign "$SIGNING_IDENTITY" \
        --entitlements "$entitlements_path" \
        --options runtime \
        --timestamp \
        --verbose \
        "$app_path"
    
    # Verify bundle signature
    if codesign --verify --verbose "$app_path" &> /dev/null; then
        log_success "Application bundle signed successfully"
    else
        log_error "Failed to verify application bundle signature"
        return 1
    fi
}

# Verify signatures
verify_signatures() {
    local target="$1"
    local target_name
    target_name=$(basename "$target")
    
    log_info "Verifying signatures for: $target_name"
    
    # Check codesign verification
    if codesign --verify --verbose "$target"; then
        log_success "Code signature verification passed"
    else
        log_error "Code signature verification failed"
        return 1
    fi
    
    # Check spctl assessment (Gatekeeper)
    log_info "Checking Gatekeeper assessment..."
    if spctl --assess --verbose --type execute "$target"; then
        log_success "Gatekeeper assessment passed"
    else
        log_warning "Gatekeeper assessment failed (may require notarization)"
        return 1
    fi
    
    # Display signature information
    log_info "Signature information:"
    codesign --display --verbose=4 "$target"
}

# Sign DMG file
sign_dmg_file() {
    local dmg_path="$1"
    local dmg_name
    dmg_name=$(basename "$dmg_path")
    
    log_info "Signing DMG: $dmg_name"
    
    # Sign the DMG
    codesign \
        --sign "$SIGNING_IDENTITY" \
        --timestamp \
        --verbose \
        "$dmg_path"
    
    # Verify DMG signature
    if codesign --verify --verbose "$dmg_path" &> /dev/null; then
        log_success "DMG signed successfully"
    else
        log_error "Failed to verify DMG signature"
        return 1
    fi
    
    # Check DMG with spctl
    log_info "Checking DMG with Gatekeeper..."
    if spctl --assess --verbose --type install "$dmg_path"; then
        log_success "DMG Gatekeeper assessment passed"
    else
        log_warning "DMG Gatekeeper assessment failed (may require notarization)"
    fi
}

# Process DMG signing
process_dmg() {
    local dmg_path="$1"
    
    if [[ ! -f "$dmg_path" ]]; then
        log_error "DMG file not found: $dmg_path"
        exit 1
    fi
    
    log_info "Processing DMG: $(basename "$dmg_path")"
    
    # Create temporary directory for extracted app
    local temp_dir
    temp_dir=$(mktemp -d)
    local mount_output
    local device
    local mount_point
    
    # Mount DMG
    log_info "Mounting DMG for signing..."
    mount_output=$(hdiutil attach -readonly -nobrowse "$dmg_path" | grep '/dev/disk')
    device=$(echo "$mount_output" | awk '{print $1}')
    mount_point=$(echo "$mount_output" | sed 's/.*\(\/Volumes\/.*\)/\1/')
    
    if [[ -z "$mount_point" ]]; then
        log_error "Failed to mount DMG"
        exit 1
    fi
    
    # Find app bundle in mounted DMG
    local app_bundle
    app_bundle=$(find "$mount_point" -name "*.app" -type d | head -n 1)
    
    if [[ -z "$app_bundle" ]]; then
        log_error "No application bundle found in DMG"
        hdiutil detach "$device" 2>/dev/null
        exit 1
    fi
    
    log_info "Found app bundle: $(basename "$app_bundle")"
    
    # Copy app bundle to temp directory
    cp -R "$app_bundle" "$temp_dir/"
    local temp_app="$temp_dir/$(basename "$app_bundle")"
    
    # Unmount DMG
    hdiutil detach "$device" 2>/dev/null
    
    if [[ "$VERIFY_ONLY" == true ]]; then
        log_info "Verification mode - checking existing signatures..."
        verify_signatures "$temp_app"
    else
        # Create entitlements file
        local entitlements_path="$temp_dir/entitlements.plist"
        create_entitlements "$entitlements_path"
        
        # Sign the app bundle
        sign_app_bundle "$temp_app" "$entitlements_path"
        
        # Verify signatures
        verify_signatures "$temp_app"
        
        # Create new signed DMG
        log_info "Creating new signed DMG..."
        local signed_dmg="${dmg_path%.*}-signed.dmg"
        local volume_name
        volume_name=$(basename "$dmg_path" .dmg)
        
        # Create new DMG with signed app
        hdiutil create -srcfolder "$temp_app" -volname "$volume_name" \
            -fs HFS+ -format UDZO "$signed_dmg"
        
        # Sign the new DMG
        sign_dmg_file "$signed_dmg"
        
        log_success "Signed DMG created: $signed_dmg"
    fi
    
    # Clean up
    rm -rf "$temp_dir"
}

# Main execution function
main() {
    log_info "CloudWorkstation DMG Code Signing"
    
    # Default DMG path if not provided
    if [[ -z "$DMG_PATH" ]]; then
        DMG_PATH="$PROJECT_ROOT/dist/dmg/CloudWorkstation-v0.4.2.dmg"
        log_info "Using default DMG path: $DMG_PATH"
    fi
    
    # Check prerequisites
    check_prerequisites
    
    # Verify signing identity (skip if verify-only)
    if [[ "$VERIFY_ONLY" == false ]]; then
        if ! verify_signing_identity "$SIGNING_IDENTITY"; then
            log_error "Please install a valid 'Developer ID Application' certificate"
            echo ""
            echo "To obtain a certificate:"
            echo "1. Join the Apple Developer Program"
            echo "2. Generate a Certificate Signing Request in Keychain Access"
            echo "3. Create a 'Developer ID Application' certificate at developer.apple.com"
            echo "4. Download and install the certificate"
            echo ""
            exit 1
        fi
    fi
    
    # Process the DMG
    process_dmg "$DMG_PATH"
    
    if [[ "$VERIFY_ONLY" == true ]]; then
        log_success "Signature verification completed"
    else
        log_success "DMG signing completed successfully!"
        echo ""
        echo "Next steps:"
        echo "1. Test the signed DMG on a different Mac"
        echo "2. Notarize with Apple: ./scripts/notarize-dmg.sh"
        echo "3. Distribute via GitHub releases"
    fi
}

# Run main function
main "$@"