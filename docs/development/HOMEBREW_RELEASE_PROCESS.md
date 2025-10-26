# Homebrew Release Deployment Process

This document provides a step-by-step process for deploying Prism releases via Homebrew tap with prebuilt binaries.

## Prerequisites

- GitHub CLI authenticated with SSH access
- Access to `scttfrdmn/prism` and `scttfrdmn/homebrew-prism` repositories
- Local Prism project built successfully

## Step-by-Step Release Process

### 1. Update Version and Build Release Binaries

```bash
# Update version in all deployment assets
cd /Users/scttfrdmn/src/prism

# Update version files
# - pkg/version/version.go: Version = "X.Y.Z-N"
# - Makefile: VERSION := X.Y.Z-N  
# - Formula/prism.rb: version "X.Y.Z-N"
# - INSTALL.md, demo.sh, DEMO_SEQUENCE.md: Update version references

# Build release binaries for all platforms
make clean
make release

# Verify release binaries exist and have correct versions
ls -la bin/release/
./bin/release/darwin-arm64/cws --version
./bin/release/darwin-arm64/cwsd --version
# Should show: Prism CLI vX.Y.Z-N and Prism Daemon vX.Y.Z-N
```

### 2. Create Release Archives

```bash
# Create archives from release binaries (make release already built cross-platform)
cd bin/release

# Create tar.gz archives with correct structure for Homebrew
tar -czf prism-darwin-arm64.tar.gz -C darwin-arm64 prism cwsd
tar -czf prism-darwin-amd64.tar.gz -C darwin-amd64 prism cwsd

# Verify archive contents (should show binaries in root, not subdirectories)
tar -tzf prism-darwin-arm64.tar.gz
# Should show:
# cws
# cwsd

# Verify binary functionality from archives
tar -xzf prism-darwin-arm64.tar.gz -C /tmp/test-extract/
/tmp/test-extract/cws --version
# Should show: Prism CLI vX.Y.Z-N
```

### 3. Calculate SHA256 Checksums

```bash
# Get checksums for formula
shasum -a 256 /tmp/release-archives/prism-darwin-arm64.tar.gz
shasum -a 256 /tmp/release-archives/prism-darwin-amd64.tar.gz

# Save these checksums for the formula update
```

### 4. Update Homebrew Formula

Edit `/Users/scttfrdmn/src/prism/Formula/prism.rb`:

```ruby
class Cloudworkstation < Formula
  desc "Enterprise research management platform - Launch cloud research environments in seconds"
  homepage "https://github.com/scttfrdmn/prism"
  license "MIT"
  head "https://github.com/scttfrdmn/prism.git", branch: "main"
  
  version "X.Y.Z"  # Update version number

  # Use prebuilt binaries for faster installation  
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/scttfrdmn/prism/releases/download/vX.Y.Z/prism-darwin-arm64.tar.gz"
      sha256 "ARM64_SHA256_HERE"  # Insert actual checksum
    else
      url "https://github.com/scttfrdmn/prism/releases/download/vX.Y.Z/prism-darwin-amd64.tar.gz"
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
    system "mkdir", "-p", "#{ENV["HOME"]}/.prism"
  end

  test do
    # Test that binaries exist and are executable
    assert_predicate bin/"cws", :exist?
    assert_predicate bin/"cwsd", :exist?
    
    # Test version command
    assert_match "Prism v", shell_output("#{bin}/cws --version")
    assert_match "Prism v", shell_output("#{bin}/cwsd --version")
  end

  service do
    run [opt_bin/"cwsd"]
    keep_alive true
    log_path var/"log/prism/cwsd.log"
    error_log_path var/"log/prism/cwsd.log"
    working_dir HOMEBREW_PREFIX
  end
end
```

### 5. Deploy to GitHub Releases

```bash
# Create GitHub release (if new version)
cd /Users/scttfrdmn/src/prism
gh release create vX.Y.Z --title "Prism vX.Y.Z" --notes "Release notes here"

# Or, if updating existing release, delete old assets first
gh release delete-asset vX.Y.Z prism-darwin-arm64.tar.gz -y
gh release delete-asset vX.Y.Z prism-darwin-amd64.tar.gz -y

# Upload new binary archives
gh release upload vX.Y.Z /tmp/release-archives/prism-darwin-arm64.tar.gz
gh release upload vX.Y.Z /tmp/release-archives/prism-darwin-amd64.tar.gz

# Verify upload
gh release view vX.Y.Z --json assets --jq '.assets[].name'
# Should show:
# prism-darwin-arm64.tar.gz
# prism-darwin-amd64.tar.gz
```

### 6. Update and Deploy Homebrew Tap

```bash
# Copy updated formula to tap repository
cp /Users/scttfrdmn/src/prism/Formula/prism.rb /opt/homebrew/Library/Taps/scttfrdmn/homebrew-prism/

# Commit and push to tap repository
cd /opt/homebrew/Library/Taps/scttfrdmn/homebrew-prism
git add prism.rb
git commit -m "🚀 RELEASE: Update Prism to vX.Y.Z with prebuilt binaries

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
brew uninstall prism 2>/dev/null || true
brew untap scttfrdmn/prism

# Fresh installation from GitHub
brew tap scttfrdmn/prism
brew install prism

# Verify installation
prism --version
cwsd --version
brew test prism

# Test service functionality
brew services start prism
sleep 2
prism daemon status
pgrep -f cwsd | wc -l  # Should be 1

# Verify no duplicate startups
launchctl list | grep prism  # Should show single entry
brew services list | grep prism  # Should show single service
```

### 8. Service Startup Verification

**Critical Checks for No Duplicates**:
```bash
# Check for single service entry
launchctl list | grep prism  # Should show 1 line

# Check for single LaunchAgent file
find ~/Library/LaunchAgents/ -name "*prism*"  # Should show 1 file

# Check for single process
pgrep -f cwsd | wc -l  # Should return 1

# Check service restart behavior
brew services restart prism
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
4. **Test Assertions**: Match actual binary output (`Prism v`)
5. **Service Labels**: Use unique Homebrew labels to prevent conflicts

### 🔄 **For Cross-Compilation (Future)**:
```bash
# Build for both architectures
make cross-compile

# This creates:
# prism-darwin-amd64.tar.gz
# prism-darwin-arm64.tar.gz

# Then follow steps 3-8 above
```

This process ensures repeatable, professional deployment of Prism via Homebrew with proper service management and no startup duplicates.