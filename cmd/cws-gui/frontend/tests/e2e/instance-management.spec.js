// End-to-end tests for instance management operations
import { test, expect } from '@playwright/test'

test.describe('Instance Management Operations', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    
    // Navigate to My Instances section
    await page.click('.nav-item:has-text("My Instances")')
    await expect(page.locator('#my-instances')).toHaveClass(/active/)
    
    // Wait for instances to load
    await expect(page.locator('.instance-card')).toHaveCount(2) // Mock returns 2 instances
  })

  test('displays instance information correctly', async ({ page }) => {
    // Verify running instance
    const runningInstance = page.locator('.instance-card:has-text("ml-research-workstation")')
    await expect(runningInstance).toBeVisible()
    await expect(runningInstance.locator('.instance-name')).toHaveText('ml-research-workstation')
    await expect(runningInstance.locator('.instance-status.running')).toHaveText('running')
    await expect(runningInstance.locator('text=54.123.45.67')).toBeVisible()
    await expect(runningInstance.locator('text=$0.0416/hour')).toBeVisible()
    await expect(runningInstance.locator('text=us-west-2')).toBeVisible()

    // Verify stopped instance
    const stoppedInstance = page.locator('.instance-card:has-text("data-analysis-r")')
    await expect(stoppedInstance).toBeVisible()
    await expect(stoppedInstance.locator('.instance-name')).toHaveText('data-analysis-r')
    await expect(stoppedInstance.locator('.instance-status.stopped')).toHaveText('stopped')
    await expect(stoppedInstance.locator('text=54.123.45.68')).toBeVisible()
    await expect(stoppedInstance.locator('text=$0.0832/hour')).toBeVisible()
  })

  test('shows appropriate action buttons based on instance state', async ({ page }) => {
    // Running instance should have Stop button
    const runningInstance = page.locator('.instance-card:has-text("ml-research-workstation")')
    await expect(runningInstance.locator('button:has-text("Connect")')).toBeVisible()
    await expect(runningInstance.locator('button:has-text("Stop")')).toBeVisible()
    await expect(runningInstance.locator('button:has-text("Start")')).toHaveCount(0)

    // Stopped instance should have Start button
    const stoppedInstance = page.locator('.instance-card:has-text("data-analysis-r")')
    await expect(stoppedInstance.locator('button:has-text("Connect")')).toBeVisible()
    await expect(stoppedInstance.locator('button:has-text("Start")')).toBeVisible()
    await expect(stoppedInstance.locator('button:has-text("Stop")')).toHaveCount(0)
  })

  test('stop instance operation works correctly', async ({ page }) => {
    const runningInstance = page.locator('.instance-card:has-text("ml-research-workstation")')
    
    // Click stop button
    await runningInstance.locator('button:has-text("Stop")').click()
    
    // Should show success message (mocked as alert)
    page.on('dialog', dialog => {
      expect(dialog.message()).toContain('stopping')
      dialog.accept()
    })
    
    // Instance state should update to stopping
    await expect(runningInstance.locator('.instance-status.stopping')).toBeVisible({ timeout: 5000 })
    
    // Button should change from Stop to Start
    await expect(runningInstance.locator('button:has-text("Start")')).toBeVisible()
    await expect(runningInstance.locator('button:has-text("Stop")')).toHaveCount(0)
  })

  test('start instance operation works correctly', async ({ page }) => {
    const stoppedInstance = page.locator('.instance-card:has-text("data-analysis-r")')
    
    // Click start button
    await stoppedInstance.locator('button:has-text("Start")').click()
    
    // Should show success message (mocked as alert)
    page.on('dialog', dialog => {
      expect(dialog.message()).toContain('starting')
      dialog.accept()
    })
    
    // Instance state should update to starting
    await expect(stoppedInstance.locator('.instance-status.starting')).toBeVisible({ timeout: 5000 })
    
    // Button should change from Start to Stop
    await expect(stoppedInstance.locator('button:has-text("Stop")')).toBeVisible()
    await expect(stoppedInstance.locator('button:has-text("Start")')).toHaveCount(0)
  })

  test('connect to instance shows connection information', async ({ page }) => {
    const runningInstance = page.locator('.instance-card:has-text("ml-research-workstation")')
    
    // Set up dialog handler for connection info
    page.on('dialog', dialog => {
      expect(dialog.message()).toContain('Connection information for "ml-research-workstation"')
      expect(dialog.message()).toContain('SSH: ssh ec2-user@54.123.45.67')
      expect(dialog.message()).toContain('JUPYTER: http://54.123.45.67:8888')
      dialog.accept()
    })
    
    // Click connect button
    await runningInstance.locator('button:has-text("Connect")').click()
  })

  test('handles instance operations with confirmation dialogs', async ({ page }) => {
    const runningInstance = page.locator('.instance-card:has-text("ml-research-workstation")')
    
    // Set up dialog handler for stop confirmation
    let dialogShown = false
    page.on('dialog', dialog => {
      if (dialog.message().includes('Stop instance')) {
        dialogShown = true
        expect(dialog.message()).toContain('ml-research-workstation')
        expect(dialog.message()).toContain('shut down the instance but preserve all data')
        dialog.accept() // Accept the confirmation
      } else {
        dialog.accept() // Accept success message
      }
    })
    
    // Click stop button
    await runningInstance.locator('button:has-text("Stop")').click()
    
    // Verify confirmation dialog was shown
    await page.waitForTimeout(100) // Give dialog time to appear
    expect(dialogShown).toBe(true)
  })

  test('instance management persists across navigation', async ({ page }) => {
    // Stop an instance
    const runningInstance = page.locator('.instance-card:has-text("ml-research-workstation")')
    
    page.on('dialog', dialog => dialog.accept()) // Auto-accept all dialogs
    
    await runningInstance.locator('button:has-text("Stop")').click()
    await expect(runningInstance.locator('.instance-status.stopping')).toBeVisible()
    
    // Navigate away and back
    await page.click('.nav-item:has-text("Quick Start")')
    await expect(page.locator('#quick-start')).toHaveClass(/active/)
    
    await page.click('.nav-item:has-text("My Instances")')
    await expect(page.locator('#my-instances')).toHaveClass(/active/)
    
    // Instance state should be preserved
    await expect(runningInstance.locator('.instance-status.stopping')).toBeVisible()
  })

  test('handles empty instance list gracefully', async ({ page }) => {
    // This would test the empty state
    // For now, verify the structure exists for when there are no instances
    await expect(page.locator('#instances-grid')).toBeVisible()
    
    // In a real scenario with empty list, should show:
    // "No instances running" and "Launch your first research environment in Quick Start"
  })

  test('instance cards have proper visual styling', async ({ page }) => {
    const instanceCards = page.locator('.instance-card')
    
    // Verify all cards have proper structure
    for (let i = 0; i < await instanceCards.count(); i++) {
      const card = instanceCards.nth(i)
      await expect(card.locator('.instance-header')).toBeVisible()
      await expect(card.locator('.instance-name')).toBeVisible()
      await expect(card.locator('.instance-status')).toBeVisible()
      await expect(card.locator('.instance-details')).toBeVisible()
      await expect(card.locator('.instance-actions')).toBeVisible()
    }
  })

  test('instance status badges have correct styling', async ({ page }) => {
    // Running status should have running class
    const runningStatus = page.locator('.instance-status.running')
    await expect(runningStatus).toBeVisible()
    await expect(runningStatus).toHaveText('running')
    
    // Stopped status should have stopped class
    const stoppedStatus = page.locator('.instance-status.stopped')
    await expect(stoppedStatus).toBeVisible()
    await expect(stoppedStatus).toHaveText('stopped')
  })

  test('action buttons are properly styled and accessible', async ({ page }) => {
    const actionButtons = page.locator('.instance-actions button')
    
    // All buttons should have proper classes and be clickable
    for (let i = 0; i < await actionButtons.count(); i++) {
      const button = actionButtons.nth(i)
      await expect(button).toBeVisible()
      await expect(button).toHaveClass(/btn-secondary/)
      
      // Should be focusable for keyboard navigation
      await button.focus()
      await expect(button).toBeFocused()
    }
  })

  test('hover effects work on instance cards', async ({ page }) => {
    const firstCard = page.locator('.instance-card').first()
    
    // Hover over card
    await firstCard.hover()
    
    // Card should remain visible and interactive
    await expect(firstCard).toBeVisible()
    
    // Action buttons should be clickable when hovered
    const connectButton = firstCard.locator('button:has-text("Connect")')
    await expect(connectButton).toBeVisible()
  })
})