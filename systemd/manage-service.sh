#!/bin/bash
set -euo pipefail

# CloudWorkstation service management script

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if systemctl is available
if ! command -v systemctl >/dev/null 2>&1; then
    log_error "systemctl not found - this script requires systemd"
    exit 1
fi

SERVICE_NAME="cwsd"

# Service management functions
start_service() {
    log_info "Starting CloudWorkstation service..."
    if systemctl start $SERVICE_NAME; then
        log_success "Service started successfully"
        systemctl status $SERVICE_NAME --no-pager -l
    else
        log_error "Failed to start service"
        exit 1
    fi
}

stop_service() {
    log_info "Stopping CloudWorkstation service..."
    if systemctl stop $SERVICE_NAME; then
        log_success "Service stopped successfully"
    else
        log_error "Failed to stop service"
        exit 1
    fi
}

restart_service() {
    log_info "Restarting CloudWorkstation service..."
    if systemctl restart $SERVICE_NAME; then
        log_success "Service restarted successfully"
        systemctl status $SERVICE_NAME --no-pager -l
    else
        log_error "Failed to restart service"
        exit 1
    fi
}

enable_service() {
    log_info "Enabling CloudWorkstation service for auto-start..."
    if systemctl enable $SERVICE_NAME; then
        log_success "Service enabled for auto-start"
    else
        log_error "Failed to enable service"
        exit 1
    fi
}

disable_service() {
    log_info "Disabling CloudWorkstation service auto-start..."
    if systemctl disable $SERVICE_NAME; then
        log_success "Service auto-start disabled"
    else
        log_error "Failed to disable service"
        exit 1
    fi
}

status_service() {
    systemctl status $SERVICE_NAME --no-pager -l
}

logs_service() {
    log_info "Showing CloudWorkstation service logs (Ctrl+C to exit)..."
    journalctl -u $SERVICE_NAME -f --no-pager
}

show_config() {
    log_info "Current configuration:"
    echo
    
    if [[ -f /etc/cloudworkstation/config.json ]]; then
        echo "Main config (/etc/cloudworkstation/config.json):"
        cat /etc/cloudworkstation/config.json | jq . 2>/dev/null || cat /etc/cloudworkstation/config.json
        echo
    else
        log_warning "Main config file not found"
    fi
    
    if [[ -f /etc/cloudworkstation/aws/config ]]; then
        echo "AWS config (/etc/cloudworkstation/aws/config):"
        cat /etc/cloudworkstation/aws/config
        echo
    else
        log_warning "AWS config file not found"
    fi
    
    if [[ -f /var/lib/cloudworkstation/.cloudworkstation/idle.json ]]; then
        echo "Idle config (/var/lib/cloudworkstation/.cloudworkstation/idle.json):"
        cat /var/lib/cloudworkstation/.cloudworkstation/idle.json | jq . 2>/dev/null || head -20 /var/lib/cloudworkstation/.cloudworkstation/idle.json
        echo
    fi
}

show_state() {
    log_info "Current service state:"
    echo
    
    # Service status
    echo "Service status:"
    systemctl is-active $SERVICE_NAME || echo "inactive"
    systemctl is-enabled $SERVICE_NAME || echo "disabled"
    echo
    
    # Runtime state
    if [[ -f /var/lib/cloudworkstation/.cloudworkstation/autonomous_state.json ]]; then
        echo "Autonomous state:"
        cat /var/lib/cloudworkstation/.cloudworkstation/autonomous_state.json | jq . 2>/dev/null || head -20 /var/lib/cloudworkstation/.cloudworkstation/autonomous_state.json
    else
        log_info "No autonomous state file found"
    fi
}

enable_autonomous() {
    log_info "Enabling autonomous idle detection..."
    
    # Update config to enable autonomous execution
    if command -v jq >/dev/null 2>&1; then
        tmp_file=$(mktemp)
        jq '.autonomous.auto_execute = true | .autonomous.dry_run = false' /etc/cloudworkstation/config.json > "$tmp_file"
        mv "$tmp_file" /etc/cloudworkstation/config.json
        
        log_success "Enabled autonomous execution"
        log_warning "The service will now automatically hibernate/stop idle instances"
        
        # Restart service to pick up new config
        restart_service
    else
        log_error "jq command not found - please manually edit /etc/cloudworkstation/config.json"
        log_info "Set: autonomous.auto_execute = true, autonomous.dry_run = false"
    fi
}

disable_autonomous() {
    log_info "Disabling autonomous idle detection..."
    
    # Update config to disable autonomous execution
    if command -v jq >/dev/null 2>&1; then
        tmp_file=$(mktemp)
        jq '.autonomous.auto_execute = false | .autonomous.dry_run = true' /etc/cloudworkstation/config.json > "$tmp_file"
        mv "$tmp_file" /etc/cloudworkstation/config.json
        
        log_success "Disabled autonomous execution (dry run mode)"
        
        # Restart service to pick up new config
        restart_service
    else
        log_error "jq command not found - please manually edit /etc/cloudworkstation/config.json"
        log_info "Set: autonomous.auto_execute = false, autonomous.dry_run = true"
    fi
}

show_help() {
    echo "CloudWorkstation Service Management"
    echo
    echo "Usage: $0 COMMAND"
    echo
    echo "Commands:"
    echo "  start               Start the service"
    echo "  stop                Stop the service"
    echo "  restart             Restart the service"
    echo "  enable              Enable auto-start on boot"
    echo "  disable             Disable auto-start on boot"
    echo "  status              Show service status"
    echo "  logs                Show service logs (follow mode)"
    echo "  config              Show current configuration"
    echo "  state               Show current runtime state"
    echo "  enable-autonomous   Enable autonomous idle detection"
    echo "  disable-autonomous  Disable autonomous idle detection"
    echo "  help                Show this help"
    echo
    echo "Examples:"
    echo "  $0 start                    # Start the service"
    echo "  $0 enable && $0 start       # Enable auto-start and start now"
    echo "  $0 logs                     # Watch logs in real-time"
    echo "  $0 enable-autonomous        # Enable automatic idle actions"
    echo
}

# Main command handling
case "${1:-help}" in
    start)
        start_service
        ;;
    stop)
        stop_service
        ;;
    restart)
        restart_service
        ;;
    enable)
        enable_service
        ;;
    disable)
        disable_service
        ;;
    status)
        status_service
        ;;
    logs)
        logs_service
        ;;
    config)
        show_config
        ;;
    state)
        show_state
        ;;
    enable-autonomous)
        enable_autonomous
        ;;
    disable-autonomous)
        disable_autonomous
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "Unknown command: ${1:-}"
        echo
        show_help
        exit 1
        ;;
esac