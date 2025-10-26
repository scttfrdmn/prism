// Debug test to understand GUI testing issues
import { test, expect } from '@playwright/test'

test.describe('Debug Tests', () => {
  test('check what elements exist on the page', async ({ page }) => {
    await page.goto('/')
    
    // Wait for page to load
    await page.waitForTimeout(2000)
    
    // Check what's actually on the page
    const title = await page.title()
    console.log('Page title:', title)
    
    // Check if main elements exist
    const appElement = page.locator('#app')
    await expect(appElement).toBeVisible()
    
    // Check for header elements
    const headerTitle = page.locator('h1.app-title')
    await expect(headerTitle).toBeVisible()
    
    // Check for navigation
    const navItems = page.locator('.nav-item')
    const navCount = await navItems.count()
    console.log('Navigation items count:', navCount)
    
    // Check for settings button
    const settingsButton = page.locator('button[title="Settings"]')
    const settingsExists = await settingsButton.count()
    console.log('Settings button exists:', settingsExists > 0)
    
    // Check if settings modal exists (should be hidden)
    const settingsModal = page.locator('#settings-modal')
    const modalExists = await settingsModal.count()
    console.log('Settings modal exists:', modalExists > 0)
    
    if (modalExists > 0) {
      const modalClasses = await settingsModal.getAttribute('class')
      console.log('Settings modal classes:', modalClasses)
    }
    
    // Try to list all JavaScript functions available
    const jsContext = await page.evaluate(() => {
      return {
        showSection: typeof showSection !== 'undefined',
        showSettings: typeof showSettings !== 'undefined',
        hideSettings: typeof hideSettings !== 'undefined',
        toggleTheme: typeof toggleTheme !== 'undefined'
      }
    })
    console.log('JavaScript functions available:', jsContext)
  })

  test('test direct settings modal manipulation', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(2000)
    
    // Try to open settings modal directly via DOM manipulation
    await page.evaluate(() => {
      const modal = document.getElementById('settings-modal')
      if (modal) {
        modal.classList.remove('hidden')
      }
    })
    
    // Check if modal is now visible
    const settingsModal = page.locator('#settings-modal')
    await expect(settingsModal).not.toHaveClass(/hidden/)
  })

  test('test theme toggle functionality', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(2000)
    
    // Check current theme
    const currentTheme = await page.evaluate(() => {
      return document.documentElement.getAttribute('data-theme') || 'default'
    })
    console.log('Current theme:', currentTheme)
    
    // Try to call toggleTheme function
    const themeToggleAvailable = await page.evaluate(() => {
      return typeof toggleTheme !== 'undefined'
    })
    console.log('toggleTheme function available:', themeToggleAvailable)
    
    if (themeToggleAvailable) {
      await page.evaluate(() => toggleTheme())
      
      const newTheme = await page.evaluate(() => {
        return document.documentElement.getAttribute('data-theme') || 'default'
      })
      console.log('New theme after toggle:', newTheme)
    }
  })
})