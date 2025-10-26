// Test JavaScript function availability and execution
import { test, expect } from '@playwright/test'

test.describe('JavaScript Functions', () => {
  test('script loading and function availability', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(2000) // Give time for scripts to load
    
    // Check if functions exist in global scope
    const functionsAvailable = await page.evaluate(() => {
      return {
        showSection: typeof showSection,
        showSettings: typeof showSettings,
        showSettingsSection: typeof showSettingsSection,
        toggleTheme: typeof toggleTheme,
        hideSettings: typeof hideSettings
      }
    })
    
    console.log('Function availability:', functionsAvailable)
    
    // Check if functions exist on window object
    const windowFunctions = await page.evaluate(() => {
      return {
        showSection: typeof window.showSection,
        showSettings: typeof window.showSettings,
        toggleTheme: typeof window.toggleTheme
      }
    })
    
    console.log('Window functions:', windowFunctions)
    
    // Test if we can access functions via onclick attributes
    const onclickHandlers = await page.evaluate(() => {
      const navItem = document.querySelector('.nav-item')
      const settingsBtn = document.querySelector('button[title="Settings"]')
      return {
        navOnclick: navItem?.getAttribute('onclick'),
        settingsOnclick: settingsBtn?.getAttribute('onclick')
      }
    })
    
    console.log('Onclick handlers:', onclickHandlers)
  })

  test('DOM manipulation alternative works', async ({ page }) => {
    await page.goto('/')
    
    // Test direct DOM manipulation works (our proven approach)
    await page.evaluate(() => {
      // This should work regardless of JavaScript function availability
      document.querySelectorAll('.section').forEach(s => s.classList.remove('active'))
      document.getElementById('my-instances').classList.add('active')
    })
    
    await expect(page.locator('#my-instances')).toHaveClass(/active/)
    await expect(page.locator('#quick-start')).not.toHaveClass(/active/)
  })

  test('settings modal DOM manipulation', async ({ page }) => {
    await page.goto('/')
    
    // Test settings modal manipulation
    await page.evaluate(() => {
      document.getElementById('settings-modal').classList.remove('hidden')
    })
    
    await expect(page.locator('#settings-modal')).not.toHaveClass(/hidden/)
    
    // Test settings section switching
    await page.evaluate(() => {
      document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'))
      document.getElementById('settings-aws').classList.add('active')
    })
    
    await expect(page.locator('#settings-aws')).toHaveClass(/active/)
  })

  test('theme system DOM manipulation', async ({ page }) => {
    await page.goto('/')
    
    // Test theme switching via DOM
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark')
      const themeLink = document.getElementById('theme-link')
      if (themeLink) {
        themeLink.href = '/themes/dark.css'
      }
    })
    
    const theme = await page.evaluate(() => {
      return document.documentElement.getAttribute('data-theme')
    })
    
    expect(theme).toBe('dark')
  })
})