#!/bin/bash
# CloudWorkstation Uninstall Manager
# Comprehensive cleanup script for CloudWorkstation removal across platforms
#
# This script provides thorough cleanup for various installation methods:
# - Homebrew installations (macOS)
# - Direct binary installations
# - Source installations
# - Package manager installations (future: APT, RPM, etc.)
#
# Usage:
#   ./scripts/uninstall-manager.sh [OPTIONS]
#
# Options:
#   --force          Force removal without confirmation prompts
#   --keep-config    Preserve user configuration files
#   --keep-logs      Preserve log files
#   --dry-run        Show what would be removed without actually removing
#   --verbose        Show detailed output
#   --help           Show this help message

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Default options
FORCE_REMOVAL=false
KEEP_CONFIG=false
KEEP_LOGS=false
DRY_RUN=false
VERBOSE=false

# Platform detection
PLATFORM="$(uname -s)"
case "$PLATFORM" in
    Darwin) OS_TYPE="macos" ;;
    Linux)  OS_TYPE="linux" ;;
    CYGWIN*|MINGW*|MSYS*) OS_TYPE="windows" ;;
    *) OS_TYPE="unknown" ;;
esac

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${NC}   $1${NC}"
    fi
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                FORCE_REMOVAL=true
                shift
                ;;
            --keep-config)
                KEEP_CONFIG=true
                shift
                ;;
            --keep-logs)
                KEEP_LOGS=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

show_help() {
    cat << EOF
CloudWorkstation Uninstall Manager

This script provides comprehensive cleanup for CloudWorkstation installations,
ensuring no orphaned processes or files remain after uninstallation.

Usage: $0 [OPTIONS]

Options:
  --force          Force removal without confirmation prompts
  --keep-config    Preserve user configuration files (~/.cloudworkstation)
  --keep-logs      Preserve log files
  --dry-run        Show what would be removed without actually removing
  --verbose        Show detailed output
  --help, -h       Show this help message

Examples:
  $0                           # Interactive uninstall with prompts
  $0 --force                   # Force uninstall without prompts
  $0 --dry-run --verbose       # See what would be removed
  $0 --keep-config --keep-logs # Uninstall but keep user data

Installation Methods Supported:
  • Homebrew (macOS): brew uninstall cloudworkstation
  • Direct binary installations
  • Source code installations
  • Package managers (future support)

Safety Features:
  • Process detection and graceful shutdown
  • Multiple confirmation prompts (unless --force)
  • Dry-run mode for preview
  • Selective preservation of user data
  • Comprehensive cleanup verification

EOF
}

# Confirmation prompt
confirm_action() {
    local message="$1"
    
    if [[ "$FORCE_REMOVAL" == "true" ]]; then
        return 0
    fi
    
    echo -e "${YELLOW}$message${NC}"
    read -p "Continue? [y/N]: " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Operation cancelled by user"
        exit 0
    fi
}

# Execute or preview command
execute_command() {
    local cmd="$1"
    local desc="$2"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would execute: $cmd"
        if [[ -n "$desc" ]]; then
            log_verbose "Purpose: $desc"
        fi
        return 0
    fi
    
    log_verbose "Executing: $cmd"
    if [[ -n "$desc" ]]; then
        log_verbose "Purpose: $desc"
    fi
    
    eval "$cmd"
}

# Remove file or directory safely
safe_remove() {
    local path="$1"
    local desc="$2"
    
    if [[ ! -e "$path" ]]; then
        log_verbose "Path does not exist: $path"
        return 0
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would remove: $path"
        if [[ -n "$desc" ]]; then
            log_verbose "Description: $desc"
        fi
        return 0
    fi
    
    log_verbose "Removing: $path"
    if [[ -n "$desc" ]]; then
        log_verbose "Description: $desc"
    fi
    
    rm -rf "$path"
    if [[ $? -eq 0 ]]; then
        log_success "Removed: $path"
    else
        log_warning "Failed to remove: $path"
    fi
}

# Detect installation method
detect_installation() {
    local installation_type="none"
    local binary_locations=()
    
    # Check Homebrew installation
    if command -v brew >/dev/null 2>&1; then
        if brew list cloudworkstation >/dev/null 2>&1; then
            installation_type="homebrew"
            binary_locations+=("$(brew --prefix)/bin/cws")
            binary_locations+=("$(brew --prefix)/bin/prismd")
        fi
    fi
    
    # Check system PATH installations
    if command -v cws >/dev/null 2>&1; then
        installation_type="system"
        binary_locations+=("$(which cws)")
    fi
    
    if command -v cwsd >/dev/null 2>&1; then
        if [[ "$installation_type" != "system" ]]; then
            installation_type="system"
        fi
        binary_locations+=("$(which cwsd)")
    fi
    
    # Check source installation in project directory
    if [[ -x "$PROJECT_ROOT/bin/cws" ]] || [[ -x "$PROJECT_ROOT/bin/prismd" ]]; then
        if [[ "$installation_type" == "none" ]]; then
            installation_type="source"
        fi
        [[ -x "$PROJECT_ROOT/bin/cws" ]] && binary_locations+=("$PROJECT_ROOT/bin/cws")
        [[ -x "$PROJECT_ROOT/bin/prismd" ]] && binary_locations+=("$PROJECT_ROOT/bin/prismd")
    fi
    
    echo "$installation_type"
    if [[ "$VERBOSE" == "true" ]]; then
        log_verbose "Binary locations found:"
        for location in "${binary_locations[@]}"; do
            log_verbose "  $location"
        done
    fi
}

# Find daemon processes
find_daemon_processes() {
    local pids=()
    
    case "$OS_TYPE" in
        macos|linux)
            # Use pgrep to find cwsd processes
            while IFS= read -r pid; do
                [[ -n "$pid" ]] && pids+=("$pid")
            done < <(pgrep -f "cwsd" 2>/dev/null || true)
            ;;
        windows)
            # Use tasklist on Windows
            while IFS= read -r line; do
                if [[ "$line" =~ cwsd.*[[:space:]]+([0-9]+)[[:space:]] ]]; then
                    pids+=("${BASH_REMATCH[1]}")
                fi
            done < <(tasklist 2>/dev/null | grep -i cwsd || true)
            ;;
    esac
    
    echo "${pids[@]}"
}

# Stop daemon processes
stop_daemon_processes() {
    log_info "Checking for running daemon processes..."
    
    local pids
    pids=($(find_daemon_processes))
    
    if [[ ${#pids[@]} -eq 0 ]]; then
        log_success "No daemon processes found"
        return 0
    fi
    
    log_warning "Found ${#pids[@]} daemon process(es): ${pids[*]}"
    
    # Try graceful shutdown first via API
    if command -v cws >/dev/null 2>&1; then
        log_info "Attempting graceful shutdown via API..."
        if ! execute_command "cws daemon stop" "Graceful daemon shutdown"; then
            log_warning "API shutdown failed, proceeding with direct process termination"
        else
            sleep 2
            # Check if processes are still running
            local remaining_pids
            remaining_pids=($(find_daemon_processes))
            if [[ ${#remaining_pids[@]} -eq 0 ]]; then
                log_success "Daemon processes stopped gracefully"
                return 0
            fi
        fi
    fi
    
    # Direct process termination
    log_info "Stopping daemon processes directly..."
    for pid in "${pids[@]}"; do
        log_verbose "Stopping PID $pid"
        
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "[DRY-RUN] Would stop process $pid"
            continue
        fi
        
        # Try SIGTERM first
        if kill -TERM "$pid" 2>/dev/null; then
            log_verbose "Sent SIGTERM to PID $pid"
        else
            log_warning "Failed to send SIGTERM to PID $pid"
        fi
    done
    
    # Wait for graceful shutdown
    log_info "Waiting for processes to stop..."
    sleep 5
    
    # Check for remaining processes and force kill if necessary
    local remaining_pids
    remaining_pids=($(find_daemon_processes))
    
    if [[ ${#remaining_pids[@]} -gt 0 ]]; then
        log_warning "Force killing remaining processes: ${remaining_pids[*]}"
        for pid in "${remaining_pids[@]}"; do
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "[DRY-RUN] Would force kill process $pid"
            else
                if kill -KILL "$pid" 2>/dev/null; then
                    log_verbose "Force killed PID $pid"
                else
                    log_warning "Failed to force kill PID $pid"
                fi
            fi
        done
    fi
    
    # Final verification
    sleep 2
    local final_pids
    final_pids=($(find_daemon_processes))
    
    if [[ ${#final_pids[@]} -eq 0 ]]; then
        log_success "All daemon processes stopped"
    else
        log_error "Some processes may still be running: ${final_pids[*]}"
        return 1
    fi
}

# Uninstall via Homebrew
uninstall_homebrew() {
    log_info "Uninstalling CloudWorkstation via Homebrew..."
    
    if ! command -v brew >/dev/null 2>&1; then
        log_error "Homebrew not found"
        return 1
    fi
    
    if ! brew list cloudworkstation >/dev/null 2>&1; then
        log_warning "CloudWorkstation not installed via Homebrew"
        return 0
    fi
    
    # Stop Homebrew service first
    execute_command "brew services stop cloudworkstation 2>/dev/null || true" "Stop Homebrew service"
    
    # Uninstall package (this will call our uninstall block)
    execute_command "brew uninstall cloudworkstation" "Uninstall CloudWorkstation package"
    
    log_success "Homebrew uninstallation completed"
}

# Remove system binaries
remove_system_binaries() {
    log_info "Removing system binaries..."
    
    local binaries=("cws" "cwsd")
    local removed_any=false
    
    for binary in "${binaries[@]}"; do
        local binary_path
        binary_path=$(which "$binary" 2>/dev/null || echo "")
        
        if [[ -n "$binary_path" && -x "$binary_path" ]]; then
            log_verbose "Found binary: $binary_path"
            
            # Check if it's a Homebrew binary (skip if Homebrew will handle it)
            if [[ "$binary_path" =~ /opt/homebrew/bin/ ]] || [[ "$binary_path" =~ /usr/local/bin/ ]]; then
                if command -v brew >/dev/null 2>&1 && brew list cloudworkstation >/dev/null 2>&1; then
                    log_verbose "Skipping Homebrew binary: $binary_path"
                    continue
                fi
            fi
            
            safe_remove "$binary_path" "CloudWorkstation binary"
            removed_any=true
        fi
    done
    
    if [[ "$removed_any" == "true" ]]; then
        log_success "System binaries removed"
    else
        log_info "No system binaries found to remove"
    fi
}

# Clean up configuration files
cleanup_config_files() {
    if [[ "$KEEP_CONFIG" == "true" ]]; then
        log_info "Skipping configuration cleanup (--keep-config specified)"
        return 0
    fi
    
    log_info "Cleaning up configuration files..."
    
    local config_dir="$HOME/.cloudworkstation"
    if [[ -d "$config_dir" ]]; then
        if [[ "$FORCE_REMOVAL" == "false" ]]; then
            echo
            log_warning "This will remove your CloudWorkstation configuration:"
            log_warning "  • AWS profiles and credentials"
            log_warning "  • Daemon configuration"
            log_warning "  • Instance state files"
            confirm_action "Remove configuration directory: $config_dir"
        fi
        
        safe_remove "$config_dir" "Configuration directory"
    else
        log_verbose "Configuration directory not found: $config_dir"
    fi
}

# Clean up log files
cleanup_log_files() {
    if [[ "$KEEP_LOGS" == "true" ]]; then
        log_info "Skipping log cleanup (--keep-logs specified)"
        return 0
    fi
    
    log_info "Cleaning up log files..."
    
    local log_paths=()
    
    case "$OS_TYPE" in
        macos)
            log_paths+=(
                "$HOME/Library/Logs/cloudworkstation"
                "/usr/local/var/log/cloudworkstation"
                "/opt/homebrew/var/log/cloudworkstation"
            )
            ;;
        linux)
            log_paths+=(
                "$HOME/.local/share/cloudworkstation/logs"
                "/var/log/cloudworkstation"
                "/tmp/cloudworkstation.log"
            )
            ;;
        windows)
            log_paths+=(
                "$HOME/AppData/Local/CloudWorkstation/logs"
                "/tmp/cloudworkstation.log"
            )
            ;;
    esac
    
    local removed_any=false
    for log_path in "${log_paths[@]}"; do
        if [[ -e "$log_path" ]]; then
            safe_remove "$log_path" "Log files"
            removed_any=true
        fi
    done
    
    if [[ "$removed_any" == "true" ]]; then
        log_success "Log files cleaned up"
    else
        log_verbose "No log files found to remove"
    fi
}

# Clean up service files
cleanup_service_files() {
    log_info "Cleaning up service files..."
    
    case "$OS_TYPE" in
        macos)
            local service_files=(
                "$HOME/Library/LaunchAgents/homebrew.mxcl.cloudworkstation.plist"
                "$HOME/Library/LaunchAgents/com.cloudworkstation.daemon.plist"
                "/Library/LaunchDaemons/com.cloudworkstation.daemon.plist"
            )
            ;;
        linux)
            local service_files=(
                "$HOME/.local/share/systemd/user/cloudworkstation.service"
                "/etc/systemd/system/cloudworkstation.service"
                "/etc/init.d/cloudworkstation"
            )
            ;;
        windows)
            local service_files=()
            # Windows services handled differently
            ;;
    esac
    
    local removed_any=false
    for service_file in "${service_files[@]}"; do
        if [[ -e "$service_file" ]]; then
            safe_remove "$service_file" "Service file"
            removed_any=true
        fi
    done
    
    if [[ "$removed_any" == "true" ]]; then
        log_success "Service files cleaned up"
    else
        log_verbose "No service files found to remove"
    fi
}

# Clean up temporary files
cleanup_temp_files() {
    log_info "Cleaning up temporary files..."
    
    local temp_patterns=(
        "/tmp/cloudworkstation*"
        "/var/tmp/cloudworkstation*"
        "$HOME/.tmp/cloudworkstation*"
    )
    
    local removed_any=false
    for pattern in "${temp_patterns[@]}"; do
        for file in $pattern; do
            if [[ -e "$file" ]]; then
                safe_remove "$file" "Temporary file"
                removed_any=true
            fi
        done
    done
    
    if [[ "$removed_any" == "true" ]]; then
        log_success "Temporary files cleaned up"
    else
        log_verbose "No temporary files found to remove"
    fi
}

# Verify cleanup completion
verify_cleanup() {
    log_info "Verifying cleanup completion..."
    
    local issues=()
    
    # Check for remaining processes
    local remaining_pids
    remaining_pids=($(find_daemon_processes))
    if [[ ${#remaining_pids[@]} -gt 0 ]]; then
        issues+=("Daemon processes still running: ${remaining_pids[*]}")
    fi
    
    # Check for remaining binaries
    for binary in "cws" "cwsd"; do
        if command -v "$binary" >/dev/null 2>&1; then
            issues+=("Binary still in PATH: $binary")
        fi
    done
    
    # Check for remaining config (if not kept)
    if [[ "$KEEP_CONFIG" == "false" ]] && [[ -d "$HOME/.cloudworkstation" ]]; then
        issues+=("Configuration directory still exists: $HOME/.cloudworkstation")
    fi
    
    if [[ ${#issues[@]} -eq 0 ]]; then
        log_success "Cleanup verification passed"
        return 0
    else
        log_warning "Cleanup verification found issues:"
        for issue in "${issues[@]}"; do
            log_warning "  • $issue"
        done
        return 1
    fi
}

# Main uninstallation process
main() {
    echo "CloudWorkstation Uninstall Manager"
    echo "=================================="
    echo
    
    parse_args "$@"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "DRY-RUN MODE: No actual changes will be made"
        echo
    fi
    
    # Detect installation
    local installation_type
    installation_type=$(detect_installation)
    
    log_info "Platform: $OS_TYPE"
    log_info "Installation type: $installation_type"
    echo
    
    if [[ "$installation_type" == "none" ]]; then
        log_info "No CloudWorkstation installation detected"
        exit 0
    fi
    
    # Final confirmation
    if [[ "$FORCE_REMOVAL" == "false" ]] && [[ "$DRY_RUN" == "false" ]]; then
        echo
        log_warning "This will completely remove CloudWorkstation from your system"
        log_warning "Installation type: $installation_type"
        if [[ "$KEEP_CONFIG" == "false" ]]; then
            log_warning "Configuration files will be removed"
        fi
        if [[ "$KEEP_LOGS" == "false" ]]; then
            log_warning "Log files will be removed"
        fi
        confirm_action "Proceed with uninstallation?"
        echo
    fi
    
    # Execute uninstallation steps
    local exit_code=0
    
    # 1. Stop daemon processes
    if ! stop_daemon_processes; then
        log_error "Failed to stop daemon processes"
        exit_code=1
    fi
    
    # 2. Uninstall based on installation type
    case "$installation_type" in
        homebrew)
            if ! uninstall_homebrew; then
                log_error "Homebrew uninstallation failed"
                exit_code=1
            fi
            ;;
        system|source)
            remove_system_binaries
            ;;
    esac
    
    # 3. Clean up files
    cleanup_config_files
    cleanup_log_files
    cleanup_service_files
    cleanup_temp_files
    
    # 4. Verify cleanup
    if ! verify_cleanup; then
        log_warning "Cleanup verification found issues (see above)"
        exit_code=1
    fi
    
    echo
    if [[ $exit_code -eq 0 ]]; then
        log_success "CloudWorkstation has been successfully uninstalled"
        echo
        log_info "Thank you for using CloudWorkstation!"
        if [[ "$KEEP_CONFIG" == "false" ]]; then
            log_info "AWS credentials and profiles remain unchanged"
        fi
    else
        log_error "Uninstallation completed with warnings"
        log_info "You may need to manually remove remaining components"
    fi
    
    exit $exit_code
}

# Execute main function with all arguments
main "$@"