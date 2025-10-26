# Prism GUI Refinements & Polish TODO List

## Version: Post-Wails 3.x MVP Implementation 
Created: 2025-08-18  
**Status**: Ready for future cleanup and polish work

---

## 🎨 **UI/UX Refinements**

### Progressive Disclosure Improvements
- [ ] **Advanced Options Toggle**: Implement collapsible advanced launch options (custom instance types, spot pricing, VPC settings)
- [ ] **Template Details Modal**: Expandable template information with dependency chains, validation status, and troubleshooting guides
- [ ] **Instance Action Menus**: Dropdown actions (hibernate/resume, create snapshots, view logs, connect via different protocols)
- [ ] **Settings Expansion**: Multi-tab settings with advanced preferences, AWS profile management, and notification preferences

### Visual Polish
- [ ] **Loading States**: Replace simple spinners with skeleton screens and progress indicators
- [ ] **Animations**: Add subtle micro-interactions for template selection, form transitions, and status changes
- [ ] **Empty States**: Improve empty state illustrations and messaging with actionable next steps
- [ ] **Error Handling**: Professional error modals with suggested solutions and retry mechanisms

### Theme System Enhancements
- [ ] **Theme Customization UI**: In-app theme editor for custom CSS variables without file editing
- [ ] **Theme Validation**: CSS validation and preview system for custom themes
- [ ] **Additional Themes**: Add research-specific themes (biology-green, physics-blue, chemistry-purple)
- [ ] **Auto Dark Mode**: System-based automatic theme switching

---

## 🔧 **Technical Improvements**

### API Integration
- [ ] **Real Daemon Integration**: Connect service.go TODO methods to actual Prism daemon API
- [ ] **Error Handling**: Comprehensive error states and recovery mechanisms
- [ ] **Caching**: Smart template and instance data caching with refresh strategies
- [ ] **Websocket Updates**: Real-time instance status updates via websocket instead of polling

### Performance Optimizations
- [ ] **Lazy Loading**: Template icons and metadata lazy loading for faster startup
- [ ] **Virtual Scrolling**: For large template and instance lists
- [ ] **Bundle Optimization**: Code splitting and dynamic imports for Vite frontend
- [ ] **Memory Management**: Proper cleanup and garbage collection for long-running sessions

### Build System
- [ ] **Production Builds**: Optimized production builds with asset minimization
- [ ] **Cross-Platform**: Windows and Linux build configurations
- [ ] **Distribution**: Automated app signing and packaging for all platforms
- [ ] **Hot Reload**: Development mode with hot reload for faster iteration

---

## 🚀 **Feature Additions**

### Research Workflow Integration
- [ ] **Project Templates**: Multi-template project setups (ML pipeline with storage, compute, and visualization)
- [ ] **Collaboration Features**: Share instance configurations and templates with team members
- [ ] **Usage Analytics**: Built-in cost tracking and usage analytics dashboard
- [ ] **Backup Integration**: Automatic EBS snapshot scheduling and management

### Advanced Instance Management
- [ ] **Batch Operations**: Select and perform operations on multiple instances simultaneously
- [ ] **Instance Monitoring**: CPU, memory, and GPU usage graphs directly in the GUI
- [ ] **Log Viewer**: Built-in log streaming and filtering capabilities
- [ ] **File Manager**: Simple file upload/download interface for research data

### Template Marketplace Integration (Future Phase 5)
- [ ] **Template Discovery**: Browse and search community-contributed research templates
- [ ] **Template Sharing**: Publish and share custom templates with the research community
- [ ] **Template Reviews**: Rating and review system for community templates
- [ ] **Template Dependencies**: Visual dependency graphs and compatibility checking

---

## 📱 **Platform Integration**

### Desktop Experience
- [ ] **System Tray Integration**: Always-on system tray with quick actions and status
- [ ] **Native Notifications**: Desktop notifications for instance state changes and cost alerts
- [ ] **Keyboard Shortcuts**: Comprehensive keyboard navigation and shortcuts
- [ ] **Menu Bar Integration**: Native menu bar with standard application menus (File, Edit, View, Window, Help)

### Accessibility
- [ ] **Screen Reader Support**: ARIA labels and semantic HTML for assistive technologies
- [ ] **Keyboard Navigation**: Full keyboard accessibility for all interface elements
- [ ] **High Contrast**: High contrast theme variants for accessibility compliance
- [ ] **Font Scaling**: Support for system font scaling preferences

---

## 🧪 **Testing & Quality**

### Testing Suite
- [ ] **Unit Tests**: Frontend JavaScript unit tests with Vitest (matches our Vite build system)
- [ ] **Integration Tests**: Go backend service integration tests with testify
- [ ] **E2E Tests**: End-to-end GUI workflow testing with Playwright
- [ ] **Visual Regression**: Automated screenshot testing for theme consistency with Percy or Chromatic
- [ ] **Component Tests**: Isolated component testing with Testing Library
- [ ] **API Contract Tests**: Ensure GUI service matches daemon API with Pact or similar

### Code Quality
- [ ] **ESLint Configuration**: Modern JavaScript linting rules for frontend code
- [ ] **TypeScript Migration**: Migrate JavaScript frontend to TypeScript for better type safety
- [ ] **Go Formatting**: Ensure consistent Go code formatting and linting
- [ ] **Documentation**: Comprehensive code documentation and API references

---

## 📚 **Documentation**

### User Documentation
- [ ] **GUI User Guide**: Comprehensive user guide for the new Wails 3.x interface
- [ ] **Theme Customization Guide**: Step-by-step theme creation and customization tutorial
- [ ] **Troubleshooting Guide**: Common issues and solutions for GUI problems
- [ ] **Video Tutorials**: Screen recordings demonstrating key workflows

### Developer Documentation
- [ ] **Architecture Documentation**: Technical overview of Wails 3.x implementation
- [ ] **API Documentation**: Complete daemon API reference for GUI integration
- [ ] **Build Instructions**: Detailed development environment setup and build processes
- [ ] **Contribution Guidelines**: Guidelines for GUI contributions and theme development

---

## 🎯 **Priority Levels**

### **High Priority** (Essential for production)
- Real daemon API integration
- Error handling and recovery
- Basic theme system completion
- Cross-platform builds

### **Medium Priority** (Quality of life)
- Advanced options toggle
- Loading state improvements
- System tray integration
- Keyboard shortcuts

### **Low Priority** (Polish and enhancement)
- Additional themes
- Animation improvements
- Template marketplace integration
- Advanced monitoring features

---

## 📝 **Implementation Notes**

### **Current Technical Debt**
- `service.go` contains TODO methods that need daemon API integration
- Frontend error handling is basic alert() calls - needs professional modals
- No real-time updates - relies on 30-second polling intervals
- Missing production build configuration and optimization

### **Architecture Decisions**
- Wails 3.x chosen for superior HTML/CSS skinning and menu bar integration
- Vite build system for modern frontend development experience
- Go backend service pattern for clean daemon API abstraction
- Progressive disclosure UX pattern for academic researcher usability

### **Design System**
- Inter font family for professional academic appearance
- CSS custom properties system for comprehensive theme customization
- Responsive grid layouts for various window sizes
- Consistent spacing and color systems across all themes

---

---

## 🧪 **Web Testing Strategy for Wails GUI**

### **Why Web Testing Tools Work**
Since Wails 3.x renders our GUI using a WebView (Chromium-based), we can leverage the entire modern web testing ecosystem:
- **DOM Access**: Full access to HTML elements, CSS styles, and JavaScript state
- **Browser APIs**: Standard web APIs work normally (localStorage, fetch, etc.)
- **DevTools Integration**: Chrome DevTools work for debugging during development
- **Familiar Tooling**: Same tools used for React, Vue, Angular applications

### **Recommended Testing Stack**

#### **1. Unit Testing: Vitest + Testing Library**
```javascript
// tests/unit/template-selection.test.js
import { describe, test, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/dom'
import { loadTemplates, selectTemplate } from '../src/main.js'

describe('Template Selection', () => {
  test('loads and displays templates', async () => {
    // Mock Wails service
    window.wails = {
      PrismService: {
        GetTemplates: vi.fn().mockResolvedValue([
          { Name: 'Python ML', Description: 'Machine learning environment' }
        ])
      }
    }
    
    await loadTemplates()
    expect(screen.getByText('Python ML')).toBeInTheDocument()
  })
  
  test('shows launch form after template selection', async () => {
    selectTemplate('Python ML')
    await waitFor(() => {
      expect(screen.getByText('Launch Research Environment')).toBeVisible()
    })
  })
})
```

#### **2. End-to-End Testing: Playwright**
```javascript
// tests/e2e/launch-workflow.spec.js
import { test, expect } from '@playwright/test'

test.describe('Instance Launch Workflow', () => {
  test('complete launch process', async ({ page }) => {
    // Start Wails GUI application
    await page.goto('http://localhost:3000') // Dev server or built app
    
    // Step 1: Select template
    await page.click('text=Python Machine Learning')
    await expect(page.locator('.template-card.selected')).toBeVisible()
    
    // Step 2: Fill launch form
    await expect(page.locator('#launch-form')).toBeVisible()
    await page.fill('#instance-name', 'test-ml-workstation')
    await page.selectOption('#instance-size', 'L')
    
    // Step 3: Launch instance
    await page.click('#launch-btn')
    await expect(page.locator('text=Successfully launched')).toBeVisible()
    
    // Step 4: Verify instance appears in dashboard
    await page.click('text=My Instances')
    await expect(page.locator('text=test-ml-workstation')).toBeVisible()
  })
})
```

#### **3. Visual Regression Testing: Percy**
```javascript
// tests/visual/themes.spec.js  
import { test } from '@playwright/test'
import percySnapshot from '@percy/playwright'

test.describe('Theme Visual Testing', () => {
  const themes = ['core', 'dark', 'academic', 'minimal']
  
  themes.forEach(theme => {
    test(`${theme} theme renders correctly`, async ({ page }) => {
      await page.goto('http://localhost:3000')
      await page.evaluate((theme) => applyTheme(theme), theme)
      await percySnapshot(page, `GUI - ${theme} theme`)
    })
  })
})
```

#### **4. Component Testing: Testing Library**
```javascript
// tests/components/instance-card.test.js
import { test, expect } from 'vitest'
import { render, screen } from '@testing-library/dom'

test('instance card displays correct information', () => {
  const instance = {
    Name: 'my-research',
    State: 'running', 
    IP: '54.123.45.67',
    Cost: 0.0416
  }
  
  renderInstanceCard(instance)
  
  expect(screen.getByText('my-research')).toBeInTheDocument()
  expect(screen.getByText('running')).toHaveClass('instance-status', 'running')
  expect(screen.getByText('$0.0416/hour')).toBeInTheDocument()
})
```

### **Testing Configuration Files**

#### **Vitest Config** (`vitest.config.js`)
```javascript
import { defineConfig } from 'vite'

export default defineConfig({
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./tests/setup.js'],
    coverage: {
      reporter: ['text', 'json', 'html']
    }
  }
})
```

#### **Playwright Config** (`playwright.config.js`)
```javascript
import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests/e2e',
  use: {
    baseURL: 'http://localhost:3000',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure'
  },
  webServer: {
    command: 'npm run dev',
    port: 3000,
    reuseExistingServer: !process.env.CI
  }
})
```

### **Integration with Wails Development**

#### **Testing During Development**
```bash
# Terminal 1: Start Wails in dev mode
wails3 dev

# Terminal 2: Run tests against live application  
npm run test:e2e
npm run test:visual
```

#### **CI/CD Integration**
```yaml
# .github/workflows/gui-tests.yml
name: GUI Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      # Build Wails application
      - name: Build GUI
        run: |
          cd cmd/cws-gui/frontend
          npm ci
          npm run build
          
      # Run unit tests  
      - name: Unit Tests
        run: npm run test:unit
        
      # Run E2E tests
      - name: E2E Tests  
        run: |
          npx playwright install
          npm run test:e2e
          
      # Visual regression tests
      - name: Visual Tests
        run: npm run test:visual
        env:
          PERCY_TOKEN: ${{ secrets.PERCY_TOKEN }}
```

### **Testing Real Daemon Integration**

#### **Mock Daemon for Testing**
```javascript
// tests/mocks/daemon-server.js
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

const handlers = [
  http.get('http://localhost:8947/api/v1/templates', () => {
    return HttpResponse.json([
      { Name: 'Python ML', Description: 'ML environment' },
      { Name: 'R Research', Description: 'R environment' }
    ])
  }),
  
  http.get('http://localhost:8947/api/v1/instances', () => {
    return HttpResponse.json([
      { Name: 'test-instance', State: 'running', IP: '1.2.3.4' }
    ])
  })
]

export const server = setupServer(...handlers)
```

### **Advanced Testing Scenarios**

#### **Error State Testing**
```javascript
test('handles daemon connection failure gracefully', async ({ page }) => {
  // Simulate daemon being down
  await page.route('http://localhost:8947/**', route => route.abort())
  
  await page.goto('http://localhost:3000')
  await expect(page.locator('text=Daemon unavailable')).toBeVisible()
  await expect(page.locator('button:has-text("Retry")')).toBeVisible()
})
```

#### **Theme Switching Testing**
```javascript
test('theme persistence across sessions', async ({ page }) => {
  await page.goto('http://localhost:3000')
  
  // Switch to dark theme
  await page.click('button[title="Toggle Theme"]')
  await expect(page.locator('[data-theme="dark"]')).toBeVisible()
  
  // Reload page
  await page.reload()
  
  // Theme should persist
  await expect(page.locator('[data-theme="dark"]')).toBeVisible()
})
```

### **Benefits of Web-Based Testing**

1. **Familiar Tools**: Same tools developers already know for web applications
2. **Rich Ecosystem**: Mature testing libraries with extensive documentation  
3. **Fast Feedback**: Quick test execution without full app compilation
4. **Visual Testing**: Screenshot comparison for UI consistency
5. **Accessibility Testing**: Web a11y tools work out of the box
6. **Performance Testing**: Web performance metrics and lighthouse audits
7. **Cross-Platform**: Tests run on CI/CD without platform-specific setup

**Total Refinement Items**: 65+ improvements across UI/UX, technical, feature, platform, testing, and documentation categories.

This comprehensive refinement list provides a roadmap for transforming the current Wails 3.x MVP into a production-ready, professional-grade GUI for academic researchers.