# Changelog

All notable changes to Prism will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.4] - 2025-10-18

### Added
- **Universal Version System**: Dynamic OS version selection at launch time
  - `--version` flag for specifying OS versions (e.g., `--version 24.04`, `--version 22.04`)
  - Support for version aliases: `latest`, `lts`, `previous-lts`
  - 4-level hierarchical AMI structure: distro → version → region → architecture
  - AWS SSM Parameter Store integration for Ubuntu, Amazon Linux, Debian
  - Static fallback AMIs for Rocky Linux, RHEL, Alpine
- **AMI Freshness Checking**: Proactive validation of static AMI IDs
  - `prism ami check-freshness` command to validate AMI mappings
  - Automatic detection of outdated AMIs against latest SSM values
  - Clear reporting with recommended update actions
  - Support for all distributions (SSM-backed and static)
- **Enhanced AMI Discovery**: Intelligent AMI resolution with automatic updates
  - Daemon startup warm-up with bulk AMI discovery
  - Hybrid discovery: SSM Parameter Store with static fallback
  - Regional AMI caching for improved performance

### Enhanced
- **Version Resolution**: 3-tier priority system (User → Template → Default)
- **Template System**: Version constraints in template dependencies
- **Documentation**: Complete VERSION_SYSTEM_IMPLEMENTATION.md guide

### Technical
- Added `pkg/aws/ami_discovery.go` (416 lines) - AMI discovery and freshness checking
- Added `pkg/templates/resolver.go` (267 lines) - Version resolution and aliases
- Added `pkg/templates/dependencies.go` (300 lines) - Dependency resolution
- Enhanced `pkg/templates/parser.go` with hierarchical AMI structure
- Added `Version` field to `LaunchRequest` for version specification
- Integrated AMI discovery into daemon initialization
- Added REST API endpoint `/api/v1/ami/check-freshness`
- Added CLI commands: `prism ami check-freshness`

### Benefits
- **No Template Explosion**: Single template supports multiple OS versions
- **Always Current**: SSM integration provides latest AMIs automatically
- **Version Flexibility**: Choose any supported OS version at launch time
- **Proactive Maintenance**: Monthly freshness checks identify outdated AMIs
- **Clear Communication**: Users know exactly which version they're getting

### Supported Distributions
- Ubuntu: 24.04, 22.04, 20.04 (SSM-backed)
- Rocky Linux: 10, 9 (static fallback)
- Amazon Linux: 2023, 2 (SSM-backed)
- Debian: 12 (SSM-backed)
- RHEL: 9 (static fallback)
- Alpine: 3.20 (static fallback)

## [0.5.3] - 2025-10-17

### Development Workflow
- **Simplified Git Hooks**: Streamlined pre-commit checks to run in < 5 seconds (down from 2-5 minutes)
  - Fast auto-formatting only (gofmt, goimports, go mod tidy)
  - Heavy checks (lint, tests) moved to explicit make targets for pre-push validation
- **Enhanced Makefile**: Go Report Card linting integration with comprehensive quality tools
  - gofmt, goimports, go vet, gocyclo, misspell, staticcheck, golangci-lint
  - Quick Start workflow documentation for new developers
- **Documentation Cleanup**: Organized 20+ historical documents into structured archive
  - Created docs/archive/ with planning/, implementation/, deprecated/ subdirectories
  - Preserved historical context while cleaning main docs/ directory

### Quality Improvements
- **Cost Display Precision**: Enhanced cost output from 3 to 4 decimal places for sub-cent accuracy
- **Version Synchronization**: Fixed Makefile version mismatch (aligned with runtime version)

### Infrastructure
- **Build System**: Maintained zero compilation errors for production binaries
- **Testing**: Core functionality verification with automated smoke tests
- **GoReleaser Integration**: Complete distribution automation with multi-platform support
  - Automated builds for Linux, macOS, Windows (AMD64 + ARM64)
  - Homebrew tap integration (scttfrdmn/homebrew-tap)
  - Scoop bucket support for Windows package management
  - Debian/RPM/APK packages for Linux distributions
  - Docker multi-arch images with manifest support
  - Makefile targets for local testing (snapshot mode)
  - Simplified Homebrew formula with auto-starting daemon messaging

## [0.4.1] - 2025-08-08

### Critical Bug Fixes
- **GUI Content Display**: Fixed blank white areas in Dashboard, Instances, Templates, and Storage sections
- **Version Verification**: Fixed daemon version reporting (was hardcoded "0.1.0", now reports actual version)
- **CLI Version Panic**: Fixed crash when GitCommit string shorter than 8 characters  
- **Storage API Mismatch**: Fixed JSON unmarshaling errors in EFS/EBS volume endpoints
- **GUI Threading**: Eliminated threading warnings and improved stability
- **Daemon Version Checking**: Added proper version verification after daemon startup

### User Experience Improvements
- **System Tray Integration**: Enhanced window management and data refresh when shown from tray
- **Navigation Highlighting**: Fixed sidebar navigation button highlighting without rebuilding
- **Connection Status**: Improved daemon connection status detection with proper timeouts
- **Error Messages**: More helpful and actionable error messages throughout the application

### Documentation
- **Major Cleanup**: Organized 50+ scattered documentation files into clean structure
  - Root: 14 essential project files
  - docs/: 41 current documentation files organized by category  
  - docs/archive/: 42 historical files properly archived
- **Updated Navigation**: Comprehensive documentation index with clear categorization
- **User Guides**: Improved organization of user-facing documentation

### Technical Improvements  
- **API Consistency**: Storage and volume endpoints now return arrays instead of maps
- **Version System**: Robust version verification across CLI and GUI interfaces
- **Build System**: Clean compilation with zero errors across all platforms
- **Homebrew Integration**: Complete end-to-end Homebrew installation validation

## [0.4.0] - 2025-07-15

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

## [0.4.3] - 2025-08-19

### Added
- Template inheritance system with multi-level stacking support
- Comprehensive template validation with 8+ validation rules
- Enhanced build system with cross-compilation fixes
- Complete hibernation ecosystem with cost optimization
- Idle detection system with automated hibernation policies
- Professional GUI interface with system tray integration
- CLI version output consistency with daemon formatting
- EFS multi-instance sharing with cross-template collaboration

### Enhanced
- Version synchronization across all components (CLI, daemon, GUI)
- Cross-compilation support using existing crosscompile build tags
- Template system with stackable inheritance (e.g., Rocky9 + Conda)
- Hibernation policies with intelligent fallback to stop when unsupported
- Cost optimization with session-preserving hibernation capabilities
- GitHub release workflow with automated distribution packages
- Homebrew tap with complete installation testing cycle

### Fixed
- CLI version display format to match daemon professional output
- Cross-compilation keychain errors using platform-specific alternatives
- Template validation preventing invalid package managers and self-reference
- Mock API client version consistency in tests
- Version variable synchronization between Makefile and runtime
- Distribution package checksums and binary verification

### Documentation
- Updated all version references from 0.4.2 to 0.4.3
- Template inheritance and validation technical guides
- Hibernation ecosystem implementation documentation
- Complete release preparation and distribution strategy
- Homebrew tap setup and maintenance procedures
- Windows MSI installer comprehensive documentation

## [Unreleased] - 0.5.0 Multi-User System

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
- Batch device management for security administration
- Device registry integration for centralized control
- Multi-device revocation and validation tools

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
- BATCH_DEVICE_MANAGEMENT.md with device security documentation
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
- Integration with new Prism API context-aware methods
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