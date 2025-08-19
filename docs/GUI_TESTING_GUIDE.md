# CloudWorkstation GUI Testing Guide

## Version: v0.4.3 - Wails 3.x Implementation  
**Last Updated**: 2025-08-18  
**Testing Stack**: Vitest + Playwright + Percy + MSW

---

## 🎯 **Testing Philosophy**

CloudWorkstation's Wails 3.x GUI leverages modern web testing tools to ensure reliability, visual consistency, and accessibility across all 5 professional themes. Our testing strategy provides comprehensive coverage while maintaining fast feedback cycles.

### **Key Advantages of Web-Based Testing**
- **Familiar Tooling**: Same tools used for modern web applications
- **Rich Ecosystem**: Mature libraries with extensive documentation
- **Fast Execution**: Quick test runs without full desktop app compilation
- **Visual Regression**: Automatic screenshot comparison for UI consistency
- **Accessibility**: Built-in a11y testing with industry-standard tools
- **Cross-Platform CI**: Tests run seamlessly in cloud environments

---

## 🏗️ **Testing Architecture**

```
┌─────────────────────┐    ┌──────────────────────┐    ┌─────────────────────┐
│   Unit Tests        │    │   E2E Tests          │    │   Visual Tests      │
│   (Vitest)          │    │   (Playwright)       │    │   (Percy/Chromatic) │
├─────────────────────┤    ├──────────────────────┤    ├─────────────────────┤
│ • Template logic    │    │ • Complete workflows │    │ • Theme consistency │
│ • Instance mgmt     │    │ • Navigation flows   │    │ • Component states  │
│ • Theme switching   │    │ • Form validation    │    │ • Responsive design │
│ • Utility functions │    │ • Error handling     │    │ • Cross-browser     │
└─────────────────────┘    └──────────────────────┘    └─────────────────────┘
            │                          │                          │
            └──────────────────────────┼──────────────────────────┘
                                      │
                    ┌─────────────────────────────────┐
                    │      Mock Service Worker        │
                    │      (MSW - API Mocking)        │
                    ├─────────────────────────────────┤
                    │ • Daemon API responses          │
                    │ • Error simulation              │
                    │ • Network condition testing     │
                    │ • Consistent test data          │
                    └─────────────────────────────────┘
```

---

## 🛠️ **Setup and Installation**

### **Prerequisites**
```bash
# Node.js 20+ and npm
node --version  # Should be 20+
npm --version   # Should be 9+

# Go 1.24+ for backend integration tests
go version      # Should be 1.24+
```

### **Install Testing Dependencies**
```bash
cd cmd/cws-gui/frontend

# Install all testing dependencies
npm install

# Install Playwright browsers
npm run playwright:install

# Verify installation
npm run test:unit -- --version
npx playwright --version
```

### **Environment Setup**
```bash
# Create test environment file
cat > .env.test << 'EOF'
# Mock daemon URL for testing
VITE_DAEMON_URL=http://localhost:8947

# Test-specific settings
NODE_ENV=test
VITEST_POOL_THREADS=false
EOF

# Install browser dependencies (CI environments)
npx playwright install-deps
```

---

## 🧪 **Test Categories and Usage**

### **1. Unit Tests (Vitest + Testing Library)**

**Purpose**: Test individual functions and components in isolation

**Run Commands**:
```bash
# Run all unit tests
npm run test:unit

# Run with watch mode (development)
npm run test

# Generate coverage report  
npm run test:coverage

# Run specific test file
npx vitest template-selection.test.js

# Run tests matching pattern
npx vitest --grep "template selection"
```

**Example Test Structure**:
```javascript
// tests/unit/template-selection.test.js
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { screen } from '@testing-library/dom'

describe('Template Selection', () => {
  beforeEach(() => {
    setupDOM() // Helper to create test DOM
  })

  test('loads and displays templates correctly', () => {
    renderTemplates(mockTemplates)
    expect(screen.getByText('Python Machine Learning')).toBeInTheDocument()
  })
})
```

**Coverage Thresholds**:
- **Branches**: 80%
- **Functions**: 80%  
- **Lines**: 80%
- **Statements**: 80%

### **2. End-to-End Tests (Playwright)**

**Purpose**: Test complete user workflows and interactions

**Run Commands**:
```bash
# Run all E2E tests
npm run test:e2e

# Run E2E tests in headed mode (development)
npx playwright test --headed

# Run specific browser
npx playwright test --project=chromium

# Run specific test file
npx playwright test launch-workflow.spec.js

# Debug mode with Playwright Inspector
npx playwright test --debug
```

**Test Categories**:
- **Launch Workflow**: Complete instance launch from template selection to dashboard
- **Navigation**: Section switching, progressive disclosure, menu interactions
- **Instance Management**: Start, stop, connect operations with state verification
- **Form Validation**: Input validation, error handling, user feedback
- **Theme Switching**: Theme persistence, visual consistency, settings modal

**Example E2E Test**:
```javascript
// tests/e2e/launch-workflow.spec.js
test('complete launch process', async ({ page }) => {
  await page.goto('/')
  
  // Step 1: Select template
  await page.click('.template-card:has-text("Python ML")')
  await expect(page.locator('.template-card.selected')).toBeVisible()
  
  // Step 2: Fill launch form
  await page.fill('#instance-name', 'test-workstation')
  await page.selectOption('#instance-size', 'L')
  
  // Step 3: Launch instance
  await page.click('#launch-btn')
  await expect(page.locator('text=Successfully launched')).toBeVisible()
})
```

### **3. Visual Regression Tests (Percy)**

**Purpose**: Ensure visual consistency across themes and viewport sizes

**Setup Percy**:
```bash
# Sign up at percy.io and get token
export PERCY_TOKEN=your_percy_token_here

# Run visual tests  
npm run test:visual

# Run for specific theme
npx playwright test themes.spec.js --grep="core theme"
```

**Visual Test Coverage**:
- **5 Themes**: Core, Dark, Academic, Minimal, Custom
- **3 Viewports**: Desktop (1280px), Tablet (768px), Mobile (375px)
- **UI States**: Normal, selected, loading, error, empty
- **Components**: Template cards, instance cards, modals, forms

**Percy Configuration** (`.percy.yml`):
```yaml
version: 2
discovery:
  allowed-hostnames:
    - localhost
snapshot:
  widths: [375, 768, 1280]
  min-height: 1024
  percy-css: |
    /* Disable animations for consistent screenshots */
    *, *::before, *::after {
      animation-duration: 0s !important;
      animation-delay: 0s !important;
      transition-duration: 0s !important;
      transition-delay: 0s !important;
    }
```

### **4. API Contract Tests (MSW)**

**Purpose**: Ensure GUI service layer matches daemon API contracts

**Mock Server Usage**:
```javascript
// tests/mocks/daemon-server.js
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'

const server = setupServer(
  http.get('http://localhost:8947/api/v1/templates', () => {
    return HttpResponse.json(mockTemplates)
  }),
  
  http.post('http://localhost:8947/api/v1/instances/launch', async ({ request }) => {
    const body = await request.json()
    return HttpResponse.json({
      name: body.name,
      state: 'launching',
      instance_id: 'i-' + Math.random().toString(36)
    })
  })
)
```

### **5. Accessibility Tests (axe-core)**

**Purpose**: Ensure WCAG 2.1 compliance and screen reader compatibility

**Run Accessibility Tests**:
```bash
# Install axe-playwright
npm install --save-dev @axe-core/playwright

# Create accessibility test
cat > tests/e2e/accessibility.spec.js << 'EOF'
import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

test('passes accessibility standards', async ({ page }) => {
  await page.goto('/')
  const results = await new AxeBuilder({ page }).analyze()
  expect(results.violations).toEqual([])
})
EOF

# Run accessibility tests
npx playwright test accessibility.spec.js
```

### **6. Performance Tests (Lighthouse)**

**Purpose**: Ensure GUI meets performance benchmarks

**Performance Thresholds**:
- **Performance**: 90+
- **Accessibility**: 95+
- **Best Practices**: 90+
- **SEO**: 80+

```bash
# Install lighthouse
npm install --save-dev lighthouse playwright-lighthouse

# Run performance tests
npx playwright test performance.spec.js
```

---

## 🚀 **Development Workflow**

### **Local Development Testing**
```bash
# Terminal 1: Start Wails GUI in dev mode
cd cmd/cws-gui
wails3 dev

# Terminal 2: Run tests against live application
cd frontend
npm run test        # Unit tests with watch mode
npm run test:e2e    # E2E tests against dev server
npm run test:visual # Visual tests with Percy
```

### **Pre-Commit Testing**
```bash
# Run complete test suite before committing
npm run test:all

# Quick smoke test
npm run test:unit && npm run test:e2e -- --grep="critical"

# Check test coverage
npm run test:coverage
open coverage/index.html  # View coverage report
```

### **Test-Driven Development (TDD)**
```bash
# 1. Write failing test
npm run test -- --grep "new feature"

# 2. Implement feature
# Edit source files...

# 3. Run test until it passes
npm run test -- --grep "new feature"

# 4. Refactor with confidence
npm run test:all
```

---

## 🔧 **Configuration Files**

### **Vitest Configuration** (`vitest.config.js`)
```javascript
import { defineConfig } from 'vite'

export default defineConfig({
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./tests/setup.js'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      thresholds: {
        global: {
          branches: 80,
          functions: 80,
          lines: 80,
          statements: 80
        }
      }
    }
  }
})
```

### **Playwright Configuration** (`playwright.config.js`)
```javascript
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure'
  },
  
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'firefox', use: { ...devices['Desktop Firefox'] } },
    { name: 'webkit', use: { ...devices['Desktop Safari'] } }
  ],
  
  webServer: {
    command: 'npm run dev',
    port: 3000,
    reuseExistingServer: !process.env.CI
  }
})
```

---

## 🤖 **CI/CD Integration**

### **GitHub Actions Workflow**

The GUI testing workflow (`.github/workflows/gui-tests.yml`) includes:

- **Unit Tests**: Fast feedback on every push
- **E2E Tests**: Complete workflow validation
- **Visual Tests**: Theme consistency on PRs
- **Integration Tests**: Real backend integration
- **Accessibility Tests**: WCAG compliance
- **Performance Tests**: Lighthouse benchmarks

### **Test Stages**:
```yaml
jobs:
  unit-tests:     # Fast unit test execution
  e2e-tests:      # Cross-browser E2E testing  
  visual-tests:   # Percy visual regression (PR only)
  integration:    # Real daemon integration
  accessibility:  # a11y compliance testing
  performance:    # Lighthouse benchmarks
  test-summary:   # Aggregate results
```

### **Quality Gates**:
- ✅ **All unit tests must pass** (required)
- ✅ **Critical E2E workflows must pass** (required)
- ✅ **Visual regression approval** (required for UI changes)
- ✅ **Accessibility compliance** (required)
- ✅ **Performance thresholds met** (advisory)

---

## 🐛 **Debugging Tests**

### **Unit Test Debugging**
```bash
# Run single test with detailed output
npx vitest template-selection.test.js --reporter=verbose

# Debug with browser devtools
npx vitest --inspect-brk template-selection.test.js

# Add debug breakpoints
test('debug example', () => {
  debugger; // Breakpoint here
  expect(true).toBe(true)
})
```

### **E2E Test Debugging**
```bash
# Debug with Playwright Inspector
npx playwright test --debug launch-workflow.spec.js

# Run with visible browser
npx playwright test --headed launch-workflow.spec.js

# Slow motion for visual debugging
npx playwright test --headed --slowMo=1000

# Generate trace files for failed tests
npx playwright test --trace=on
npx playwright show-trace trace.zip
```

### **Visual Test Debugging**
```bash
# Compare visual changes locally
percy snapshot compare

# Upload snapshots manually
percy upload ./screenshots

# Debug Percy builds
percy build:status build-id
```

---

## 📊 **Test Reporting and Metrics**

### **Coverage Reports**
```bash
# Generate HTML coverage report
npm run test:coverage
open coverage/index.html

# Coverage by file type
npm run test:coverage -- --reporter=text-summary
```

### **E2E Test Reports**  
```bash
# Generate HTML report
npx playwright test
npx playwright show-report

# JSON report for CI integration
npx playwright test --reporter=json
```

### **Performance Metrics**
```bash
# Lighthouse JSON reports
npm run test:performance -- --output=json

# Web vitals tracking
npm install --save-dev web-vitals
```

### **Test Metrics Tracked**:
- **Test Coverage**: Line, branch, function, statement coverage
- **Test Execution Time**: Unit (~10s), E2E (~2min), Visual (~5min)
- **Flakiness Rate**: Retry success rate and failure patterns
- **Visual Changes**: Screenshot diff count and approval rate
- **Performance Scores**: Core Web Vitals and Lighthouse metrics

---

## 🔍 **Best Practices**

### **Writing Reliable Tests**

1. **Use Data Attributes for Stable Selectors**:
```html
<button data-testid="launch-button" class="btn-primary">Launch</button>
```
```javascript
await page.click('[data-testid="launch-button"]')
```

2. **Wait for Conditions, Not Timeouts**:
```javascript
// ❌ Avoid arbitrary waits
await page.waitForTimeout(1000)

// ✅ Wait for specific conditions
await expect(page.locator('#launch-form')).toBeVisible()
```

3. **Mock External Dependencies**:
```javascript
// Mock Wails service calls
global.wails = {
  CloudWorkstationService: {
    GetTemplates: vi.fn().mockResolvedValue(mockTemplates)
  }
}
```

### **Test Organization**

1. **Descriptive Test Names**:
```javascript
// ❌ Generic
test('template test', () => {})

// ✅ Specific
test('shows launch form after template selection', () => {})
```

2. **Setup and Teardown**:
```javascript
beforeEach(() => {
  setupDOM()
  resetMocks()
})

afterEach(() => {
  cleanup()
})
```

3. **Test Data Management**:
```javascript
// Centralized mock data
export const mockTemplates = [
  { name: 'Python ML', category: 'Machine Learning' },
  { name: 'R Research', category: 'Data Science' }
]
```

### **Performance Optimization**

1. **Parallel Test Execution**:
```javascript
// playwright.config.js
export default defineConfig({
  fullyParallel: true,
  workers: process.env.CI ? 1 : undefined
})
```

2. **Smart Test Retries**:
```javascript
// vitest.config.js
export default defineConfig({
  test: {
    retry: process.env.CI ? 2 : 0
  }
})
```

3. **Selective Test Running**:
```bash
# Run only changed tests
npm run test -- --changed

# Run only critical tests
npm run test:e2e -- --grep="@critical"
```

---

## 🚨 **Troubleshooting**

### **Common Issues**

**Issue**: Unit tests failing with DOM errors
```bash
# Solution: Ensure jsdom environment
npm run test -- --environment=jsdom
```

**Issue**: E2E tests timing out
```bash
# Solution: Increase timeout and add wait conditions
await expect(page.locator('#element')).toBeVisible({ timeout: 10000 })
```

**Issue**: Visual tests showing false positives
```bash
# Solution: Disable animations in Percy CSS
/* percy-specific CSS to disable animations */
```

**Issue**: Mock service worker not intercepting requests
```bash
# Solution: Ensure MSW is started before tests
beforeAll(() => server.listen())
afterAll(() => server.close())
```

### **Debug Checklist**

- [ ] Is the mock daemon server running and responding?
- [ ] Are all test dependencies installed (`npm ci`)?
- [ ] Is the correct Node.js version being used (20+)?
- [ ] Are Playwright browsers installed (`npx playwright install`)?
- [ ] Is the test environment configured properly (`.env.test`)?
- [ ] Are there any console errors or network failures?
- [ ] Is the GUI development server running on the expected port?

---

## 📚 **Resources and Documentation**

### **Framework Documentation**
- [Vitest Testing Framework](https://vitest.dev/)
- [Playwright End-to-End Testing](https://playwright.dev/)
- [Testing Library Utilities](https://testing-library.com/)
- [Mock Service Worker](https://mswjs.io/)
- [Percy Visual Testing](https://percy.io/)

### **CloudWorkstation-Specific Resources**
- [GUI Refinements TODO List](./GUI_REFINEMENTS_TODOLIST.md)
- [Daemon API Reference](./DAEMON_API_REFERENCE.md)
- [Wails 3.x Documentation](https://wails.io/docs/next/)

### **Testing Strategy References**
- [Testing Trophy Philosophy](https://kentcdodds.com/blog/the-testing-trophy-and-testing-classifications)
- [Web Accessibility Testing](https://web.dev/accessibility-testing/)
- [Visual Regression Testing Guide](https://percy.io/blog/visual-testing-guide)

---

**Total Test Coverage**: 65+ test scenarios across unit, E2E, visual, accessibility, and performance categories.

This comprehensive testing guide ensures the CloudWorkstation Wails 3.x GUI maintains professional quality, visual consistency, and accessibility standards while providing fast feedback loops for developers.