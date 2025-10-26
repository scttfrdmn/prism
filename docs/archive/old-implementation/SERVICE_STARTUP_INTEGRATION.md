# Prism Service Startup Integration

This document describes the comprehensive system startup integration for Prism daemon across different installation methods and operating systems.

## Overview

Prism provides automatic daemon startup on system boot through platform-specific service management systems:

- **macOS**: launchd services (user and system mode)
- **Linux**: systemd services with security hardening
- **Windows**: Windows Service Manager integration

## Platform-Specific Implementation

### macOS (launchd)

**Service Files:**
- `scripts/com.prism.daemon.plist` - launchd plist template
- `scripts/macos-service-manager.sh` - Complete service management
- `scripts/macos-dmg-postinstall.sh` - DMG installer integration

**Installation Methods:**
1. **Homebrew**: `brew services start prism`
2. **Manual**: `./scripts/macos-service-manager.sh install`
3. **DMG**: Automatic via post-install script

**Service Modes:**
- **User Mode**: Service runs when user is logged in
- **System Mode**: Service runs at system startup (requires sudo)

**Features:**
- Automatic restart on crashes
- Resource limits and security constraints
- Configurable log paths and environment
- Integration with macOS system preferences

### Linux (systemd)

**Service Files:**
- `systemd/cwsd.service` - systemd unit file
- `scripts/linux-service-manager.sh` - Complete service management
- `systemd/install-service.sh` - Installation script

**Installation:**
```bash
# Install system service
sudo ./scripts/linux-service-manager.sh install

# Enable auto-startup
sudo systemctl enable cwsd

# Start service
sudo systemctl start cwsd
```

**Security Features:**
- Dedicated `prism` user
- Restricted file system access
- Security hardening (NoNewPrivileges, ProtectSystem, etc.)
- Resource limits and process constraints

**Service Configuration:**
- Configuration: `/etc/prism/`
- Data directory: `/var/lib/prism/`
- Logs: `/var/log/prism/` and systemd journal

### Windows (Service Manager)

**Service Files:**
- `scripts/windows-service-wrapper.go` - Go-based Windows service wrapper
- `scripts/windows-service-manager.ps1` - PowerShell management
- Windows Event Log integration

**Installation:**
```powershell
# Install service (requires Administrator)
.\scripts\windows-service-manager.ps1 install

# Check status
.\scripts\windows-service-manager.ps1 status
```

**Service Features:**
- Automatic startup configuration
- Windows Event Log integration
- Service recovery on failures
- Proper Windows service lifecycle management

## Cross-Platform Management

### Universal Service Manager

The `scripts/service-manager.sh` script provides unified service management across all platforms:

```bash
# Works on any supported platform
./scripts/service-manager.sh install    # Install service
./scripts/service-manager.sh status     # Check status
./scripts/service-manager.sh logs       # View logs
./scripts/service-manager.sh validate   # Validate configuration
```

### Makefile Integration

Service management is integrated into the build system:

```bash
make service-install      # Install service
make service-status       # Check status
make service-logs         # View logs
make install-complete     # Install binaries + service
```

## Installation Method Integration

### Homebrew (macOS)

The Homebrew formula includes full service integration:

```ruby
service do
  run [opt_bin/"cwsd"]
  keep_alive true
  log_path var/"log/prism/cwsd.log"
  error_log_path var/"log/prism/cwsd.log"
  working_dir HOMEBREW_PREFIX
end
```

**Usage:**
```bash
brew install prism
brew services start prism  # Auto-startup enabled
```

### Chocolatey (Windows)

Chocolatey package includes Windows service installation:

```powershell
# Automatic during installation
choco install prism

# Service is installed and started automatically
```

### DMG Installer (macOS)

DMG packages include post-install script for automatic service setup:
- Verifies installation
- Creates user directories
- Installs user-mode service
- Provides setup instructions

### Linux Package Managers

Integration planned for:
- **APT packages** (Debian/Ubuntu)
- **RPM packages** (RHEL/CentOS/Fedora)
- **Snap packages** (Universal Linux)

## Service Configuration

### Default Configuration

Each platform includes default service configuration:

**macOS launchd:**
```xml
<key>EnvironmentVariables</key>
<dict>
    <key>PRISM_SERVICE_MODE</key>
    <string>true</string>
    <key>HOME</key>
    <string>/Users/username</string>
</dict>
```

**Linux systemd:**
```ini
Environment=PRISM_SERVICE_MODE=true
Environment=PRISM_CONFIG_DIR=/etc/prism
Environment=PRISM_STATE_DIR=/var/lib/prism
```

**Windows Service:**
```go
cmd.Env = append(os.Environ(),
    "PRISM_SERVICE_MODE=true",
    "PRISM_LOG_PATH=C:\\ProgramData\\Prism\\Logs",
)
```

### Security Configuration

All services implement security best practices:

**Linux systemd security:**
- `NoNewPrivileges=yes`
- `ProtectSystem=strict`
- `PrivateTmp=yes`
- `RestrictRealtime=yes`

**macOS security:**
- Resource limits (file handles, processes)
- Restricted user context
- Secure file permissions

**Windows security:**
- Runs as Local System with minimal privileges
- Event logging for audit trails
- Service recovery configuration

## Logging and Monitoring

### Log Locations

**macOS:**
- User mode: `~/Library/Logs/prism/`
- System mode: `/var/log/prism/`

**Linux:**
- systemd journal: `journalctl -u cwsd`
- Log files: `/var/log/prism/`

**Windows:**
- Windows Event Log (Application)
- Log files: `%ProgramData%\Prism\Logs\`

### Log Management

```bash
# Cross-platform log viewing
./scripts/service-manager.sh logs        # View logs
./scripts/service-manager.sh follow      # Follow in real-time

# Platform-specific
journalctl -u cwsd -f                    # Linux
tail -f ~/Library/Logs/prism/ # macOS
Get-EventLog -LogName Application -Source PrismDaemon  # Windows
```

## Troubleshooting

### Common Issues

**Service Won't Start:**
```bash
./scripts/service-manager.sh validate   # Check configuration
./scripts/service-manager.sh status     # Check current status
```

**Permission Issues:**
- macOS: Check user permissions and keychain access
- Linux: Verify `prism` user exists
- Windows: Ensure Administrator privileges for service management

**Configuration Issues:**
```bash
./scripts/service-manager.sh validate   # Comprehensive validation
```

### Manual Service Management

If automated installation fails, services can be managed manually:

**macOS:**
```bash
launchctl load ~/Library/LaunchAgents/com.prism.daemon.plist
launchctl start com.prism.daemon
```

**Linux:**
```bash
sudo systemctl enable cwsd
sudo systemctl start cwsd
```

**Windows:**
```powershell
sc create PrismDaemon binPath="C:\path\to\prism-service.exe"
sc start PrismDaemon
```

## Development and Testing

### Testing Service Installation

```bash
# Test on current platform
make service-install
make service-status
make service-validate

# Test cross-platform
./scripts/service-manager.sh info       # System information
./scripts/service-manager.sh validate  # Configuration validation
```

### Service Development

Service wrapper development:
- Go-based for cross-platform compatibility
- Platform-specific native service integration
- Comprehensive error handling and logging
- Graceful shutdown and restart handling

## Future Enhancements

### Planned Features

1. **Container Integration**: Docker/Podman service management
2. **Cloud Service Integration**: AWS ECS/Lambda service deployment
3. **High Availability**: Multi-instance service clustering
4. **Monitoring Integration**: Prometheus/Grafana metrics
5. **Configuration Management**: Centralized service configuration

### Package Manager Integration

Planned integration with additional package managers:
- **Snap packages** (Universal Linux)
- **Flatpak** (Linux desktop applications)
- **winget** (Windows Package Manager)
- **Scoop** (Windows command-line installer)

## Conclusion

Prism provides comprehensive system startup integration across all supported platforms, ensuring that the daemon starts automatically on system boot while maintaining security best practices and providing robust management tools for different installation methods.