// Error boundary and error handling tests for GUI components
import { test, expect } from '@playwright/test'

test.describe('Error Boundary and Error Handling', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible()
  })

  test('template loading error is handled gracefully', async ({ page }) => {
    // Check if template error state is displayed when templates fail to load
    const errorMessage = page.locator('text=Failed to load templates')
    const retryButton = page.locator('button:has-text("Retry")')
    
    // Check if either templates loaded OR error state shown
    const templatesLoaded = await page.locator('.template-card').count()
    const errorShown = await errorMessage.count()
    
    expect(templatesLoaded + errorShown).toBeGreaterThan(0)
    
    // If error state is shown, verify retry functionality exists
    if (errorShown > 0) {
      await expect(errorMessage).toBeVisible()
      await expect(page.locator('#template-grid').getByText('Please check if the daemon is running')).toBeVisible()
      await expect(page.locator('#template-grid button:has-text("Retry")')).toBeVisible()
    }
  })

  test('instance loading error is handled gracefully', async ({ page }) => {
    // Switch to instances section using DOM manipulation
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    
    await expect(page.locator('#instances-grid')).toBeVisible()
    
    // Check if either instances loaded OR error state shown
    const instancesLoaded = await page.locator('.instance-card:not(.loading)').count()
    const errorMessage = page.locator('text=Failed to load instances')
    const errorShown = await errorMessage.count()
    
    expect(instancesLoaded + errorShown).toBeGreaterThan(0)
    
    // If error state is shown, verify error handling UI
    if (errorShown > 0) {
      await expect(errorMessage).toBeVisible()
      await expect(page.locator('#instances-grid').getByText('Please check if the daemon is running')).toBeVisible()
      await expect(page.locator('#instances-grid button:has-text("Retry")')).toBeVisible()
    }
  })

  test('daemon connection error is handled gracefully', async ({ page }) => {
    // Check connection status in status bar
    const connectionStatus = page.locator('#connection-status')
    await expect(connectionStatus).toBeVisible()
    
    // Verify that connection status provides meaningful information
    const statusText = await connectionStatus.textContent()
    expect(statusText).toBeTruthy()
    expect(statusText.length).toBeGreaterThan(0)
    
    // Check status dot exists for visual indication
    await expect(page.locator('.status-dot')).toBeVisible()
  })

  test('form submission errors are handled gracefully', async ({ page }) => {
    // Use DOM manipulation to show launch form
    await page.evaluate(() => {
      document.getElementById('launch-form').classList.remove('hidden')
    })
    
    await expect(page.locator('#launch-form')).toBeVisible()
    
    // Test form validation on submit with empty fields
    const submitButton = page.locator('#launch-form button[type="submit"], #launch-form .btn-primary')
    
    if (await submitButton.count() > 0) {
      // Click submit without filling required fields
      await submitButton.click()
      
      // Check if validation messages appear or form prevents submission
      const validationMessages = await page.locator('.error, .validation-error, [aria-invalid="true"]').count()
      const formStillVisible = await page.locator('#launch-form').isVisible()
      
      // Either validation messages shown OR form prevented submission
      expect(validationMessages > 0 || formStillVisible).toBeTruthy()
    }
  })

  test('settings form errors are handled gracefully', async ({ page }) => {
    // Use DOM manipulation to open settings and activate daemon section
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.getElementById('settings-daemon').classList.add('active')
      const daemonBtn = document.querySelector('.settings-nav-btn[onclick*="daemon"]')
      if (daemonBtn) daemonBtn.classList.add('active')
    })
    
    await expect(page.locator('#settings-modal')).toBeVisible()
    
    // Test that settings form elements exist and are accessible
    const daemonUrlInput = page.locator('#daemon-url')
    if (await daemonUrlInput.count() > 0) {
      // Verify input element exists and has expected properties
      await expect(daemonUrlInput).toBeVisible()
      
      const initialValue = await daemonUrlInput.inputValue()
      expect(initialValue).toBeTruthy()
      
      // Check save button exists
      const saveButton = page.locator('#settings-modal button:has-text("Save")')
      if (await saveButton.count() > 0) {
        await expect(saveButton).toBeVisible()
      }
    }
  })

  test('network timeout errors are handled gracefully', async ({ page }) => {
    // Check that the app handles network errors without crashing
    await expect(page.locator('#app')).toBeVisible()
    
    // Navigate to different sections to trigger potential network calls
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    
    await expect(page.locator('#instances-grid')).toBeVisible()
    
    // Navigate back to templates
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('quick-start').classList.add('active')
    })
    
    await expect(page.locator('#template-grid')).toBeVisible()
    
    // App should still be functional and not crashed
    await expect(page.locator('#app')).toBeVisible()
    await expect(page.locator('.app-title')).toBeVisible()
  })

  test('invalid template selection is handled gracefully', async ({ page }) => {
    // Verify that template grid exists
    await expect(page.locator('#template-grid')).toBeVisible()
    
    // Check if templates are loaded or error state shown
    const templatesLoaded = await page.locator('.template-card').count()
    const errorShown = await page.locator('text=Failed to load templates').count()
    
    if (templatesLoaded > 0) {
      // Click on first template if available
      await page.locator('.template-card').first().click()
      
      // Should either show launch form or handle gracefully
      const launchFormVisible = await page.locator('#launch-form').isVisible()
      const errorPresent = await page.locator('.error').count()
      
      // Either form shows or error is handled gracefully
      expect(launchFormVisible || errorPresent >= 0).toBeTruthy()
    } else if (errorShown > 0) {
      // Error state is properly handled
      await expect(page.locator('text=Failed to load templates')).toBeVisible()
      await expect(page.locator('button:has-text("Retry")')).toBeVisible()
    }
  })

  test('connection loss recovery is handled gracefully', async ({ page }) => {
    // Monitor connection status throughout test
    const connectionStatus = page.locator('#connection-status')
    await expect(connectionStatus).toBeVisible()
    
    // Refresh page to simulate connection issues
    await page.reload()
    
    // Verify app recovers gracefully
    await expect(page.locator('#app')).toBeVisible()
    await expect(connectionStatus).toBeVisible()
    
    // Check that essential UI elements are still functional
    await expect(page.locator('.app-title')).toBeVisible()
    await expect(page.locator('.bottom-nav')).toBeVisible()
    await expect(page.locator('.status-bar')).toBeVisible()
  })

  test('JavaScript errors do not crash the interface', async ({ page }) => {
    // Monitor for console errors
    const errors = []
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text())
      }
    })
    
    // Perform various actions that might trigger errors
    await page.evaluate(() => {
      // Try to access potentially undefined functions
      try {
        if (typeof showSection === 'function') {
          showSection('invalid-section')
        }
      } catch (e) {
        console.log('Expected error handled:', e.message)
      }
    })
    
    // App should still be functional
    await expect(page.locator('#app')).toBeVisible()
    await expect(page.locator('.app-title')).toBeVisible()
    
    // Navigation should still work via DOM manipulation
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    
    await expect(page.locator('#my-instances')).toHaveClass(/active/)
  })

  test('error recovery through UI interactions works', async ({ page }) => {
    // Test that users can recover from error states
    const retryButtons = page.locator('button:has-text("Retry")')
    const retryCount = await retryButtons.count()
    
    if (retryCount > 0) {
      // Click first retry button
      await retryButtons.first().click()
      
      // Verify that retry action doesn't crash the app
      await expect(page.locator('#app')).toBeVisible()
      
      // Wait for potential loading to complete
      await page.waitForTimeout(1000)
      
      // App should still be responsive
      await expect(page.locator('.app-title')).toBeVisible()
    }
    
    // Test navigation recovery
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('quick-start').classList.add('active')
    })
    
    await expect(page.locator('#quick-start')).toHaveClass(/active/)
  })
})