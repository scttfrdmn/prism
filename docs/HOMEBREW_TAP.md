# CloudWorkstation Homebrew Tap

This document describes how to set up and maintain the Homebrew tap for CloudWorkstation.

## Overview

Homebrew taps are third-party repositories of formulas. For CloudWorkstation, we maintain a tap repository at:
https://github.com/scttfrdmn/homebrew-cloudworkstation

## Setup Instructions

### 1. Create the Tap Repository

If you haven't already created the tap repository:

1. Create a new GitHub repository named `homebrew-cloudworkstation`
2. Initialize with a README.md explaining the purpose of the tap
3. Create a `Formula` directory to store the formula files

```bash
mkdir -p Formula
```

### 2. Add the Formula

Copy the CloudWorkstation formula to the repository:

```bash
cp packaging/homebrew/cloudworkstation.rb Formula/
```

### 3. Configure Automated Updates

The formula is automatically updated by the GitHub Action workflow in `.github/workflows/homebrew-update.yml` when new releases are published. This requires:

1. A GitHub Personal Access Token with `repo` scope added as a secret named `TAP_REPO_TOKEN`
2. Proper versioning in the main repository

## Using the Tap

Users can install CloudWorkstation from the tap with:

```bash
# Add the tap (only needed once)
brew tap scttfrdmn/cloudworkstation

# Install CloudWorkstation
brew install cloudworkstation
```

## Testing the Formula Locally

To test the formula locally before releasing:

```bash
# Install from the local formula file
brew install --build-from-source ./packaging/homebrew/cloudworkstation.rb

# Test installation from the tap
brew install scttfrdmn/cloudworkstation/cloudworkstation
```

## Updating the Formula Manually

The formula is updated automatically on release, but you can manually update it:

1. Build the release archives for all platforms:
   ```bash
   make release
   ```

2. Run the update script:
   ```bash
   ./scripts/update_homebrew.sh v0.4.3 ./dist/v0.4.3
   ```

3. Commit and push the updated formula to the tap repository:
   ```bash
   cp packaging/homebrew/cloudworkstation.rb /path/to/homebrew-cloudworkstation/Formula/
   cd /path/to/homebrew-cloudworkstation
   git add Formula/cloudworkstation.rb
   git commit -m "Update formula for v0.4.3"
   git push
   ```

## Formula Structure

The CloudWorkstation formula includes:

- **Versioning**: The formula automatically detects the latest version from GitHub releases
- **Architecture-specific builds**: Different binaries for macOS/Linux and arm64/amd64
- **Dependencies**: Go is required to build from source
- **Completion scripts**: Bash, Zsh, and Fish completion scripts
- **Manual pages**: Installation of man pages
- **Configuration**: Setup of default configuration

## CI Integration

The formula is automatically updated by the GitHub Actions workflow when a new release is created. The workflow:

1. Downloads the release artifacts
2. Calculates SHA256 checksums
3. Updates the formula with new version and checksums
4. Commits and pushes the updated formula to the tap repository

## Troubleshooting

Common issues:

- **Missing SHA256 checksums**: Ensure the release artifacts are properly uploaded
- **Formula audit failures**: Run `brew audit --strict cloudworkstation.rb` to check for issues
- **Installation failures**: Check dependencies and path issues

For audit failures, use:
```bash
brew audit --strict --online packaging/homebrew/cloudworkstation.rb
```