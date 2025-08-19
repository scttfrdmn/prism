// End-to-end tests for complete instance launch workflow
import { test, expect } from '@playwright/test'

test.describe('Instance Launch Workflow', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the GUI application
    await page.goto('/')
    
    // Wait for initial load
    await expect(page.locator('#app')).toBeVisible()
    await expect(page.locator('.header')).toBeVisible()
  })

  test('complete launch process from template selection to instance dashboard', async ({ page }) => {
    // Step 1: Verify we're on Quick Start section by default
    await expect(page.locator('#quick-start')).toHaveClass(/active/)
    await expect(page.locator('h2:has-text("Quick Start")')).toBeVisible()
    
    // Step 2: Wait for templates to load
    await expect(page.locator('.template-card')).toHaveCount(3) // We have 3 mock templates
    await expect(page.locator('text=Python Machine Learning')).toBeVisible()
    await expect(page.locator('text=R Research Environment')).toBeVisible()
    
    // Step 3: Select a template
    await page.click('.template-card:has-text("Python Machine Learning")')
    
    // Verify template selection
    await expect(page.locator('.template-card.selected')).toBeVisible()
    await expect(page.locator('.template-card.selected:has-text("Python Machine Learning")')).toBeVisible()
    
    // Step 4: Verify launch form appears (Progressive Disclosure)
    await expect(page.locator('#launch-form')).toBeVisible()
    await expect(page.locator('#launch-form')).not.toHaveClass('hidden')
    await expect(page.locator('#selected-template-name')).toHaveText('Python Machine Learning (Simplified)')
    
    // Step 5: Fill out launch form
    const instanceNameInput = page.locator('#instance-name')
    await expect(instanceNameInput).toBeVisible()
    
    // Clear auto-generated name and enter custom name
    await instanceNameInput.clear()
    await instanceNameInput.fill('test-ml-workstation')
    
    // Select instance size
    await page.selectOption('#instance-size', 'L')
    
    // Step 6: Launch instance
    const launchButton = page.locator('#launch-btn')
    await expect(launchButton).toBeVisible()
    await expect(launchButton).toHaveText(/Launch Research Environment/)
    
    await launchButton.click()
    
    // Step 7: Verify launch success (mock will succeed)
    await expect(page.locator('text=Successfully launched')).toBeVisible({ timeout: 10000 })
    
    // Step 8: Verify automatic navigation to My Instances
    await expect(page.locator('#my-instances')).toHaveClass(/active/)
    
    // Step 9: Verify instance appears in dashboard
    await expect(page.locator('text=test-ml-workstation')).toBeVisible()
    await expect(page.locator('.instance-card:has-text("test-ml-workstation")')).toBeVisible()
    
    // Step 10: Verify instance details are displayed
    const newInstanceCard = page.locator('.instance-card:has-text("test-ml-workstation")')
    await expect(newInstanceCard.locator('.instance-status.running')).toBeVisible()
    await expect(newInstanceCard.locator('text=Connect')).toBeVisible()
    await expect(newInstanceCard.locator('text=Stop')).toBeVisible()
  })

  test('form validation prevents invalid launches', async ({ page }) => {
    // Select template
    await page.click('.template-card:has-text("Python Machine Learning")')
    await expect(page.locator('#launch-form')).toBeVisible()
    
    // Try to launch with empty instance name
    await page.locator('#instance-name').clear()
    await page.click('#launch-btn')
    
    // Should show validation error (mocked as alert for now)
    page.on('dialog', dialog => {
      expect(dialog.message()).toContain('Please enter an instance name')
      dialog.accept()
    })
    
    // Try with invalid instance name
    await page.locator('#instance-name').fill('Invalid Name With Spaces!')
    await page.click('#launch-btn')
    
    page.on('dialog', dialog => {
      expect(dialog.message()).toContain('lowercase letters, numbers, and hyphens')
      dialog.accept()
    })
  })

  test('template switching updates launch form correctly', async ({ page }) => {
    // Select first template
    await page.click('.template-card:has-text("Python Machine Learning")')
    await expect(page.locator('#selected-template-name')).toHaveText('Python Machine Learning (Simplified)')
    
    // Switch to second template
    await page.click('.template-card:has-text("R Research Environment")')
    
    // Verify only one template is selected
    await expect(page.locator('.template-card.selected')).toHaveCount(1)
    await expect(page.locator('.template-card.selected:has-text("R Research Environment")')).toBeVisible()
    
    // Verify launch form updates
    await expect(page.locator('#selected-template-name')).toHaveText('R Research Environment (Simplified)')
  })

  test('clear selection hides launch form', async ({ page }) => {
    // Select template and verify form shows
    await page.click('.template-card:has-text("Python Machine Learning")')
    await expect(page.locator('#launch-form')).toBeVisible()
    
    // Clear selection
    await page.click('button:has-text("Change Template")')
    
    // Verify form is hidden and no template selected
    await expect(page.locator('#launch-form')).toHaveClass('hidden')
    await expect(page.locator('.template-card.selected')).toHaveCount(0)
  })

  test('auto-generated instance names are smart', async ({ page }) => {
    // Select template
    await page.click('.template-card:has-text("Python Machine Learning")')
    
    // Check that instance name is auto-generated
    const instanceNameInput = page.locator('#instance-name')
    const autoName = await instanceNameInput.inputValue()
    
    // Should contain template-based prefix and date suffix
    expect(autoName).toMatch(/^python-machine-learning.*-\d{4}$/)
    expect(autoName.length).toBeLessThanOrEqual(25) // Reasonable length limit
  })

  test('instance size selection works correctly', async ({ page }) => {
    await page.click('.template-card:has-text("Python Machine Learning")')
    
    const sizeSelect = page.locator('#instance-size')
    
    // Verify default selection
    await expect(sizeSelect).toHaveValue('M')
    
    // Test all size options
    const sizes = ['S', 'M', 'L', 'XL']
    for (const size of sizes) {
      await sizeSelect.selectOption(size)
      await expect(sizeSelect).toHaveValue(size)
    }
  })
})