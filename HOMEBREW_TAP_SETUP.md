# CloudWorkstation Homebrew Tap Setup Guide

## Overview

This guide explains how to set up a proper Homebrew tap for CloudWorkstation, allowing users to install with standard `brew tap` and `brew install` commands.

## Current Status

✅ **Release Ready**: v0.4.1 is tagged and pushed to GitHub  
✅ **Formula Created**: Production-ready Homebrew formula with correct SHA256  
⏳ **Tap Creation**: Need to create separate tap repository  

## Step 1: Create Homebrew Tap Repository

Create a new GitHub repository named `homebrew-cloudworkstation`:

```bash
# Repository should be: github.com/scttfrdmn/homebrew-cloudworkstation
# This follows Homebrew's naming convention: homebrew-<tapname>
```

## Step 2: Setup Tap Repository

```bash
# Clone the new tap repository
git clone git@github.com:scttfrdmn/homebrew-cloudworkstation.git
cd homebrew-cloudworkstation

# Copy the formula
cp /path/to/cloudworkstation/packaging/homebrew/cloudworkstation.rb .

# Create initial commit
git add cloudworkstation.rb
git commit -m "Initial CloudWorkstation formula for Homebrew tap

- CloudWorkstation v0.4.1 CLI tool for academic research
- Multi-interface support: CLI, TUI, GUI
- Complete with templates, documentation, and service support"

# Push to GitHub
git push origin main
```

## Step 3: User Installation Instructions

Once the tap is set up, users can install CloudWorkstation with:

```bash
# Add the tap
brew tap scttfrdmn/cloudworkstation

# Install CloudWorkstation
brew install cloudworkstation

# Verify installation
cws version
cws templates
```

## Step 4: Test the Tap

Test the complete installation flow:

```bash
# Remove any existing installations
brew uninstall cloudworkstation 2>/dev/null || true
brew untap scttfrdmn/cloudworkstation 2>/dev/null || true

# Test fresh installation
brew tap scttfrdmn/cloudworkstation
brew install cloudworkstation

# Test functionality
cws version                    # Should show v0.4.1
cws templates                  # Should list available templates
cws daemon status              # Should show daemon status
```

## Step 5: Maintenance

To update the formula for new releases:

```bash
# In the tap repository
# Update the formula with new version and SHA256
# Commit and push changes

# Users can then update with:
brew update
brew upgrade cloudworkstation
```

## Current Formula Details

**Location**: `packaging/homebrew/cloudworkstation.rb`  
**Version**: v0.4.1  
**Source**: GitHub release tarball  
**SHA256**: `e4ac4cc646dcedf2df172877db473f091d9f694ffc28912a5a1dc8b738233545`

**Features**:
- Builds from source using Go
- Installs all three interfaces: CLI, TUI, GUI  
- Includes templates and documentation
- Supports macOS service integration
- Comprehensive testing with version verification

## Formula Structure

```ruby
class Cloudworkstation < Formula
  desc "CLI tool for launching pre-configured cloud workstations for academic research"
  homepage "https://github.com/scttfrdmn/cloudworkstation"
  url "https://github.com/scttfrdmn/cloudworkstation/archive/v0.4.1.tar.gz"
  sha256 "e4ac4cc646dcedf2df172877db473f091d9f694ffc28912a5a1dc8b738233545"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "mod", "tidy"
    system "make", "build"
    bin.install "bin/cws", "bin/cwsd", "bin/cws-gui"
    doc.install "README.md", "CLAUDE.md", "CHANGELOG.md"
    share.install "templates"
  end

  test do
    assert_predicate bin/"cws", :exist?
    assert_match "CloudWorkstation v#{version}", shell_output("#{bin}/cws version 2>&1", 0)
    system "#{bin}/cws", "templates"
  end

  service do
    run [opt_bin/"cwsd"]
    keep_alive true
    log_path var/"log/cloudworkstation/cwsd.log"
  end
end
```

## Next Steps

1. **Create tap repository** on GitHub
2. **Copy formula** to tap repository  
3. **Test installation** end-to-end
4. **Document installation** in main README
5. **Announce availability** to users

Once completed, CloudWorkstation will be installable via standard Homebrew commands, providing a professional installation experience for macOS and Linux users.