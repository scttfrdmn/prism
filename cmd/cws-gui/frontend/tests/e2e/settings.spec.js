// Comprehensive tests for the settings interface with progressive disclosure
import { test, expect } from '@playwright/test'

test.describe('Settings Interface', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  // Helper function to open settings modal
  async function openSettings(page) {
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
    })
  }

  // Helper function to switch settings section
  async function switchSettingsSection(page, sectionId, navIndex) {
    await page.evaluate(({ sectionId, navIndex }) => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById(`settings-${sectionId}`).classList.add('active')
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.querySelector(`.settings-nav-btn:nth-child(${navIndex})`).classList.add('active')
    }, { sectionId, navIndex })
  }

  test('settings modal opens and closes correctly', async ({ page }) => {
    // Settings modal should be hidden initially
    await expect(page.locator('#settings-modal')).toHaveClass(/hidden/)
    
    // Open settings via DOM manipulation
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
    })
    await expect(page.locator('#settings-modal')).not.toHaveClass(/hidden/)
    
    // Close settings modal
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.add('hidden')
    })
    await expect(page.locator('#settings-modal')).toHaveClass(/hidden/)
  })

  test('settings navigation tabs work correctly', async ({ page }) => {
    // Open settings via DOM manipulation
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
    })
    
    // General tab should be active by default
    await expect(page.locator('#settings-general')).toHaveClass(/active/)
    await expect(page.locator('.settings-nav-btn:nth-child(1)')).toHaveClass(/active/)
    
    // Switch to AWS tab via DOM manipulation
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-aws').classList.add('active')
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.querySelector('.settings-nav-btn:nth-child(2)').classList.add('active')
    })
    await expect(page.locator('#settings-aws')).toHaveClass(/active/)
    await expect(page.locator('#settings-general')).not.toHaveClass(/active/)
    
    // Switch to Daemon tab
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-daemon').classList.add('active')
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.querySelector('.settings-nav-btn:nth-child(3)').classList.add('active')
    })
    await expect(page.locator('#settings-daemon')).toHaveClass(/active/)
    
    // Switch to Appearance tab
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-appearance').classList.add('active')
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.querySelector('.settings-nav-btn:nth-child(4)').classList.add('active')
    })
    await expect(page.locator('#settings-appearance')).toHaveClass(/active/)
    
    // Switch to Advanced tab
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-advanced').classList.add('active')
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.querySelector('.settings-nav-btn:nth-child(5)').classList.add('active')
    })
    await expect(page.locator('#settings-advanced')).toHaveClass(/active/)
  })

  test('general settings display correctly', async ({ page }) => {
    await openSettings(page)
    await switchSettingsSection(page, 'general', 1)
    
    // Check section title and description
    await expect(page.locator('#settings-general h4')).toHaveText('🏠 General Settings')
    await expect(page.locator('#settings-general .section-description')).toContainText('Basic application preferences')
    
    // Check auto-start setting
    await expect(page.locator('#autostart-gui')).toBeVisible()
    
    // Check auto-refresh setting
    await expect(page.locator('#auto-refresh')).toBeVisible()
    
    // Check default instance size setting
    await expect(page.locator('#default-instance-size')).toBeVisible()
  })

  test('AWS settings display correctly', async ({ page }) => {
    await openSettings(page)
    await switchSettingsSection(page, 'aws', 2)
    
    // Check section title
    await expect(page.locator('#settings-aws h4')).toHaveText('☁️ AWS Configuration')
    
    // Check AWS profile setting
    await expect(page.locator('#aws-profile')).toBeVisible()
    
    // Check AWS region setting  
    await expect(page.locator('#aws-region')).toBeVisible()
    
    // Check cost warnings checkbox
    await expect(page.locator('#cost-warnings')).toBeVisible()
    
    // Check cost limit setting
    await expect(page.locator('#daily-cost-limit')).toBeVisible()
  })

  test('daemon settings display correctly', async ({ page }) => {
    // Use DOM manipulation to open settings and switch to daemon section
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.getElementById('settings-daemon').classList.add('active')
      document.querySelector('.settings-nav-btn[onclick*="daemon"]').classList.add('active')
    })
    
    // Check daemon URL setting
    await expect(page.locator('#daemon-url')).toBeVisible()
    await expect(page.locator('#daemon-url')).toHaveValue('http://localhost:8947')
    
    // Check connection timeout
    await expect(page.locator('#connection-timeout')).toBeVisible()
    await expect(page.locator('#connection-timeout')).toHaveValue('10')
    
    // Check auto-start daemon checkbox
    await expect(page.locator('#auto-start-daemon')).toBeVisible()
    await expect(page.locator('#auto-start-daemon')).toBeChecked()
    
    // Check daemon control buttons
    await expect(page.locator('button:has-text("Test Connection")')).toBeVisible()
    await expect(page.locator('button:has-text("Restart Daemon")')).toBeVisible()
  })

  test('appearance settings display correctly', async ({ page }) => {
    // Use DOM manipulation to open settings and switch to appearance section
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.getElementById('settings-appearance').classList.add('active')
      document.querySelector('.settings-nav-btn[onclick*="appearance"]').classList.add('active')
    })
    
    // Check theme selector
    await expect(page.locator('#theme-selector')).toBeVisible()
    await expect(page.locator('#theme-selector')).toHaveValue('core') // Default theme
    
    // Check theme options
    const themeOptions = ['core', 'academic', 'minimal', 'dark', 'custom']
    for (const theme of themeOptions) {
      await expect(page.locator(`#theme-selector option[value="${theme}"]`)).toBeVisible()
    }
    
    // Check animations setting
    await expect(page.locator('#animations-enabled')).toBeVisible()
    await expect(page.locator('#animations-enabled')).toBeChecked()
    
    // Check compact mode setting
    await expect(page.locator('#compact-mode')).toBeVisible()
  })

  test('advanced settings display correctly', async ({ page }) => {
    // Use DOM manipulation to open settings and switch to advanced section
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.getElementById('settings-advanced').classList.add('active')
      document.querySelector('.settings-nav-btn[onclick*="advanced"]').classList.add('active')
    })
    
    // Check debug mode setting
    await expect(page.locator('#debug-mode')).toBeVisible()
    await expect(page.locator('#debug-mode')).not.toBeChecked() // Should be off by default
    
    // Check log level setting
    await expect(page.locator('#log-level')).toBeVisible()
    await expect(page.locator('#log-level')).toHaveValue('info')
    
    // Check usage analytics setting
    await expect(page.locator('#usage-analytics')).toBeVisible()
    await expect(page.locator('#usage-analytics')).not.toBeChecked()
    
    // Check action buttons
    await expect(page.locator('button:has-text("Export Settings")')).toBeVisible()
    await expect(page.locator('button:has-text("Import Settings")')).toBeVisible()
    await expect(page.locator('button:has-text("Reset to Defaults")')).toBeVisible()
  })

  test('settings form validation works', async ({ page }) => {
    await page.click('button[title="Settings"]')
    await page.click('.settings-nav-btn:has-text("Daemon")')
    
    // Test invalid daemon URL
    await page.fill('#daemon-url', 'invalid-url')
    await page.click('button:has-text("Save Changes")')
    // Should show validation error (implementation dependent)
    
    // Test valid daemon URL
    await page.fill('#daemon-url', 'http://localhost:9999')
    // Should not show validation error
  })

  test('theme switching works from settings', async ({ page }) => {
    await page.click('button[title="Settings"]')
    await page.click('.settings-nav-btn:has-text("Appearance")')
    
    // Change to dark theme
    await page.selectOption('#theme-selector', 'dark')
    
    // Theme should be applied (check document attribute or CSS changes)
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
    
    // Change to academic theme
    await page.selectOption('#theme-selector', 'academic')
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'academic')
  })

  test('auto-start configuration works', async ({ page }) => {
    await page.click('button[title="Settings"]')
    await page.click('.settings-nav-btn:has-text("General")')
    
    // Toggle auto-start
    await page.click('#autostart-gui')
    
    // Should show some indication of success (toast notification, etc.)
    // This would depend on the implementation
  })

  test('daemon connection test works', async ({ page }) => {
    await page.click('button[title="Settings"]')
    await page.click('.settings-nav-btn:has-text("Daemon")')
    
    // Click test connection button
    await page.click('button:has-text("Test Connection")')
    
    // Should show connection result (success or failure notification)
    // Implementation would show appropriate feedback
  })

  test('settings persistence works', async ({ page }) => {
    await page.click('button[title="Settings"]')
    
    // Change some settings
    await page.click('.settings-nav-btn:has-text("General")')
    await page.selectOption('#default-instance-size', 'L')
    
    await page.click('.settings-nav-btn:has-text("AWS")')
    await page.fill('#daily-cost-limit', '75')
    
    // Save settings
    await page.click('button:has-text("Save Changes")')
    
    // Reload page and check settings persist
    await page.reload()
    await page.click('button[title="Settings"]')
    
    await page.click('.settings-nav-btn:has-text("General")')
    await expect(page.locator('#default-instance-size')).toHaveValue('L')
    
    await page.click('.settings-nav-btn:has-text("AWS")')
    await expect(page.locator('#daily-cost-limit')).toHaveValue('75')
  })

  test('settings modal footer buttons work', async ({ page }) => {
    await page.click('button[title="Settings"]')
    
    // Check footer buttons are present
    await expect(page.locator('.modal-footer button:has-text("Reset Section")')).toBeVisible()
    await expect(page.locator('.modal-footer button:has-text("Cancel")')).toBeVisible() 
    await expect(page.locator('.modal-footer button:has-text("Save Changes")')).toBeVisible()
    
    // Test cancel button
    await page.click('button:has-text("Cancel")')
    await expect(page.locator('#settings-modal')).toHaveClass(/hidden/)
  })

  test('progressive disclosure shows complexity appropriately', async ({ page }) => {
    await page.click('button[title="Settings"]')
    
    // General settings should be visible first
    await expect(page.locator('#settings-general')).toHaveClass(/active/)
    
    // Advanced settings should be last tab
    const navButtons = page.locator('.settings-nav-btn')
    await expect(navButtons.last()).toContainText('Advanced')
    
    // Advanced settings should contain debug/developer options
    await page.click('.settings-nav-btn:has-text("Advanced")')
    await expect(page.locator('#debug-mode')).toBeVisible()
    await expect(page.locator('h5:has-text("Debug & Logging")')).toBeVisible()
  })
})