# DMG Build Guide

Complete guide for building professional macOS DMG installer packages for Prism.

## Quick Start

```bash
# Install prerequisites
make dmg-setup

# Build development DMG (fastest)
make dmg-dev

# Build universal DMG (Intel + Apple Silicon)
make dmg-universal

# Complete signed and notarized DMG
make dmg-all
```

## Prerequisites

### System Requirements

- **macOS:** Required for DMG creation
- **Xcode:** Command line tools installed
- **Python 3:** For icon generation (with Pillow)
- **Developer ID:** For signing (optional)
- **Apple ID:** For notarization (optional)

### Setup

```bash
# Install Xcode command line tools
xcode-select --install

# Install Python dependencies
pip3 install Pillow

# Verify prerequisites
make dmg-setup
```

## Build Process Overview

The DMG build process consists of several stages:

1. **Binary Building** - Compile Go binaries for target architectures
2. **App Bundle Creation** - Create macOS application bundle structure
3. **Resource Copying** - Include templates, icons, and scripts
4. **DMG Creation** - Generate disk image with custom layout
5. **Code Signing** - Sign with Developer ID (optional)
6. **Notarization** - Submit to Apple for approval (optional)

## Build Targets

### Development Builds

```bash
# Fast build for testing (CLI + daemon only, no GUI)
make dmg-dev

# Standard build (current architecture)
make dmg

# Test DMG integrity
make dmg-test
```

### Production Builds

```bash
# Universal binary (Intel + Apple Silicon)
make dmg-universal

# Signed DMG
make dmg-signed

# Universal signed DMG
make dmg-universal-signed

# Complete pipeline: build → sign → notarize
make dmg-all
```

### Utility Targets

```bash
# Clean build artifacts
make dmg-clean

# Setup prerequisites
make dmg-setup

# Test existing DMG
make dmg-test
```

## Manual Build Process

### 1. Build Binaries

```bash
# For current architecture
make build

# For universal binary
GOOS=darwin GOARCH=amd64 make build
GOOS=darwin GOARCH=arm64 make build
# Combine with lipo
```

### 2. Create DMG

```bash
# Basic DMG creation
./scripts/build-dmg.sh

# Development DMG (no GUI)
./scripts/build-dmg.sh --dev

# Universal DMG
./scripts/build-dmg.sh --universal
```

### 3. Sign DMG (Optional)

```bash
# With default identity
./scripts/sign-dmg.sh

# With specific identity
./scripts/sign-dmg.sh --dev-id "Developer ID Application: Your Name (TEAMID)"

# Verify existing signatures
./scripts/sign-dmg.sh --verify-only
```

### 4. Notarize DMG (Optional)

```bash
# Setup credentials first
xcrun notarytool store-credentials prism \
  --apple-id your@email.com \
  --team-id TEAMID

# Notarize DMG
./scripts/notarize-dmg.sh

# Check notarization status
./scripts/notarize-dmg.sh --check-status UUID
```

## Code Signing Setup

### 1. Developer ID Certificate

1. **Join Apple Developer Program** ($99/year)
2. **Generate Certificate Signing Request** in Keychain Access
3. **Create Developer ID Application certificate** at developer.apple.com
4. **Download and install** certificate in Keychain

### 2. Verify Certificate

```bash
# List available certificates
security find-identity -v -p codesigning

# Should show: "Developer ID Application: Your Name (TEAMID)"
```

### 3. Environment Variables (CI/CD)

```bash
# Export certificate for CI
security export -t cert -f pkcs12 -k login.keychain -P password cert.p12

# GitHub Secrets:
DEVELOPER_ID_APPLICATION_P12=<base64 encoded p12>
DEVELOPER_ID_APPLICATION_PASSWORD=<p12 password>
DEVELOPER_ID_APPLICATION_IDENTITY=<full certificate name>
```

## Notarization Setup

### 1. App-Specific Password

1. **Sign in** to appleid.apple.com
2. **Generate app-specific password** for notarization
3. **Store securely** for CLI use

### 2. Store Credentials

```bash
# Store in keychain
xcrun notarytool store-credentials prism \
  --apple-id your@email.com \
  --team-id TEAMID123 \
  --password

# Verify stored credentials
xcrun notarytool history --keychain-profile prism
```

### 3. Environment Variables (CI/CD)

```bash
# GitHub Secrets:
APPLE_ID=your@email.com
APPLE_ID_PASSWORD=<app-specific password>
APPLE_TEAM_ID=TEAMID123
```

## DMG Customization

### Visual Design

The DMG uses custom visual elements:

- **Background Image:** Programmatically generated with branding
- **Window Layout:** Custom positioning via AppleScript
- **Icons:** High-resolution app icon with multiple sizes
- **Typography:** System fonts with proper hierarchy

### Layout Configuration

```bash
# Window size: 600x400
# App icon position: (150, 200)  
# Applications folder: (450, 200)
# README position: (300, 350)
```

### Custom Assets

```bash
# App icon sources (auto-generated if missing)
assets/icon.png                    # Source icon
.background/dmg-background.png     # DMG background

# Generated icon sizes
icon_16x16.png through icon_1024x1024.png
icon_16x16@2x.png through icon_512x512@2x.png
```

## Troubleshooting

### Common Build Issues

**1. Missing Xcode tools**
```bash
xcode-select --install
# Ensure tools are installed in /Applications/Xcode.app or /Library/Developer
```

**2. Python/Pillow issues**
```bash
# Install via pip
pip3 install Pillow --break-system-packages

# Or via conda
conda install pillow

# Fallback: DMG will use system tools without custom icons
```

**3. Permission errors**
```bash
# Ensure scripts are executable
chmod +x scripts/*.sh

# Check disk space
df -h .
```

**4. Signing failures**
```bash
# Verify certificate
security find-identity -v -p codesigning

# Check certificate validity
security show-identity -p codesigning "Developer ID Application: Your Name"

# Clear signing cache
sudo rm -rf ~/Library/Caches/com.apple.dt.Xcode/
```

**5. Notarization issues**
```bash
# Check credentials
xcrun notarytool history --keychain-profile prism

# Verify app-specific password
# Make sure 2FA is enabled on Apple ID

# Check submission status
xcrun notarytool info SUBMISSION-UUID --keychain-profile prism
```

### Testing and Validation

**Test DMG Creation:**
```bash
# Build and test
make dmg-dev
make dmg-test

# Manual verification
hdiutil verify dist/dmg/Prism-v0.4.2.dmg
```

**Test Installation:**
```bash
# Mount and inspect
open dist/dmg/Prism-v0.4.2.dmg
# Verify all components present

# Test app bundle
/Applications/Prism.app/Contents/MacOS/Prism --help
```

**Test Signing:**
```bash
# Verify signature
codesign --verify --verbose /Applications/Prism.app

# Test Gatekeeper
spctl --assess --verbose --type execute /Applications/Prism.app
```

## CI/CD Integration

### GitHub Actions

The repository includes two workflows:

1. **build-dmg.yml** - Full release pipeline with signing/notarization
2. **test-dmg.yml** - PR testing without certificates

### Workflow Triggers

```yaml
# Full build on tags
on:
  push:
    tags: ['v*']

# Testing on PRs
on:
  pull_request:
    paths:
      - 'scripts/build-dmg.sh'
      - 'cmd/**'
      - 'pkg/**'
```

### Secret Configuration

Required GitHub Secrets for signing/notarization:

```
DEVELOPER_ID_APPLICATION_P12
DEVELOPER_ID_APPLICATION_PASSWORD  
DEVELOPER_ID_APPLICATION_IDENTITY
APPLE_ID
APPLE_ID_PASSWORD
APPLE_TEAM_ID
```

## Performance Optimization

### Build Speed

```bash
# Development builds (fastest)
make dmg-dev           # ~30 seconds

# Standard builds
make dmg               # ~60 seconds

# Universal builds
make dmg-universal     # ~90 seconds

# Signed builds
make dmg-signed        # +30 seconds

# Notarized builds
make dmg-notarized     # +300 seconds (Apple processing)
```

### Size Optimization

- **Base DMG:** ~50MB (compressed)
- **Universal:** ~80MB (Intel + Apple Silicon)
- **Templates:** ~5MB (included resources)
- **Compression:** UDZO with zlib-level 9

### Caching

```bash
# Go build cache
export GOCACHE=~/.cache/go-build

# Module cache  
export GOMODCACHE=~/go/pkg/mod

# Reuse build artifacts when possible
```

## Release Process

### 1. Prepare Release

```bash
# Update version
make bump-minor  # or bump-major, bump-patch

# Update changelog
vim CHANGELOG.md

# Commit changes
git add .
git commit -m "Release v0.4.3"
```

### 2. Create Release

```bash
# Tag release
git tag v0.4.3
git push origin v0.4.3

# GitHub Actions will automatically:
# - Build universal DMG
# - Sign with Developer ID
# - Submit for notarization
# - Create GitHub release
# - Upload DMG as release asset
```

### 3. Post-Release

```bash
# Test release
curl -L -O https://github.com/scttfrdmn/prism/releases/latest/download/Prism-v0.4.3.dmg

# Update documentation
# Update Homebrew formula (if needed)
# Announce release
```

## Advanced Topics

### Custom App Bundle

The DMG creates a complete macOS application bundle:

```
Prism.app/
├── Contents/
│   ├── Info.plist              # Bundle metadata
│   ├── MacOS/
│   │   ├── Prism    # Launcher script
│   │   ├── prism                 # CLI binary
│   │   ├── cwsd                # Daemon binary
│   │   └── cws-gui            # GUI binary
│   ├── Resources/
│   │   ├── Prism.icns
│   │   ├── templates/
│   │   └── scripts/
│   └── Frameworks/            # Dependencies (if needed)
```

### Post-Installation Hooks

The launcher script (`Contents/MacOS/Prism`) handles:

- First-run welcome dialog
- CLI tools installation
- PATH configuration
- LaunchAgent setup
- AWS profile wizard

### Uninstallation

Professional uninstaller included:

- Removes application bundle
- Removes CLI tools
- Stops and removes LaunchAgent
- Cleans shell configuration
- Optional user data removal

---

This DMG build system provides a professional, Apple-compliant installation experience for Prism users.