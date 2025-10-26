#!/bin/bash

# CloudWorkstation macOS Uninstaller
# Removes all CloudWorkstation components from macOS system
# Usage: ./scripts/macos-uninstall.sh [--complete] [--keep-data]

set -euo pipefail

# Configuration
readonly INSTALL_DIR="/usr/local/bin"
readonly PRISM_DIR="$HOME/.cloudworkstation"
readonly LAUNCH_AGENT_PLIST="$HOME/Library/LaunchAgents/com.cloudworkstation.daemon.plist"
readonly APP_APPLICATIONS="/Applications/CloudWorkstation.app"
readonly DESKTOP_SHORTCUT="$HOME/Desktop/CloudWorkstation.command"

# Uninstall options
COMPLETE_REMOVAL=false
KEEP_USER_DATA=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --complete)
            COMPLETE_REMOVAL=true
            shift
            ;;
        --keep-data)
            KEEP_USER_DATA=true
            shift
            ;;
        -h|--help)
            echo "CloudWorkstation macOS Uninstaller"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --complete    Remove everything including user data and preferences"
            echo "  --keep-data   Keep user data and AWS profiles (remove only binaries)"
            echo "  --help        Show this help"
            echo ""
            echo "Default behavior removes application files but keeps user data."
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
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

# Show confirmation dialog
show_confirmation_dialog() {
    local message="$1"
    osascript << EOD
        display dialog "$message" with title "CloudWorkstation Uninstall" buttons {"Cancel", "Uninstall"} default button "Cancel" with icon caution
EOD
}

# Show completion dialog
show_completion_dialog() {
    local message="$1"
    osascript << EOD
        display dialog "$message" with title "CloudWorkstation Uninstall Complete" buttons {"OK"} default button "OK" with icon note
EOD
}

# Stop and unload daemon
stop_daemon() {
    log_info "Stopping CloudWorkstation daemon..."
    
    # Stop any running daemon processes
    if pgrep -f "cwsd" > /dev/null; then
        log_info "Stopping running daemon processes..."
        pkill -f "cwsd" || true
        sleep 2
    fi
    
    # Unload LaunchAgent
    if [[ -f "$LAUNCH_AGENT_PLIST" ]]; then
        log_info "Unloading LaunchAgent..."
        launchctl unload "$LAUNCH_AGENT_PLIST" 2>/dev/null || true
        rm -f "$LAUNCH_AGENT_PLIST"
        log_success "LaunchAgent removed"
    fi
    
    log_success "Daemon stopped and unloaded"
}

# Remove command line tools
remove_cli_tools() {
    log_info "Removing command-line tools..."
    
    local removed_tools=()
    
    # Remove cws binary
    if [[ -f "$INSTALL_DIR/cws" ]]; then
        if [[ -w "$INSTALL_DIR" ]]; then
            rm -f "$INSTALL_DIR/cws"
            removed_tools+=("cws")
        else
            # Need sudo for removal
            if sudo rm -f "$INSTALL_DIR/cws" 2>/dev/null; then
                removed_tools+=("cws")
            else
                log_error "Failed to remove $INSTALL_DIR/cws"
            fi
        fi
    fi
    
    # Remove cwsd binary
    if [[ -f "$INSTALL_DIR/cwsd" ]]; then
        if [[ -w "$INSTALL_DIR" ]]; then
            rm -f "$INSTALL_DIR/cwsd"
            removed_tools+=("cwsd")
        else
            # Need sudo for removal
            if sudo rm -f "$INSTALL_DIR/cwsd" 2>/dev/null; then
                removed_tools+=("cwsd")
            else
                log_error "Failed to remove $INSTALL_DIR/cwsd"
            fi
        fi
    fi
    
    if [[ ${#removed_tools[@]} -gt 0 ]]; then
        log_success "Removed CLI tools: ${removed_tools[*]}"
    else
        log_info "No CLI tools found to remove"
    fi
}

# Remove application bundle
remove_app_bundle() {
    log_info "Removing application bundle..."
    
    if [[ -d "$APP_APPLICATIONS" ]]; then
        # Stop the application if it's running
        osascript << 'EOF' 2>/dev/null || true
            tell application "CloudWorkstation" to quit
EOF
        sleep 1
        
        # Remove the app bundle
        rm -rf "$APP_APPLICATIONS"
        log_success "Application bundle removed from Applications folder"
    else
        log_info "No application bundle found in Applications folder"
    fi
}

# Remove desktop shortcut
remove_desktop_shortcut() {
    log_info "Removing desktop shortcut..."
    
    if [[ -f "$DESKTOP_SHORTCUT" ]]; then
        rm -f "$DESKTOP_SHORTCUT"
        log_success "Desktop shortcut removed"
    else
        log_info "No desktop shortcut found"
    fi
}

# Clean shell configuration
clean_shell_config() {
    log_info "Cleaning shell configuration..."
    
    local shell_configs=("$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.zshrc" "$HOME/.profile" "$HOME/.config/fish/config.fish")
    local cleaned_files=()
    
    for config_file in "${shell_configs[@]}"; do
        if [[ -f "$config_file" ]] && grep -q "cloudworkstation\|$INSTALL_DIR.*cws" "$config_file"; then
            # Create backup
            cp "$config_file" "$config_file.bak.$(date +%Y%m%d%H%M%S)"
            
            # Remove CloudWorkstation-related lines
            if [[ "$config_file" == *"config.fish" ]]; then
                sed -i '' '/set -gx PATH.*cws/d; /CloudWorkstation/d' "$config_file" 2>/dev/null || true
            else
                sed -i '' '/export PATH.*cws/d; /CloudWorkstation/d' "$config_file" 2>/dev/null || true
            fi
            
            cleaned_files+=("$(basename "$config_file")")
        fi
    done
    
    if [[ ${#cleaned_files[@]} -gt 0 ]]; then
        log_success "Cleaned shell configurations: ${cleaned_files[*]}"
        log_info "Backup files created with .bak extension"
    else
        log_info "No shell configuration cleanup needed"
    fi
}

# Remove user data and preferences
remove_user_data() {
    log_info "Removing user data and preferences..."
    
    local removed_items=()
    
    # Remove CloudWorkstation directory
    if [[ -d "$PRISM_DIR" ]]; then
        rm -rf "$PRISM_DIR"
        removed_items+=(".cloudworkstation directory")
    fi
    
    # Remove macOS preferences
    local prefs_files=(
        "$HOME/Library/Preferences/com.cloudworkstation.app.plist"
        "$HOME/Library/Preferences/com.cloudworkstation.daemon.plist"
        "$HOME/Library/Preferences/com.cloudworkstation.cli.plist"
    )
    
    for pref_file in "${prefs_files[@]}"; do
        if [[ -f "$pref_file" ]]; then
            rm -f "$pref_file"
            removed_items+=("$(basename "$pref_file")")
        fi
    done
    
    # Remove application support files
    local app_support_dir="$HOME/Library/Application Support/CloudWorkstation"
    if [[ -d "$app_support_dir" ]]; then
        rm -rf "$app_support_dir"
        removed_items+=("Application Support files")
    fi
    
    # Remove caches
    local cache_dir="$HOME/Library/Caches/com.cloudworkstation.app"
    if [[ -d "$cache_dir" ]]; then
        rm -rf "$cache_dir"
        removed_items+=("Cache files")
    fi
    
    if [[ ${#removed_items[@]} -gt 0 ]]; then
        log_success "Removed user data: ${removed_items[*]}"
    else
        log_info "No user data found to remove"
    fi
}

# Remove logs and temporary files
remove_logs_and_temp() {
    log_info "Removing logs and temporary files..."
    
    local temp_locations=(
        "/tmp/cwsd.log"
        "/tmp/cloudworkstation-*"
        "$HOME/Library/Logs/CloudWorkstation"
    )
    
    local removed_items=()
    
    for location in "${temp_locations[@]}"; do
        if [[ -e "$location" ]] || [[ -n "$(ls $location 2>/dev/null)" ]]; then
            rm -rf $location 2>/dev/null || true
            removed_items+=("$(basename "$location")")
        fi
    done
    
    if [[ ${#removed_items[@]} -gt 0 ]]; then
        log_success "Removed temporary files: ${removed_items[*]}"
    else
        log_info "No temporary files found to remove"
    fi
}

# Verify removal
verify_removal() {
    log_info "Verifying removal..."
    
    local remaining_items=()
    
    # Check for remaining binaries
    if [[ -f "$INSTALL_DIR/cws" ]] || [[ -f "$INSTALL_DIR/cwsd" ]]; then
        remaining_items+=("CLI tools")
    fi
    
    # Check for app bundle
    if [[ -d "$APP_APPLICATIONS" ]]; then
        remaining_items+=("Application bundle")
    fi
    
    # Check for daemon
    if pgrep -f "cwsd" > /dev/null; then
        remaining_items+=("Running daemon")
    fi
    
    # Check for LaunchAgent
    if [[ -f "$LAUNCH_AGENT_PLIST" ]]; then
        remaining_items+=("LaunchAgent")
    fi
    
    if [[ "$KEEP_USER_DATA" == false ]] && [[ -d "$PRISM_DIR" ]]; then
        remaining_items+=("User data")
    fi
    
    if [[ ${#remaining_items[@]} -gt 0 ]]; then
        log_warning "Some items could not be removed: ${remaining_items[*]}"
        return 1
    else
        log_success "All components removed successfully"
        return 0
    fi
}

# Main uninstall function
main() {
    log_info "CloudWorkstation macOS Uninstaller"
    log_info "Complete removal: $COMPLETE_REMOVAL"
    log_info "Keep user data: $KEEP_USER_DATA"
    
    # Show confirmation dialog
    local confirmation_message="Are you sure you want to uninstall CloudWorkstation?"
    
    if [[ "$COMPLETE_REMOVAL" == true ]]; then
        confirmation_message="$confirmation_message

WARNING: This will remove ALL CloudWorkstation data including:
• Application files and binaries
• User data and AWS profiles
• Configuration and preferences
• Logs and temporary files

This action cannot be undone."
    elif [[ "$KEEP_USER_DATA" == true ]]; then
        confirmation_message="$confirmation_message

This will remove only the application files while preserving:
• User data and AWS profiles
• Configuration files
• Templates and customizations"
    else
        confirmation_message="$confirmation_message

This will remove application files but keep user data that you can restore later."
    fi
    
    local response
    response=$(show_confirmation_dialog "$confirmation_message") || {
        log_info "Uninstall cancelled by user"
        exit 0
    }
    
    if [[ "$response" != *"Uninstall"* ]]; then
        log_info "Uninstall cancelled by user"
        exit 0
    fi
    
    log_info "Starting CloudWorkstation uninstallation..."
    
    # Stop daemon and services first
    stop_daemon
    
    # Remove application components
    remove_cli_tools
    remove_app_bundle
    remove_desktop_shortcut
    clean_shell_config
    
    # Handle user data based on options
    if [[ "$COMPLETE_REMOVAL" == true ]]; then
        remove_user_data
        remove_logs_and_temp
    elif [[ "$KEEP_USER_DATA" == false ]]; then
        # Default: remove data but ask for confirmation
        local data_response
        data_response=$(show_confirmation_dialog "Do you also want to remove your CloudWorkstation data and AWS profiles? Choose 'Cancel' to keep your data for future installations.") || true
        
        if [[ "$data_response" == *"Uninstall"* ]]; then
            remove_user_data
        else
            log_info "Keeping user data and preferences"
        fi
        remove_logs_and_temp
    else
        log_info "Keeping user data as requested"
        remove_logs_and_temp
    fi
    
    # Verify removal
    if verify_removal; then
        show_completion_dialog "CloudWorkstation has been successfully uninstalled.

If you kept your user data, it will be restored if you reinstall CloudWorkstation in the future.

Thank you for using CloudWorkstation!"
        
        log_success "CloudWorkstation uninstallation completed successfully!"
    else
        show_completion_dialog "CloudWorkstation uninstallation completed with some warnings. Please check the console for details."
        log_warning "Uninstallation completed with warnings"
    fi
    
    echo ""
    echo "Uninstallation summary:"
    echo "• Application files: Removed"
    echo "• CLI tools: Removed"
    echo "• Daemon service: Stopped and removed"
    if [[ "$COMPLETE_REMOVAL" == true ]] || ([[ "$KEEP_USER_DATA" == false ]] && [[ "$?" == 0 ]]); then
        echo "• User data: Removed"
    else
        echo "• User data: Preserved in $PRISM_DIR"
    fi
    echo ""
    echo "To reinstall CloudWorkstation, download the latest DMG from:"
    echo "https://github.com/scttfrdmn/prism/releases"
}

# Run main function
main "$@"