# Prism Testing Plan

**Goal**: Establish comprehensive testing strategy with functional coverage and Go Report Card A+ grade compliance.

**Philosophy**: "Functional testing as necessary, not tests for testing sake" - focus on actual behavior validation, not implementation details.

---

## Testing Strategy Overview

### Phase 1: Fix Existing Tests (Priority: CRITICAL)
**Status**: 🟡 IN PROGRESS - Major progress made, remaining issues identified
**Estimated Time**: 2-4 hours (50% complete)
**Progress**: CLI mock client fixed, TUI mocks need updates

#### Completed
- ✅ **Fixed GetCostTrends Missing Method**
  - Added to: `/pkg/api/mock/mock_client.go` (main mock)
  - Added to: `/internal/cli/mock_api_client.go` (CLI mock)
  - Test Suite Status: 27 passing, 3 failing

#### Remaining Issues
1. **TUI Model Test Mocks Need Update** (Priority: HIGH)
   - Missing method: `ApplyRightsizingRecommendation(ctx context.Context, instanceName string) error`
   - Files affected:
     - `/internal/tui/models/instances_test.go` - mockAPIClient
     - `/internal/tui/models/dashboard_test.go` - mockAPIClientDashboard
     - `/internal/tui/models/commands_test.go` - references mockAPIClient
     - Additional test files with local mocks
   - Note: This is a stub method that returns nil (see `/internal/tui/api/client.go:ApplyRightsizingRecommendation`)
   - Fix: Add stub method to each mock client:
     ```go
     func (m *mockAPIClient) ApplyRightsizingRecommendation(ctx context.Context, instanceName string) error {
         return nil
     }
     ```

2. **CLI Test Failures** (Priority: MEDIUM - functional tests, not compilation)
   - 7 test failures in `/internal/cli/...`
   - Tests compile and run, but assertions fail
   - Example failures: TestSimplified_AvailableCommands, TestScalingCommands_Rightsizing
   - These are behavioral test failures, not interface issues

3. **Research Package Test** (Priority: LOW - timing issue)
   - 1 test failure in `/pkg/research/manager_test.go`
   - Appears to be timing-related (time.Time comparison failure)
   - Not blocking other work

#### Tasks
- [x] Update main MockClient with GetCostTrends
- [x] Update CLI MockAPIClient with GetCostTrends
- [ ] Add ApplyRightsizingRecommendation to TUI mock clients
- [ ] Fix behavioral test failures in CLI package
- [ ] Fix timing issue in research package test
- [ ] Verify `go test ./...` passes with zero errors
- [ ] Document mock update process for future additions

#### Test Suite Summary
- **27 passing** test suites (88% pass rate)
- **3 failing** test suites:
  1. `internal/cli` - 7 behavioral test failures
  2. `internal/tui/models` - compilation error (missing mock method)
  3. `pkg/research` - 1 timing-related test failure

---

### Phase 2: Go Report Card A+ Compliance (Priority: HIGH)
**Status**: 🟡 READY - Can begin after Phase 1 complete
**Estimated Time**: 2-3 hours
**Tool**: `/Users/scttfrdmn/go/bin/goreportcard-cli`

#### Go Report Card Criteria
1. **gofmt** - Code formatting compliance
2. **go vet** - Static analysis for common issues
3. **gocyclo** - Cyclomatic complexity (max 15 recommended)
4. **golint** - Coding style suggestions
5. **ineffassign** - Ineffectual assignments
6. **misspell** - Spelling errors
7. **Test Coverage** - Percentage of code covered by tests

#### Tasks
- [ ] Run: `goreportcard-cli -v .`
- [ ] Fix any gofmt violations: `gofmt -w .`
- [ ] Fix any go vet issues: `go vet ./...`
- [ ] Review gocyclo complexity: `gocyclo -over 15 .`
- [ ] Fix golint suggestions: `golint ./...`
- [ ] Fix ineffassign issues: `ineffassign ./...`
- [ ] Fix misspell errors: `misspell -w .`
- [ ] Re-run goreportcard-cli until A+ achieved

---

### Phase 3: Functional Test Coverage - Backend (Priority: HIGH)
**Status**: 🟡 READY - Can begin after Phase 1 complete
**Estimated Time**: 4-6 hours
**Focus**: Test actual behavior, not implementation details

#### New Features Requiring Tests (Sessions 10-12)

##### 1. Rightsizing System
**Files to Test**:
- `/pkg/daemon/rightsizing_handlers.go` - API endpoints
- Handler logic for recommendations and application

**Test Scenarios** (Functional):
```go
// Test: GetRightsizingRecommendations returns valid data
// Test: ApplyRightsizingRecommendation modifies instance
// Test: GetRightsizingStats calculates savings correctly
// Test: Error handling when instance not found
```

##### 2. Policy Framework
**Files to Test**:
- `/pkg/daemon/policy_handlers.go` - API endpoints
- Policy enforcement and assignment logic

**Test Scenarios** (Functional):
```go
// Test: Policy enforcement can be enabled/disabled
// Test: Template access check respects assigned policies
// Test: Policy set assignment persists correctly
// Test: Error handling for invalid policy names
```

##### 3. Marketplace System
**Files to Test**:
- `/pkg/daemon/marketplace_handlers.go` - API endpoints
- Template installation and validation

**Test Scenarios** (Functional):
```go
// Test: Template search returns filtered results
// Test: Template installation succeeds with valid template
// Test: Category filtering works correctly
// Test: Error handling for duplicate installations
```

##### 4. Idle Detection System
**Files to Test**:
- `/pkg/daemon/idle_handlers.go` - API endpoints
- Idle policy evaluation logic

**Test Scenarios** (Functional):
```go
// Test: Idle policies list returns all configured policies
// Test: Instance schedules track idle time correctly
// Test: Policy application respects thresholds
// Test: Hibernation action triggered when conditions met
```

##### 5. Budget Management (Session 10)
**Files to Test**:
- `/pkg/daemon/budget_handlers.go` - API endpoints
- Cost calculation and alert logic

**Test Scenarios** (Functional):
```go
// Test: Budget alerts trigger at correct thresholds
// Test: Cost trends calculate daily/weekly/monthly correctly
// Test: Budget forecasting uses historical data
// Test: Error handling for negative budgets
```

##### 6. AMI Management (Session 11)
**Files to Test**:
- `/pkg/daemon/ami_handlers.go` - API endpoints
- AMI build and regional management

**Test Scenarios** (Functional):
```go
// Test: AMI build creates image successfully
// Test: Regional AMI list shows correct availability
// Test: AMI copy to region succeeds
// Test: Error handling for invalid AMI IDs
```

#### Testing Approach
- **Unit Tests**: Test individual handler functions with mocks
- **Table-Driven Tests**: Test multiple scenarios efficiently
- **Error Cases**: Test all error paths (not just happy paths)
- **Boundary Conditions**: Test edge cases (empty lists, max values, etc.)

#### Tasks
- [ ] Add tests for rightsizing handlers (4-6 test functions)
- [ ] Add tests for policy handlers (5-7 test functions)
- [ ] Add tests for marketplace handlers (4-6 test functions)
- [ ] Add tests for idle detection handlers (4-6 test functions)
- [ ] Add tests for budget handlers (5-7 test functions)
- [ ] Add tests for AMI handlers (4-6 test functions)
- [ ] Verify test coverage: `go test -cover ./...`

---

### Phase 4: TypeScript/React Testing (Priority: MEDIUM)
**Status**: 🟢 PLANNED - Can begin after Phase 3
**Estimated Time**: 3-5 hours
**Framework**: Jest + React Testing Library

#### Setup Required
```bash
cd cmd/cws-gui/frontend
npm install --save-dev @testing-library/react @testing-library/jest-dom jest @types/jest
```

#### Components Requiring Tests (Sessions 10-12)

##### 1. RightsizingView Component
**Test Scenarios** (Functional):
```typescript
// Test: Renders recommendations table with data
// Test: Apply recommendation opens confirmation modal
// Test: Tabs switch between recommendations and savings
// Test: Details view displays when recommendation clicked
```

##### 2. PolicyView Component
**Test Scenarios** (Functional):
```typescript
// Test: Enforcement toggle updates state
// Test: Template access checker displays results
// Test: Policy assignment modal works correctly
```

##### 3. MarketplaceView Component
**Test Scenarios** (Functional):
```typescript
// Test: Search filters templates by name/description/tags
// Test: Category filter narrows results
// Test: Install button triggers installation workflow
// Test: Template cards display rating stars correctly
```

##### 4. IdleDetectionView Component
**Test Scenarios** (Functional):
```typescript
// Test: Policy details view shows threshold information
// Test: Tabs switch between policies and schedules
// Test: Cost savings display shows correct percentage
```

##### 5. LogsView Component
**Test Scenarios** (Functional):
```typescript
// Test: Log lines display in scrollable container
// Test: Copy to clipboard works
// Test: Download logs creates .log file
// Test: Log type selection triggers refresh
```

##### 6. BudgetView Component (Session 10)
**Test Scenarios** (Functional):
```typescript
// Test: Budget progress bars show correct percentages
// Test: Cost trends chart displays data
// Test: Alert configuration saves correctly
```

##### 7. AMIManagementView Component (Session 11)
**Test Scenarios** (Functional):
```typescript
// Test: AMI builds table shows status correctly
// Test: Regional availability displays per-region status
// Test: Build AMI modal validates input
```

#### Testing Approach
- **Component Tests**: Test user interactions and state changes
- **Integration Tests**: Test API calls and data flow
- **Accessibility Tests**: Ensure WCAG AA compliance maintained
- **Visual Regression**: Optional - screenshot comparison for UI changes

#### Tasks
- [ ] Set up Jest + React Testing Library
- [ ] Configure test environment and utilities
- [ ] Add tests for RightsizingView (4-5 test cases)
- [ ] Add tests for PolicyView (3-4 test cases)
- [ ] Add tests for MarketplaceView (4-5 test cases)
- [ ] Add tests for IdleDetectionView (3-4 test cases)
- [ ] Add tests for LogsView (3-4 test cases)
- [ ] Add tests for BudgetView (4-5 test cases)
- [ ] Add tests for AMIManagementView (4-5 test cases)
- [ ] Run test suite: `npm test`

---

## Phase 5: Integration & E2E Testing (FUTURE)
**Status**: 🔵 FUTURE - Planned after Phase 1-4 complete
**Estimated Time**: TBD
**Scope**: AWS integration testing with real resources

### Planned Approach
- Test actual AWS API calls with test credentials
- Test instance launch/stop/hibernate workflows
- Test EFS/EBS volume creation and attachment
- Test template application end-to-end
- Test project budget enforcement with real costs

**Note**: This phase will be discussed separately after unit/functional testing complete.

---

## Test Coverage Goals

### Go Backend
- **Target**: 70%+ coverage for critical paths
- **Focus Areas**: API handlers, state management, AWS operations
- **Excluded**: Mock implementations, test utilities, generated code

### TypeScript Frontend
- **Target**: 60%+ coverage for components
- **Focus Areas**: User interactions, state management, API integration
- **Excluded**: Third-party components (Cloudscape), mock data

---

## Success Criteria

### Phase 1 Complete
- ✅ All existing tests pass: `go test ./...` returns zero errors
- ✅ MockAPIClient interface complete with all 50+ methods
- ✅ Clean build with no test-related compilation errors

### Phase 2 Complete
- ✅ Go Report Card grade: **A+**
- ✅ Zero gofmt violations
- ✅ Zero go vet warnings
- ✅ No cyclomatic complexity over 15
- ✅ Zero ineffassign issues
- ✅ Zero misspell errors

### Phase 3 Complete
- ✅ 25-35 new functional tests added for backend features
- ✅ All API handlers have test coverage
- ✅ Error paths tested for all new features
- ✅ Test coverage report shows 70%+ for new code

### Phase 4 Complete
- ✅ 25-30 new component tests added for GUI features
- ✅ Critical user workflows tested
- ✅ All new views have test coverage
- ✅ Test suite runs successfully: `npm test`

---

## Implementation Strategy

### Day 1: Fix Foundation (Phase 1)
1. Update MockAPIClient interface completely
2. Fix all broken tests
3. Verify clean `go test ./...` run

### Day 2: Quality Compliance (Phase 2)
1. Run goreportcard-cli analysis
2. Fix all identified issues systematically
3. Achieve A+ grade

### Day 3-4: Backend Testing (Phase 3)
1. Add tests for rightsizing + policy systems
2. Add tests for marketplace + idle detection
3. Add tests for budget + AMI management

### Day 5: Frontend Testing (Phase 4)
1. Set up Jest + React Testing Library
2. Add tests for all 7 new GUI components
3. Verify test suite passes

---

## Maintenance Strategy

### Preventing Future Test Breakage
1. **Mock Update Process**: Document how to update mocks when adding new API methods
2. **Test Review**: Require test updates with all API changes
3. **CI Integration**: Run tests on every commit (future GitHub Actions)
4. **Coverage Tracking**: Monitor test coverage trends over time

### Test Quality Guidelines
1. **Functional Focus**: Test behavior, not implementation
2. **Clear Names**: Test names describe what is being tested
3. **Independent Tests**: Tests don't depend on each other
4. **Fast Execution**: Keep test suite under 30 seconds
5. **Deterministic**: Tests produce same result every time

---

## Timeline

| Phase | Duration | Dependencies | Status |
|-------|----------|--------------|--------|
| Phase 1: Fix Existing Tests | 2-4 hours | None | 🔴 CRITICAL |
| Phase 2: Go Report Card A+ | 2-3 hours | Phase 1 | 🟡 READY |
| Phase 3: Backend Tests | 4-6 hours | Phase 1 | 🟡 READY |
| Phase 4: Frontend Tests | 3-5 hours | Phase 3 | 🟢 PLANNED |
| Phase 5: Integration Tests | TBD | Phase 1-4 | 🔵 FUTURE |

**Total Estimated Time**: 11-18 hours (Phases 1-4)

---

## Next Steps

1. **Immediate**: Begin Phase 1 - Update MockAPIClient interface
2. **Review**: Confirm testing approach and priorities
3. **Execute**: Work through phases systematically
4. **Report**: Track progress and test coverage metrics
5. **Iterate**: Adjust based on findings during implementation

---

**Document Version**: 1.0
**Created**: October 2025 (Session 12)
**Last Updated**: October 2025
**Status**: Ready for implementation
