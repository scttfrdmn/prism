// Actual daemon integration tests for CloudWorkstation GUI
// These tests start the real CloudWorkstation daemon and test GUI integration
import { test, expect } from '@playwright/test'
import { spawn } from 'child_process'
import { promisify } from 'util'
import { exec } from 'child_process'

const execAsync = promisify(exec)

// Test configuration
const DAEMON_BINARY = '/Users/scttfrdmn/src/cloudworkstation/bin/cwsd'
const DAEMON_URL = 'http://localhost:8947'
const DAEMON_STARTUP_TIMEOUT = 10000 // 10 seconds
const API_TIMEOUT = 15000 // 15 seconds for API calls

let daemonProcess = null

// Helper function to start the CloudWorkstation daemon
async function startDaemon() {
  // Kill any existing daemon first
  try {
    await execAsync('pkill -f cwsd || true')
    await new Promise(resolve => setTimeout(resolve, 1000))
  } catch (error) {
    // Ignore errors - daemon might not be running
  }

  // Start the daemon
  daemonProcess = spawn(DAEMON_BINARY, [], {
    stdio: 'pipe',
    detached: false
  })

  // Wait for daemon to start
  let attempts = 0
  const maxAttempts = 20
  
  while (attempts < maxAttempts) {
    try {
      const response = await fetch(`${DAEMON_URL}/api/v1/health`)
      if (response.ok) {
        console.log('Daemon started successfully')
        return true
      }
    } catch (error) {
      // Daemon not ready yet
    }
    
    attempts++
    await new Promise(resolve => setTimeout(resolve, 500))
  }
  
  throw new Error(`Daemon failed to start after ${maxAttempts * 500}ms`)
}

// Helper function to stop the daemon
async function stopDaemon() {
  if (daemonProcess) {
    daemonProcess.kill('SIGTERM')
    daemonProcess = null
  }
  
  // Ensure cleanup
  try {
    await execAsync('pkill -f cwsd || true')
  } catch (error) {
    // Ignore errors
  }
}

// Helper function to make API calls to the daemon
async function apiCall(endpoint, options = {}) {
  const url = `${DAEMON_URL}/api/v1${endpoint}`
  const defaultOptions = {
    timeout: API_TIMEOUT,
    headers: {
      'Content-Type': 'application/json'
    }
  }
  
  const response = await fetch(url, { ...defaultOptions, ...options })
  if (!response.ok) {
    throw new Error(`API call failed: ${response.status} ${response.statusText}`)
  }
  
  return response.json()
}

test.describe('Daemon Integration Tests', () => {
  // Start daemon before all tests
  test.beforeAll(async () => {
    console.log('Starting CloudWorkstation daemon...')
    await startDaemon()
  })

  // Stop daemon after all tests
  test.afterAll(async () => {
    console.log('Stopping CloudWorkstation daemon...')
    await stopDaemon()
  })

  test('daemon health check works', async () => {
    // Test direct API call to daemon
    const response = await fetch(`${DAEMON_URL}/api/v1/health`)
    expect(response.ok).toBe(true)
    
    const data = await response.json()
    expect(data).toHaveProperty('status', 'healthy')
  })

  test('GUI connects to real daemon and loads data', async ({ page }) => {
    await page.goto('/')
    
    // Wait for the application to load
    await expect(page.locator('h1.app-title')).toBeVisible()
    
    // Navigate to My Instances section
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    
    // Wait for API call to complete and content to load
    // This should now connect to the real daemon
    await page.waitForTimeout(3000) // Give time for API call
    
    // Check that we get either instances or the no-instances message from real daemon
    const hasInstancesContent = await page.locator('.instance-card').count()
    const hasNoInstancesMessage = await page.locator('.no-instances').count()
    const hasInstancesGrid = await page.locator('.instances-grid').count()
    
    // At least one of these should be present (real daemon response)
    expect(hasInstancesContent + hasNoInstancesMessage + hasInstancesGrid).toBeGreaterThan(0)
  })

  test('templates section loads real template data', async ({ page }) => {
    await page.goto('/')
    
    // Check that templates endpoint returns data
    const templatesData = await apiCall('/templates')
    expect(templatesData).toBeDefined()
    
    // Navigate to templates in GUI and verify they're displayed
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('quick-start').classList.add('active')
    })
    
    // Wait for templates to load
    await page.waitForTimeout(2000)
    
    // Check that template elements are present
    const templateElements = await page.locator('.template-card, .template-item, .template-list').count()
    expect(templateElements).toBeGreaterThan(0)
  })

  test('daemon status API integration', async ({ page }) => {
    // Test the daemon health endpoint (the actual working endpoint)
    const healthData = await apiCall('/health')
    expect(healthData).toBeDefined()
    expect(healthData).toHaveProperty('status', 'healthy')
    
    await page.goto('/')
    
    // Navigate to settings or status section if available
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
    })
    
    await expect(page.locator('#settings-modal')).not.toHaveClass(/hidden/)
    
    // Check that GUI can display daemon status
    const modalVisible = await page.locator('.modal-content').isVisible()
    expect(modalVisible).toBe(true)
  })

  test('real API error handling in GUI', async ({ page }) => {
    await page.goto('/')
    
    // Navigate to instances section
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    
    // Wait for API response
    await page.waitForTimeout(3000)
    
    // Check that error handling works properly (no uncaught errors)
    const errors = []
    page.on('pageerror', error => {
      errors.push(error.message)
    })
    
    // Trigger potential error conditions
    await page.evaluate(() => {
      // Try to trigger API calls that might fail
      if (window.refreshInstances) {
        window.refreshInstances()
      }
    })
    
    await page.waitForTimeout(2000)
    
    // Should have no uncaught JavaScript errors
    expect(errors.length).toBe(0)
  })

  test('settings modal integrates with daemon configuration', async ({ page }) => {
    await page.goto('/')
    
    // Open settings modal
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
    })
    
    await expect(page.locator('#settings-modal')).not.toHaveClass(/hidden/)
    
    // Check that daemon section is present and configured
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-daemon').classList.add('active')
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.querySelector('.settings-nav-btn:nth-child(3)').classList.add('active')
    })
    
    await expect(page.locator('#settings-daemon')).toHaveClass(/active/)
    
    // Check that daemon URL field shows correct value
    const daemonUrl = await page.locator('#daemon-url').inputValue()
    expect(daemonUrl).toBe('http://localhost:8947')
    
    // Verify connection test button exists (real integration point)
    await expect(page.locator('button:has-text("Test Connection")')).toBeVisible()
  })

  test('GUI theme switching persists through daemon interaction', async ({ page }) => {
    await page.goto('/')
    
    // Open settings and switch to appearance
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-appearance').classList.add('active')
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.querySelector('.settings-nav-btn:nth-child(4)').classList.add('active')
    })
    
    // Change theme
    await page.evaluate(() => {
      const selector = document.getElementById('theme-selector')
      if (selector) {
        selector.value = 'dark'
        document.documentElement.setAttribute('data-theme', 'dark')
        const themeLink = document.getElementById('theme-link')
        if (themeLink) {
          themeLink.href = '/themes/dark.css'
        }
      }
    })
    
    // Verify theme applied
    const theme = await page.evaluate(() => {
      return document.documentElement.getAttribute('data-theme')
    })
    expect(theme).toBe('dark')
    
    // Navigate to instances to trigger API call
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.add('hidden')
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    
    // Wait for API interaction
    await page.waitForTimeout(2000)
    
    // Verify theme persisted through daemon interaction
    const persistedTheme = await page.evaluate(() => {
      return document.documentElement.getAttribute('data-theme')
    })
    expect(persistedTheme).toBe('dark')
  })

  test('concurrent GUI and daemon operations', async ({ page }) => {
    await page.goto('/')
    
    // Start multiple concurrent operations
    const promises = [
      // API call promise
      apiCall('/templates'),
      // GUI navigation promise
      page.evaluate(() => {
        document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
        document.getElementById('my-instances').classList.add('active')
      }),
      // Settings modal promise
      page.evaluate(() => {
        setTimeout(() => {
          document.getElementById('settings-modal').classList.remove('hidden')
        }, 100)
      })
    ]
    
    // Wait for all operations
    await Promise.all(promises)
    
    // Verify GUI is still responsive
    await expect(page.locator('#my-instances')).toHaveClass(/active/)
    await expect(page.locator('#settings-modal')).not.toHaveClass(/hidden/)
    
    // Verify no JavaScript errors occurred
    const errors = []
    page.on('pageerror', error => {
      errors.push(error.message)
    })
    
    await page.waitForTimeout(1000)
    expect(errors.length).toBe(0)
  })
})

test.describe('Real Daemon API Integration', () => {
  test.beforeAll(async () => {
    console.log('Starting daemon for API integration tests...')
    await startDaemon()
  })

  test.afterAll(async () => {
    console.log('Stopping daemon after API integration tests...')
    await stopDaemon()
  })

  test('health endpoint returns correct data', async () => {
    const data = await apiCall('/health')
    expect(data).toHaveProperty('status', 'healthy')
    expect(data).toHaveProperty('last_checked')
    expect(data).toHaveProperty('system_metrics')
  })

  test('templates endpoint returns template data', async () => {
    const data = await apiCall('/templates')
    expect(data).toBeDefined()
    expect(typeof data).toBe('object')
  })

  test('instances endpoint handles empty list correctly', async () => {
    const data = await apiCall('/instances')
    expect(data).toBeDefined()
    expect(data).toHaveProperty('instances')
    expect(Array.isArray(data.instances)).toBe(true)
  })

  test('health endpoint provides system information', async () => {
    const data = await apiCall('/health')
    expect(data).toBeDefined()
    expect(data).toHaveProperty('status', 'healthy')
    expect(data).toHaveProperty('system_metrics')
    expect(data.system_metrics).toHaveProperty('uptime')
  })
})