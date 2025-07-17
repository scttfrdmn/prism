# Changelog

All notable changes to CloudWorkstation will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.1] - 2025-07-15

### Added
- **Graphical User Interface (GUI)** - Point-and-click interface for easier use
  - System tray integration for desktop monitoring
  - Visual dashboard with instance status and costs
  - Template browser with visual cards and descriptions
  - Storage management with visual indicators
  - Dark and light themes support
- **Package manager distribution** for easier installation
  - Homebrew formula for macOS and Linux
  - Chocolatey package for Windows
  - Conda package for all platforms
- **Multi-architecture support**
  - AMD64 (Intel/AMD) for all platforms
  - ARM64 (Apple Silicon, AWS Graviton) for macOS and Linux
- **Multi-profile foundation** for the upcoming v0.4.2 features
  - Profile management package (`pkg/profile`)
  - Profile switching infrastructure
  - AWS credential provider integration
- **Complete API client with context support**
  - Context-aware API methods for proper timeouts
  - Improved error handling with context propagation
  - Full compatibility with both CLI and GUI clients

### Changed
- Updated API client interfaces to use context support
- Improved documentation with GUI User Guide
- Enhanced error handling with clear user feedback
- Updated build system for multi-architecture support
- Restructured package layout for better distribution

### Fixed
- Compatibility between CLI and GUI components
- API method signatures for proper context handling
- Build system for cross-platform package generation
- Documentation to reflect current features and installation methods

## [Unreleased] - 0.4.3 Secure Profiles

### Added
- Secure invitation system with device binding
- Cross-platform keychain integration for secure credential storage
- S3-based registry for tracking authorized devices
- Multi-level permissions model for invitation delegation
- Device management interface in GUI, TUI and CLI
- Administrator utilities for device management
- Batch invitation system for managing multiple invitations at once
- CSV import/export for bulk invitation management
- Concurrent invitation processing with worker pools

### Enhanced
- Profile management with security attributes
- GUI invitation dialog with device binding options
- TUI profile component with security indicators
- CLI invitation commands with security features

### Documentation
- SECURE_PROFILE_IMPLEMENTATION.md with technical details
- SECURE_INVITATION_ARCHITECTURE.md with design documentation
- ADMINISTRATOR_GUIDE.md with security management instructions
- BATCH_INVITATION_GUIDE.md with bulk invitation instructions
- Updated comments throughout the codebase

## [0.4.2] - 2025-07-16

### Added
- Multi-profile support for multiple AWS accounts
- Profile-aware client for state isolation
- Invitation-based profile sharing
- Profile switching in GUI, TUI and CLI

### Enhanced
- API client with context support
- Error handling with detailed context information
- Performance optimizations with connection pooling
- GUI interface with profile management

### Documentation
- Profile export/import documentation
- Multi-profile guide with technical details

## [Unreleased] - 0.4.0 Development

### Added
- Redesigned Terminal User Interface (TUI) for improved visual management
  - Dashboard view with instance status and cost monitoring
  - Template browser with detailed template information
  - Interactive instance management interface
  - System status monitoring and notifications
  - Visual storage and volume management
  - Keyboard shortcuts for common operations
- Integration with new CloudWorkstation API context-aware methods
- Consistent help system with keyboard shortcut reference
- Better terminal compatibility across platforms
- Tab-based navigation between sections
- Progressive disclosure of advanced features

### Changed
- Updated API client interface to use context support
- Improved TUI components with active/inactive state handling
- Enhanced error handling with clear user feedback
- Updated Bubbles and BubbleTea dependencies to latest versions
- More consistent user experience between CLI and TUI

### Fixed
- Fixed spinner rendering issues during API operations
- Improved terminal compatibility with various terminal emulators
- Better error messages for API connection failures