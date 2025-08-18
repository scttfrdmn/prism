#!/bin/bash

# CloudWorkstation macOS Post-Installation Script
# Handles PATH setup, service installation, and initial configuration
# This script runs automatically after DMG installation

set -euo pipefail

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly APP_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"  # From Resources/scripts to app root
readonly MACOS_DIR="$APP_DIR/Contents/MacOS"
readonly INSTALL_DIR="/usr/local/bin"
readonly CLOUDWORKSTATION_DIR="$HOME/.cloudworkstation"

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

# Show installation dialog
show_install_dialog() {
    local message="$1"
    osascript << EOD
        display dialog "$message" with title "CloudWorkstation Installation" buttons {"Skip", "Install"} default button "Install" with icon note
EOD
}

# Show info dialog
show_info_dialog() {
    local message="$1"
    osascript << EOD
        display dialog "$message" with title "CloudWorkstation Setup" buttons {"OK"} default button "OK" with icon note
EOD
}

# Show error dialog
show_error_dialog() {
    local message="$1"
    osascript << EOD
        display dialog "$message" with title "CloudWorkstation Setup Error" buttons {"OK"} default button "OK" with icon stop
EOD
}

# Check if command line tools are already installed
check_cli_tools_installed() {
    if [[ -f "$INSTALL_DIR/cws" ]] && [[ -f "$INSTALL_DIR/cwsd" ]]; then
        # Check if they're the same version as in the app bundle
        local installed_version
        local bundle_version
        
        installed_version=$("$INSTALL_DIR/cws" --version 2>/dev/null | grep -o "v[0-9]\+\.[0-9]\+\.[0-9]\+" || echo "unknown")
        bundle_version=$("$MACOS_DIR/cws" --version 2>/dev/null | grep -o "v[0-9]\+\.[0-9]\+\.[0-9]\+" || echo "unknown")
        
        if [[ "$installed_version" == "$bundle_version" ]] && [[ "$installed_version" != "unknown" ]]; then
            return 0  # Same version already installed
        fi
    fi
    return 1  # Not installed or different version
}

# Install command line tools
install_cli_tools() {
    log_info "Installing CloudWorkstation command-line tools..."
    
    # Check if installation directory exists
    if [[ ! -d "$INSTALL_DIR" ]]; then
        # Try to create it
        if ! sudo mkdir -p "$INSTALL_DIR" 2>/dev/null; then
            log_error "Cannot create $INSTALL_DIR. Using /usr/bin instead."
            INSTALL_DIR="/usr/bin"
        fi
    fi
    
    # Check write permissions
    if [[ ! -w "$INSTALL_DIR" ]]; then
        log_info "Administrator privileges required for installation to $INSTALL_DIR"
        
        # Use AppleScript to get password
        local password_result
        password_result=$(osascript << 'EOD'
            display dialog "Administrator password required to install CloudWorkstation CLI tools:" with title "CloudWorkstation Installation" default answer "" with hidden answer buttons {"Cancel", "OK"} default button "OK"
EOD
        ) || {
            log_warning "Installation cancelled by user"
            return 1
        }
        
        if [[ "$password_result" == *"OK"* ]]; then
            local password
            password=$(echo "$password_result" | sed 's/.*text returned://; s/, button returned:.*//')
            
            # Install with sudo
            if echo "$password" | sudo -S cp "$MACOS_DIR/cws" "$INSTALL_DIR/" 2>/dev/null && \
               echo "$password" | sudo -S cp "$MACOS_DIR/cwsd" "$INSTALL_DIR/" 2>/dev/null && \
               echo "$password" | sudo -S chmod +x "$INSTALL_DIR/cws" "$INSTALL_DIR/cwsd" 2>/dev/null; then
                log_success "CLI tools installed to $INSTALL_DIR"
                return 0
            else
                log_error "Installation failed"
                return 1
            fi
        else
            log_warning "Installation cancelled"
            return 1
        fi
    else
        # Install without sudo
        if cp "$MACOS_DIR/cws" "$INSTALL_DIR/" && \
           cp "$MACOS_DIR/cwsd" "$INSTALL_DIR/" && \
           chmod +x "$INSTALL_DIR/cws" "$INSTALL_DIR/cwsd"; then
            log_success "CLI tools installed to $INSTALL_DIR"
            return 0
        else
            log_error "Installation failed"
            return 1
        fi
    fi
}

# Setup shell PATH configuration
setup_shell_path() {
    log_info "Setting up shell PATH configuration..."
    
    # Determine user shell
    local user_shell="${SHELL##*/}"
    local profile_file=""
    
    case "$user_shell" in
        bash)
            if [[ -f "$HOME/.bash_profile" ]]; then
                profile_file="$HOME/.bash_profile"
            else
                profile_file="$HOME/.bashrc"
            fi
            ;;
        zsh)
            profile_file="$HOME/.zshrc"
            ;;
        fish)
            profile_file="$HOME/.config/fish/config.fish"
            ;;
        *)
            log_warning "Unknown shell: $user_shell. Using .profile"
            profile_file="$HOME/.profile"
            ;;
    esac
    
    # Add PATH if not already present
    local path_export=""
    if [[ "$user_shell" == "fish" ]]; then
        path_export="set -gx PATH $INSTALL_DIR \$PATH"
    else
        path_export="export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
    
    if [[ -f "$profile_file" ]] && grep -q "$INSTALL_DIR" "$profile_file"; then
        log_info "PATH already configured in $profile_file"
    else
        echo "" >> "$profile_file"
        echo "# CloudWorkstation CLI tools" >> "$profile_file"
        echo "$path_export" >> "$profile_file"
        log_success "PATH configured in $profile_file"
    fi
}

# Create application directories
create_app_directories() {
    log_info "Creating application directories..."
    
    # Create main config directory
    mkdir -p "$CLOUDWORKSTATION_DIR"
    mkdir -p "$CLOUDWORKSTATION_DIR/profiles"
    mkdir -p "$CLOUDWORKSTATION_DIR/templates"
    mkdir -p "$CLOUDWORKSTATION_DIR/logs"
    mkdir -p "$CLOUDWORKSTATION_DIR/cache"
    
    # Set appropriate permissions
    chmod 755 "$CLOUDWORKSTATION_DIR"
    chmod 700 "$CLOUDWORKSTATION_DIR/profiles"  # Sensitive AWS credentials
    
    log_success "Application directories created"
}

# Install LaunchAgent for daemon auto-start
install_launch_agent() {
    log_info "Installing LaunchAgent for daemon auto-start..."
    
    local launch_agents_dir="$HOME/Library/LaunchAgents"
    local plist_file="$launch_agents_dir/com.cloudworkstation.daemon.plist"
    
    mkdir -p "$launch_agents_dir"
    
    # Create LaunchAgent plist
    cat > "$plist_file" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.cloudworkstation.daemon</string>
    
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/cwsd</string>
    </array>
    
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    
    <key>RunAtLoad</key>
    <true/>
    
    <key>WorkingDirectory</key>
    <string>$CLOUDWORKSTATION_DIR</string>
    
    <key>StandardErrorPath</key>
    <string>$CLOUDWORKSTATION_DIR/logs/daemon.log</string>
    
    <key>StandardOutPath</key>
    <string>$CLOUDWORKSTATION_DIR/logs/daemon.log</string>
    
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>$INSTALL_DIR:/usr/local/bin:/usr/bin:/bin</string>
    </dict>
    
    <key>ThrottleInterval</key>
    <integer>10</integer>
</dict>
</plist>
EOF
    
    # Set correct permissions
    chmod 644 "$plist_file"
    
    # Load the LaunchAgent
    if launchctl load "$plist_file" 2>/dev/null; then
        log_success "LaunchAgent installed and loaded"
    else
        log_warning "LaunchAgent installed but failed to load (will load on next login)"
    fi
}

# Copy templates to user directory
copy_templates() {
    log_info "Copying templates to user directory..."
    
    local templates_source="$APP_DIR/Contents/Resources/templates"
    local templates_dest="$CLOUDWORKSTATION_DIR/templates"
    
    if [[ -d "$templates_source" ]]; then
        cp -r "$templates_source"/* "$templates_dest/" 2>/dev/null || {
            log_warning "Some templates could not be copied"
        }
        log_success "Templates copied to $templates_dest"
    else
        log_warning "No templates found in app bundle"
    fi
}

# Create desktop shortcut
create_desktop_shortcut() {
    log_info "Creating desktop shortcut..."
    
    local desktop_dir="$HOME/Desktop"
    local shortcut_path="$desktop_dir/CloudWorkstation.command"
    
    if [[ -d "$desktop_dir" ]]; then
        cat > "$shortcut_path" << EOF
#!/bin/bash
# CloudWorkstation Desktop Shortcut
# Double-click to open CloudWorkstation terminal interface

clear
echo "CloudWorkstation v0.4.2"
echo "======================"
echo ""
echo "Available commands:"
echo "  cws --help        Show help"
echo "  cws templates     List available templates"
echo "  cws profiles      Manage AWS profiles"
echo "  cws list          List running instances"
echo ""
echo "Getting started:"
echo "  cws profiles create my-profile    # Setup AWS"
echo "  cws launch python-ml my-project  # Launch workstation"
echo ""

# Keep terminal open
bash
EOF
        
        chmod +x "$shortcut_path"
        log_success "Desktop shortcut created"
    else
        log_info "Desktop directory not found, skipping shortcut creation"
    fi
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."
    
    local errors=0
    
    # Check CLI tools
    if command -v cws &> /dev/null && command -v cwsd &> /dev/null; then
        log_success "CLI tools are accessible via PATH"
    else
        log_error "CLI tools not found in PATH"
        ((errors++))
    fi
    
    # Check directories
    if [[ -d "$CLOUDWORKSTATION_DIR" ]]; then
        log_success "Application directories created"
    else
        log_error "Application directories not found"
        ((errors++))
    fi
    
    # Check daemon
    if pgrep -f "cwsd" > /dev/null; then
        log_success "CloudWorkstation daemon is running"
    else
        log_info "Daemon not running (will start on next login)"
    fi
    
    return $errors
}

# Main installation function
main() {
    log_info "CloudWorkstation macOS Post-Installation Setup"
    log_info "App bundle: $APP_DIR"
    
    # Create application directories first
    create_app_directories
    
    # Ask user about CLI tools installation
    if ! check_cli_tools_installed; then
        local install_response
        install_response=$(show_install_dialog "Would you like to install CloudWorkstation command-line tools (cws, cwsd) to $INSTALL_DIR? This allows you to use CloudWorkstation from any terminal window.")
        
        if [[ "$install_response" == *"Install"* ]]; then
            if install_cli_tools; then
                setup_shell_path
                install_launch_agent
            else
                show_error_dialog "Failed to install command-line tools. You can still use the CloudWorkstation.app directly."
            fi
        else
            log_info "Skipping CLI tools installation"
        fi
    else
        log_info "CLI tools already installed and up to date"
    fi
    
    # Copy templates
    copy_templates
    
    # Create desktop shortcut (optional)
    create_desktop_shortcut
    
    # Verify installation
    if verify_installation; then
        show_info_dialog "CloudWorkstation installation completed successfully!

Available interfaces:
• CloudWorkstation.app - GUI interface
• 'cws' command - Terminal interface
• 'cwsd' daemon - Background service

Next steps:
1. Configure AWS credentials
2. Browse available templates
3. Launch your first workstation

Documentation: Help menu in CloudWorkstation.app"
        
        log_success "Post-installation setup completed successfully!"
    else
        show_error_dialog "Installation completed with some errors. Please check the console for details."
        log_error "Post-installation setup completed with errors"
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi