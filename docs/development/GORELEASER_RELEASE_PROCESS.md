# GoReleaser Release Process

This document describes the automated release process using GoReleaser, which streamlines building binaries, creating GitHub releases, and updating package managers.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Detailed Step-by-Step Process](#detailed-step-by-step-process)
5. [What GoReleaser Does Automatically](#what-goreleaser-does-automatically)
6. [Common Issues and Solutions](#common-issues-and-solutions)
7. [Verification](#verification)
8. [Rolling Back a Release](#rolling-back-a-release)

## Overview

GoReleaser automates the entire release process:
- Builds binaries for all platforms (Linux, macOS, Windows; amd64 and arm64)
- Creates archives (.tar.gz, .zip)
- Generates Linux packages (deb, rpm, apk)
- Creates GitHub release with all artifacts
- Updates Homebrew formula
- Updates Scoop manifest
- Calculates checksums

**Time to complete**: ~5-10 minutes (mostly building binaries)

## Prerequisites

### 1. GoReleaser Installation

```bash
# Install GoReleaser
brew install goreleaser

# Verify installation
goreleaser --version
# Should show: goreleaser version 2.x.x or higher
```

### 2. GitHub CLI Authentication

GoReleaser requires a GitHub token with `repo` permissions. The easiest way is to use GitHub CLI:

```bash
# Install GitHub CLI if not already installed
brew install gh

# Authenticate with GitHub
gh auth login

# Verify authentication
gh auth status
# Should show: ✓ Logged in to github.com account <your-username>

# Test token retrieval
gh auth token | cut -c1-10
# Should show first 10 characters of your token
```

### 3. Clean Git State

GoReleaser requires a clean working directory (no uncommitted changes):

```bash
# Check git status
git status

# Should show:
# On branch main
# nothing to commit, working tree clean

# If you have uncommitted changes, commit or stash them first
git add .
git commit -m "Prepare for release"
```

### 4. Version Synchronization

Ensure all version files are updated and synchronized:

```bash
# Check version consistency
grep -r "0.5.8" pkg/version/version.go cmd/prism-gui/frontend/package.json

# Verify with smoke tests
make test-smoke
```

## Quick Start

For experienced users, the complete process in three commands:

```bash
# 1. Create and push git tag
git tag -a v0.5.8 -m "Release v0.5.8: [Brief Description]"
git push origin v0.5.8

# 2. Run GoReleaser with GitHub token
export GITHUB_TOKEN=$(gh auth token)
goreleaser release --clean

# 3. Verify release
gh release view v0.5.8
```

## Detailed Step-by-Step Process

### Step 1: Create Git Tag

Create an annotated git tag for the release:

```bash
# Create annotated tag with comprehensive message
git tag -a v0.5.8 -m "Release v0.5.8: Quick Start Experience, Billing Accuracy, and Reliability

Key Features:
- Quick Start Wizard (GUI) - Launch workspace in <30 seconds
- CLI init command - Interactive onboarding wizard
- Background State Monitoring - Async daemon monitoring
- Hibernation Billing Exception - Accurate cost tracking
- AWS System Status Checks - Full readiness verification
- Workspace Terminology - Consistent user-facing language

Success Metrics:
- Time to first workspace: 15min → <30 seconds
- All components build successfully
- Complete feature parity across CLI/TUI/GUI

Full release notes: docs/RELEASE_NOTES_v0.5.8.md"

# Verify tag was created
git tag -l -n10 v0.5.8
```

### Step 2: Push Tag to GitHub

```bash
# Push the tag to GitHub
git push origin v0.5.8

# Note: If you encounter pre-push hook failures about stale binaries,
# you can bypass with --no-verify since GoReleaser will build fresh binaries
git push --no-verify origin v0.5.8
```

### Step 3: Handle Any Uncommitted Changes

If you have uncommitted changes (common with frontend package locks):

```bash
# Check for uncommitted changes
git status

# Common culprits:
# - cmd/prism-gui/frontend/package-lock.json
# - cmd/prism-gui/frontend/node_modules/.package-lock.json

# Commit them
git add cmd/prism-gui/frontend/package-lock.json cmd/prism-gui/frontend/node_modules/.package-lock.json
git commit -m "Update frontend package lock files"
git push origin main

# Move the tag to the new commit
git tag -d v0.5.8
git tag -a v0.5.8 -m "Release v0.5.8: [Brief Description]"
git push --force origin v0.5.8
```

### Step 4: Set GitHub Token

Export the GitHub token as an environment variable:

```bash
# Export token from GitHub CLI
export GITHUB_TOKEN=$(gh auth token)

# Verify it's set (should show a token starting with gho_ or ghp_)
echo $GITHUB_TOKEN | cut -c1-10
```

### Step 5: Run GoReleaser

Run GoReleaser to build and publish the release:

```bash
# Run GoReleaser with clean flag
goreleaser release --clean

# This will take 5-10 minutes and will:
# - Build binaries for all platforms
# - Create archives and packages
# - Upload to GitHub
# - Update Homebrew and Scoop
```

**What you'll see:**
```
• cleaning distribution directory
• loading environment variables
  • using token from $GITHUB_TOKEN
• getting and validating git state
  • git state: commit=abc123... branch=main current_tag=v0.5.8 dirty=false
• building binaries
  • building: dist/prism_darwin_arm64/prism
  • building: dist/prism_linux_amd64/prism
  [... more builds ...]
• creating archives
  • archiving: dist/prism_0.5.8_darwin_arm64.tar.gz
  [... more archives ...]
• linux packages
  • creating: dist/prism_0.5.8_linux_amd64.deb
  [... more packages ...]
• publishing
  • releasing: tag=v0.5.8 repo=scttfrdmn/prism
  • uploading to release: [... artifacts ...]
  • release published: https://github.com/scttfrdmn/prism/releases/tag/v0.5.8
• release succeeded after 41s
```

### Step 6: Verify Release

Verify the release was successful:

```bash
# View release details
gh release view v0.5.8

# Check that all assets are present (should be 13 assets)
gh release view v0.5.8 --json assets --jq '.assets | map(.name)'

# Expected assets:
# - checksums.txt
# - prism_0.5.8_darwin_arm64.tar.gz
# - prism_0.5.8_darwin_x86_64.tar.gz
# - prism_0.5.8_linux_amd64.deb
# - prism_0.5.8_linux_amd64.rpm
# - prism_0.5.8_linux_amd64.apk
# - prism_0.5.8_linux_arm64.deb
# - prism_0.5.8_linux_arm64.rpm
# - prism_0.5.8_linux_arm64.apk
# - prism_0.5.8_linux_arm64.tar.gz
# - prism_0.5.8_linux_x86_64.tar.gz
# - prism_0.5.8_windows_arm64.zip
# - prism_0.5.8_windows_x86_64.zip

# Visit the release page
open https://github.com/scttfrdmn/prism/releases/tag/v0.5.8
```

## What GoReleaser Does Automatically

GoReleaser handles the entire release pipeline:

### 1. Binary Building
- Builds for all target platforms: Linux, macOS, Windows
- Builds for all architectures: amd64, arm64
- Injects version information via ldflags
- Handles CGO requirements for macOS keychain integration

### 2. Archive Creation
- Creates `.tar.gz` archives for Linux and macOS
- Creates `.zip` archives for Windows
- Includes README, LICENSE, CHANGELOG, templates, and docs
- Consistent naming: `prism_VERSION_OS_ARCH.ext`

### 3. Linux Package Generation
- Creates `.deb` packages for Debian/Ubuntu
- Creates `.rpm` packages for RHEL/Fedora/CentOS
- Creates `.apk` packages for Alpine Linux
- All packages for both amd64 and arm64

### 4. Checksum Generation
- Generates SHA256 checksums for all artifacts
- Creates `checksums.txt` file
- Useful for verification and security

### 5. GitHub Release
- Creates GitHub release with tag
- Uploads all artifacts to the release
- Generates changelog from git history
- Sets release notes and metadata

### 6. Homebrew Formula Update
- Updates formula in `scttfrdmn/homebrew-tap`
- Calculates SHA256 checksums for archives
- Updates version numbers
- Commits and pushes changes automatically

### 7. Scoop Manifest Update
- Updates manifest in `scttfrdmn/scoop-bucket`
- Updates download URLs
- Updates version numbers
- Commits and pushes changes automatically

## Common Issues and Solutions

### Issue 1: GITHUB_TOKEN Not Set

**Error:**
```
⨯ release failed: GITHUB_TOKEN environment variable not set
```

**Solution:**
```bash
# Export token from GitHub CLI
export GITHUB_TOKEN=$(gh auth token)

# Or create a personal access token at:
# https://github.com/settings/tokens
# With 'repo' scope, then:
export GITHUB_TOKEN=ghp_your_token_here
```

### Issue 2: Dirty Git State

**Error:**
```
⨯ release failed after 0s
  error=git is in a dirty state
  Please check what can be changing the following files:
   M cmd/prism-gui/frontend/node_modules/.package-lock.json
   M cmd/prism-gui/frontend/package-lock.json
```

**Solution:**
```bash
# Commit the changes
git add cmd/prism-gui/frontend/node_modules/.package-lock.json cmd/prism-gui/frontend/package-lock.json
git commit -m "Update frontend package lock files"
git push origin main

# Move the tag to the new commit
git tag -d v0.5.8
git tag -a v0.5.8 -m "Release v0.5.8: [Description]"
git push --force origin v0.5.8

# Run GoReleaser again
export GITHUB_TOKEN=$(gh auth token)
goreleaser release --clean
```

### Issue 3: Tag Already Exists on Wrong Commit

**Error:**
```
⨯ release failed: tag v0.5.8 already exists on a different commit
```

**Solution:**
```bash
# Delete the tag locally and remotely
git tag -d v0.5.8
git push origin :refs/tags/v0.5.8

# Create tag on correct commit
git tag -a v0.5.8 -m "Release v0.5.8: [Description]"
git push origin v0.5.8
```

### Issue 4: GoReleaser Configuration Errors

**Error:**
```
⨯ release failed: configuration error
```

**Solution:**
```bash
# Validate GoReleaser configuration
goreleaser check

# Fix any issues reported
# The configuration file is: .goreleaser.yaml

# Common issues:
# - Deprecated fields (warnings, but won't block release)
# - Invalid YAML syntax
# - Missing required fields
```

### Issue 5: Network/Upload Failures

**Error:**
```
⨯ release failed: failed to upload artifact
```

**Solution:**
```bash
# Check GitHub status
open https://www.githubstatus.com/

# Retry the release (GoReleaser is idempotent)
goreleaser release --clean

# If partial release exists, delete it first:
gh release delete v0.5.8 --yes
git push origin :refs/tags/v0.5.8
git push origin v0.5.8
goreleaser release --clean
```

## Verification

After a successful release, verify everything is working:

### 1. GitHub Release

```bash
# Check release exists and is public
gh release view v0.5.8

# Verify all 13 assets are present
gh release view v0.5.8 --json assets --jq '.assets | length'
# Should output: 13

# Check download URLs are accessible
curl -I https://github.com/scttfrdmn/prism/releases/download/v0.5.8/prism_0.5.8_darwin_arm64.tar.gz
# Should return: HTTP/2 200
```

### 2. Homebrew Formula

```bash
# Check Homebrew tap was updated
curl https://raw.githubusercontent.com/scttfrdmn/homebrew-tap/main/prism.rb | grep "version \"0.5.8\""

# Test installation (in a clean environment or VM)
brew tap scttfrdmn/tap
brew install prism
prism --version
# Should output: Prism CLI v0.5.8
```

### 3. Scoop Manifest

```bash
# Check Scoop bucket was updated
curl https://raw.githubusercontent.com/scttfrdmn/scoop-bucket/main/prism.json | jq .version
# Should output: "0.5.8"

# Test installation (in Windows environment)
scoop bucket add scttfrdmn https://github.com/scttfrdmn/scoop-bucket
scoop install prism
prism --version
# Should output: Prism CLI v0.5.8
```

### 4. Linux Packages

```bash
# Test deb package (Ubuntu/Debian)
wget https://github.com/scttfrdmn/prism/releases/download/v0.5.8/prism_0.5.8_linux_amd64.deb
sudo dpkg -i prism_0.5.8_linux_amd64.deb
prism --version

# Test rpm package (RHEL/Fedora/CentOS)
wget https://github.com/scttfrdmn/prism/releases/download/v0.5.8/prism_0.5.8_linux_amd64.rpm
sudo rpm -i prism_0.5.8_linux_amd64.rpm
prism --version

# Test apk package (Alpine Linux)
wget https://github.com/scttfrdmn/prism/releases/download/v0.5.8/prism_0.5.8_linux_amd64.apk
sudo apk add --allow-untrusted prism_0.5.8_linux_amd64.apk
prism --version
```

### 5. Checksum Verification

```bash
# Download checksums
wget https://github.com/scttfrdmn/prism/releases/download/v0.5.8/checksums.txt

# Download an artifact
wget https://github.com/scttfrdmn/prism/releases/download/v0.5.8/prism_0.5.8_darwin_arm64.tar.gz

# Verify checksum
sha256sum --check checksums.txt --ignore-missing
# Should output: prism_0.5.8_darwin_arm64.tar.gz: OK
```

## Rolling Back a Release

If you need to rollback a release:

### 1. Delete GitHub Release

```bash
# Delete the release (keeps the tag)
gh release delete v0.5.8 --yes

# Or delete release and tag together
gh release delete v0.5.8 --yes
git push origin :refs/tags/v0.5.8
git tag -d v0.5.8
```

### 2. Revert Homebrew Formula

```bash
# Clone the tap repository
git clone https://github.com/scttfrdmn/homebrew-tap.git
cd homebrew-tap

# Revert to previous version
git revert HEAD --no-edit
git push origin main
```

### 3. Revert Scoop Manifest

```bash
# Clone the bucket repository
git clone https://github.com/scttfrdmn/scoop-bucket.git
cd scoop-bucket

# Revert to previous version
git revert HEAD --no-edit
git push origin main
```

### 4. Communicate the Rollback

```bash
# Create a new issue explaining the rollback
gh issue create --title "Release v0.5.8 Rolled Back" \
  --body "Release v0.5.8 was rolled back due to [reason]. Users should use v0.5.7 until further notice."
```

## Advanced Usage

### Dry Run (Test Without Publishing)

```bash
# Run GoReleaser without publishing
goreleaser release --snapshot --clean

# This will:
# - Build all binaries
# - Create all archives and packages
# - Generate checksums
# - But NOT upload to GitHub or update package managers

# Check the dist/ directory for artifacts
ls -lh dist/
```

### Skip Specific Steps

```bash
# Skip validation (not recommended)
goreleaser release --clean --skip=validate

# Skip announcements
goreleaser release --clean --skip=announce

# Skip Homebrew
goreleaser release --clean --skip=homebrew

# Multiple skips
goreleaser release --clean --skip=validate,announce
```

### Verbose Output

```bash
# Run with debug output
goreleaser release --clean --debug

# Helpful for troubleshooting build issues
```

## Configuration File

The GoReleaser configuration is in `.goreleaser.yaml` at the project root.

Key sections:
- `builds`: Binary build configuration
- `archives`: Archive creation rules
- `nfpms`: Linux package generation
- `brews`: Homebrew formula configuration
- `scoops`: Scoop manifest configuration

See [GoReleaser documentation](https://goreleaser.com/customization/) for configuration options.

## Troubleshooting Resources

- GoReleaser Docs: https://goreleaser.com
- GitHub CLI Docs: https://cli.github.com/manual/
- Homebrew Formula Docs: https://docs.brew.sh/Formula-Cookbook
- Scoop Manifest Docs: https://github.com/ScoopInstaller/Scoop/wiki/App-Manifests

## Summary

The GoReleaser process replaces dozens of manual steps with a single automated command:

```bash
# Old way: ~30-60 minutes of manual work
make build
make cross-compile
make package
make checksums
make github-release
make homebrew-update
make scoop-update

# New way: ~5-10 minutes, fully automated
export GITHUB_TOKEN=$(gh auth token)
goreleaser release --clean
```

**Key Benefits:**
- ✅ Consistent, reproducible releases
- ✅ Eliminates human error
- ✅ Automatic package manager updates
- ✅ Comprehensive artifact generation
- ✅ Built-in verification and checksums
- ✅ Version injection via ldflags
- ✅ Platform-specific optimizations

---

**Last Updated**: October 27, 2025
**GoReleaser Version**: 2.12.7
**Prism Version**: v0.5.8
