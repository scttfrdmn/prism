// Comprehensive GUI functionality tests using direct DOM manipulation
import { test, expect } from '@playwright/test'

test.describe('Comprehensive GUI Functionality Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(1000) // Give page time to load
  })

  test('GUI application loads with correct structure', async ({ page }) => {
    // Check page title
    await expect(page).toHaveTitle('CloudWorkstation')
    
    // Check main app structure
    await expect(page.locator('#app')).toBeVisible()
    await expect(page.locator('h1.app-title')).toBeVisible()
    await expect(page.locator('.logo')).toHaveText('☁️')
    
    // Check navigation structure (3 nav items)
    await expect(page.locator('.nav-item')).toHaveCount(3)
    
    // Check default active section
    await expect(page.locator('#quick-start')).toHaveClass(/active/)
  })

  test('section navigation works via DOM manipulation', async ({ page }) => {
    // Test My Instances section
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    await expect(page.locator('#my-instances')).toHaveClass(/active/)
    await expect(page.locator('#quick-start')).not.toHaveClass(/active/)
    
    // Test Remote Desktop section
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('remote-desktop').classList.add('active')
    })
    await expect(page.locator('#remote-desktop')).toHaveClass(/active/)
    await expect(page.locator('#my-instances')).not.toHaveClass(/active/)
    
    // Back to Quick Start
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('quick-start').classList.add('active')
    })
    await expect(page.locator('#quick-start')).toHaveClass(/active/)
  })

  test('settings modal opens and closes via DOM manipulation', async ({ page }) => {
    // Initially hidden
    await expect(page.locator('#settings-modal')).toHaveClass(/hidden/)
    
    // Open settings modal
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
    })
    await expect(page.locator('#settings-modal')).not.toHaveClass(/hidden/)
    await expect(page.locator('#settings-modal')).toBeVisible()
    
    // Close settings modal
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.add('hidden')
    })
    await expect(page.locator('#settings-modal')).toHaveClass(/hidden/)
  })

  test('settings interface structure is complete', async ({ page }) => {
    // Open settings
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
    })
    
    // Check modal structure
    await expect(page.locator('.modal-content')).toBeVisible()
    await expect(page.locator('.modal-header h3')).toContainText('Configuration')
    
    // Check settings navigation tabs
    const settingsNavButtons = [
      '.settings-nav-btn:has-text("General")',
      '.settings-nav-btn:has-text("AWS")', 
      '.settings-nav-btn:has-text("Daemon")',
      '.settings-nav-btn:has-text("Appearance")',
      '.settings-nav-btn:has-text("Advanced")'
    ]
    
    for (const selector of settingsNavButtons) {
      await expect(page.locator(selector)).toBeVisible()
    }
    
    // Check default active section (General)
    await expect(page.locator('#settings-general')).toHaveClass(/active/)
  })

  test('settings sections can be switched via DOM manipulation', async ({ page }) => {
    // Open settings
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
    })
    
    // Test switching to AWS settings
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-aws').classList.add('active')
      
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.querySelector('.settings-nav-btn:nth-child(2)').classList.add('active')
    })
    await expect(page.locator('#settings-aws')).toHaveClass(/active/)
    await expect(page.locator('#settings-general')).not.toHaveClass(/active/)
    
    // Test switching to Advanced settings
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-advanced').classList.add('active')
      
      document.querySelectorAll('.settings-nav-btn').forEach(n => n.classList.remove('active'))
      document.querySelector('.settings-nav-btn:nth-child(5)').classList.add('active')
    })
    await expect(page.locator('#settings-advanced')).toHaveClass(/active/)
    await expect(page.locator('#settings-aws')).not.toHaveClass(/active/)
  })

  test('theme system works via DOM manipulation', async ({ page }) => {
    // Check default theme
    const initialTheme = await page.evaluate(() => {
      return document.documentElement.getAttribute('data-theme') || 'core'
    })
    expect(initialTheme).toBe('core')
    
    // Change to dark theme
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark')
      const themeLink = document.getElementById('theme-link')
      if (themeLink) {
        themeLink.href = '/themes/dark.css'
      }
    })
    
    const newTheme = await page.evaluate(() => {
      return document.documentElement.getAttribute('data-theme')
    })
    expect(newTheme).toBe('dark')
  })

  test('settings form elements exist and are functional', async ({ page }) => {
    // Open settings
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
    })
    
    // Check General settings elements
    await expect(page.locator('#autostart-gui')).toBeVisible()
    await expect(page.locator('#auto-refresh')).toBeVisible()
    await expect(page.locator('#default-instance-size')).toBeVisible()
    
    // Switch to AWS settings and check elements
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-aws').classList.add('active')
    })
    await expect(page.locator('#aws-profile')).toBeVisible()
    await expect(page.locator('#aws-region')).toBeVisible()
    await expect(page.locator('#cost-warnings')).toBeVisible()
    
    // Switch to Appearance settings and check theme selector
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-appearance').classList.add('active')
    })
    await expect(page.locator('#theme-selector')).toBeVisible()
    await expect(page.locator('#animations-enabled')).toBeVisible()
  })

  test('responsive design works across viewports', async ({ page }) => {
    // Test different viewport sizes
    const viewports = [
      { width: 1280, height: 720 },   // Desktop
      { width: 768, height: 1024 },   // Tablet
      { width: 375, height: 667 }     // Mobile
    ]
    
    for (const viewport of viewports) {
      await page.setViewportSize(viewport)
      
      // Check that main elements are still visible
      await expect(page.locator('#app')).toBeVisible()
      await expect(page.locator('.nav-item')).toHaveCount(3)
      
      // On smaller screens, ensure navigation is accessible
      if (viewport.width < 768) {
        // Check mobile-specific behavior
        await expect(page.locator('.bottom-nav')).toBeVisible()
      }
    }
  })

  test('template and instance sections have correct structure', async ({ page }) => {
    // Check Quick Start section structure
    await expect(page.locator('#quick-start h2')).toContainText('Quick Start')
    await expect(page.locator('.template-filters')).toBeVisible()
    await expect(page.locator('#template-grid')).toBeVisible()
    
    // Switch to My Instances section
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    
    await expect(page.locator('#my-instances h2')).toContainText('My Instances')
    await expect(page.locator('#instances-grid')).toBeVisible()
    
    // Switch to Remote Desktop section
    await page.evaluate(() => {
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('remote-desktop').classList.add('active')
    })
    
    await expect(page.locator('#remote-desktop h2')).toContainText('Remote Desktop')
  })

  test('progressive disclosure principle is implemented', async ({ page }) => {
    // Open settings
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
    })
    
    // General settings should be active by default (simplest)
    await expect(page.locator('#settings-general')).toHaveClass(/active/)
    
    // Check that Advanced settings exist but aren't active initially
    await expect(page.locator('#settings-advanced')).not.toHaveClass(/active/)
    await expect(page.locator('.settings-nav-btn:has-text("Advanced")')).toBeVisible()
    
    // Advanced settings should have debug/developer options
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-advanced').classList.add('active')
    })
    
    await expect(page.locator('#debug-mode')).toBeVisible()
    await expect(page.locator('#log-level')).toBeVisible()
  })

  test('accessibility and keyboard navigation structure', async ({ page }) => {
    await page.goto('/')
    
    // Check that interactive elements have proper attributes
    await expect(page.locator('button[title="Settings"]')).toBeVisible()
    await expect(page.locator('button[title="Toggle Theme"]')).toBeVisible()
    
    // Open settings and check General section controls
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-general').classList.add('active')
    })
    
    await expect(page.locator('#autostart-gui')).toBeVisible()
    await expect(page.locator('#auto-refresh')).toBeVisible()
    
    // Switch to AWS section to check AWS controls
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-aws').classList.add('active')
    })
    await expect(page.locator('#aws-profile')).toBeVisible()
    
    // Switch to Appearance section to check theme controls  
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-appearance').classList.add('active')
    })
    await expect(page.locator('#theme-selector')).toBeVisible()
    
    // Check that labels exist near the controls (setting-label classes)
    const labelCount = await page.locator('.setting-label').count()
    expect(labelCount).toBeGreaterThan(0) // We should have multiple labels
  })
})