#!/bin/bash

# CloudWorkstation Linux Service Manager
# Comprehensive service management for CloudWorkstation daemon on Linux systems

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_NAME="cloudworkstation-daemon"
SYSTEMD_SERVICE_NAME="cwsd"
SYSTEMD_SERVICE_FILE="/etc/systemd/system/${SYSTEMD_SERVICE_NAME}.service"

# Paths
DAEMON_PATH="/usr/local/bin/prismd"
CLI_PATH="/usr/local/bin/cws"
CONFIG_DIR="/etc/cloudworkstation"
STATE_DIR="/var/lib/cloudworkstation"
LOG_DIR="/var/log/cloudworkstation"
SERVICE_USER="cloudworkstation"

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

# Check if running as root
is_root() {
    [[ $EUID -eq 0 ]]
}

# Require root privileges
require_root() {
    if ! is_root; then
        error_exit "This operation requires root privileges. Please run with sudo."
    fi
}

# Detect init system
detect_init_system() {
    if command -v systemctl >/dev/null 2>&1 && systemctl --version >/dev/null 2>&1; then
        echo "systemd"
    elif command -v service >/dev/null 2>&1; then
        echo "sysv"
    elif command -v rc-service >/dev/null 2>&1; then
        echo "openrc"
    else
        echo "unknown"
    fi
}

# Check if systemd service exists
systemd_service_exists() {
    systemctl list-unit-files | grep -q "^${SYSTEMD_SERVICE_NAME}.service"
}

# Check if daemon binary exists
daemon_binary_exists() {
    [[ -x "$DAEMON_PATH" ]]
}

# Create service user and group
create_service_user() {
    log "Creating service user and group..."
    
    if ! getent group "$SERVICE_USER" >/dev/null 2>&1; then
        groupadd --system "$SERVICE_USER"
        green "✅ Created group: $SERVICE_USER"
    else
        log "Group $SERVICE_USER already exists"
    fi
    
    if ! getent passwd "$SERVICE_USER" >/dev/null 2>&1; then
        useradd --system --gid "$SERVICE_USER" \
                --home-dir "$STATE_DIR" \
                --shell /usr/sbin/nologin \
                --comment "CloudWorkstation Daemon" "$SERVICE_USER"
        green "✅ Created user: $SERVICE_USER"
    else
        log "User $SERVICE_USER already exists"
    fi
}

# Create necessary directories
create_directories() {
    log "Creating service directories..."
    
    # Configuration directory
    mkdir -p "$CONFIG_DIR/aws"
    chown root:"$SERVICE_USER" "$CONFIG_DIR"
    chmod 750 "$CONFIG_DIR"
    chown root:"$SERVICE_USER" "$CONFIG_DIR/aws"
    chmod 750 "$CONFIG_DIR/aws"
    
    # State and data directory
    mkdir -p "$STATE_DIR/.config"
    mkdir -p "$STATE_DIR/.cloudworkstation"
    mkdir -p "$STATE_DIR/.ssh"
    chown -R "$SERVICE_USER:$SERVICE_USER" "$STATE_DIR"
    chmod 700 "$STATE_DIR"
    chmod 700 "$STATE_DIR/.ssh"
    
    # Log directory
    mkdir -p "$LOG_DIR"
    chown "$SERVICE_USER:$SERVICE_USER" "$LOG_DIR"
    chmod 755 "$LOG_DIR"
    
    green "✅ Created directory structure"
}

# Generate systemd service file
generate_systemd_service() {
    log "Generating systemd service file..."
    
    cat > "$SYSTEMD_SERVICE_FILE" << EOF
[Unit]
Description=CloudWorkstation Daemon - Enterprise Research Management Platform
Documentation=https://github.com/scttfrdmn/cloudworkstation
After=network-online.target multi-user.target
Wants=network-online.target
ConditionPathExists=$DAEMON_PATH
RequiresMountsFor=$STATE_DIR $CONFIG_DIR

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$DAEMON_PATH --service
ExecReload=/bin/kill -HUP \$MAINPID
ExecStop=/bin/kill -TERM \$MAINPID
Restart=always
RestartSec=5
StartLimitInterval=60s
StartLimitBurst=3

# Working directory
WorkingDirectory=$STATE_DIR

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096
TimeoutStartSec=60s
TimeoutStopSec=30s

# Security hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
PrivateDevices=yes
ProtectControlGroups=yes
ProtectKernelModules=yes
ProtectKernelTunables=yes
RestrictRealtime=yes
RestrictSUIDSGID=yes
MemoryDenyWriteExecute=yes
RestrictNamespaces=yes

# Allow access to configuration and state directories
ReadWritePaths=$STATE_DIR
ReadWritePaths=$LOG_DIR
ReadOnlyPaths=$CONFIG_DIR

# Environment
Environment=HOME=$STATE_DIR
Environment=XDG_CONFIG_HOME=$STATE_DIR/.config
Environment=AWS_CONFIG_FILE=$CONFIG_DIR/aws/config
Environment=AWS_SHARED_CREDENTIALS_FILE=$CONFIG_DIR/aws/credentials
Environment=PRISM_SERVICE_MODE=true
Environment=PRISM_CONFIG_DIR=$CONFIG_DIR
Environment=PRISM_STATE_DIR=$STATE_DIR
Environment=PRISM_LOG_DIR=$LOG_DIR

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=cloudworkstation

[Install]
WantedBy=multi-user.target
EOF

    chmod 644 "$SYSTEMD_SERVICE_FILE"
    green "✅ Generated systemd service file"
}

# Create default configuration
create_default_config() {
    log "Creating default configuration..."
    
    # AWS config
    if [[ ! -f "$CONFIG_DIR/aws/config" ]]; then
        cat > "$CONFIG_DIR/aws/config" << 'EOF'
[default]
region = us-west-2
output = json

[profile cloudworkstation]
region = us-west-2
output = json
EOF
        chown root:"$SERVICE_USER" "$CONFIG_DIR/aws/config"
        chmod 640 "$CONFIG_DIR/aws/config"
        log "Created default AWS config"
    fi
    
    # AWS credentials template
    if [[ ! -f "$CONFIG_DIR/aws/credentials" ]]; then
        cat > "$CONFIG_DIR/aws/credentials" << 'EOF'
# AWS credentials for CloudWorkstation
# Replace with your actual AWS credentials
[default]
aws_access_key_id = YOUR_ACCESS_KEY_HERE
aws_secret_access_key = YOUR_SECRET_KEY_HERE

[cloudworkstation]
aws_access_key_id = YOUR_ACCESS_KEY_HERE
aws_secret_access_key = YOUR_SECRET_KEY_HERE
EOF
        chown root:"$SERVICE_USER" "$CONFIG_DIR/aws/credentials"
        chmod 640 "$CONFIG_DIR/aws/credentials"
        yellow "⚠️  Created template AWS credentials - YOU MUST EDIT THIS FILE"
    fi
    
    # Main configuration
    if [[ ! -f "$CONFIG_DIR/config.json" ]]; then
        cat > "$CONFIG_DIR/config.json" << 'EOF'
{
  "daemon": {
    "port": "8947",
    "log_level": "info",
    "enable_metrics": true
  },
  "aws": {
    "region": "us-west-2",
    "profile": "cloudworkstation"
  },
  "idle": {
    "enabled": true,
    "default_profile": "standard",
    "check_interval": "2m"
  },
  "security": {
    "enable_audit_log": true,
    "max_api_rate": 100
  }
}
EOF
        chown root:"$SERVICE_USER" "$CONFIG_DIR/config.json"
        chmod 640 "$CONFIG_DIR/config.json"
        log "Created default daemon configuration"
    fi
    
    green "✅ Created default configuration files"
}

# Setup SSH keys for instance management
setup_ssh_keys() {
    log "Setting up SSH keys for instance management..."
    
    local ssh_key_path="$STATE_DIR/.ssh/cloudworkstation"
    
    if [[ ! -f "$ssh_key_path" ]]; then
        sudo -u "$SERVICE_USER" ssh-keygen -t ed25519 -f "$ssh_key_path" -N "" -C "cloudworkstation-daemon"
        green "✅ Generated SSH key: $ssh_key_path"
        echo
        blue "📋 Public key (add this to your AWS EC2 key pairs):"
        cat "$ssh_key_path.pub"
        echo
    else
        log "SSH key already exists: $ssh_key_path"
    fi
}

# Install service
install_service() {
    local init_system="$(detect_init_system)"
    
    log "Installing CloudWorkstation service ($init_system)..."
    require_root
    
    case "$init_system" in
        systemd)
            if systemd_service_exists; then
                yellow "⚠️  Service already installed. Use 'reinstall' to update."
                return 0
            fi
            
            if ! daemon_binary_exists; then
                error_exit "CloudWorkstation daemon not found at $DAEMON_PATH. Please install CloudWorkstation first."
            fi
            
            create_service_user
            create_directories
            generate_systemd_service
            create_default_config
            setup_ssh_keys
            
            systemctl daemon-reload
            systemctl enable "$SYSTEMD_SERVICE_NAME"
            
            green "✅ CloudWorkstation service installed successfully"
            log "Service will start automatically on system boot"
            
            # Start the service
            start_service
            ;;
        *)
            error_exit "Unsupported init system: $init_system"
            ;;
    esac
}

# Uninstall service
uninstall_service() {
    local init_system="$(detect_init_system)"
    
    log "Uninstalling CloudWorkstation service ($init_system)..."
    require_root
    
    case "$init_system" in
        systemd)
            if ! systemd_service_exists; then
                yellow "⚠️  Service not installed"
                return 0
            fi
            
            # Stop and disable service
            stop_service
            systemctl disable "$SYSTEMD_SERVICE_NAME" || true
            
            # Remove service file
            rm -f "$SYSTEMD_SERVICE_FILE"
            systemctl daemon-reload
            systemctl reset-failed "$SYSTEMD_SERVICE_NAME" || true
            
            green "✅ CloudWorkstation service uninstalled successfully"
            
            yellow "⚠️  Configuration and data directories preserved:"
            echo "   Config: $CONFIG_DIR"
            echo "   Data: $STATE_DIR"
            echo "   Logs: $LOG_DIR"
            echo "   User: $SERVICE_USER"
            echo
            echo "To completely remove CloudWorkstation:"
            echo "   sudo userdel $SERVICE_USER"
            echo "   sudo groupdel $SERVICE_USER"
            echo "   sudo rm -rf $CONFIG_DIR $STATE_DIR $LOG_DIR"
            ;;
        *)
            error_exit "Unsupported init system: $init_system"
            ;;
    esac
}

# Reinstall service (update configuration)
reinstall_service() {
    log "Reinstalling CloudWorkstation service..."
    
    if systemd_service_exists; then
        uninstall_service
    fi
    
    install_service
}

# Start service
start_service() {
    local init_system="$(detect_init_system)"
    
    case "$init_system" in
        systemd)
            if ! systemd_service_exists; then
                error_exit "Service not installed. Run 'install' first."
            fi
            
            log "Starting CloudWorkstation service..."
            systemctl start "$SYSTEMD_SERVICE_NAME"
            green "✅ CloudWorkstation service started"
            ;;
        *)
            error_exit "Unsupported init system: $init_system"
            ;;
    esac
}

# Stop service
stop_service() {
    local init_system="$(detect_init_system)"
    
    case "$init_system" in
        systemd)
            log "Stopping CloudWorkstation service..."
            systemctl stop "$SYSTEMD_SERVICE_NAME" 2>/dev/null || true
            green "✅ CloudWorkstation service stopped"
            ;;
        *)
            error_exit "Unsupported init system: $init_system"
            ;;
    esac
}

# Restart service
restart_service() {
    log "Restarting CloudWorkstation service..."
    stop_service
    sleep 2
    start_service
}

# Get service status
show_status() {
    local init_system="$(detect_init_system)"
    
    log "CloudWorkstation Service Status ($init_system):"
    echo
    
    case "$init_system" in
        systemd)
            if systemd_service_exists; then
                green "📦 Service: Installed"
                echo "   Unit: $SYSTEMD_SERVICE_NAME.service"
                echo "   File: $SYSTEMD_SERVICE_FILE"
                echo "   Daemon: $DAEMON_PATH"
                echo "   User: $SERVICE_USER"
                echo "   Config: $CONFIG_DIR"
                echo "   Data: $STATE_DIR"
                echo "   Logs: $LOG_DIR"
                echo
                
                # Show systemd status
                if systemctl is-active "$SYSTEMD_SERVICE_NAME" >/dev/null 2>&1; then
                    green "🟢 Status: Active (Running)"
                elif systemctl is-enabled "$SYSTEMD_SERVICE_NAME" >/dev/null 2>&1; then
                    yellow "🟡 Status: Inactive (Enabled)"
                else
                    red "🔴 Status: Inactive (Disabled)"
                fi
                
                # Show detailed systemd status
                echo
                blue "📊 Systemd Status:"
                systemctl status "$SYSTEMD_SERVICE_NAME" --no-pager -l || true
                
                # Show recent log entries
                echo
                blue "📝 Recent Log Entries:"
                journalctl -u "$SYSTEMD_SERVICE_NAME" -n 10 --no-pager || true
            else
                red "❌ Service: Not installed"
            fi
            ;;
        *)
            yellow "⚠️  Unsupported init system: $init_system"
            ;;
    esac
    
    echo
}

# Show service logs
show_logs() {
    local init_system="$(detect_init_system)"
    
    case "$init_system" in
        systemd)
            if ! systemd_service_exists; then
                yellow "⚠️  Service not installed"
                return
            fi
            
            log "Showing CloudWorkstation service logs..."
            echo
            journalctl -u "$SYSTEMD_SERVICE_NAME" --no-pager
            ;;
        *)
            yellow "⚠️  Unsupported init system: $init_system"
            ;;
    esac
}

# Follow service logs
follow_logs() {
    local init_system="$(detect_init_system)"
    
    case "$init_system" in
        systemd)
            if ! systemd_service_exists; then
                yellow "⚠️  Service not installed"
                return
            fi
            
            log "Following CloudWorkstation service logs... (Press Ctrl+C to stop)"
            echo
            journalctl -u "$SYSTEMD_SERVICE_NAME" -f
            ;;
        *)
            yellow "⚠️  Unsupported init system: $init_system"
            ;;
    esac
}

# Validate service configuration
validate_service() {
    local init_system="$(detect_init_system)"
    
    log "Validating CloudWorkstation service configuration..."
    echo
    
    local errors=0
    
    # Check init system
    if [[ "$init_system" == "systemd" ]]; then
        green "✅ Init system: systemd (supported)"
    else
        red "❌ Init system: $init_system (unsupported)"
        ((errors++))
    fi
    
    # Check daemon binary
    if daemon_binary_exists; then
        green "✅ Daemon binary: Found at $DAEMON_PATH"
        
        # Check version
        local version
        if version="$($DAEMON_PATH --version 2>/dev/null)"; then
            echo "   Version: $version"
        fi
    else
        red "❌ Daemon binary: Not found at $DAEMON_PATH"
        ((errors++))
    fi
    
    # Check CLI binary
    if [[ -x "$CLI_PATH" ]]; then
        green "✅ CLI binary: Found at $CLI_PATH"
    else
        yellow "⚠️  CLI binary: Not found at $CLI_PATH (optional)"
    fi
    
    # Check service user
    if getent passwd "$SERVICE_USER" >/dev/null 2>&1; then
        green "✅ Service user: $SERVICE_USER exists"
    else
        red "❌ Service user: $SERVICE_USER does not exist"
        ((errors++))
    fi
    
    # Check directories
    for dir_info in \
        "$CONFIG_DIR:Config directory" \
        "$STATE_DIR:State directory" \
        "$LOG_DIR:Log directory"; do
        
        local dir="${dir_info%:*}"
        local name="${dir_info#*:}"
        
        if [[ -d "$dir" ]]; then
            green "✅ $name: $dir exists"
        else
            red "❌ $name: $dir does not exist"
            ((errors++))
        fi
    done
    
    # Check systemd service
    if systemd_service_exists; then
        green "✅ Systemd service: Installed"
        
        # Check if enabled
        if systemctl is-enabled "$SYSTEMD_SERVICE_NAME" >/dev/null 2>&1; then
            green "   Auto-start: Enabled"
        else
            yellow "   Auto-start: Disabled"
        fi
        
        # Validate service file
        if systemd-analyze verify "$SYSTEMD_SERVICE_FILE" >/dev/null 2>&1; then
            green "   Service file: Valid"
        else
            red "   Service file: Invalid"
            ((errors++))
        fi
    else
        red "❌ Systemd service: Not installed"
        ((errors++))
    fi
    
    # Check configuration files
    if [[ -f "$CONFIG_DIR/config.json" ]]; then
        green "✅ Main config: Found"
        
        # Validate JSON
        if python3 -m json.tool "$CONFIG_DIR/config.json" >/dev/null 2>&1; then
            green "   JSON syntax: Valid"
        else
            red "   JSON syntax: Invalid"
            ((errors++))
        fi
    else
        yellow "⚠️  Main config: Not found (will use defaults)"
    fi
    
    if [[ -f "$CONFIG_DIR/aws/credentials" ]]; then
        green "✅ AWS credentials: Found"
        
        # Check if template
        if grep -q "YOUR_ACCESS_KEY_HERE" "$CONFIG_DIR/aws/credentials" 2>/dev/null; then
            yellow "   Contains template values (needs configuration)"
        else
            green "   Appears configured"
        fi
    else
        yellow "⚠️  AWS credentials: Not found"
    fi
    
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
CloudWorkstation Linux Service Manager

USAGE:
    linux-service-manager.sh <command>

COMMANDS:
    install     Install and start CloudWorkstation service (requires sudo)
    uninstall   Stop and uninstall CloudWorkstation service (requires sudo) 
    reinstall   Update service configuration (uninstall + install, requires sudo)
    start       Start the service (requires sudo)
    stop        Stop the service (requires sudo)
    restart     Restart the service (requires sudo)
    status      Show service status and configuration
    logs        Show service logs
    follow      Follow service logs in real-time
    validate    Validate service configuration
    help        Show this help message

EXAMPLES:
    # Install system service (requires sudo)
    sudo ./linux-service-manager.sh install
    
    # Check service status (no sudo required)
    ./linux-service-manager.sh status
    
    # Follow real-time logs (no sudo required)
    ./linux-service-manager.sh follow

NOTES:
    - Service runs as dedicated 'cloudworkstation' user
    - Service starts automatically on system boot
    - Service automatically restarts if daemon crashes
    - Configuration stored in /etc/cloudworkstation/
    - Data stored in /var/lib/cloudworkstation/
    - Logs available via journalctl -u cwsd
    - Requires systemd (most modern Linux distributions)

INSTALLATION STEPS:
    1. Install the service: sudo ./linux-service-manager.sh install
    2. Edit AWS credentials: sudo nano /etc/cloudworkstation/aws/credentials
    3. Configure daemon: sudo nano /etc/cloudworkstation/config.json
    4. Check status: ./linux-service-manager.sh status
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