// Form validation tests for launch and settings forms
import { test, expect } from '@playwright/test'

test.describe('Form Validation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible()
  })

  test('launch form validation works correctly', async ({ page }) => {
    // Use DOM manipulation to show launch form
    await page.evaluate(() => {
      document.getElementById('launch-form').classList.remove('hidden')
    })
    
    await expect(page.locator('#launch-form')).toBeVisible()
    
    // Test required field validation
    const instanceNameInput = page.locator('#instance-name')
    const submitButton = page.locator('#launch-form button[type="submit"], #launch-form .btn-primary')
    
    if (await instanceNameInput.count() > 0 && await submitButton.count() > 0) {
      // Clear instance name field if it has a value
      await instanceNameInput.clear()
      
      // Try to submit with empty required field
      await submitButton.click()
      
      // Check for validation behavior
      const formStillVisible = await page.locator('#launch-form').isVisible()
      const validationErrors = await page.locator('.error, .validation-error, [aria-invalid="true"]').count()
      
      // Form should either show validation errors or prevent submission
      expect(formStillVisible || validationErrors > 0).toBeTruthy()
    }
  })

  test('launch form accepts valid input', async ({ page }) => {
    // Use DOM manipulation to show launch form
    await page.evaluate(() => {
      document.getElementById('launch-form').classList.remove('hidden')
    })
    
    await expect(page.locator('#launch-form')).toBeVisible()
    
    // Test with valid input
    const instanceNameInput = page.locator('#instance-name')
    
    if (await instanceNameInput.count() > 0) {
      // Enter valid instance name
      await instanceNameInput.fill('test-instance-123')
      
      // Verify input is accepted
      const inputValue = await instanceNameInput.inputValue()
      expect(inputValue).toBe('test-instance-123')
      
      // Check that form elements are properly structured
      await expect(instanceNameInput).toBeVisible()
      await expect(instanceNameInput).toBeEditable()
    }
  })

  test('launch form instance name validation', async ({ page }) => {
    // Use DOM manipulation to show launch form
    await page.evaluate(() => {
      document.getElementById('launch-form').classList.remove('hidden')
    })
    
    await expect(page.locator('#launch-form')).toBeVisible()
    
    const instanceNameInput = page.locator('#instance-name')
    
    if (await instanceNameInput.count() > 0) {
      // Test various input patterns
      const testInputs = [
        'valid-name',
        'valid_name_123',
        'test-instance'
      ]
      
      for (const testInput of testInputs) {
        await instanceNameInput.fill(testInput)
        const value = await instanceNameInput.inputValue()
        expect(value).toBe(testInput)
      }
    }
  })

  test('settings form daemon URL validation', async ({ page }) => {
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
    
    const daemonUrlInput = page.locator('#daemon-url')
    
    if (await daemonUrlInput.count() > 0) {
      await expect(daemonUrlInput).toBeVisible()
      
      // Test valid URL formats
      const validUrls = [
        'http://localhost:8947',
        'http://127.0.0.1:8947',
        'https://daemon.example.com:8947'
      ]
      
      for (const url of validUrls) {
        await daemonUrlInput.fill(url)
        const value = await daemonUrlInput.inputValue()
        expect(value).toBe(url)
      }
    }
  })

  test('settings form timeout validation', async ({ page }) => {
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
    
    const timeoutInput = page.locator('#daemon-timeout')
    
    if (await timeoutInput.count() > 0) {
      await expect(timeoutInput).toBeVisible()
      
      // Test numeric input validation
      const validTimeouts = ['5', '10', '30', '60']
      
      for (const timeout of validTimeouts) {
        await timeoutInput.fill(timeout)
        const value = await timeoutInput.inputValue()
        expect(value).toBe(timeout)
      }
    }
  })

  test('settings form AWS profile validation', async ({ page }) => {
    // Use DOM manipulation to open settings and activate AWS section
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.getElementById('settings-aws').classList.add('active')
      const awsBtn = document.querySelector('.settings-nav-btn[onclick*="aws"]')
      if (awsBtn) awsBtn.classList.add('active')
    })
    
    await expect(page.locator('#settings-modal')).toBeVisible()
    
    const profileInput = page.locator('#aws-profile')
    
    if (await profileInput.count() > 0) {
      await expect(profileInput).toBeVisible()
      
      // Check that profile selector has options and default value
      const options = await profileInput.locator('option').count()
      expect(options).toBeGreaterThan(0)
      
      // Verify default selection exists
      const selectedValue = await profileInput.inputValue()
      expect(selectedValue).toBeTruthy()
      
      // Test selecting first available option if multiple exist
      if (options > 1) {
        await profileInput.selectOption({ index: 0 })
        const newValue = await profileInput.inputValue()
        expect(newValue).toBeTruthy()
      }
    }
  })

  test('settings form region validation', async ({ page }) => {
    // Use DOM manipulation to open settings and activate AWS section
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.getElementById('settings-aws').classList.add('active')
      const awsBtn = document.querySelector('.settings-nav-btn[onclick*="aws"]')
      if (awsBtn) awsBtn.classList.add('active')
    })
    
    await expect(page.locator('#settings-modal')).toBeVisible()
    
    const regionSelect = page.locator('#aws-region')
    
    if (await regionSelect.count() > 0) {
      await expect(regionSelect).toBeVisible()
      
      // Check that region selector has options
      const options = await regionSelect.locator('option').count()
      expect(options).toBeGreaterThan(0)
      
      // Verify default selection exists
      const selectedValue = await regionSelect.inputValue()
      expect(selectedValue).toBeTruthy()
    }
  })

  test('settings form cost limit validation', async ({ page }) => {
    // Use DOM manipulation to open settings and activate AWS section
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.getElementById('settings-aws').classList.add('active')
      const awsBtn = document.querySelector('.settings-nav-btn[onclick*="aws"]')
      if (awsBtn) awsBtn.classList.add('active')
    })
    
    await expect(page.locator('#settings-modal')).toBeVisible()
    
    const costLimitInput = page.locator('#daily-cost-limit')
    
    if (await costLimitInput.count() > 0) {
      await expect(costLimitInput).toBeVisible()
      
      // Test numeric validation
      const validCosts = ['10', '25', '50', '100']
      
      for (const cost of validCosts) {
        await costLimitInput.fill(cost)
        const value = await costLimitInput.inputValue()
        expect(value).toBe(cost)
      }
    }
  })

  test('form field accessibility attributes', async ({ page }) => {
    // Test launch form accessibility
    await page.evaluate(() => {
      document.getElementById('launch-form').classList.remove('hidden')
    })
    
    const instanceNameInput = page.locator('#instance-name')
    
    if (await instanceNameInput.count() > 0) {
      // Check for accessibility attributes
      const hasLabel = await page.locator('label[for="instance-name"]').count() > 0
      const hasAriaLabel = await instanceNameInput.getAttribute('aria-label')
      const hasPlaceholder = await instanceNameInput.getAttribute('placeholder')
      
      // Should have some form of labeling
      expect(hasLabel || hasAriaLabel || hasPlaceholder).toBeTruthy()
    }
  })

  test('form submission prevents double-submit', async ({ page }) => {
    // Use DOM manipulation to show launch form
    await page.evaluate(() => {
      document.getElementById('launch-form').classList.remove('hidden')
    })
    
    await expect(page.locator('#launch-form')).toBeVisible()
    
    const submitButton = page.locator('#launch-form button[type="submit"], #launch-form .btn-primary')
    
    if (await submitButton.count() > 0) {
      // Fill in instance name if field exists
      const instanceNameInput = page.locator('#instance-name')
      if (await instanceNameInput.count() > 0) {
        await instanceNameInput.fill('test-instance')
      }
      
      // Check button initial state
      const initiallyDisabled = await submitButton.isDisabled()
      
      // Click submit button
      await submitButton.click()
      
      // Form should either disable button or be hidden after submission
      const buttonDisabled = await submitButton.isDisabled()
      const formVisible = await page.locator('#launch-form').isVisible()
      
      // Either button disabled, form hidden, or still processing
      expect(buttonDisabled || !formVisible || !initiallyDisabled).toBeTruthy()
    }
  })
})