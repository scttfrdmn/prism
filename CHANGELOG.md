# Changelog

All notable changes to CloudWorkstation will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2024-07-11

This release focuses on two major improvements: the AMI Builder System and Terminal User Interface (TUI). These enhancements significantly improve the user experience and performance of CloudWorkstation.

### Added
- **AMI Builder System** (replacing UserData scripts)
  - Automated AMI building with GitHub Actions
  - YAML template format for AMI definitions
  - Multi-region and multi-architecture support
  - JSON Schema validation for templates
  - Template registry for version management
  - Registry lookup API with fallback mechanism
  - Template conversion utilities
  - Comprehensive documentation for AMI Builder

- **Terminal User Interface (TUI)**
  - Dashboard view for system overview
  - Instances view with management capabilities
  - Templates view for environment selection
  - Storage management view for volumes
  - Settings page with theme switching
  - Notification system for asynchronous operations
  - Dark/light theme support
  - Search functionality across all list views
  - Customized TabBar component
  
- **Template Library Expansion**
  - r-research: R + RStudio Server + tidyverse
  - python-research: Python + Jupyter + data science
  - desktop-research: Ubuntu Desktop + NICE DCV
  - basic-ubuntu: Plain Ubuntu 22.04
  - neuroimaging: FSL + AFNI + ANTs
  - bioinformatics: BWA + GATK + Samtools
  - gis-research: QGIS + GRASS + PostGIS

### Improved
- **Performance**
  - Reduced instance launch time from 10+ minutes to under 60 seconds
  - More reliable environment setup with pre-built AMIs
  - Consistent software configuration across launches
  
- **User Experience**
  - Streamlined terminal UI with intuitive navigation
  - Visual feedback for all operations
  - Improved error messages and troubleshooting
  - Progressive disclosure of advanced features

- **Architecture**
  - Enhanced distributed architecture with daemon and client
  - Complete API integration between components
  - Clean separation of concerns

### Technical
- Go 1.24+ compatibility
- Bubble Tea framework for TUI
- GitHub Actions for CI/CD
- AWS SSM Parameter Store for registry
- Comprehensive testing framework
- Enhanced documentation

## [0.1.0-alpha] - 2023-07-10

Initial alpha release with core functionality and testing framework. This release focuses on establishing a solid foundation with distributed architecture and comprehensive testing.

### Key Features
- Distributed client-server architecture with REST API
- Complete AWS integration with instance management
- Template-based workstation provisioning
- EFS and EBS volume management
- Multi-region support
- Comprehensive testing framework
- Desktop environment support with NICE DCV

## [0.1.1] - 2023-07-08 (Development)

### Added
- Multi-region AMI builder support
  - Region validation and error handling
  - Cross-region AMI copying functionality
  - Region-specific configuration system
  - Centralized version management package
  - Security group parameter support
  - Helper scripts for AMI building
  - Integration testing with LocalStack
- Comprehensive error handling system for AMI builder
  - Typed errors with context information (ValidationError, NetworkError, etc.)
  - Detailed error reporting with troubleshooting suggestions
  - Retryable error detection
  - Error context propagation with related metadata
  - Clear user-facing error messages
- Complete template management system
  - Template import/export functionality
  - Schema validation with JSON Schema
  - Multiple source formats (file, URL, GitHub)
  - Template sharing through registry
  - Builder pattern for template creation and modification
  - Rich CLI interface for template operations
  - Template validation and verification
- Comprehensive testing infrastructure with LocalStack integration
- Docker-based AWS service emulation for integration testing
- Advanced test coverage analysis and reporting
- Integration tests for complete AWS operations (EC2, EFS, EBS)
- Enhanced unit test coverage for all core packages
- Docker Compose configuration for testing environment
- Comprehensive testing documentation (TESTING.md)
- Coverage targets: 85% AWS, 80% daemon, 75% API, 75% overall
- Build tags for separating unit and integration tests
- Individual package testing capabilities
- Error handling tests for AWS operations
- Regional pricing tests for 13+ AWS regions
- Discount combination scenario testing
- Template validation across architectures
- HTTP endpoint comprehensive testing
- Instance lifecycle testing (launch, start, stop, delete)
- Volume operations testing (EFS, EBS creation/deletion)
- Storage attachment/detachment testing
- Multi-instance management testing
- Standardized template repository with yaml templates
- End-to-end testing framework with real AWS
- Desktop environment support with NICE DCV
- Templates for various research domains:
  - python-research: Python with scientific and ML libraries
  - neuroimaging: FSL, AFNI, ANTs and MRtrix
  - bioinformatics: BWA, GATK, Samtools and R Bioconductor
  - gis-research: QGIS, GRASS, PostGIS and geospatial libraries
  - desktop-research: Ubuntu Desktop with NICE DCV
- Documentation for template format

### Improved
- Test coverage from basic unit tests to production-ready testing strategy
- AWS package coverage: 48.3% → 49.5% with comprehensive helper function tests
- Daemon package coverage: 16.4% → 27.8% with extensive HTTP handler tests
- Overall testing reliability and maintainability
- Error handling robustness across all packages
  - Structured error types for better error classification
  - Context-rich error information for debugging
  - Consistent error handling patterns throughout AMI builder
  - Retryable vs. non-retryable error distinction
- Documentation quality with detailed testing guide

### Technical
- LocalStack 3.0 integration for AWS service emulation
- Build tag system for test categorization (`// +build integration`)
- Docker Compose test environment configuration
- Coverage analysis tooling and HTML report generation
- Makefile targets for test automation
- CI/CD ready testing infrastructure

## [0.1.0] - Initial Release

### Added
- CloudWorkstation MVP with monolithic architecture
- Basic CLI interface for instance management
- Hard-coded templates (R, Python, Ubuntu)
- JSON state file management
- AWS EC2 integration
- Simple cost estimation
- Instance launch, list, connect, stop, delete operations
- Basic error handling
- Cross-platform support (macOS, Linux, Windows)

### Architecture
- Single main.go file implementation
- Direct AWS SDK calls
- Local JSON state persistence
- Template-based instance provisioning
- Cost-aware resource management