# Changelog

All notable changes to CloudWorkstation will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Multi-region AMI builder support
  - Region validation and error handling
  - Cross-region AMI copying functionality
  - Region-specific configuration system
  - Centralized version management package
  - Security group parameter support
  - Helper scripts for AMI building
  - Integration testing with LocalStack
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

### Improved
- Test coverage from basic unit tests to production-ready testing strategy
- AWS package coverage: 48.3% → 49.5% with comprehensive helper function tests
- Daemon package coverage: 16.4% → 27.8% with extensive HTTP handler tests
- Overall testing reliability and maintainability
- Error handling robustness across all packages
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