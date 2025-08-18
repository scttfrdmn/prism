#!/bin/bash

# CloudWorkstation DMG Notarization Script
# Submits DMG to Apple for notarization and staples the ticket
# Usage: ./scripts/notarize-dmg.sh [DMG_PATH] [--apple-id EMAIL] [--password PASSWORD] [--team-id TEAM]

set -euo pipefail

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Notarization settings
DMG_PATH=""
APPLE_ID=""
APP_PASSWORD=""
TEAM_ID=""
CHECK_STATUS=false
REQUEST_UUID=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --apple-id)
            APPLE_ID="$2"
            shift 2
            ;;
        --password)
            APP_PASSWORD="$2"
            shift 2
            ;;
        --team-id)
            TEAM_ID="$2"
            shift 2
            ;;
        --check-status)
            CHECK_STATUS=true
            REQUEST_UUID="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [DMG_PATH] [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  DMG_PATH                    Path to signed DMG file"
            echo "  --apple-id EMAIL            Apple ID for notarization"
            echo "  --password PASSWORD         App-specific password"
            echo "  --team-id TEAM              Team ID (for multiple teams)"
            echo "  --check-status UUID         Check notarization status"
            echo "  --help                      Show this help"
            echo ""
            echo "Examples:"
            echo "  $0 dist/dmg/CloudWorkstation-v0.4.2-signed.dmg --apple-id you@example.com --password abcd-efgh-ijkl-mnop"
            echo "  $0 --check-status 12345678-1234-1234-1234-123456789012"
            echo ""
            echo "Setup:"
            echo "1. Create app-specific password at appleid.apple.com"
            echo "2. Store in keychain: xcrun notarytool store-credentials --apple-id you@example.com --team-id TEAMID123"
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
    log_info "Checking notarization prerequisites..."
    
    # Check if running on macOS
    if [[ "$(uname)" != "Darwin" ]]; then
        log_error "Notarization requires macOS"
        exit 1
    fi
    
    # Check for required tools
    local required_tools=("xcrun" "ditto" "codesign")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' not found"
            exit 1
        fi
    done
    
    # Check for notarytool availability
    if ! xcrun notarytool --help &> /dev/null; then
        log_error "notarytool not available. Please update Xcode command line tools."
        echo "Run: xcode-select --install"
        exit 1
    fi
    
    # Check macOS version (notarytool requires macOS 10.15.7+)
    local macos_version
    macos_version=$(sw_vers -productVersion)
    if [[ "$(printf '%s\n' "10.15.7" "$macos_version" | sort -V | head -n1)" != "10.15.7" ]]; then
        log_warning "notarytool requires macOS 10.15.7 or later. Current version: $macos_version"
        log_info "Falling back to legacy altool method..."
        USE_LEGACY_ALTOOL=true
    fi
    
    log_success "Prerequisites check passed"
}

# Check stored credentials
check_stored_credentials() {
    log_info "Checking for stored notarization credentials..."
    
    # List stored credentials
    if xcrun notarytool history --keychain-profile cloudworkstation 2>/dev/null | head -1 &>/dev/null; then
        log_success "Found stored credentials with profile 'cloudworkstation'"
        return 0
    elif xcrun notarytool history --keychain-profile default 2>/dev/null | head -1 &>/dev/null; then
        log_success "Found stored credentials with profile 'default'"
        return 0
    else
        log_warning "No stored credentials found"
        return 1
    fi
}

# Store credentials in keychain
store_credentials() {
    local apple_id="$1"
    local password="$2"
    local team_id="$3"
    
    log_info "Storing notarization credentials in keychain..."
    
    local store_cmd="xcrun notarytool store-credentials cloudworkstation --apple-id $apple_id"
    
    if [[ -n "$team_id" ]]; then
        store_cmd="$store_cmd --team-id $team_id"
    fi
    
    echo "Please enter your app-specific password when prompted."
    echo "Create one at: https://appleid.apple.com/account/manage"
    echo ""
    
    if eval "$store_cmd"; then
        log_success "Credentials stored successfully"
    else
        log_error "Failed to store credentials"
        return 1
    fi
}

# Validate DMG before submission
validate_dmg() {
    local dmg_path="$1"
    
    log_info "Validating DMG before notarization..."
    
    # Check if file exists
    if [[ ! -f "$dmg_path" ]]; then
        log_error "DMG file not found: $dmg_path"
        exit 1
    fi
    
    # Check if DMG is signed
    if codesign --verify "$dmg_path" &> /dev/null; then
        log_success "DMG is properly code signed"
    else
        log_error "DMG is not code signed. Please run ./scripts/sign-dmg.sh first"
        exit 1
    fi
    
    # Check file size (Apple has a 2GB limit)
    local file_size
    file_size=$(stat -f%z "$dmg_path" 2>/dev/null || stat -c%s "$dmg_path" 2>/dev/null)
    local max_size=$((2 * 1024 * 1024 * 1024))  # 2GB
    
    if [[ "$file_size" -gt "$max_size" ]]; then
        log_error "DMG file is too large for notarization (max 2GB)"
        log_error "Current size: $(numfmt --to=iec "$file_size")"
        exit 1
    fi
    
    log_success "DMG validation passed"
}

# Submit for notarization using notarytool
submit_notarization_notarytool() {
    local dmg_path="$1"
    local dmg_name
    dmg_name=$(basename "$dmg_path")
    
    log_info "Submitting '$dmg_name' for notarization using notarytool..."
    
    # Try to use stored credentials first
    local submit_cmd=""
    if check_stored_credentials; then
        if xcrun notarytool history --keychain-profile cloudworkstation 2>/dev/null | head -1 &>/dev/null; then
            submit_cmd="xcrun notarytool submit \"$dmg_path\" --keychain-profile cloudworkstation --wait"
        else
            submit_cmd="xcrun notarytool submit \"$dmg_path\" --keychain-profile default --wait"
        fi
    elif [[ -n "$APPLE_ID" ]] && [[ -n "$APP_PASSWORD" ]]; then
        # Use provided credentials
        submit_cmd="xcrun notarytool submit \"$dmg_path\" --apple-id $APPLE_ID --password $APP_PASSWORD"
        if [[ -n "$TEAM_ID" ]]; then
            submit_cmd="$submit_cmd --team-id $TEAM_ID"
        fi
        submit_cmd="$submit_cmd --wait"
    else
        log_error "No notarization credentials available"
        echo ""
        echo "Please either:"
        echo "1. Store credentials: xcrun notarytool store-credentials cloudworkstation --apple-id your@email.com"
        echo "2. Provide credentials: $0 --apple-id your@email.com --password app-specific-password"
        echo ""
        exit 1
    fi
    
    log_info "Executing: $submit_cmd"
    echo "This may take several minutes..."
    
    # Submit and capture output
    local output
    if output=$(eval "$submit_cmd" 2>&1); then
        echo "$output"
        
        # Check if successful
        if echo "$output" | grep -q "Successfully received submission"; then
            local submission_id
            submission_id=$(echo "$output" | grep -E "id: [a-f0-9-]+" | sed 's/.*id: //')
            log_success "Notarization submitted successfully"
            log_info "Submission ID: $submission_id"
            
            # Check if notarization completed
            if echo "$output" | grep -q "status: Accepted"; then
                log_success "Notarization completed successfully!"
                return 0
            else
                log_error "Notarization failed or is still processing"
                echo "$output"
                return 1
            fi
        else
            log_error "Failed to submit for notarization"
            echo "$output"
            return 1
        fi
    else
        log_error "Notarization submission failed"
        echo "$output"
        return 1
    fi
}

# Check notarization status
check_notarization_status() {
    local request_uuid="$1"
    
    log_info "Checking notarization status for request: $request_uuid"
    
    local status_cmd=""
    if check_stored_credentials; then
        if xcrun notarytool history --keychain-profile cloudworkstation 2>/dev/null | head -1 &>/dev/null; then
            status_cmd="xcrun notarytool info $request_uuid --keychain-profile cloudworkstation"
        else
            status_cmd="xcrun notarytool info $request_uuid --keychain-profile default"
        fi
    else
        log_error "No stored credentials found for status check"
        exit 1
    fi
    
    if eval "$status_cmd"; then
        log_success "Status check completed"
    else
        log_error "Failed to check notarization status"
        return 1
    fi
}

# Staple notarization ticket
staple_ticket() {
    local dmg_path="$1"
    local dmg_name
    dmg_name=$(basename "$dmg_path")
    
    log_info "Stapling notarization ticket to '$dmg_name'..."
    
    if xcrun stapler staple "$dmg_path"; then
        log_success "Notarization ticket stapled successfully"
        
        # Verify stapling
        if xcrun stapler validate "$dmg_path"; then
            log_success "Stapled ticket validated successfully"
        else
            log_warning "Stapled ticket validation failed"
            return 1
        fi
    else
        log_error "Failed to staple notarization ticket"
        return 1
    fi
}

# Verify final notarization
verify_notarization() {
    local dmg_path="$1"
    local dmg_name
    dmg_name=$(basename "$dmg_path")
    
    log_info "Verifying final notarization for '$dmg_name'..."
    
    # Check with spctl
    if spctl --assess --verbose --type install "$dmg_path"; then
        log_success "Gatekeeper assessment passed - DMG is properly notarized"
    else
        log_error "Gatekeeper assessment failed"
        return 1
    fi
    
    # Display notarization information
    log_info "Notarization information:"
    codesign --display --verbose "$dmg_path" || true
}

# Create notarized DMG with proper naming
create_notarized_dmg() {
    local source_dmg="$1"
    local notarized_dmg="${source_dmg%-signed.dmg}-notarized.dmg"
    
    if [[ "$source_dmg" != "$notarized_dmg" ]]; then
        log_info "Creating final notarized DMG: $(basename "$notarized_dmg")"
        cp "$source_dmg" "$notarized_dmg"
        log_success "Notarized DMG created: $notarized_dmg"
        echo "Final DMG: $notarized_dmg"
    else
        echo "Final DMG: $source_dmg"
    fi
}

# Main execution function
main() {
    log_info "CloudWorkstation DMG Notarization"
    
    # Handle status check mode
    if [[ "$CHECK_STATUS" == true ]]; then
        check_prerequisites
        check_notarization_status "$REQUEST_UUID"
        exit 0
    fi
    
    # Default DMG path if not provided
    if [[ -z "$DMG_PATH" ]]; then
        # Look for signed DMG first
        local signed_dmg="$PROJECT_ROOT/dist/dmg/CloudWorkstation-v0.4.2-signed.dmg"
        local unsigned_dmg="$PROJECT_ROOT/dist/dmg/CloudWorkstation-v0.4.2.dmg"
        
        if [[ -f "$signed_dmg" ]]; then
            DMG_PATH="$signed_dmg"
        elif [[ -f "$unsigned_dmg" ]]; then
            DMG_PATH="$unsigned_dmg"
        else
            log_error "No DMG file found. Please build and sign first."
            echo "Run: make dmg-signed"
            exit 1
        fi
        
        log_info "Using DMG: $DMG_PATH"
    fi
    
    # Check prerequisites
    check_prerequisites
    
    # Validate DMG
    validate_dmg "$DMG_PATH"
    
    # Submit for notarization
    if submit_notarization_notarytool "$DMG_PATH"; then
        # Staple the ticket
        if staple_ticket "$DMG_PATH"; then
            # Verify final notarization
            verify_notarization "$DMG_PATH"
            
            # Create final notarized DMG
            create_notarized_dmg "$DMG_PATH"
            
            log_success "DMG notarization completed successfully!"
            echo ""
            echo "Your DMG is now ready for distribution:"
            echo "• Code signed with Developer ID"
            echo "• Notarized by Apple"
            echo "• Stapled notarization ticket"
            echo "• Gatekeeper approved"
            echo ""
            echo "Next steps:"
            echo "1. Test on a different Mac without Developer tools"
            echo "2. Upload to GitHub releases"
            echo "3. Update distribution documentation"
        else
            log_error "Failed to staple notarization ticket"
            exit 1
        fi
    else
        log_error "Notarization failed"
        exit 1
    fi
}

# Run main function
main "$@"