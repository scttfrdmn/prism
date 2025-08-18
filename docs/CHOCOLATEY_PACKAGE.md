# CloudWorkstation Chocolatey Package

This document describes how to set up and maintain the Chocolatey package for CloudWorkstation.

## Overview

[Chocolatey](https://chocolatey.org/) is a package manager for Windows that simplifies software installation. The CloudWorkstation Chocolatey package allows Windows users to easily install and update the application.

## Package Structure

The CloudWorkstation Chocolatey package is structured as follows:

```
packaging/chocolatey/
├── cloudworkstation.nuspec      # Package metadata
└── tools/
    ├── chocolateyinstall.ps1    # Installation script
    └── chocolateyuninstall.ps1  # Uninstallation script
```

## Package Elements

### Package Specification (cloudworkstation.nuspec)

The `.nuspec` file contains metadata about the package:

- Package identifier
- Version information
- Author details
- Project URLs
- Description and summary
- Tags for searchability
- Dependencies (if any)

### Installation Script (chocolateyinstall.ps1)

The installation script handles:

- Downloading the CloudWorkstation binary for Windows
- Verifying checksums for security
- Creating binary shims
- Setting up the configuration directory
- Displaying post-installation instructions

### Uninstallation Script (chocolateyuninstall.ps1)

The uninstallation script:

- Removes binary shims
- Preserves user configuration

## Updating the Package

The package is updated automatically by the GitHub Actions workflow when a new release is created. This process:

1. Calculates new checksums for the Windows binary
2. Updates version information in the nuspec file
3. Updates the download URL and checksum in the installation script
4. Creates a new Chocolatey package
5. Pushes it to the Chocolatey repository

To manually update the package:

```powershell
# From the project root
.\scripts\update_chocolatey.ps1 -Version "v0.4.2" -ReleaseDir ".\dist\v0.4.2"
```

## Testing the Package Locally

To test the package locally:

```powershell
# Create the package
choco pack .\packaging\chocolatey\cloudworkstation.nuspec

# Install the package locally
choco install cloudworkstation -s . -y

# Test the installation
cws --version

# Uninstall
choco uninstall cloudworkstation -y
```

## Publishing to Chocolatey.org

Once tested, the package can be published to [Chocolatey.org](https://chocolatey.org/):

```powershell
# Ensure you have a Chocolatey API key
choco apikey -k <your-api-key> -s https://push.chocolatey.org/

# Push the package
choco push cloudworkstation.0.4.2.nupkg -s https://push.chocolatey.org/
```

Note: The first submission to Chocolatey.org requires manual approval, which may take 1-2 days. Subsequent updates are usually approved much faster.

## Continuous Integration

The GitHub Actions workflow `.github/workflows/chocolatey-update.yml` automates the package update process. The workflow:

1. Triggers when a new release is published
2. Downloads the release artifacts
3. Updates the package with new version information and checksums
4. Creates and submits the package

### Required Secrets

To enable CI automation, add the following secret to your GitHub repository:

- `CHOCOLATEY_API_KEY`: Your Chocolatey API key for package submission

## Troubleshooting

Common issues:

- **Package validation failures**: Use `choco pack --debug` to get detailed errors
- **Installation failures**: Check that the binary SHA256 matches the expected checksum
- **Shim issues**: Verify that the binary name in the installation script matches the actual binary

## Best Practices

- Always test packages locally before publishing
- Use explicit versioning that matches the application version
- Keep descriptions concise and focused on the software's purpose
- Include clear installation and usage instructions
- Include version changes in the package description