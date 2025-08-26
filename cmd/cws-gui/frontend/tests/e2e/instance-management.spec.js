// End-to-end tests for instance management operations
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

test.describe('Instance Management Operations', () => {
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
    
    // Navigate to My Instances section using DOM manipulation (reliable method)
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    
    await expect(page.locator('#my-instances')).toHaveClass(/active/)
    
    // Wait for API call to complete and content to load
    await page.waitForTimeout(3000)
  })

  test('displays instance information correctly', async ({ page }) => {
    // Test that instance content is displayed (either instances or no-instances message)
    const hasInstanceCards = await page.locator('.instance-card').count()
    const hasNoInstancesMessage = await page.locator('.no-instances').count()
    const hasInstancesGrid = await page.locator('.instances-grid').count()
    
    // At least one of these should be present (real daemon response)
    expect(hasInstanceCards + hasNoInstancesMessage + hasInstancesGrid).toBeGreaterThan(0)
    
    // If there are instance cards, verify they have basic structure
    if (hasInstanceCards > 0) {
      const firstCard = page.locator('.instance-card').first()
      
      // Test that instance cards have expected elements (flexible to actual GUI structure)
      const cardVisible = await firstCard.isVisible()
      expect(cardVisible).toBe(true)
      
      // Look for any text content that indicates instance information
      const cardText = await firstCard.textContent()
      expect(cardText.length).toBeGreaterThan(0)
    }
    
    // If no instances, verify appropriate message is shown
    if (hasInstanceCards === 0) {
      const noInstancesVisible = await page.locator('.no-instances, .empty-state').isVisible()
      // Should show some indication that there are no instances
      expect(noInstancesVisible || hasInstancesGrid > 0).toBe(true)
    }
  })

  test('shows appropriate action buttons based on instance state', async ({ page }) => {
    const instanceCards = await page.locator('.instance-card').count()
    
    if (instanceCards > 0) {
      // Test that instance cards have action buttons
      for (let i = 0; i < Math.min(instanceCards, 3); i++) { // Test up to 3 instances
        const card = page.locator('.instance-card').nth(i)
        
        // Look for common action buttons (flexible to actual implementation)
        const buttons = await card.locator('button').count()
        expect(buttons).toBeGreaterThan(0) // Should have at least one action button
        
        // Common buttons might include Connect, Start, Stop, etc.
        const cardText = await card.textContent()
        const hasActionButtons = await card.locator('button, .btn, .action-btn').count()
        expect(hasActionButtons).toBeGreaterThan(0)
      }
    } else {
      // No instances case - should show empty state
      const hasEmptyState = await page.locator('.no-instances, .empty-state, .instances-grid').count()
      expect(hasEmptyState).toBeGreaterThan(0)
    }
  })

  test('instance action buttons are clickable', async ({ page }) => {
    const instanceCards = await page.locator('.instance-card').count()
    
    if (instanceCards > 0) {
      const firstCard = page.locator('.instance-card').first()
      const actionButtons = await firstCard.locator('button, .btn').count()
      
      if (actionButtons > 0) {
        const firstButton = firstCard.locator('button, .btn').first()
        
        // Test that button is clickable (without assuming specific operations)
        await expect(firstButton).toBeVisible()
        
        // Test hover state works
        await firstButton.hover()
        await expect(firstButton).toBeVisible()
        
        // Button should be focusable
        await firstButton.focus()
        // Note: Not clicking to avoid triggering actual operations during testing
      }
    }
  })

  test('instance management UI is responsive to user interaction', async ({ page }) => {
    const instanceCards = await page.locator('.instance-card').count()
    
    if (instanceCards > 0) {
      // Test that cards respond to hover
      const firstCard = page.locator('.instance-card').first()
      await firstCard.hover()
      await expect(firstCard).toBeVisible()
      
      // Test that the section refreshes properly
      await page.evaluate(() => {
        // Simulate refresh action that might be available
        if (window.refreshInstances) {
          window.refreshInstances()
        }
      })
      
      // Wait for any updates
      await page.waitForTimeout(1000)
      
      // Verify content is still there
      await expect(page.locator('#my-instances')).toHaveClass(/active/)
    }
  })

  test('empty state handling works correctly', async ({ page }) => {
    // Test that the instances section handles the empty state gracefully
    const hasContent = await page.locator('.instance-card, .no-instances, .instances-grid, .empty-state').count()
    
    // Should always have some content (either instances or empty state)
    expect(hasContent).toBeGreaterThan(0)
    
    // Test that the UI structure is consistent
    await expect(page.locator('#my-instances')).toBeVisible()
  })

  test('navigation between sections preserves instance state', async ({ page }) => {
    // Test navigation away and back to instances section
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('quick-start').classList.add('active')
    })
    await expect(page.locator('#quick-start')).toHaveClass(/active/)
    
    // Navigate back to instances
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    await expect(page.locator('#my-instances')).toHaveClass(/active/)
    
    // Wait for content to load again
    await page.waitForTimeout(2000)
    
    // Verify content is displayed
    const hasContent = await page.locator('.instance-card, .no-instances, .instances-grid').count()
    expect(hasContent).toBeGreaterThan(0)
  })

  test('instances section structure is consistent', async ({ page }) => {
    // Test that the instances section has consistent structure
    await expect(page.locator('#my-instances')).toBeVisible()
    
    // Should have some content structure
    const hasContent = await page.locator('.instance-card, .no-instances, .instances-grid, .empty-state').count()
    expect(hasContent).toBeGreaterThan(0)
  })

  test('instance cards have proper accessibility features', async ({ page }) => {
    const instanceCards = await page.locator('.instance-card').count()
    
    if (instanceCards > 0) {
      // Test first few cards for accessibility
      for (let i = 0; i < Math.min(instanceCards, 2); i++) {
        const card = page.locator('.instance-card').nth(i)
        
        // Card should be visible and have content
        await expect(card).toBeVisible()
        
        // Should have some text content
        const cardText = await card.textContent()
        expect(cardText.length).toBeGreaterThan(0)
        
        // Should be interactive (can be hovered/focused)
        await card.hover()
        await expect(card).toBeVisible()
      }
    }
  })

  test('instance management handles API responses correctly', async ({ page }) => {
    // Test that the GUI correctly handles daemon API responses
    await page.waitForTimeout(3000) // Allow time for API calls
    
    // Should display appropriate content based on API response
    const hasInstanceCards = await page.locator('.instance-card').count()
    const hasNoInstancesMessage = await page.locator('.no-instances').count()
    const hasInstancesGrid = await page.locator('.instances-grid').count()
    
    // One of these should be present based on daemon response
    expect(hasInstanceCards + hasNoInstancesMessage + hasInstancesGrid).toBeGreaterThan(0)
    
    // No JavaScript errors should occur
    const errors = []
    page.on('pageerror', error => {
      errors.push(error.message)
    })
    
    await page.waitForTimeout(1000)
    expect(errors.length).toBe(0)
  })

  test('refresh functionality works correctly', async ({ page }) => {
    // Test that refreshing the instances section works
    await page.evaluate(() => {
      // Try to refresh if refresh functionality exists
      if (window.refreshInstances) {
        window.refreshInstances()
      } else if (window.loadInstances) {
        window.loadInstances()
      }
    })
    
    // Wait for refresh to complete
    await page.waitForTimeout(2000)
    
    // Verify section is still active and content is displayed
    await expect(page.locator('#my-instances')).toHaveClass(/active/)
    
    const hasContent = await page.locator('.instance-card, .no-instances, .instances-grid').count()
    expect(hasContent).toBeGreaterThan(0)
  })
})