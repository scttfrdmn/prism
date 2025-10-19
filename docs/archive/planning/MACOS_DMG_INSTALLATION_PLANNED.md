# macOS DMG Installation Guide

Professional macOS installer for CloudWorkstation with native experience, code signing, and notarization.

## Quick Start

1. **Download:** Get the latest `CloudWorkstation-v0.5.5.dmg` from [GitHub Releases](https://github.com/scttfrdmn/cloudworkstation/releases)
2. **Install:** Double-click DMG, drag CloudWorkstation.app to Applications
3. **Launch:** Open CloudWorkstation from Applications or Spotlight
4. **Setup:** Follow the guided setup for AWS configuration

## Installation Methods

### Method 1: DMG Installer (Recommended)

**Best for:** Desktop users who want a native macOS experience with GUI and CLI access.

```bash
# Download and install
curl -L -O https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/CloudWorkstation-v0.5.5.dmg
open CloudWorkstation-v0.5.5.dmg
# Drag CloudWorkstation.app to Applications folder
```

**Includes:**
- Native macOS application bundle
- Automatic CLI tools installation (`cws`, `cwsd`)
- LaunchAgent for daemon auto-start
- Professional uninstaller
- Universal binary (Intel + Apple Silicon)

### Method 2: Homebrew (Traditional)

**Best for:** Command-line users who prefer package managers.

```bash
brew tap scttfrdmn/tap
brew install cloudworkstation
```

### Method 3: Direct Binary Download

**Best for:** Automated deployments or minimal installations.

```bash
# Intel Macs
curl -L https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation-darwin-amd64.tar.gz | tar xz

# Apple Silicon Macs
curl -L https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/cloudworkstation-darwin-arm64.tar.gz | tar xz
```

## DMG Installation Process

### 1. Download and Verification

```bash
# Download DMG
curl -L -O https://github.com/scttfrdmn/cloudworkstation/releases/latest/download/CloudWorkstation-v0.4.2.dmg

# Verify integrity (optional)
hdiutil verify CloudWorkstation-v0.4.2.dmg

# Check code signature (if signed)
codesign --verify --verbose CloudWorkstation-v0.4.2.dmg
```

### 2. Installation

1. **Mount DMG:** Double-click `CloudWorkstation-v0.4.2.dmg`
2. **Install:** Drag `CloudWorkstation.app` to `Applications` folder
3. **Eject DMG:** Unmount the disk image

### 3. First Launch

1. **Open:** Launch CloudWorkstation from Applications or Spotlight
2. **Security:** Allow unsigned app if prompted (first launch only)
3. **Setup Wizard:** Choose your preferred setup:
   - **GUI Interface:** Visual management with desktop integration
   - **Command Line Setup:** Terminal-based installation with PATH configuration

### 4. CLI Tools Installation (Optional)

The DMG installer can automatically install command-line tools:

- **During App Launch:** Choose "Command Line Setup" in welcome screen
- **Manual Installation:** Open CloudWorkstation.app → File → Install CLI Tools
- **Automatic:** CLI tools install to `/usr/local/bin/` with PATH setup

## What's Installed

### Application Bundle Structure

```
/Applications/CloudWorkstation.app/
├── Contents/
│   ├── Info.plist                    # App metadata and configuration
│   ├── MacOS/
│   │   ├── CloudWorkstation          # Main launcher script
│   │   ├── cws                       # CLI client binary
│   │   ├── cwsd                      # Daemon binary
│   │   └── cws-gui                   # GUI binary (full build only)
│   ├── Resources/
│   │   ├── CloudWorkstation.icns     # Application icon
│   │   ├── templates/                # Built-in templates
│   │   └── scripts/
│   │       ├── install-cli-tools.sh # CLI installation
│   │       └── service-manager.sh   # Service management
│   └── Frameworks/                   # Dependencies (if needed)
```

### System Integration

**Command Line Tools:**
- `/usr/local/bin/cws` - CLI client
- `/usr/local/bin/cwsd` - Daemon binary

**User Data Directory:**
- `~/.cloudworkstation/` - Configuration and data
- `~/.cloudworkstation/profiles/` - AWS profiles (secure)
- `~/.cloudworkstation/templates/` - User templates
- `~/.cloudworkstation/logs/` - Application logs

**LaunchAgent:**
- `~/Library/LaunchAgents/com.cloudworkstation.daemon.plist` - Auto-start daemon

**Shell Integration:**
- PATH configuration in `~/.zshrc`, `~/.bashrc`, etc.
- Tab completion (optional)

## Configuration

### Initial Setup

1. **Launch CloudWorkstation**
2. **AWS Configuration:**
   ```bash
   # Via GUI: Settings → AWS Configuration
   # Via CLI:
   cws profiles create my-profile
   ```
3. **Verify Setup:**
   ```bash
   cws --version
   cws templates
   cws profiles list
   ```

### Advanced Configuration

**Daemon Configuration:**
```bash
# Check daemon status
launchctl list com.cloudworkstation.daemon

# Manual daemon control
cwsd --help
```

**Profile Management:**
```bash
# Create profile
cws profiles create research-profile --region us-west-2

# Switch profiles
cws profiles use research-profile

# Export profile
cws profiles export research-profile > profile-backup.json
```

## Security Features

### Code Signing and Notarization

- **Developer ID:** Signed with Apple Developer ID certificate
- **Notarized:** Submitted to Apple for security review
- **Gatekeeper:** Approved for macOS security systems
- **Hardened Runtime:** Enhanced security protections

### Security Verification

```bash
# Verify app signature
codesign --verify --verbose /Applications/CloudWorkstation.app

# Check Gatekeeper approval
spctl --assess --verbose --type execute /Applications/CloudWorkstation.app

# View certificate details
codesign --display --verbose=4 /Applications/CloudWorkstation.app
```

### Keychain Integration

CloudWorkstation integrates with macOS Keychain for secure credential storage:

- AWS credentials stored in Keychain
- Encrypted profile data
- Secure inter-process communication

## Troubleshooting

### Common Issues

**1. "Cannot be opened because it is from an unidentified developer"**
```bash
# Allow in System Preferences > Security & Privacy
# Or via command line:
sudo xattr -rd com.apple.quarantine /Applications/CloudWorkstation.app
```

**2. CLI commands not found**
```bash
# Check PATH
echo $PATH | grep /usr/local/bin

# Reinstall CLI tools
open /Applications/CloudWorkstation.app
# Choose "Command Line Setup"
```

**3. Daemon not starting**
```bash
# Check LaunchAgent
launchctl list | grep cloudworkstation

# Manual start
cwsd

# Reload LaunchAgent
launchctl unload ~/Library/LaunchAgents/com.cloudworkstation.daemon.plist
launchctl load ~/Library/LaunchAgents/com.cloudworkstation.daemon.plist
```

**4. Permission issues**
```bash
# Fix permissions
sudo chown -R $(whoami) ~/.cloudworkstation/
chmod 700 ~/.cloudworkstation/profiles/
```

### Diagnostic Information

```bash
# System information
make service-info

# Check installation
which cws
cws --version

# Daemon status
cws daemon status

# View logs
tail -f ~/.cloudworkstation/logs/daemon.log
```

### Getting Help

1. **In-App Help:** CloudWorkstation.app → Help Menu
2. **Command Line:** `cws --help`
3. **Documentation:** [GitHub Wiki](https://github.com/scttfrdmn/cloudworkstation/wiki)
4. **Issues:** [GitHub Issues](https://github.com/scttfrdmn/cloudworkstation/issues)

## Uninstallation

### Complete Removal

The DMG includes a professional uninstaller:

```bash
# Via included script
/Applications/CloudWorkstation.app/Contents/Resources/scripts/uninstall.sh

# Or download uninstaller
curl -L -O https://raw.githubusercontent.com/scttfrdmn/cloudworkstation/main/scripts/macos-uninstall.sh
chmod +x macos-uninstall.sh
./macos-uninstall.sh
```

### Manual Removal

```bash
# Stop and remove daemon
launchctl unload ~/Library/LaunchAgents/com.cloudworkstation.daemon.plist
rm ~/Library/LaunchAgents/com.cloudworkstation.daemon.plist

# Remove application
rm -rf /Applications/CloudWorkstation.app

# Remove CLI tools
sudo rm /usr/local/bin/cws /usr/local/bin/cwsd

# Remove user data (optional)
rm -rf ~/.cloudworkstation/

# Clean shell configuration
# Edit ~/.zshrc, ~/.bashrc to remove CloudWorkstation PATH entries
```

### Uninstall Options

- `--complete` - Remove everything including user data
- `--keep-data` - Keep AWS profiles and configuration
- Default: Remove app but keep user data for future installations

## System Requirements

### Minimum Requirements

- **OS:** macOS 10.15 (Catalina) or later
- **Architecture:** Intel (x86_64) or Apple Silicon (arm64)
- **Memory:** 512MB available RAM
- **Storage:** 200MB free disk space
- **Network:** Internet connection for AWS operations

### Recommended Requirements

- **OS:** macOS 12.0 (Monterey) or later
- **Memory:** 2GB available RAM
- **Storage:** 1GB free disk space
- **AWS:** Valid AWS account with appropriate permissions

### Compatibility

**macOS Versions:**
- ✅ macOS 14.0+ (Sonoma) - Fully supported
- ✅ macOS 13.0+ (Ventura) - Fully supported
- ✅ macOS 12.0+ (Monterey) - Fully supported
- ✅ macOS 11.0+ (Big Sur) - Supported
- ✅ macOS 10.15+ (Catalina) - Supported (minimum)

**Architectures:**
- ✅ Apple Silicon (M1, M2, M3) - Native universal binary
- ✅ Intel x86_64 - Native support
- ✅ Rosetta 2 - Intel binaries run on Apple Silicon

## Comparison with Other Installation Methods

| Feature | DMG Installer | Homebrew | Direct Binary |
|---------|---------------|----------|---------------|
| GUI Application | ✅ | ❌ | ❌ |
| CLI Tools | ✅ | ✅ | ✅ |
| Auto PATH Setup | ✅ | ✅ | ❌ |
| Auto-start Daemon | ✅ | ❌ | ❌ |
| Native macOS Experience | ✅ | ❌ | ❌ |
| Uninstaller | ✅ | ✅ | ❌ |
| Code Signed | ✅ | ✅ | ❌ |
| Auto Updates | 🔜 | ✅ | ❌ |
| Offline Installation | ✅ | ❌ | ✅ |

## Build Information

This DMG installer is built using:

- **Build System:** Professional DMG creation pipeline
- **Signing:** Apple Developer ID Application certificate
- **Notarization:** Apple notary service
- **CI/CD:** GitHub Actions automation
- **Testing:** Comprehensive integrity and functionality tests

### Build Targets

```bash
# Development DMG (fast)
make dmg-dev

# Universal DMG (Intel + Apple Silicon)
make dmg-universal

# Signed DMG
make dmg-signed

# Complete pipeline (build → sign → notarize)
make dmg-all
```

## Contributing to DMG Installation

### Reporting Issues

1. **Installation Issues:** Use the "DMG Installation" issue template
2. **Include Diagnostics:** Run `make service-info` and include output
3. **System Information:** Include macOS version and architecture

### Testing

```bash
# Test DMG creation
make test-dmg

# Validate installation
./scripts/test-dmg-installation.sh
```

### Development

The DMG creation system consists of:

- `scripts/build-dmg.sh` - Main DMG creation
- `scripts/sign-dmg.sh` - Code signing
- `scripts/notarize-dmg.sh` - Apple notarization  
- `scripts/macos-postinstall.sh` - Post-installation setup
- `scripts/macos-uninstall.sh` - Complete removal
- `.github/workflows/build-dmg.yml` - CI/CD automation

---

**CloudWorkstation macOS DMG Installer** - Professional installation experience for academic researchers launching cloud workstations in seconds.