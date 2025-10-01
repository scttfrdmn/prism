# CloudWorkstation Release Process

This document outlines the comprehensive release process for CloudWorkstation, covering all steps from version updates to GitHub releases and Homebrew formula updates.

## Table of Contents

1. [Pre-Release Preparation](#pre-release-preparation)
2. [Version Management](#version-management)
3. [Cross-Platform Build Process](#cross-platform-build-process)
4. [Release Artifact Creation](#release-artifact-creation)
5. [Git Tag Management](#git-tag-management)
6. [GitHub Release Creation](#github-release-creation)
7. [Homebrew Formula Updates](#homebrew-formula-updates)
8. [Post-Release Verification](#post-release-verification)
9. [Troubleshooting Common Issues](#troubleshooting-common-issues)
10. [Release Checklist](#release-checklist)

## Pre-Release Preparation

### 1. Code Quality Assurance

Before starting the release process, ensure all code changes are complete and tested:

```bash
# Run comprehensive test suite
make test

# Build and test all components
make build

# Run any specific test suites (example from v0.4.6)
cd cmd/cws-gui/frontend
npm run build
npx vitest run src/App.behavior.test.tsx --reporter=verbose

# Verify TUI functionality
./bin/cws tui

# Verify GUI functionality
./bin/cws-gui

# Verify CLI functionality
./bin/cws templates
./bin/cws list
```

### 2. Documentation Updates

Ensure all documentation reflects the new features and changes:

- Update README.md if needed
- Update CHANGELOG.md with new version details
- Review and update any relevant documentation files
- Create comprehensive release notes (see example below)

### 3. Clean Working Directory

Ensure your working directory is clean and all changes are committed:

```bash
git status
# Should show no uncommitted changes

git log --oneline -10
# Review recent commits to ensure everything is included
```

## Version Management

### 1. Update Version Files

CloudWorkstation uses several files that need version updates:

**pkg/version/version.go:**
```go
package version

// Version is the current version of the CLI
const Version = "0.4.6"  // Update this value
```

**Makefile:**
```makefile
VERSION := 0.4.6  # Update this value
```

### 2. Version Update Commands

```bash
# Update version in pkg/version/version.go
sed -i '' 's/const Version = ".*"/const Version = "0.4.6"/' pkg/version/version.go

# Update version in Makefile
sed -i '' 's/VERSION := .*/VERSION := 0.4.6/' Makefile

# Verify updates
grep -n "Version.*=" pkg/version/version.go
grep -n "VERSION :=" Makefile
```

### 3. Commit Version Changes

```bash
git add pkg/version/version.go Makefile
git commit -m "📦 RELEASE: Update version to v0.4.6"
git push origin main
```

## Cross-Platform Build Process

### 1. Clean Build Environment

```bash
# Clean previous builds
make clean
rm -rf bin/ dist/
```

### 2. Cross-Platform Compilation

CloudWorkstation supports multiple platforms and architectures:

```bash
# Build for all platforms
make cross-compile

# This creates binaries for:
# - darwin/amd64 (macOS Intel)
# - darwin/arm64 (macOS Apple Silicon)
# - linux/amd64 (Linux x86_64)
# - linux/arm64 (Linux ARM64)
# - windows/amd64 (Windows x86_64)
```

### 3. Manual Cross-Compilation (if needed)

```bash
# macOS Intel
GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=0.4.6" -o bin/release/darwin-amd64/cws ./cmd/cws
GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=0.4.6" -o bin/release/darwin-amd64/cwsd ./cmd/cwsd

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=0.4.6" -o bin/release/darwin-arm64/cws ./cmd/cws
GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=0.4.6" -o bin/release/darwin-arm64/cwsd ./cmd/cwsd

# Linux x86_64
GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=0.4.6" -o bin/release/linux-amd64/cws ./cmd/cws
GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=0.4.6" -o bin/release/linux-amd64/cwsd ./cmd/cwsd

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=0.4.6" -o bin/release/linux-arm64/cws ./cmd/cws
GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=0.4.6" -o bin/release/linux-arm64/cwsd ./cmd/cwsd

# Windows x86_64
GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=0.4.6" -o bin/release/windows-amd64/cws.exe ./cmd/cws
GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/scttfrdmn/cloudworkstation/pkg/version.Version=0.4.6" -o bin/release/windows-amd64/cwsd.exe ./cmd/cwsd
```

### 4. GUI Build Process (Platform-Specific)

If including GUI components, build them separately:

```bash
# Build GUI (requires Fyne and platform-specific dependencies)
go build -o bin/release/darwin-amd64/cws-gui ./cmd/cws-gui
go build -o bin/release/darwin-arm64/cws-gui ./cmd/cws-gui
# Note: GUI may not be available on all platforms
```

## Release Artifact Creation

### 1. Create Distribution Directory

```bash
mkdir -p dist/v0.4.6
```

### 2. Package Platform Archives

Create compressed archives for each platform:

```bash
# macOS Intel
cd bin/release/darwin-amd64
tar -czf ../../../dist/v0.4.6/cloudworkstation-v0.4.6-darwin-amd64.tar.gz cws cwsd
cd ../../..

# macOS Apple Silicon
cd bin/release/darwin-arm64
tar -czf ../../../dist/v0.4.6/cloudworkstation-v0.4.6-darwin-arm64.tar.gz cws cwsd
cd ../../..

# Linux x86_64
cd bin/release/linux-amd64
tar -czf ../../../dist/v0.4.6/cloudworkstation-v0.4.6-linux-amd64.tar.gz cws cwsd
cd ../../..

# Linux ARM64
cd bin/release/linux-arm64
tar -czf ../../../dist/v0.4.6/cloudworkstation-v0.4.6-linux-arm64.tar.gz cws cwsd
cd ../../..

# Windows x86_64
cd bin/release/windows-amd64
zip -r ../../../dist/v0.4.6/cloudworkstation-v0.4.6-windows-amd64.zip cws.exe cwsd.exe
cd ../../..
```

### 3. Generate Checksums

Create SHA256 checksums for all archives:

```bash
cd dist/v0.4.6
shasum -a 256 *.tar.gz *.zip > checksums.txt
cd ../..

# Verify checksums file
cat dist/v0.4.6/checksums.txt
```

### 4. Create Release Notes

Create comprehensive release notes (example from v0.4.6):

```bash
cat > dist/v0.4.6/RELEASE_NOTES.md << 'EOF'
# CloudWorkstation v0.4.6 Release Notes

**Release Date**: September 28, 2025
**Tag**: `v0.4.6`
**Branch**: `feature/cloudscape-migration` → `main`

## 🎯 Major Features

### Complete EFS Multi-Modal Integration
CloudWorkstation v0.4.6 delivers comprehensive EFS volume management...

[Include detailed release notes with features, improvements, and breaking changes]
EOF
```

## Git Tag Management

### 1. Create Annotated Git Tag

```bash
# Create annotated tag with release message
git tag -a v0.4.6 -m "$(cat <<'EOF'
CloudWorkstation v0.4.6: Complete EFS Multi-Modal Integration

🎯 Major Features:
• Complete EFS volume management across CLI, TUI, and GUI interfaces
• Multi-instance file sharing for collaborative research environments
• Professional Cloudscape-based GUI with real-time mount status
• Interactive TUI with tabbed navigation and keyboard-driven operations

🔧 Technical Improvements:
• 573+ lines of professional React/TypeScript volume management
• Enhanced TUI architecture with interactive mount/unmount capabilities
• Advanced Cloudscape components integration
• Complete API coverage for GetVolumes, MountVolume, UnmountVolume

📊 Phase 4 Enterprise Features Complete:
✅ Project-Based Organization: Complete project lifecycle with role-based access
✅ Advanced Budget Management: Real-time cost tracking and automated controls
✅ Multi-User Collaboration: Granular permissions and member management
✅ EFS Volume Sharing: Multi-instance file sharing for collaborative research
✅ Multi-Modal Access: Professional interfaces for all user preferences

This release completes CloudWorkstation's Phase 4 enterprise research platform,
providing comprehensive multi-instance file sharing capabilities while maintaining
core simplicity and power for individual researchers.

Ready for institutional deployment and collaborative research workflows.
EOF
)"
```

### 2. Push Tag to Remote

```bash
git push origin v0.4.6
```

### 3. Verify Tag Creation

```bash
git tag -l "v0.4.6"
git show v0.4.6
```

## GitHub Release Creation

### 1. Create GitHub Release via API

```bash
# Create GitHub release using gh CLI
gh release create v0.4.6 \
  --title "CloudWorkstation v0.4.6: Complete EFS Multi-Modal Integration" \
  --notes-file dist/v0.4.6/RELEASE_NOTES.md \
  --prerelease=false \
  dist/v0.4.6/*.tar.gz \
  dist/v0.4.6/*.zip \
  dist/v0.4.6/checksums.txt \
  dist/v0.4.6/RELEASE_NOTES.md
```

### 2. Manual GitHub Release Creation

If using the GitHub web interface:

1. Navigate to https://github.com/scttfrdmn/cloudworkstation/releases
2. Click "Draft a new release"
3. Choose tag: `v0.4.6`
4. Release title: `CloudWorkstation v0.4.6: Complete EFS Multi-Modal Integration`
5. Upload all files from `dist/v0.4.6/`
6. Paste release notes from `RELEASE_NOTES.md`
7. Click "Publish release"

### 3. Verify Release Creation

```bash
# List recent releases
gh release list

# View specific release
gh release view v0.4.6
```

## Homebrew Formula Updates

### 1. Navigate to Homebrew Repository

```bash
# Clone or navigate to homebrew tap repository
cd /path/to/homebrew-cloudworkstation
# or
git clone https://github.com/scttfrdmn/homebrew-cloudworkstation.git
cd homebrew-cloudworkstation
```

### 2. Update Formula with New Version and Checksums

Update `cloudworkstation.rb` with the new version and checksums:

```ruby
class Cloudworkstation < Formula
  desc "Academic research computing platform - Launch cloud research environments"
  homepage "https://github.com/scttfrdmn/cloudworkstation"
  license "MIT"
  head "https://github.com/scttfrdmn/cloudworkstation.git", branch: "main"

  version "0.4.6"  # Update version

  # Use prebuilt binaries for faster installation
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.6/cloudworkstation-v0.4.6-darwin-arm64.tar.gz"
      sha256 "5d8a11d9031cbdbd65e937034c3d50151fe49976cd2b8a631c2e68b74b93f0e8"  # Update checksum
    else
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.6/cloudworkstation-v0.4.6-darwin-amd64.tar.gz"
      sha256 "8171765b3ce9dc0c4305dcf88b277d95db092cd3f8c1928449fab9753a22279d"  # Update checksum
    end
  end

  # ... rest of formula remains the same
end
```

### 3. Update Formula Documentation

Update the caveats section to reflect new version features:

```ruby
def caveats
  s = <<~EOS
    CloudWorkstation #{version} has been installed with full functionality!

    📦 Installed Components:
      • CLI (cws) - Command-line interface with all latest features
      • TUI (cws tui) - Terminal user interface
      • Daemon (cwsd) - Background service
  EOS

  if OS.mac?
    s += <<~EOS
      • GUI (cws-gui) - Desktop application with system tray
    EOS
  end

  s += <<~EOS

    🚀 Quick Start:
      cws profiles add personal research --aws-profile aws --region us-west-2
      cws profiles switch personal
      cws launch "Python Machine Learning (Simplified)" my-project

    📚 Documentation:
      cws help                    # Full command reference (Cobra CLI)
      cws templates               # List available templates
      cws daemon status           # Check daemon status

    🔧 Service Management (Auto-Start on Boot):
      brew services start cloudworkstation   # Auto-start daemon with Homebrew
      brew services stop cloudworkstation    # Stop daemon service
      brew services restart cloudworkstation # Restart daemon service

    🎨 Version 0.4.6 EFS Multi-Modal Integration:
      • Complete EFS volume management across CLI, TUI, and GUI interfaces
      • Multi-instance file sharing for collaborative research environments
      • Professional Cloudscape-based GUI with real-time mount status
      • Interactive TUI with tabbed navigation and keyboard-driven operations

      Example EFS usage:
        cws volumes list                    # List EFS volumes
        cws volumes mount shared-data my-instance  # Mount volume to instance
        cws tui                            # Access storage tab (Press 4)

    Note: Version 0.4.6 completes Phase 4 enterprise research platform features.
  EOS
end
```

### 4. Commit and Push Formula Updates

```bash
# Add and commit changes
git add cloudworkstation.rb
git commit -m "📦 HOMEBREW: Update v0.4.6 formula with EFS multi-modal integration

- Updated version to 0.4.6
- Updated macOS checksums for prebuilt binaries
- Updated caveats with v0.4.6 EFS integration features
- Maintains prebuilt binary installation for fast setup"

# Push to origin
git push origin main
```

### 5. Handle Merge Conflicts (if needed)

If you encounter merge conflicts during push:

```bash
# Pull latest changes
git pull origin main

# Resolve conflicts manually
# Edit cloudworkstation.rb to resolve conflicts
# Keep the newer version info and correct checksums

# Complete the merge
git add cloudworkstation.rb
git rebase --continue

# Push resolved changes
git push origin main
```

## Post-Release Verification

### 1. Test Homebrew Installation

```bash
# Test installation from updated formula
brew uninstall cloudworkstation  # if previously installed
brew install scttfrdmn/cloudworkstation/cloudworkstation

# Verify installation
which cws
which cwsd
cws --version
cwsd --version
```

### 2. Test Downloaded Binaries

```bash
# Download and test release artifacts
wget https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.6/cloudworkstation-v0.4.6-darwin-arm64.tar.gz
tar -xzf cloudworkstation-v0.4.6-darwin-arm64.tar.gz
./cws --version
./cwsd --version
```

### 3. Verify Checksums

```bash
# Verify artifact integrity
shasum -c checksums.txt
```

### 4. Test New Features

Test the major features introduced in the release:

```bash
# Test EFS volume management (v0.4.6 example)
cws volumes list
cws tui  # Test storage interface

# Test GUI functionality
cws-gui  # Test volume management interface
```

## Troubleshooting Common Issues

### 1. Build Failures

**Issue**: Cross-platform compilation fails
```bash
# Solution: Clean and rebuild
make clean
go mod tidy
make cross-compile
```

**Issue**: Missing dependencies
```bash
# Solution: Update dependencies
go mod download
go mod tidy
```

### 2. Git Tag Issues

**Issue**: Tag already exists
```bash
# Solution: Delete and recreate tag
git tag -d v0.4.6
git push origin :refs/tags/v0.4.6
git tag -a v0.4.6 -m "Release message"
git push origin v0.4.6
```

### 3. Homebrew Formula Issues

**Issue**: Merge conflicts in formula
```bash
# Solution: Manual resolution
git pull origin main
# Edit cloudworkstation.rb manually
git add cloudworkstation.rb
git rebase --continue
git push origin main
```

**Issue**: Incorrect checksums
```bash
# Solution: Regenerate and update
shasum -a 256 dist/v0.4.6/*.tar.gz *.zip
# Update checksums in formula manually
```

### 4. GitHub Release Issues

**Issue**: gh CLI authentication
```bash
# Solution: Re-authenticate
gh auth login
gh auth status
```

**Issue**: Upload failures
```bash
# Solution: Retry with individual files
gh release upload v0.4.6 dist/v0.4.6/cloudworkstation-v0.4.6-darwin-arm64.tar.gz
```

## Release Checklist

Use this checklist for every release:

### Pre-Release
- [ ] All tests pass (`make test`)
- [ ] All builds successful (`make build`)
- [ ] Documentation updated
- [ ] Working directory clean
- [ ] Version numbers updated in all files

### Version Management
- [ ] `pkg/version/version.go` updated
- [ ] `Makefile` VERSION updated
- [ ] Version changes committed and pushed

### Build Process
- [ ] Cross-platform builds completed
- [ ] All target platforms built successfully
- [ ] GUI components built (if applicable)
- [ ] Build artifacts organized in `bin/release/`

### Release Artifacts
- [ ] Distribution directory created
- [ ] All platform archives created
- [ ] Checksums generated
- [ ] Release notes written
- [ ] All files in `dist/vX.X.X/`

### Git Management
- [ ] Git tag created with detailed message
- [ ] Tag pushed to remote
- [ ] Tag verified

### GitHub Release
- [ ] GitHub release created
- [ ] All artifacts uploaded
- [ ] Release notes attached
- [ ] Release published (not draft)
- [ ] Release verified accessible

### Homebrew Update
- [ ] Formula version updated
- [ ] Checksums updated in formula
- [ ] Formula documentation updated
- [ ] Formula committed and pushed
- [ ] Merge conflicts resolved (if any)

### Post-Release Verification
- [ ] Homebrew installation tested
- [ ] Downloaded binaries tested
- [ ] Checksums verified
- [ ] New features tested
- [ ] Version commands return correct version

### Documentation
- [ ] Release announced (if applicable)
- [ ] Documentation updated for new version
- [ ] CHANGELOG.md updated

## Release Automation Opportunities

Future improvements to consider:

1. **GitHub Actions**: Automate cross-platform builds
2. **Release Scripts**: Shell scripts to automate version updates
3. **Homebrew Automation**: Auto-update formula via GitHub Actions
4. **Testing Automation**: Automated post-release testing
5. **Notification System**: Slack/Discord notifications for releases

## Version History

This process was used successfully for:
- **v0.4.6**: Complete EFS Multi-Modal Integration (September 28, 2025)
- **v0.4.5**: Production-ready GUI testing and comprehensive pre-release validation
- **v0.4.4**: Enhanced security and prebuilt binaries for fast installation

---

**Last Updated**: September 28, 2025
**Process Version**: 1.0
**Maintainer**: CloudWorkstation Development Team