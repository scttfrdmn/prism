// Screenshot capture script for persona documentation
// This script captures key GUI screens for the persona walkthroughs
// Run with: npm run test:e2e -- capture-screenshots.spec.js

import { test, expect } from '@playwright/test';
import path from 'path';
import fs from 'fs';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

// ES module equivalents of __dirname
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Output directory for screenshots
const SCREENSHOT_DIR = path.join(__dirname, '../../../../../docs/USER_SCENARIOS/images/01-solo-researcher');

// Ensure output directory exists
test.beforeAll(async () => {
  if (!fs.existsSync(SCREENSHOT_DIR)) {
    fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
  }
  console.log(`📸 Screenshots will be saved to: ${SCREENSHOT_DIR}`);
});

test.describe('GUI Screenshots for Persona Documentation', () => {

  test('01 - Template Gallery (Home Page)', async ({ page }) => {
    // Capture browser console logs
    page.on('console', msg => console.log('BROWSER:', msg.type(), msg.text()));

    // Capture page errors
    page.on('pageerror', error => console.log('PAGE ERROR:', error.message));

    // Capture network requests
    page.on('request', request => {
      console.log('→ REQUEST:', request.method(), request.url());
    });

    // Capture network responses with full details
    page.on('response', async response => {
      const url = response.url();
      const status = response.status();
      console.log('← RESPONSE:', status, url);

      // Log API responses in detail
      if (url.includes('/api/v1/')) {
        try {
          const contentType = response.headers()['content-type'];
          console.log('  Content-Type:', contentType);

          if (contentType && contentType.includes('application/json')) {
            const body = await response.text();
            console.log('  Body length:', body.length);
            console.log('  Body preview:', body.substring(0, 200));

            // Try to parse and show structure
            try {
              const data = JSON.parse(body);
              if (url.includes('/templates')) {
                console.log('  Templates count:', typeof data === 'object' ? Object.keys(data).length : 0);
                console.log('  Templates keys:', typeof data === 'object' ? Object.keys(data).slice(0, 5) : []);
              }
            } catch (e) {
              console.log('  Failed to parse JSON:', e.message);
            }
          }
        } catch (e) {
          console.log('  Failed to read response:', e.message);
        }
      }
    });

    // Navigate to home page
    await page.goto('/');

    // Close Quick Start wizard if it appears
    const skipButton = page.getByText('Skip Tour');
    if (await skipButton.isVisible().catch(() => false)) {
      await skipButton.click();
      await page.waitForTimeout(500);
    }

    // Click Templates in sidebar to show template gallery
    await page.click('text=Templates');

    // Wait for templates to load (increased timeout for test environments where AWS API calls take longer)
    await page.waitForSelector('[data-testid="template-card"]', { timeout: 20000 });

    // Take full page screenshot
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, 'gui-template-gallery.png'),
      fullPage: false
    });

    console.log('✅ Captured: gui-template-gallery.png');
  });

  test('02 - Quick Start Wizard (if available)', async ({ page }) => {
    await page.goto('/');

    // Look for Quick Start wizard button
    const quickStartButton = await page.getByText('Quick Start', { exact: false }).first();

    if (await quickStartButton.isVisible()) {
      await quickStartButton.click();

      // Wait for wizard dialog
      await page.waitForSelector('[role="dialog"]', { timeout: 5000 }).catch(() => {
        console.log('⚠️  No Quick Start wizard dialog found');
      });

      // Capture wizard
      await page.screenshot({
        path: path.join(SCREENSHOT_DIR, 'gui-quick-start-wizard.png'),
        fullPage: false
      });

      console.log('✅ Captured: gui-quick-start-wizard.png');
    } else {
      console.log('⚠️  Quick Start wizard not available yet');
    }
  });

  test('03 - Template Card Detail View', async ({ page }) => {
    await page.goto('/');

    // Wait for templates to load (increased timeout for test environments where AWS API calls take longer)
    await page.waitForSelector('[data-testid="template-card"]', { timeout: 20000 });

    // Click on first template card to show details
    const firstTemplate = await page.locator('[data-testid="template-card"]').first();
    await firstTemplate.click();

    // Wait a moment for any animations
    await page.waitForTimeout(500);

    // Capture with template detail visible
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, 'gui-template-detail.png'),
      fullPage: false
    });

    console.log('✅ Captured: gui-template-detail.png');
  });

  test('04 - Workspaces Tab (Instances List)', async ({ page }) => {
    await page.goto('/');

    // Click on Workspaces/Instances tab
    const workspacesTab = await page.getByText('Workspaces', { exact: false })
      .or(page.getByText('Instances', { exact: false }))
      .first();

    if (await workspacesTab.isVisible()) {
      await workspacesTab.click();
      await page.waitForTimeout(1000); // Wait for tab content to load

      await page.screenshot({
        path: path.join(SCREENSHOT_DIR, 'gui-workspaces-list.png'),
        fullPage: false
      });

      console.log('✅ Captured: gui-workspaces-list.png');
    } else {
      console.log('⚠️  Workspaces tab not found');
    }
  });

  test('05 - Projects Tab (if available)', async ({ page }) => {
    await page.goto('/');

    // Look for Projects tab
    const projectsTab = await page.getByText('Projects', { exact: false }).first();

    if (await projectsTab.isVisible()) {
      await projectsTab.click();
      await page.waitForTimeout(1000);

      await page.screenshot({
        path: path.join(SCREENSHOT_DIR, 'gui-projects-dashboard.png'),
        fullPage: false
      });

      console.log('✅ Captured: gui-projects-dashboard.png');
    } else {
      console.log('⚠️  Projects tab not available');
    }
  });

  test('06 - Storage Tab', async ({ page }) => {
    await page.goto('/');

    // Look for Storage tab
    const storageTab = await page.getByText('Storage', { exact: false }).first();

    if (await storageTab.isVisible()) {
      await storageTab.click();
      await page.waitForTimeout(1000);

      await page.screenshot({
        path: path.join(SCREENSHOT_DIR, 'gui-storage-management.png'),
        fullPage: false
      });

      console.log('✅ Captured: gui-storage-management.png');
    } else {
      console.log('⚠️  Storage tab not available');
    }
  });

  test('07 - Settings/Profiles Tab', async ({ page }) => {
    await page.goto('/');

    // Look for Settings or Profiles tab
    const settingsTab = await page.getByText('Settings', { exact: false })
      .or(page.getByText('Profiles', { exact: false }))
      .first();

    if (await settingsTab.isVisible()) {
      await settingsTab.click();
      await page.waitForTimeout(1000);

      await page.screenshot({
        path: path.join(SCREENSHOT_DIR, 'gui-settings-profiles.png'),
        fullPage: false
      });

      console.log('✅ Captured: gui-settings-profiles.png');
    } else {
      console.log('⚠️  Settings tab not available');
    }
  });

  test('08 - Launch Dialog (Template Selection)', async ({ page }) => {
    await page.goto('/');

    // Wait for templates (increased timeout for test environments where AWS API calls take longer)
    await page.waitForSelector('[data-testid="template-card"]', { timeout: 20000 });

    // Find and click Launch button on first template
    const launchButton = await page.locator('button:has-text("Launch")').first();

    if (await launchButton.isVisible()) {
      await launchButton.click();

      // Wait for launch dialog
      await page.waitForSelector('[role="dialog"]', { timeout: 5000 }).catch(() => {
        console.log('⚠️  Launch dialog not found');
      });

      await page.waitForTimeout(500);

      await page.screenshot({
        path: path.join(SCREENSHOT_DIR, 'gui-launch-dialog.png'),
        fullPage: false
      });

      console.log('✅ Captured: gui-launch-dialog.png');
    } else {
      console.log('⚠️  Launch button not found');
    }
  });

});

test.describe('CLI Screenshots (Terminal Output)', () => {

  test.skip('CLI screenshots require separate terminal capture', async () => {
    console.log(`
📋 CLI Screenshot Instructions:

To capture CLI screenshots, run these commands in your terminal:

1. Start daemon (if not running):
   ./bin/prismd &

2. Capture Quick Start wizard:
   prism init
   # Then use: screencapture -w -o cli-init-wizard.png

3. Capture workspace list:
   prism list
   # Then use: screencapture -w -o cli-list-workspaces.png

4. Capture connection info:
   prism connect <workspace-name>
   # Then use: screencapture -w -o cli-connect-output.png

5. Capture template list:
   prism templates
   # Then use: screencapture -w -o cli-templates-list.png

Save screenshots to: ${SCREENSHOT_DIR}
    `);
  });

});
