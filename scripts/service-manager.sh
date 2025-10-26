#!/bin/bash

# CloudWorkstation Cross-Platform Service Manager
# Universal service management for CloudWorkstation daemon across macOS, Linux, and Windows

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Color output functions
red() { echo -e "\033[31m$*\033[0m"; }
green() { echo -e "\033[32m$*\033[0m"; }
yellow() { echo -e "\033[33m$*\033[0m"; }
blue() { echo -e "\033[34m$*\033[0m"; }
cyan() { echo -e "\033[36m$*\033[0m"; }

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

# Error handling
error_exit() {
    red "ERROR: $1" >&2
    exit 1
}

# Detect operating system
detect_os() {
    case "$(uname -s)" in
        Darwin)
            echo "macos"
            ;;
        Linux)
            echo "linux"
            ;;
        CYGWIN*|MINGW*|MSYS*)
            echo "windows"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

# Get platform-specific service manager
get_platform_manager() {
    local os="$1"
    
    case "$os" in
        macos)
            echo "$SCRIPT_DIR/macos-service-manager.sh"
            ;;
        linux)
            echo "$SCRIPT_DIR/linux-service-manager.sh"
            ;;
        windows)
            echo "$SCRIPT_DIR/windows-service-manager.ps1"
            ;;
        *)
            error_exit "Unsupported operating system: $os"
            ;;
    esac
}

# Check if platform manager exists
check_platform_manager() {
    local manager="$1"
    local os="$2"
    
    if [[ ! -f "$manager" ]]; then
        error_exit "Platform service manager not found: $manager"
    fi
    
    if [[ "$os" != "windows" && ! -x "$manager" ]]; then
        error_exit "Platform service manager not executable: $manager"
    fi
}

# Execute platform-specific command
execute_platform_command() {
    local os="$1"
    local manager="$2"
    shift 2
    local args=("$@")
    
    log "Executing $os service management: ${args[*]}"
    
    case "$os" in
        macos|linux)
            exec "$manager" "${args[@]}"
            ;;
        windows)
            if command -v pwsh >/dev/null 2>&1; then
                exec pwsh -File "$manager" "${args[@]}"
            elif command -v powershell >/dev/null 2>&1; then
                exec powershell -File "$manager" "${args[@]}"
            else
                error_exit "PowerShell not found. Please install PowerShell to manage Windows services."
            fi
            ;;
        *)
            error_exit "Unsupported operating system: $os"
            ;;
    esac
}

# Show platform-specific help
show_platform_help() {
    local os="$(detect_os)"
    local manager="$(get_platform_manager "$os")"
    
    log "CloudWorkstation Cross-Platform Service Manager"
    log "Detected OS: $os"
    echo
    
    case "$os" in
        macos)
            blue "📱 macOS Service Management"
            echo "   Delegating to: $manager"
            echo
            "$manager" help
            ;;
        linux)
            blue "🐧 Linux Service Management"
            echo "   Delegating to: $manager"
            echo
            "$manager" help
            ;;
        windows)
            blue "🪟 Windows Service Management"
            echo "   Delegating to: $manager"
            echo
            if command -v pwsh >/dev/null 2>&1; then
                pwsh -File "$manager" help
            elif command -v powershell >/dev/null 2>&1; then
                powershell -File "$manager" help
            else
                yellow "⚠️  PowerShell not available. Windows service management requires PowerShell."
            fi
            ;;
        unknown)
            red "❌ Unsupported operating system"
            echo
            show_general_help
            ;;
    esac
}

# Show general help information
show_general_help() {
    cat << 'EOF'
CloudWorkstation Cross-Platform Service Manager

This script provides unified service management across different operating systems.
It automatically detects your platform and delegates to the appropriate service manager.

SUPPORTED PLATFORMS:
    macOS       - Uses launchd for service management
    Linux       - Uses systemd for service management  
    Windows     - Uses Windows Service Manager (requires PowerShell)

USAGE:
    service-manager.sh <command> [options]

COMMON COMMANDS:
    install     Install and start CloudWorkstation service
    uninstall   Stop and uninstall CloudWorkstation service
    start       Start the service
    stop        Stop the service
    restart     Restart the service
    status      Show service status and configuration
    logs        Show service logs
    follow      Follow service logs in real-time (where supported)
    validate    Validate service configuration
    help        Show platform-specific help

EXAMPLES:
    # Install service on any platform
    ./service-manager.sh install
    
    # Check status on any platform
    ./service-manager.sh status
    
    # Follow logs in real-time
    ./service-manager.sh follow

PLATFORM-SPECIFIC NOTES:

macOS:
    - Service runs when user is logged in (user mode) or at system startup (system mode)
    - Supports both Homebrew and manual installation
    - Uses launchd for service management
    - Logs stored in ~/Library/Logs/prism/ or /var/log/prism/

Linux:
    - Service runs at system startup as dedicated user
    - Requires sudo for installation and service control
    - Uses systemd for service management
    - Logs available via journalctl
    - Configuration in /etc/prism/

Windows:
    - Service runs as Windows Service at system startup
    - Requires Administrator privileges
    - Uses Windows Service Control Manager
    - Logs written to Windows Event Log
    - Configuration in %ProgramData%\CloudWorkstation\

REQUIREMENTS:
    macOS:      - macOS 10.14+ with launchd
    Linux:      - systemd-based Linux distribution  
    Windows:    - Windows 10/Server 2016+ with PowerShell

For platform-specific help and advanced options, run:
    ./service-manager.sh help
EOF
}

# Show system information
show_system_info() {
    local os="$(detect_os)"
    
    log "CloudWorkstation System Information:"
    echo
    
    case "$os" in
        macos)
            green "🍎 macOS System"
            echo "   OS Version: $(sw_vers -productVersion)"
            echo "   Build: $(sw_vers -buildVersion)"
            echo "   Architecture: $(uname -m)"
            echo "   Launchd: $(launchctl version 2>/dev/null | head -n1 || echo "Available")"
            
            if command -v brew >/dev/null 2>&1; then
                echo "   Homebrew: $(brew --version | head -n1)"
            else
                echo "   Homebrew: Not installed"
            fi
            ;;
        linux)
            green "🐧 Linux System"
            echo "   Kernel: $(uname -r)"
            echo "   Architecture: $(uname -m)"
            
            if [[ -f /etc/os-release ]]; then
                . /etc/os-release
                echo "   Distribution: $NAME $VERSION_ID"
            fi
            
            if command -v systemctl >/dev/null 2>&1; then
                echo "   Systemd: $(systemctl --version | head -n1)"
            else
                echo "   Systemd: Not available"
            fi
            ;;
        windows)
            green "🪟 Windows System"
            echo "   OS: $(uname -s)"
            echo "   Version: $(uname -r)"
            echo "   Architecture: $(uname -m)"
            
            if command -v pwsh >/dev/null 2>&1; then
                echo "   PowerShell: $(pwsh -Command '$PSVersionTable.PSVersion' 2>/dev/null || echo "Available")"
            elif command -v powershell >/dev/null 2>&1; then
                echo "   PowerShell: $(powershell -Command '$PSVersionTable.PSVersion' 2>/dev/null || echo "Available")"
            else
                echo "   PowerShell: Not available"
            fi
            ;;
        unknown)
            red "❓ Unknown System"
            echo "   OS: $(uname -s)"
            echo "   Kernel: $(uname -r)"
            echo "   Architecture: $(uname -m)"
            ;;
    esac
    
    echo
    echo "Platform service manager: $(get_platform_manager "$os")"
}

# Main command handling
main() {
    local command="${1:-help}"
    shift || true
    
    # Handle special commands that don't delegate
    case "$command" in
        info|system-info)
            show_system_info
            return 0
            ;;
        help-general)
            show_general_help
            return 0
            ;;
    esac
    
    # Detect platform and get appropriate manager
    local os="$(detect_os)"
    local manager="$(get_platform_manager "$os")"
    
    # Validate platform manager exists
    check_platform_manager "$manager" "$os"
    
    # Handle help command specially to show platform-specific help
    if [[ "$command" == "help" || "$command" == "--help" || "$command" == "-h" ]]; then
        show_platform_help
        return 0
    fi
    
    # Delegate all other commands to platform-specific manager
    execute_platform_command "$os" "$manager" "$command" "$@"
}

# Check if this script is being sourced or executed
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Script is being executed directly
    main "$@"
fi