# Changelog

All notable changes to CloudWorkstation will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Research domain template expansion with 24 domains from AWS Research Wizard
- CloudSnooze-inspired idle detection system
- Multi-repository support with override capabilities
- Enhanced YAML template format with research domain extensions
- Domain-specific idle detection profiles
- Repository management commands (`cws repo`)
- Template dependency resolution across repositories
- Default repository structure at github.com/scttfrdmn/cloudworkstation-repository

### Changed
- Updated template organization with domain categories
- Enhanced documentation for all new features
- Improved cost management with idle detection
- Reorganized template structure with base, stacks, and domains

## [0.2.0] - 2024-07-10

### Added
- AMI Builder System for creating custom AMIs
- YAML template parser with JSON Schema validation
- GitHub Actions workflow for automated AMI builds
- Template registry for version management
- Multi-region support for AMIs
- TUI (Terminal User Interface) with BubbleTea framework
- TabBar component submitted as PR to charmbracelet/bubbles
- Integration tests between TUI and daemon
- User documentation for TUI features
- Documentation for AMI Builder system

### Changed
- Replaced hard-coded AMI IDs with registry lookup
- Enhanced error handling and validation
- Improved template system with dependency resolution
- Updated architecture documentation
- Reorganized code structure for better separation of concerns

### Fixed
- Template validation for edge cases
- API interface and registry type issues
- TUI component compilation issues
- Search functionality in TUI

## [0.1.0-alpha] - 2024-07-09

### Added
- Initial alpha release
- Split monolithic architecture into daemon + CLI client
- REST API backend with full endpoint coverage
- Thin CLI client maintaining identical UX
- Modular package structure ready for GUI
- Cross-platform build system with Makefile
- Complete API interface for all operations
- State management abstraction layer

### Changed
- Code structure from single main.go to modular packages
- Command line interface to use REST API

### Fixed
- JSON state file handling
- Error propagation between components