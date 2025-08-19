// Visual regression tests for theme consistency
import { test } from '@playwright/test'
import percySnapshot from '@percy/playwright'

const themes = [
  { name: 'core', description: 'Core theme with professional blue palette' },
  { name: 'dark', description: 'Dark theme for low-light environments' },
  { name: 'academic', description: 'Academic theme with scholarly colors' },
  { name: 'minimal', description: 'Minimal theme with clean design' },
  { name: 'custom', description: 'Customizable theme template' }
]

const viewports = [
  { name: 'desktop', width: 1280, height: 720 },
  { name: 'tablet', width: 768, height: 1024 },
  { name: 'mobile', width: 375, height: 667 }
]

test.describe('Theme Visual Regression', () => {
  themes.forEach(theme => {
    test(`${theme.name} theme - Quick Start section`, async ({ page }) => {
      await page.goto('/')
      
      // Apply theme
      await page.evaluate((themeName) => {
        document.documentElement.setAttribute('data-theme', themeName)
        const themeLink = document.getElementById('theme-link')
        if (themeLink) {
          themeLink.href = `/themes/${themeName}.css`
        }
      }, theme.name)
      
      // Wait for theme to load
      await page.waitForTimeout(500)
      
      // Ensure we're on Quick Start section
      await page.click('.nav-item:has-text("Quick Start")')
      
      // Wait for templates to load
      await page.waitForSelector('.template-card', { timeout: 5000 })
      
      // Take screenshot
      await percySnapshot(page, `${theme.name} theme - Quick Start section`, {
        widths: [1280, 768, 375]
      })
    })

    test(`${theme.name} theme - My Instances section`, async ({ page }) => {
      await page.goto('/')
      
      // Apply theme
      await page.evaluate((themeName) => {
        document.documentElement.setAttribute('data-theme', themeName)
        const themeLink = document.getElementById('theme-link')
        if (themeLink) {
          themeLink.href = `/themes/${themeName}.css`
        }
      }, theme.name)
      
      // Wait for theme to load
      await page.waitForTimeout(500)
      
      // Navigate to My Instances
      await page.click('.nav-item:has-text("My Instances")')
      
      // Wait for instances to load
      await page.waitForSelector('.instance-card', { timeout: 5000 })
      
      // Take screenshot
      await percySnapshot(page, `${theme.name} theme - My Instances section`, {
        widths: [1280, 768, 375]
      })
    })

    test(`${theme.name} theme - Launch form`, async ({ page }) => {
      await page.goto('/')
      
      // Apply theme
      await page.evaluate((themeName) => {
        document.documentElement.setAttribute('data-theme', themeName)
        const themeLink = document.getElementById('theme-link')
        if (themeLink) {
          themeLink.href = `/themes/${themeName}.css`
        }
      }, theme.name)
      
      // Wait for theme to load
      await page.waitForTimeout(500)
      
      // Select a template to show launch form
      await page.click('.template-card:first-child')
      await page.waitForSelector('#launch-form:not(.hidden)', { timeout: 5000 })
      
      // Fill form for better visual
      await page.fill('#instance-name', 'visual-test-instance')
      await page.selectOption('#instance-size', 'L')
      
      // Take screenshot
      await percySnapshot(page, `${theme.name} theme - Launch form`, {
        widths: [1280, 768, 375]
      })
    })

    test(`${theme.name} theme - Settings modal`, async ({ page }) => {
      await page.goto('/')
      
      // Apply theme
      await page.evaluate((themeName) => {
        document.documentElement.setAttribute('data-theme', themeName)
        const themeLink = document.getElementById('theme-link')
        if (themeLink) {
          themeLink.href = `/themes/${themeName}.css`
        }
      }, theme.name)
      
      // Wait for theme to load
      await page.waitForTimeout(500)
      
      // Open settings modal
      await page.click('button[title="Settings"]')
      await page.waitForSelector('#settings-modal:not(.hidden)', { timeout: 5000 })
      
      // Take screenshot
      await percySnapshot(page, `${theme.name} theme - Settings modal`, {
        widths: [1280, 768, 375]
      })
    })
  })

  test('Theme switching visual consistency', async ({ page }) => {
    await page.goto('/')
    
    // Start with core theme
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'core')
    })
    await page.waitForTimeout(500)
    
    // Take initial screenshot
    await percySnapshot(page, 'Theme switching - Core initial state')
    
    // Toggle to dark theme
    await page.click('button[title="Toggle Theme"]')
    await page.waitForTimeout(500)
    
    // Verify dark theme is applied
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-theme') === 'dark'
    })
    
    // Take dark theme screenshot
    await percySnapshot(page, 'Theme switching - Dark theme active')
    
    // Toggle back to core
    await page.click('button[title="Toggle Theme"]')
    await page.waitForTimeout(500)
    
    // Take final screenshot
    await percySnapshot(page, 'Theme switching - Back to core theme')
  })

  test('Responsive layout consistency across themes', async ({ page }) => {
    for (const viewport of viewports) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      
      for (const theme of themes.slice(0, 2)) { // Test core and dark for responsiveness
        await page.goto('/')
        
        // Apply theme
        await page.evaluate((themeName) => {
          document.documentElement.setAttribute('data-theme', themeName)
          const themeLink = document.getElementById('theme-link')
          if (themeLink) {
            themeLink.href = `/themes/${themeName}.css`
          }
        }, theme.name)
        
        await page.waitForTimeout(500)
        
        // Take screenshot for this viewport and theme combination
        await percySnapshot(page, `${theme.name} theme - ${viewport.name} layout`, {
          widths: [viewport.width]
        })
      }
    }
  })

  test('Component state variations', async ({ page }) => {
    await page.goto('/')
    
    // Test template selection states
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'core')
    })
    await page.waitForTimeout(500)
    
    // Normal state
    await percySnapshot(page, 'Component states - Templates normal')
    
    // Selected state
    await page.click('.template-card:first-child')
    await page.waitForTimeout(200)
    await percySnapshot(page, 'Component states - Template selected')
    
    // Test instance states
    await page.click('.nav-item:has-text("My Instances")')
    await page.waitForSelector('.instance-card')
    
    // Instance cards with different states
    await percySnapshot(page, 'Component states - Instance cards')
  })

  test('Loading and empty states', async ({ page }) => {
    await page.goto('/')
    
    // Apply core theme
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'core')
    })
    
    // Test empty states by modifying the DOM
    await page.evaluate(() => {
      // Simulate empty templates
      const templateGrid = document.getElementById('template-grid')
      if (templateGrid) {
        templateGrid.innerHTML = `
          <div class="template-card">
            <div class="text-center">
              <p>No templates available</p>
              <small>Please ensure the daemon is running</small>
            </div>
          </div>
        `
      }
    })
    
    await percySnapshot(page, 'Empty states - No templates available')
    
    // Test empty instances
    await page.click('.nav-item:has-text("My Instances")')
    await page.evaluate(() => {
      const instanceGrid = document.getElementById('instances-grid')
      if (instanceGrid) {
        instanceGrid.innerHTML = `
          <div class="instance-card">
            <div class="text-center">
              <p>No instances running</p>
              <small>Launch your first research environment in Quick Start</small>
            </div>
          </div>
        `
      }
    })
    
    await percySnapshot(page, 'Empty states - No instances running')
  })

  test('Error states visual consistency', async ({ page }) => {
    await page.goto('/')
    
    // Apply dark theme for error state testing
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark')
    })
    await page.waitForTimeout(500)
    
    // Simulate error state
    await page.evaluate(() => {
      const templateGrid = document.getElementById('template-grid')
      if (templateGrid) {
        templateGrid.innerHTML = `
          <div class="template-card">
            <div class="text-center">
              <p>Failed to load templates</p>
              <small>Please check if the daemon is running</small>
              <br><br>
              <button class="btn-secondary">Retry</button>
            </div>
          </div>
        `
      }
    })
    
    await percySnapshot(page, 'Error states - Failed to load templates')
    
    // Test connection error state
    await page.evaluate(() => {
      const status = document.getElementById('connection-status')
      if (status) {
        status.innerHTML = '<span class="status-dot disconnected"></span> Daemon unavailable'
        status.querySelector('.status-dot').classList.remove('connected', 'connecting')
        status.querySelector('.status-dot').classList.add('disconnected')
      }
    })
    
    await percySnapshot(page, 'Error states - Daemon unavailable')
  })
})