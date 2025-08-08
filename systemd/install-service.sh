#!/bin/bash
set -euo pipefail

# CloudWorkstation systemd service installer
# This script sets up the CloudWorkstation daemon as a system service

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

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

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   log_error "This script must be run as root (use sudo)"
   exit 1
fi

# Detect system init system
if command -v systemctl >/dev/null 2>&1; then
    INIT_SYSTEM="systemd"
elif command -v service >/dev/null 2>&1; then
    INIT_SYSTEM="sysv"
else
    log_error "No supported init system found (systemd or sysv)"
    exit 1
fi

log_info "Detected init system: $INIT_SYSTEM"

# Create cloudworkstation user and group
create_user() {
    log_info "Creating cloudworkstation user and group..."
    
    if ! getent group cloudworkstation >/dev/null 2>&1; then
        groupadd --system cloudworkstation
        log_success "Created group: cloudworkstation"
    else
        log_info "Group cloudworkstation already exists"
    fi
    
    if ! getent passwd cloudworkstation >/dev/null 2>&1; then
        useradd --system --gid cloudworkstation --home-dir /var/lib/cloudworkstation \
                --shell /usr/sbin/nologin --comment "CloudWorkstation Daemon" cloudworkstation
        log_success "Created user: cloudworkstation"
    else
        log_info "User cloudworkstation already exists"
    fi
}

# Create directories
create_directories() {
    log_info "Creating directories..."
    
    # Configuration directory
    mkdir -p /etc/cloudworkstation/aws
    chown root:cloudworkstation /etc/cloudworkstation
    chmod 750 /etc/cloudworkstation
    chown root:cloudworkstation /etc/cloudworkstation/aws
    chmod 750 /etc/cloudworkstation/aws
    
    # State and data directory
    mkdir -p /var/lib/cloudworkstation/.config
    mkdir -p /var/lib/cloudworkstation/.cloudworkstation
    mkdir -p /var/lib/cloudworkstation/.ssh
    chown -R cloudworkstation:cloudworkstation /var/lib/cloudworkstation
    chmod 700 /var/lib/cloudworkstation
    chmod 700 /var/lib/cloudworkstation/.ssh
    
    # Log directory
    mkdir -p /var/log/cloudworkstation
    chown cloudworkstation:cloudworkstation /var/log/cloudworkstation
    chmod 755 /var/log/cloudworkstation
    
    log_success "Created directory structure"
}

# Install binaries
install_binaries() {
    log_info "Installing CloudWorkstation binaries..."
    
    # Build binaries if they don't exist
    if [[ ! -f "$PROJECT_ROOT/bin/cwsd" ]]; then
        log_info "Building CloudWorkstation daemon..."
        cd "$PROJECT_ROOT"
        make build
    fi
    
    # Copy binaries
    cp "$PROJECT_ROOT/bin/cwsd" /usr/local/bin/cwsd
    cp "$PROJECT_ROOT/bin/cws" /usr/local/bin/cws
    chmod 755 /usr/local/bin/cwsd
    chmod 755 /usr/local/bin/cws
    
    # Verify binaries
    if ! /usr/local/bin/cwsd --version >/dev/null 2>&1; then
        log_error "Failed to verify cwsd binary"
        exit 1
    fi
    
    log_success "Installed binaries to /usr/local/bin/"
}

# Install systemd service
install_systemd_service() {
    log_info "Installing systemd service..."
    
    # Copy service file
    cp "$SCRIPT_DIR/cwsd.service" /etc/systemd/system/
    chmod 644 /etc/systemd/system/cwsd.service
    
    # Reload systemd
    systemctl daemon-reload
    
    log_success "Installed systemd service"
}

# Configure AWS credentials
configure_aws() {
    log_info "Setting up AWS configuration..."
    
    # Create default config files if they don't exist
    if [[ ! -f /etc/cloudworkstation/aws/config ]]; then
        cat > /etc/cloudworkstation/aws/config << 'EOF'
[default]
region = us-west-2
output = json

[profile cloudworkstation]
region = us-west-2
output = json
EOF
        chown root:cloudworkstation /etc/cloudworkstation/aws/config
        chmod 640 /etc/cloudworkstation/aws/config
        log_info "Created default AWS config"
    fi
    
    if [[ ! -f /etc/cloudworkstation/aws/credentials ]]; then
        cat > /etc/cloudworkstation/aws/credentials << 'EOF'
# AWS credentials for CloudWorkstation
# Replace with your actual AWS credentials
[default]
aws_access_key_id = YOUR_ACCESS_KEY_HERE
aws_secret_access_key = YOUR_SECRET_KEY_HERE

[cloudworkstation]
aws_access_key_id = YOUR_ACCESS_KEY_HERE
aws_secret_access_key = YOUR_SECRET_KEY_HERE
EOF
        chown root:cloudworkstation /etc/cloudworkstation/aws/credentials
        chmod 640 /etc/cloudworkstation/aws/credentials
        log_warning "Created template AWS credentials file - YOU MUST EDIT THIS"
        log_warning "Edit /etc/cloudworkstation/aws/credentials with your AWS keys"
    fi
}

# Create SSH keys
setup_ssh_keys() {
    log_info "Setting up SSH keys for instance monitoring..."
    
    SSH_KEY_PATH="/var/lib/cloudworkstation/.ssh/cloudworkstation"
    
    if [[ ! -f "$SSH_KEY_PATH" ]]; then
        sudo -u cloudworkstation ssh-keygen -t ed25519 -f "$SSH_KEY_PATH" -N "" -C "cloudworkstation-daemon"
        log_success "Generated SSH key: $SSH_KEY_PATH"
        log_info "Public key:"
        cat "$SSH_KEY_PATH.pub"
        log_warning "You must add this public key to your AWS EC2 key pairs"
    else
        log_info "SSH key already exists: $SSH_KEY_PATH"
    fi
}

# Create default configuration
create_config() {
    log_info "Creating default configuration..."
    
    if [[ ! -f /etc/cloudworkstation/config.json ]]; then
        cat > /etc/cloudworkstation/config.json << 'EOF'
{
  "autonomous": {
    "auto_execute": false,
    "monitor_interval": "2m",
    "ssh_username": "ubuntu",
    "ssh_key_path": "/var/lib/cloudworkstation/.ssh/cloudworkstation",
    "require_tag_confirmation": true,
    "max_actions_per_hour": 10,
    "dry_run": true
  },
  "daemon": {
    "port": "8947",
    "log_level": "info"
  },
  "idle": {
    "enabled": true,
    "default_profile": "standard"
  }
}
EOF
        chown root:cloudworkstation /etc/cloudworkstation/config.json
        chmod 640 /etc/cloudworkstation/config.json
        log_success "Created default configuration"
        log_warning "Autonomous execution is DISABLED by default for safety"
        log_warning "Set auto_execute=true and dry_run=false to enable automatic actions"
    fi
}

# Main installation
main() {
    log_info "Installing CloudWorkstation as a system service..."
    
    # Detect distribution
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        log_info "Detected OS: $NAME $VERSION_ID"
    fi
    
    # Install based on init system
    case $INIT_SYSTEM in
        systemd)
            create_user
            create_directories
            install_binaries
            install_systemd_service
            configure_aws
            setup_ssh_keys
            create_config
            
            log_success "CloudWorkstation service installed successfully!"
            echo
            log_info "Next steps:"
            echo "  1. Edit /etc/cloudworkstation/aws/credentials with your AWS keys"
            echo "  2. Add the SSH public key to your AWS EC2 key pairs"
            echo "  3. Configure /etc/cloudworkstation/config.json as needed"
            echo "  4. Enable and start the service:"
            echo "     sudo systemctl enable cwsd"
            echo "     sudo systemctl start cwsd"
            echo "  5. Check status:"
            echo "     sudo systemctl status cwsd"
            echo "     sudo journalctl -u cwsd -f"
            ;;
        *)
            log_error "Unsupported init system: $INIT_SYSTEM"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"