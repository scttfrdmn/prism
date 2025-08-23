// Basic smoke test for CloudWorkstation GUI
import { test, expect } from '@playwright/test'

test.describe('Basic Smoke Tests', () => {
  test('application loads successfully', async ({ page }) => {
    // Navigate to the app
    await page.goto('/')
    
    // Check that the app title is visible
    await expect(page.locator('h1.app-title')).toBeVisible()
    await expect(page.locator('.logo')).toHaveText('☁️')
    
    // Check that navigation is present
    await expect(page.locator('.nav-item')).toHaveCount(4)
    
    // Check that the default tab (Launch Instance) is active
    await expect(page.locator('#launch-instance')).toHaveClass(/active/)
  })
  
  test('navigation between tabs works', async ({ page }) => {
    await page.goto('/')
    
    // Click on My Instances tab
    await page.click('.nav-item:has-text("My Instances")')
    await expect(page.locator('#my-instances')).toHaveClass(/active/)
    
    // Click on Settings tab
    await page.click('.nav-item:has-text("Settings")')
    await expect(page.locator('#settings')).toHaveClass(/active/)
    
    // Click back to Launch Instance
    await page.click('.nav-item:has-text("Launch Instance")')
    await expect(page.locator('#launch-instance')).toHaveClass(/active/)
  })
  
  test('connects to daemon API', async ({ page }) => {
    await page.goto('/')
    
    // Navigate to My Instances to trigger API call
    await page.click('.nav-item:has-text("My Instances")')
    
    // Wait for either instances or the no-instances message
    const hasContent = await page.locator('.instance-card, .no-instances').waitFor({ 
      timeout: 5000,
      state: 'visible' 
    }).then(() => true).catch(() => false)
    
    expect(hasContent).toBe(true)
  })
})