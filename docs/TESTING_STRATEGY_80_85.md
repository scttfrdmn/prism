# CloudWorkstation Testing Strategy: 80-85% Coverage Target

## Executive Summary

**Current State**: 20.5% overall coverage with 60 test files and critical gaps
**Target State**: 80-85% coverage focused on business-critical functionality
**Strategy**: Strategic testing investment prioritizing high-impact, maintainable tests

## Coverage Analysis & Strategic Priorities

### Tier 1: Critical Infrastructure (Target: 80%+)
**Priority**: HIGHEST - Core functionality that breaks the entire system

#### `pkg/aws` (15.3% → 80%)
**Impact**: HIGH - AWS operations are core to entire platform
**Current Gaps**: AMI caching, cost analysis, instance management completely untested
```go
// Critical untested functions:
- NewAMICache, GetAMI, SetAMI (0% coverage)
- CalculateAMICost (0% coverage)
- Instance lifecycle operations
```
**Test Strategy**: Mock AWS SDK, focus on business logic and error handling

#### `pkg/templates` (14.2% → 75%)
**Impact**: HIGH - Template system is core differentiator
**Current Gaps**: Template inheritance, validation, script generation
```go
// Critical untested functions:
- Template inheritance merging logic
- Validation rules (8+ validation types)
- Script generation for different package managers
```
**Test Strategy**: Template fixture files, validation edge cases, inheritance scenarios

#### `pkg/daemon` (20.1% → 75%)
**Impact**: HIGH - Daemon is the API backbone
**Current Gaps**: HTTP handlers, API routing, request processing
**Test Strategy**: HTTP test server, integration tests for critical endpoints

### Tier 2: Business Logic (Target: 70-80%)

#### `pkg/project` (44.4% → 80%)
**Impact**: MEDIUM-HIGH - Enterprise features, budget management
**Current Issues**: Failing budget tracker test needs immediate fix
**Test Strategy**: Fix existing test failures, add budget calculation edge cases

#### `internal/cli` (23.5% → 70%)
**Impact**: MEDIUM-HIGH - Primary user interface
**Current Gaps**: Command parsing, API integration, error handling
**Test Strategy**: CLI integration tests with mock API client

### Tier 3: Supporting Systems (Target: 60-70%)

#### Interface Layers (0% → 60%)
- `internal/tui`: Terminal interface testing
- GUI: Skip - UI testing complex for solo dev, focus on business logic

#### Feature Modules (0% → 60%)
- `pkg/idle`: Hibernation policies
- `pkg/marketplace`: Template discovery
- `pkg/research`: User management

## Strategic Testing Roadmap

### Phase 1: Foundation (Week 1)
1. **Fix Failing Tests**: Resolve `pkg/project` budget tracker test
2. **AWS Core**: Test critical AWS operations with mocked SDK
3. **Template Core**: Test inheritance system and validation

### Phase 2: Integration (Week 2)
4. **CLI Integration**: Command execution with mock backends
5. **Daemon API**: HTTP handler testing with test servers
6. **Project Management**: Enterprise features with edge cases

### Phase 3: Feature Completion (Week 3)
7. **Hibernation System**: Idle detection and policy testing
8. **Research Users**: User lifecycle management
9. **Marketplace**: Template discovery and installation

## Testing Principles for Solo Development

### 1. **High-Impact Focus**
- Test business logic, not infrastructure
- Mock external dependencies (AWS SDK, file system)
- Focus on edge cases that cause user-facing failures

### 2. **Maintainable Test Architecture**
```go
// Good: Business logic test
func TestTemplateInheritance(t *testing.T) {
    base := createBaseTemplate()
    child := createChildTemplate()
    merged := mergeTemplates(base, child)
    assert.Equal(t, expectedResult, merged)
}

// Avoid: Infrastructure-heavy test
func TestAWSRealAPICall(t *testing.T) {
    // Skip - too brittle for solo dev
}
```

### 3. **Test Categories**
- **Unit Tests**: Business logic, calculations, transformations (70% of tests)
- **Integration Tests**: API flows, command execution (20% of tests)
- **End-to-End**: Critical user journeys only (10% of tests)

### 4. **Coverage Quality Metrics**
```bash
# Target metrics for 80-85% coverage:
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1
# Expected: total: (statements) 82.3%

# Package-specific targets:
pkg/aws:        80%+ (critical infrastructure)
pkg/templates:  75%+ (core differentiator)
pkg/daemon:     75%+ (API backbone)
pkg/project:    80%+ (enterprise features)
internal/cli:   70%+ (user interface)
```

## Implementation Strategy

### Mock Strategy
```go
// AWS SDK mocking
type MockAWSManager struct{}
func (m *MockAWSManager) LaunchInstance(...) (*Instance, error) {
    return mockInstance, nil
}

// API client mocking (already exists)
type MockAPIClient struct{}
func (m *MockAPIClient) GetTemplates() ([]Template, error) {
    return testTemplates, nil
}
```

### Test Fixtures
- Template YAML files for inheritance testing
- Mock AWS responses for pricing/AMI data
- Sample project configurations for budget testing

### CI/CD Integration
```bash
# Pre-commit hook (already exists)
make test-coverage
# Fail if coverage drops below 80%
```

## Success Criteria

### Coverage Targets by Package
- **Overall Project**: 80-85% (from current 20.5%)
- **Critical Path**: 80%+ coverage on Tier 1 packages
- **Business Logic**: 70%+ coverage on Tier 2 packages
- **Supporting**: 60%+ coverage on Tier 3 packages

### Quality Gates
- ✅ All existing tests pass consistently
- ✅ No regressions in critical functionality
- ✅ Test execution time under 30 seconds (solo dev workflow)
- ✅ Coverage reports integrated into development workflow

### Timeline
- **Week 1**: Foundation tests (Tier 1 packages to 80%)
- **Week 2**: Integration tests (Tier 2 packages to 70%)
- **Week 3**: Feature completion (Tier 3 packages to 60%)
- **Target**: 80-85% overall coverage within 3 weeks

This strategic approach balances thorough testing with solo development efficiency, ensuring high-quality code without excessive maintenance overhead.