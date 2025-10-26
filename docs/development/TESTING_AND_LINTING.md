# Testing and Linting Guide for Prism

## Overview

Prism employs comprehensive testing and linting strategies to ensure code quality, reliability, and maintainability. This guide covers our testing infrastructure, linting configuration, and automated quality checks.

## Testing Infrastructure

### Test Coverage Summary

Current test coverage across packages:
- **pkg/pricing**: 97.2% ✅
- **pkg/security**: 58.3% 
- **pkg/profile/security**: 47.3%
- **pkg/project**: 45.6%
- **pkg/api/client**: 37.9%
- **pkg/api/errors**: 36.8%
- **pkg/profile**: 24.9%
- **pkg/aws**: 24.1%
- **pkg/daemon**: 24.0%

### Running Tests

#### Quick Unit Tests
```bash
# Run all tests with short flag (30s timeout)
go test -short -timeout 30s ./...

# Run tests for specific package
go test ./pkg/aws/...

# Run with coverage
go test -cover ./...
```

#### Comprehensive Tests
```bash
# Full test suite
make test

# With coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### GUI Testing
```bash
# Backend service tests
cd cmd/cws-gui
go test -v

# Frontend E2E tests (requires daemon running)
cd cmd/cws-gui/frontend
npm test:e2e
```

### Test Organization

```
├── pkg/
│   ├── aws/             # AWS service tests
│   ├── api/client/       # API client integration tests
│   ├── daemon/           # Daemon server tests
│   └── project/          # Project management tests
├── internal/cli/         # CLI command tests
└── cmd/cws-gui/
    ├── gui_test.go       # GUI backend tests
    └── frontend/tests/   # Frontend E2E tests
```

## Linting Configuration

### golangci-lint Setup

Install golangci-lint:
```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### Running Linters

```bash
# Run all linters
golangci-lint run ./...

# Run fast linters only
golangci-lint run --fast ./...

# Run specific linter
golangci-lint run --disable-all --enable errcheck ./...

# Fix issues automatically (where possible)
golangci-lint run --fix ./...
```

### Enabled Linters

#### Core Linters
- **errcheck**: Unchecked errors
- **gosimple**: Code simplification
- **govet**: Suspicious constructs
- **ineffassign**: Ineffectual assignments
- **staticcheck**: Static analysis
- **unused**: Unused code

#### Quality Linters
- **bodyclose**: HTTP response body closure
- **dupl**: Duplicate code detection
- **gocognit**: Cognitive complexity (max: 20)
- **gocyclo**: Cyclomatic complexity (max: 15)
- **gofmt**: Go formatting
- **goimports**: Import organization
- **gosec**: Security issues
- **misspell**: Spelling errors
- **revive**: Comprehensive style guide

### Common Linting Issues

#### Unchecked Errors
```go
// Bad
defer resp.Body.Close()

// Good
defer func() {
    if err := resp.Body.Close(); err != nil {
        log.Printf("Error closing response body: %v", err)
    }
}()
```

#### Resource Cleanup
```go
// Always check Close() errors
if err := conn.Close(); err != nil {
    return fmt.Errorf("failed to close connection: %w", err)
}
```

## Git Hooks

### Pre-commit Hook (Fast Checks)

Automatically runs on `git commit`:
- Go format check
- Fast linting (30s timeout)
- Quick build verification
- Unit tests
- TODO/FIXME detection

### Pre-push Hook (Comprehensive)

Automatically runs on `git push`:
- Full build test
- All Go tests (2m timeout)
- Comprehensive linting
- Frontend tests (if applicable)
- Integration tests

### Setup Git Hooks

```bash
# Configure git hooks
./scripts/setup-git-hooks.sh

# Bypass hooks in emergencies
git commit --no-verify
git push --no-verify
```

## Continuous Improvement

### Test Coverage Goals

Target coverage by package type:
- **Core packages** (aws, daemon): >70%
- **API packages**: >60%
- **Utility packages**: >80%
- **CLI/UI packages**: >40%

### Tracking Linting Issues

```bash
# Generate linting report
golangci-lint run --out-format json > lint-report.json

# Count issues by linter
golangci-lint run | grep -o '\[\w\+\]' | sort | uniq -c
```

### Performance Benchmarks

```bash
# Run benchmarks
go test -bench=. ./...

# Compare benchmarks
go test -bench=. -benchmem ./pkg/pricing > new.txt
benchcmp old.txt new.txt
```

## Best Practices

### Writing Tests

1. **Table-driven tests** for multiple scenarios
2. **Mock external dependencies** (AWS, HTTP)
3. **Test error conditions** explicitly
4. **Use `t.Parallel()** where appropriate
5. **Clean up resources** in defer/cleanup

### Code Quality

1. **Handle all errors** explicitly
2. **Close all resources** (files, connections)
3. **Avoid complex functions** (cognitive complexity < 20)
4. **Document exported functions**
5. **Use meaningful variable names**

### Security

1. **No hardcoded credentials**
2. **Validate all inputs**
3. **Use secure defaults**
4. **Audit command execution**
5. **Check TLS configurations**

## Troubleshooting

### Test Timeouts

```bash
# Increase timeout for slow tests
go test -timeout 5m ./...

# Skip integration tests
go test -short ./...
```

### Linting Performance

```bash
# Cache linting results
golangci-lint cache clean
golangci-lint run --build-tags=integration

# Run on changed files only
golangci-lint run --new-from-rev=HEAD~1
```

### Coverage Gaps

```bash
# Find uncovered lines
go test -coverprofile=coverage.out ./pkg/aws
go tool cover -html=coverage.out

# Generate coverage for specific functions
go test -cover -run TestSpecificFunction ./pkg/aws
```

## Resources

- [golangci-lint Documentation](https://golangci-lint.run)
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Testify Testing Framework](https://github.com/stretchr/testify)
- [Playwright E2E Testing](https://playwright.dev)

---

*Last updated: v0.4.5*