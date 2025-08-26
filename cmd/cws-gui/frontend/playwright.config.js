import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  // Test directory
  testDir: './tests/e2e',
  
  // Run tests in files in parallel
  fullyParallel: true,
  
  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,
  
  // Retry on CI only
  retries: process.env.CI ? 2 : 0,
  
  // Opt out of parallel tests on CI
  workers: process.env.CI ? 1 : undefined,
  
  // Reporter to use
  reporter: [
    ['html'],
    ['json', { outputFile: 'playwright-report.json' }],
    process.env.CI ? ['github'] : ['list']
  ],
  
  // Global setup and teardown
  globalSetup: './tests/e2e/global-setup.js',
  
  // Global test configuration
  use: {
    // For Wails apps, we test against the served frontend
    // The frontend will connect to the daemon API at localhost:8947
    baseURL: 'http://localhost:3000', // Vite dev server
    
    // Browser context options
    viewport: { width: 1280, height: 720 },
    
    // Capture screenshot on failure
    screenshot: 'only-on-failure',
    
    // Record video on failure
    video: 'retain-on-failure',
    
    // Collect trace on failure
    trace: 'on-first-retry',
    
    // Timeout for each test
    actionTimeout: 10000
  },

  // Configure projects for major browsers (desktop only - CloudWorkstation is not mobile)
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    }
  ],

  // Start the Vite dev server for the frontend
  webServer: {
    command: 'npm run dev',
    port: 3000,
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  }
})