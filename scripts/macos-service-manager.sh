#!/bin/bash

# CloudWorkstation macOS Service Manager
# Manages launchd service for CloudWorkstation daemon across user and system contexts

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLIST_TEMPLATE="$SCRIPT_DIR/com.prism.daemon.plist"
SERVICE_NAME="com.prism.daemon"

# Color output functions
red() { echo -e "\033[31m$*\033[0m"; }
green() { echo -e "\033[32m$*\033[0m"; }
yellow() { echo -e "\033[33m$*\033[0m"; }
blue() { echo -e "\033[34m$*\033[0m"; }

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

# Error handling
error_exit() {
    red "ERROR: $1" >&2
    exit 1
}

# Check if running as root
is_root() {
    [[ $EUID -eq 0 ]]
}

# Get installation context (user vs system)
get_install_context() {
    if is_root; then
        echo "system"
    else
        echo "user"
    fi
}

# Get appropriate paths based on context
get_paths() {
    local context="$1"
    
    if [[ "$context" == "system" ]]; then
        DAEMON_PATH="/usr/local/bin"
        LOG_PATH="/var/log/prism"
        PLIST_PATH="/Library/LaunchDaemons"
        CONFIG_PATH="/etc/prism"
        HOME_PATH="/var/lib/prism"
        USER_NAME="nobody"
    else
        DAEMON_PATH="$(brew --prefix)/bin" 2>/dev/null || DAEMON_PATH="/usr/local/bin"
        LOG_PATH="$HOME/Library/Logs/prism"
        PLIST_PATH="$HOME/Library/LaunchAgents"
        CONFIG_PATH="$HOME/.prism"
        HOME_PATH="$HOME"
        USER_NAME="$(whoami)"
    fi
}

# Create necessary directories
create_directories() {
    local context="$1"
    
    log "Creating necessary directories for $context installation..."
    
    if [[ "$context" == "system" ]]; then
        mkdir -p "$LOG_PATH" "$CONFIG_PATH" "$HOME_PATH" "$PLIST_PATH"
        chown "$USER_NAME:staff" "$LOG_PATH" "$CONFIG_PATH" "$HOME_PATH" 2>/dev/null || true
    else
        mkdir -p "$LOG_PATH" "$CONFIG_PATH" "$PLIST_PATH"
    fi
    
    green "✅ Directories created successfully"
}

# Generate plist from template
generate_plist() {
    local target_plist="$PLIST_PATH/$SERVICE_NAME.plist"
    
    log "Generating plist configuration at $target_plist..."
    
    # Replace template variables
    sed -e "s|{{DAEMON_PATH}}|$DAEMON_PATH|g" \
        -e "s|{{LOG_PATH}}|$LOG_PATH|g" \
        -e "s|{{HOME_PATH}}|$HOME_PATH|g" \
        -e "s|{{CONFIG_PATH}}|$CONFIG_PATH|g" \
        -e "s|{{USER_NAME}}|$USER_NAME|g" \
        "$PLIST_TEMPLATE" > "$target_plist"
    
    # Set appropriate permissions
    if [[ "$(get_install_context)" == "system" ]]; then
        chown root:wheel "$target_plist"
        chmod 644 "$target_plist"
    else
        chmod 644 "$target_plist"
    fi
    
    green "✅ Plist configuration generated successfully"
}

# Check if daemon binary exists
check_daemon_binary() {
    if [[ ! -x "$DAEMON_PATH/cwsd" ]]; then
        error_exit "CloudWorkstation daemon (cwsd) not found at $DAEMON_PATH. Please install CloudWorkstation first."
    fi
    
    log "✅ Found CloudWorkstation daemon at $DAEMON_PATH/cwsd"
}

# Install service
install_service() {
    local context="$(get_install_context)"
    get_paths "$context"
    
    log "Installing CloudWorkstation service ($context mode)..."
    
    # Check if already installed
    if service_installed; then
        yellow "⚠️  Service already installed. Use 'reinstall' to update configuration."
        return 0
    fi
    
    check_daemon_binary
    create_directories "$context"
    generate_plist
    
    # Load the service
    if [[ "$context" == "system" ]]; then
        launchctl load "$PLIST_PATH/$SERVICE_NAME.plist"
    else
        launchctl load "$PLIST_PATH/$SERVICE_NAME.plist"
    fi
    
    green "✅ CloudWorkstation service installed and started successfully"
    log "Service will start automatically on system boot"
    
    # Show status
    show_status
}

# Uninstall service
uninstall_service() {
    local context="$(get_install_context)"
    get_paths "$context"
    
    log "Uninstalling CloudWorkstation service ($context mode)..."
    
    if ! service_installed; then
        yellow "⚠️  Service not installed"
        return 0
    fi
    
    # Stop and unload service
    stop_service
    launchctl unload "$PLIST_PATH/$SERVICE_NAME.plist" 2>/dev/null || true
    
    # Remove plist file
    rm -f "$PLIST_PATH/$SERVICE_NAME.plist"
    
    green "✅ CloudWorkstation service uninstalled successfully"
}

# Reinstall service (update configuration)
reinstall_service() {
    log "Reinstalling CloudWorkstation service..."
    
    if service_installed; then
        uninstall_service
    fi
    
    install_service
}

# Start service
start_service() {
    local context="$(get_install_context)"
    get_paths "$context"
    
    if ! service_installed; then
        error_exit "Service not installed. Run 'install' first."
    fi
    
    log "Starting CloudWorkstation service..."
    
    if [[ "$context" == "system" ]]; then
        launchctl start "$SERVICE_NAME"
    else
        launchctl start "$SERVICE_NAME"
    fi
    
    green "✅ CloudWorkstation service started"
}

# Stop service
stop_service() {
    local context="$(get_install_context)"
    
    log "Stopping CloudWorkstation service..."
    
    if [[ "$context" == "system" ]]; then
        launchctl stop "$SERVICE_NAME" 2>/dev/null || true
    else
        launchctl stop "$SERVICE_NAME" 2>/dev/null || true
    fi
    
    green "✅ CloudWorkstation service stopped"
}

# Restart service
restart_service() {
    log "Restarting CloudWorkstation service..."
    stop_service
    sleep 2
    start_service
}

# Check if service is installed
service_installed() {
    local context="$(get_install_context)"
    get_paths "$context"
    
    [[ -f "$PLIST_PATH/$SERVICE_NAME.plist" ]]
}

# Check service status
show_status() {
    local context="$(get_install_context)"
    get_paths "$context"
    
    log "CloudWorkstation Service Status ($context mode):"
    echo
    
    if service_installed; then
        green "📦 Service: Installed"
        echo "   Plist: $PLIST_PATH/$SERVICE_NAME.plist"
        echo "   Daemon: $DAEMON_PATH/cwsd"
        echo "   Logs: $LOG_PATH/prism-daemon.log"
        echo
        
        # Check if service is loaded and running
        if launchctl list | grep -q "$SERVICE_NAME"; then
            local service_info="$(launchctl list "$SERVICE_NAME" 2>/dev/null || echo "")"
            if [[ -n "$service_info" ]]; then
                green "🟢 Status: Running"
                echo "$service_info" | while IFS= read -r line; do
                    echo "   $line"
                done
            else
                yellow "🟡 Status: Loaded but not running"
            fi
        else
            red "🔴 Status: Not loaded"
        fi
        
        # Show recent log entries
        if [[ -f "$LOG_PATH/prism-daemon.log" ]]; then
            echo
            blue "📝 Recent Log Entries:"
            tail -n 5 "$LOG_PATH/prism-daemon.log" | sed 's/^/   /'
        fi
    else
        red "❌ Service: Not installed"
    fi
    
    echo
}

# Show service logs
show_logs() {
    local context="$(get_install_context)"
    get_paths "$context"
    
    local log_file="$LOG_PATH/prism-daemon.log"
    
    if [[ -f "$log_file" ]]; then
        log "Showing CloudWorkstation service logs..."
        echo
        if command -v less >/dev/null 2>&1; then
            less "$log_file"
        else
            cat "$log_file"
        fi
    else
        yellow "⚠️  Log file not found at $log_file"
    fi
}

# Follow service logs
follow_logs() {
    local context="$(get_install_context)"
    get_paths "$context"
    
    local log_file="$LOG_PATH/prism-daemon.log"
    
    if [[ -f "$log_file" ]]; then
        log "Following CloudWorkstation service logs... (Press Ctrl+C to stop)"
        echo
        tail -f "$log_file"
    else
        yellow "⚠️  Log file not found at $log_file"
        echo "Service may not be running or configured correctly."
    fi
}

# Validate service configuration
validate_service() {
    local context="$(get_install_context)"
    get_paths "$context"
    
    log "Validating CloudWorkstation service configuration..."
    echo
    
    local errors=0
    
    # Check daemon binary
    if [[ -x "$DAEMON_PATH/cwsd" ]]; then
        green "✅ Daemon binary: Found at $DAEMON_PATH/cwsd"
    else
        red "❌ Daemon binary: Not found at $DAEMON_PATH/cwsd"
        ((errors++))
    fi
    
    # Check plist file
    if service_installed; then
        green "✅ Service plist: Installed"
        
        # Validate plist syntax
        if plutil -lint "$PLIST_PATH/$SERVICE_NAME.plist" >/dev/null 2>&1; then
            green "✅ Plist syntax: Valid"
        else
            red "❌ Plist syntax: Invalid"
            ((errors++))
        fi
    else
        red "❌ Service plist: Not installed"
        ((errors++))
    fi
    
    # Check directories
    for dir in "$LOG_PATH" "$CONFIG_PATH"; do
        if [[ -d "$dir" ]]; then
            green "✅ Directory: $dir exists"
        else
            yellow "⚠️  Directory: $dir does not exist (will be created if needed)"
        fi
    done
    
    echo
    if [[ $errors -eq 0 ]]; then
        green "🎉 Service configuration is valid!"
    else
        red "❌ Found $errors configuration errors"
        return 1
    fi
}

# Show help
show_help() {
    cat << 'EOF'
CloudWorkstation macOS Service Manager

USAGE:
    macos-service-manager.sh <command>

COMMANDS:
    install     Install and start CloudWorkstation service
    uninstall   Stop and uninstall CloudWorkstation service  
    reinstall   Update service configuration (uninstall + install)
    start       Start the service
    stop        Stop the service
    restart     Restart the service (stop + start)
    status      Show service status and configuration
    logs        Show service logs
    follow      Follow service logs in real-time
    validate    Validate service configuration
    help        Show this help message

EXAMPLES:
    # Install service for current user (Homebrew installation)
    ./macos-service-manager.sh install
    
    # Install system-wide service (requires sudo)
    sudo ./macos-service-manager.sh install
    
    # Check service status
    ./macos-service-manager.sh status
    
    # View real-time logs
    ./macos-service-manager.sh follow

NOTES:
    - User installation: Service runs when user is logged in
    - System installation: Service runs at system startup (requires sudo)
    - Service automatically restarts if daemon crashes
    - Logs are written to ~/Library/Logs/prism/ (user) or /var/log/prism/ (system)
EOF
}

# Main command handling
main() {
    case "${1:-help}" in
        install)
            install_service
            ;;
        uninstall)
            uninstall_service
            ;;
        reinstall)
            reinstall_service
            ;;
        start)
            start_service
            ;;
        stop)
            stop_service
            ;;
        restart)
            restart_service
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs
            ;;
        follow)
            follow_logs
            ;;
        validate)
            validate_service
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            red "Unknown command: $1"
            echo
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"