# Homebrew Release Deployment Process

This document provides a step-by-step process for deploying CloudWorkstation releases via Homebrew tap with prebuilt binaries.

## Prerequisites

- GitHub CLI authenticated with SSH access
- Access to `scttfrdmn/cloudworkstation` and `scttfrdmn/homebrew-cloudworkstation` repositories
- Local CloudWorkstation project built successfully

## Step-by-Step Release Process

### 1. Update Version and Build Release Binaries

```bash
# Update version in all deployment assets
cd /Users/scttfrdmn/src/cloudworkstation

# Update version files
# - pkg/version/version.go: Version = "X.Y.Z-N"
# - Makefile: VERSION := X.Y.Z-N  
# - Formula/cloudworkstation.rb: version "X.Y.Z-N"
# - INSTALL.md, demo.sh, DEMO_SEQUENCE.md: Update version references

# Build release binaries for all platforms
make clean
make release

# Verify release binaries exist and have correct versions
ls -la bin/release/
./bin/release/darwin-arm64/cws --version
./bin/release/darwin-arm64/cwsd --version
# Should show: CloudWorkstation CLI vX.Y.Z-N and CloudWorkstation Daemon vX.Y.Z-N
```

### 2. Create Release Archives

```bash
# Create archives from release binaries (make release already built cross-platform)
cd bin/release

# Create tar.gz archives with correct structure for Homebrew
tar -czf cloudworkstation-darwin-arm64.tar.gz -C darwin-arm64 cws cwsd
tar -czf cloudworkstation-darwin-amd64.tar.gz -C darwin-amd64 cws cwsd

# Verify archive contents (should show binaries in root, not subdirectories)
tar -tzf cloudworkstation-darwin-arm64.tar.gz
# Should show:
# cws
# cwsd

# Verify binary functionality from archives
tar -xzf cloudworkstation-darwin-arm64.tar.gz -C /tmp/test-extract/
/tmp/test-extract/cws --version
# Should show: CloudWorkstation CLI vX.Y.Z-N
```

### 3. Calculate SHA256 Checksums

```bash
# Get checksums for formula
shasum -a 256 /tmp/release-archives/cloudworkstation-darwin-arm64.tar.gz
shasum -a 256 /tmp/release-archives/cloudworkstation-darwin-amd64.tar.gz

# Save these checksums for the formula update
```

### 4. Update Homebrew Formula

Edit `/Users/scttfrdmn/src/cloudworkstation/Formula/cloudworkstation.rb`:

```ruby
class Cloudworkstation < Formula
  desc "Enterprise research management platform - Launch cloud research environments in seconds"
  homepage "https://github.com/scttfrdmn/cloudworkstation"
  license "MIT"
  head "https://github.com/scttfrdmn/cloudworkstation.git", branch: "main"
  
  version "X.Y.Z"  # Update version number

  # Use prebuilt binaries for faster installation  
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/vX.Y.Z/cloudworkstation-darwin-arm64.tar.gz"
      sha256 "ARM64_SHA256_HERE"  # Insert actual checksum
    else
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/vX.Y.Z/cloudworkstation-darwin-amd64.tar.gz"
      sha256 "AMD64_SHA256_HERE"  # Insert actual checksum
    end
  end

  def install
    # Install prebuilt binaries directly from working directory
    bin.install "cws"
    bin.install "cwsd"
  end

  def post_install
    # Ensure configuration directory exists
    system "mkdir", "-p", "#{ENV["HOME"]}/.cloudworkstation"
  end

  test do
    # Test that binaries exist and are executable
    assert_predicate bin/"cws", :exist?
    assert_predicate bin/"cwsd", :exist?
    
    # Test version command
    assert_match "CloudWorkstation v", shell_output("#{bin}/cws --version")
    assert_match "CloudWorkstation v", shell_output("#{bin}/cwsd --version")
  end

  service do
    run [opt_bin/"cwsd"]
    keep_alive true
    log_path var/"log/cloudworkstation/cwsd.log"
    error_log_path var/"log/cloudworkstation/cwsd.log"
    working_dir HOMEBREW_PREFIX
  end
end
```

### 5. Deploy to GitHub Releases

```bash
# Create GitHub release (if new version)
cd /Users/scttfrdmn/src/cloudworkstation
gh release create vX.Y.Z --title "CloudWorkstation vX.Y.Z" --notes "Release notes here"

# Or, if updating existing release, delete old assets first
gh release delete-asset vX.Y.Z cloudworkstation-darwin-arm64.tar.gz -y
gh release delete-asset vX.Y.Z cloudworkstation-darwin-amd64.tar.gz -y

# Upload new binary archives
gh release upload vX.Y.Z /tmp/release-archives/cloudworkstation-darwin-arm64.tar.gz
gh release upload vX.Y.Z /tmp/release-archives/cloudworkstation-darwin-amd64.tar.gz

# Verify upload
gh release view vX.Y.Z --json assets --jq '.assets[].name'
# Should show:
# cloudworkstation-darwin-arm64.tar.gz
# cloudworkstation-darwin-amd64.tar.gz
```

### 6. Update and Deploy Homebrew Tap

```bash
# Copy updated formula to tap repository
cp /Users/scttfrdmn/src/cloudworkstation/Formula/cloudworkstation.rb /opt/homebrew/Library/Taps/scttfrdmn/homebrew-cloudworkstation/

# Commit and push to tap repository
cd /opt/homebrew/Library/Taps/scttfrdmn/homebrew-cloudworkstation
git add cloudworkstation.rb
git commit -m "🚀 RELEASE: Update CloudWorkstation to vX.Y.Z with prebuilt binaries

- Update version to X.Y.Z
- Update SHA256 checksums for new binary archives
- [Add specific changes for this release]

🎉 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"

git push origin main
```

### 7. Test End-to-End Deployment

```bash
# Clean test: remove existing installation
brew uninstall cloudworkstation 2>/dev/null || true
brew untap scttfrdmn/cloudworkstation

# Fresh installation from GitHub
brew tap scttfrdmn/cloudworkstation
brew install cloudworkstation

# Verify installation
cws --version
cwsd --version
brew test cloudworkstation

# Test service functionality
brew services start cloudworkstation
sleep 2
cws daemon status
pgrep -f cwsd | wc -l  # Should be 1

# Verify no duplicate startups
launchctl list | grep cloudworkstation  # Should show single entry
brew services list | grep cloudworkstation  # Should show single service
```

### 8. Service Startup Verification

**Critical Checks for No Duplicates**:
```bash
# Check for single service entry
launchctl list | grep cloudworkstation  # Should show 1 line

# Check for single LaunchAgent file
find ~/Library/LaunchAgents/ -name "*cloudworkstation*"  # Should show 1 file

# Check for single process
pgrep -f cwsd | wc -l  # Should return 1

# Check service restart behavior
brew services restart cloudworkstation
sleep 2
pgrep -f cwsd | wc -l  # Should still be 1
```

## Important Notes

### ✅ **What This Process Ensures:**
- **Fast Installation**: Prebuilt binaries (~1 second vs minutes of compilation)
- **No Duplicates**: Single service entry with unique labels
- **Professional Quality**: Full service integration with proper logging
- **Cross-Platform**: Architecture-specific binaries (ARM64/AMD64)
- **Verifiable**: SHA256 checksums for security
- **Testable**: Built-in formula tests for validation

### ⚠️ **Common Gotchas:**
1. **Archive Structure**: Binaries must be in `darwin-arm64/` and `darwin-amd64/` subdirectories
2. **Hardware Detection**: Use `Hardware::CPU.arm?` not `Hardware::CPU.arm64?`
3. **Installation Paths**: Homebrew extracts to working directory root, not subdirectories
4. **Test Assertions**: Match actual binary output (`CloudWorkstation v`)
5. **Service Labels**: Use unique Homebrew labels to prevent conflicts

### 🔄 **For Cross-Compilation (Future)**:
```bash
# Build for both architectures
make cross-compile

# This creates:
# cloudworkstation-darwin-amd64.tar.gz
# cloudworkstation-darwin-arm64.tar.gz

# Then follow steps 3-8 above
```

This process ensures repeatable, professional deployment of CloudWorkstation via Homebrew with proper service management and no startup duplicates.