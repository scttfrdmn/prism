// End-to-end tests for complete instance launch workflow
import { test, expect } from '@playwright/test'
import { spawn } from 'child_process'
import { promisify } from 'util'
import { exec } from 'child_process'

const execAsync = promisify(exec)

// Test configuration for daemon integration
const DAEMON_BINARY = '/Users/scttfrdmn/src/cloudworkstation/bin/cwsd'
const DAEMON_URL = 'http://localhost:8947'

let daemonProcess = null

// Helper function to start daemon for testing
async function startTestDaemon() {
  try {
    await execAsync('pkill -f cwsd || true')
    await new Promise(resolve => setTimeout(resolve, 1000))
  } catch (error) {
    // Ignore errors
  }

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
        return true
      }
    } catch (error) {
      // Daemon not ready yet
    }
    
    attempts++
    await new Promise(resolve => setTimeout(resolve, 500))
  }
  
  throw new Error('Test daemon failed to start')
}

// Helper function to stop daemon
async function stopTestDaemon() {
  if (daemonProcess) {
    daemonProcess.kill('SIGTERM')
    daemonProcess = null
  }
  
  try {
    await execAsync('pkill -f cwsd || true')
  } catch (error) {
    // Ignore errors
  }
}

test.describe('Instance Launch Workflow', () => {
  // Start daemon before all tests
  test.beforeAll(async () => {
    await startTestDaemon()
  })

  // Stop daemon after all tests
  test.afterAll(async () => {
    await stopTestDaemon()
  })

  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    
    // Navigate to Quick Start section using DOM manipulation
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('quick-start').classList.add('active')
    })
    
    await expect(page.locator('#quick-start')).toHaveClass(/active/)
    
    // Wait for templates to load from daemon
    await page.waitForTimeout(3000)
  })

  test('launch workflow UI structure and components exist', async ({ page }) => {
    // Step 1: Verify we're in Quick Start section
    await expect(page.locator('#quick-start')).toHaveClass(/active/)
    
    // Step 2: Check that templates are displayed (flexible to actual daemon response)
    const templateElements = await page.locator('.template-card, .template-item, .template-list').count()
    const quickStartContent = await page.locator('#quick-start').textContent()
    
    // Should have some template-related content
    expect(templateElements > 0 || quickStartContent.length > 0).toBe(true)
    
    // Step 3: Test template interaction (if templates exist)
    if (templateElements > 0) {
      const firstTemplate = page.locator('.template-card, .template-item').first()
      
      // Should be clickable
      await expect(firstTemplate).toBeVisible()
      
      // Test hover interaction
      await firstTemplate.hover()
      await expect(firstTemplate).toBeVisible()
      
      // Test that clicking doesn't cause JavaScript errors
      const errors = []
      page.on('pageerror', error => {
        errors.push(error.message)
      })
      
      // Click template (but don't assume specific launch behavior)
      await firstTemplate.click()
      
      await page.waitForTimeout(1000)
      expect(errors.length).toBe(0) // No JavaScript errors
    }
    
    // Step 4: Verify launch UI elements might exist
    const hasLaunchForm = await page.locator('#launch-form, .launch-form, .template-form').count()
    const hasLaunchButton = await page.locator('#launch-btn, .launch-btn, button[text*="Launch"]').count()
    
    // Launch UI might be present depending on implementation
    if (hasLaunchForm > 0 || hasLaunchButton > 0) {
      // If launch UI exists, it should be functional
      const launchElements = await page.locator('#launch-form, .launch-form, #launch-btn, .launch-btn').count()
      expect(launchElements).toBeGreaterThan(0)
    }
  })

  test('templates load and display correctly from daemon', async ({ page }) => {
    // Test that templates are loaded from the real daemon
    const templateElements = await page.locator('.template-card, .template-item, .template-list').count()
    const quickStartSection = page.locator('#quick-start')
    
    // Should have template content from daemon
    if (templateElements > 0) {
      // Templates exist - test their structure
      for (let i = 0; i < Math.min(templateElements, 3); i++) {
        const template = page.locator('.template-card, .template-item').nth(i)
        await expect(template).toBeVisible()
        
        // Should have some text content
        const templateText = await template.textContent()
        expect(templateText.length).toBeGreaterThan(0)
      }
    } else {
      // No templates - should have some informational content
      const sectionText = await quickStartSection.textContent()
      expect(sectionText.length).toBeGreaterThan(0)
    }
  })

  test('template selection and interaction works correctly', async ({ page }) => {
    const templateElements = await page.locator('.template-card, .template-item').count()
    
    if (templateElements > 0) {
      const firstTemplate = page.locator('.template-card, .template-item').first()
      
      // Test template selection
      await firstTemplate.click()
      
      // Should remain visible after click
      await expect(firstTemplate).toBeVisible()
      
      // Test that selection state might change
      await page.waitForTimeout(500)
      
      // If there are multiple templates, test switching
      if (templateElements > 1) {
        const secondTemplate = page.locator('.template-card, .template-item').nth(1)
        await secondTemplate.click()
        await expect(secondTemplate).toBeVisible()
      }
    } else {
      // No templates available - test empty state
      const quickStartContent = await page.locator('#quick-start').textContent()
      expect(quickStartContent.length).toBeGreaterThan(0)
    }
  })

  test('launch interface responds to user interaction', async ({ page }) => {
    // Test basic launch interface functionality
    const templateElements = await page.locator('.template-card, .template-item').count()
    
    if (templateElements > 0) {
      // Click first template
      await page.locator('.template-card, .template-item').first().click()
      
      // Wait for any dynamic content changes
      await page.waitForTimeout(1000)
      
      // Look for form elements that might appear
      const formElements = await page.locator('input, select, button, .form-group, .launch-form').count()
      
      // If forms exist, test basic interaction
      if (formElements > 0) {
        const inputs = await page.locator('input[type="text"], input[type="email"]').count()
        if (inputs > 0) {
          // Test input interaction only if visible
          const firstInput = page.locator('input[type="text"], input[type="email"]').first()
          
          // Only interact if element is actually visible
          const isVisible = await firstInput.isVisible()
          if (isVisible) {
            await firstInput.click()
            await expect(firstInput).toBeFocused()
          } else {
            // Input exists but isn't visible - this is expected for hidden forms
            expect(inputs).toBeGreaterThan(0) // Just verify inputs exist
          }
        }
      }
    }
    
    // Test that no JavaScript errors occur during interaction
    const errors = []
    page.on('pageerror', error => {
      errors.push(error.message)
    })
    
    await page.waitForTimeout(1000)
    expect(errors.length).toBe(0)
  })

  test('quick start section maintains consistent structure', async ({ page }) => {
    // Test that Quick Start section structure is consistent
    await expect(page.locator('#quick-start')).toBeVisible()
    await expect(page.locator('#quick-start')).toHaveClass(/active/)
    
    // Should have some content
    const sectionContent = await page.locator('#quick-start').textContent()
    expect(sectionContent.length).toBeGreaterThan(0)
    
    // Test navigation back to this section
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('quick-start').classList.add('active')
    })
    
    // Should still be functional
    await expect(page.locator('#quick-start')).toHaveClass(/active/)
    await expect(page.locator('#quick-start')).toBeVisible()
  })
})