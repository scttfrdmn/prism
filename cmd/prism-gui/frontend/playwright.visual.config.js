import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  // Test directory for visual tests
  testDir: './tests/visual',
  
  // Run tests in files in parallel
  fullyParallel: true,
  
  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,
  
  // Retry on CI only  
  retries: process.env.CI ? 2 : 0,
  
  // Use fewer workers for visual tests to avoid flakiness
  workers: 1,
  
  // Reporter to use
  reporter: [
    ['html', { outputFolder: 'visual-test-results' }],
    ['json', { outputFile: 'visual-test-report.json' }]
  ],
  
  // Global test configuration
  use: {
    // Base URL for tests
    baseURL: 'http://localhost:3000',
    
    // Browser context options optimized for visual testing
    viewport: { width: 1280, height: 720 },
    
    // Capture screenshot always for visual comparison
    screenshot: 'only-on-failure',
    
    // Disable video for visual tests (not needed)
    video: 'off',
    
    // Collect trace on failure
    trace: 'on-first-retry',
    
    // Wait for fonts and animations to settle
    actionTimeout: 15000,
    
    // Wait for network idle for consistent screenshots
    waitForLoadState: 'networkidle'
  },

  // Configure projects for visual testing
  projects: [
    {
      name: 'chromium-visual',
      use: { 
        ...devices['Desktop Chrome'],
        // Ensure consistent rendering for visual tests
        deviceScaleFactor: 1,
        // Disable animations for consistent screenshots
        reducedMotion: 'reduce'
      },
    }
  ],

  // Run your local dev server before starting the tests
  webServer: {
    command: 'npm run dev',
    port: 3000,
    reuseExistingServer: !process.env.CI,
    timeout: 60000
  }
})