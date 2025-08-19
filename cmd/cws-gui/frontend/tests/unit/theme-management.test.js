// Unit tests for theme management functionality
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { screen } from '@testing-library/dom'

// Mock theme management functions
function applyTheme(themeName) {
  // Update theme link
  const themeLink = document.getElementById('theme-link')
  if (themeLink) {
    themeLink.href = `/themes/${themeName}.css`
  }
  
  // Update document attribute for theme-specific styling
  document.documentElement.setAttribute('data-theme', themeName)
  
  // Update theme icon
  const themeIcon = document.getElementById('theme-icon')
  if (themeIcon) {
    themeIcon.textContent = themeName === 'dark' ? '☀️' : '🌙'
  }
  
  // Save preference
  localStorage.setItem('cws-theme', themeName)
  
  // Update theme selector if visible
  const selector = document.getElementById('theme-selector')
  if (selector) {
    selector.value = themeName
  }
}

function toggleTheme() {
  const currentTheme = document.documentElement.getAttribute('data-theme') || 'core'
  const newTheme = currentTheme === 'core' ? 'dark' : 'core'
  applyTheme(newTheme)
}

function initializeTheme() {
  const savedTheme = localStorage.getItem('cws-theme') || 'core'
  applyTheme(savedTheme)
}

function setupDOM() {
  document.body.innerHTML = `
    <div id="app">
      <header class="header">
        <div class="header-actions">
          <button class="btn-icon" onclick="toggleTheme()" title="Toggle Theme">
            <span id="theme-icon">🌙</span>
          </button>
        </div>
      </header>
      
      <link rel="stylesheet" href="/themes/core.css" id="theme-link">
      
      <div id="settings-modal" class="modal hidden">
        <div class="modal-content">
          <div class="modal-body">
            <select id="theme-selector">
              <option value="core">Core (Default)</option>
              <option value="academic">Academic</option>
              <option value="minimal">Minimal</option>
              <option value="dark">Dark</option>
              <option value="custom">Custom</option>
            </select>
          </div>
        </div>
      </div>
    </div>
  `
}

describe('Theme Management', () => {
  beforeEach(() => {
    setupDOM()
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  test('applies theme correctly', () => {
    applyTheme('dark')
    
    // Check document attribute
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    
    // Check theme link href
    expect(document.getElementById('theme-link').href).toContain('/themes/dark.css')
    
    // Check localStorage
    expect(localStorage.setItem).toHaveBeenCalledWith('cws-theme', 'dark')
    
    // Check theme icon
    expect(document.getElementById('theme-icon').textContent).toBe('☀️')
    
    // Check theme selector
    expect(document.getElementById('theme-selector').value).toBe('dark')
  })

  test('applies core theme correctly', () => {
    applyTheme('core')
    
    expect(document.documentElement.getAttribute('data-theme')).toBe('core')
    expect(document.getElementById('theme-link').href).toContain('/themes/core.css')
    expect(localStorage.setItem).toHaveBeenCalledWith('cws-theme', 'core')
    expect(document.getElementById('theme-icon').textContent).toBe('🌙')
    expect(document.getElementById('theme-selector').value).toBe('core')
  })

  test('toggles between core and dark themes', () => {
    // Start with core theme
    applyTheme('core')
    expect(document.documentElement.getAttribute('data-theme')).toBe('core')
    
    // Toggle to dark
    toggleTheme()
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(document.getElementById('theme-icon').textContent).toBe('☀️')
    
    // Toggle back to core
    toggleTheme()
    expect(document.documentElement.getAttribute('data-theme')).toBe('core')
    expect(document.getElementById('theme-icon').textContent).toBe('🌙')
  })

  test('initializes with saved theme from localStorage', () => {
    // Mock saved theme
    localStorage.setItem('cws-theme', 'academic')
    localStorage.getItem.mockReturnValue('academic')
    
    initializeTheme()
    
    expect(document.documentElement.getAttribute('data-theme')).toBe('academic')
    expect(document.getElementById('theme-link').href).toContain('/themes/academic.css')
    expect(document.getElementById('theme-selector').value).toBe('academic')
  })

  test('initializes with default theme when no saved theme', () => {
    localStorage.getItem.mockReturnValue(null)
    
    initializeTheme()
    
    expect(document.documentElement.getAttribute('data-theme')).toBe('core')
    expect(document.getElementById('theme-link').href).toContain('/themes/core.css')
  })

  test('handles all available themes', () => {
    const themes = ['core', 'dark', 'academic', 'minimal', 'custom']
    
    themes.forEach(theme => {
      applyTheme(theme)
      
      expect(document.documentElement.getAttribute('data-theme')).toBe(theme)
      expect(document.getElementById('theme-link').href).toContain(`/themes/${theme}.css`)
      expect(localStorage.setItem).toHaveBeenCalledWith('cws-theme', theme)
      expect(document.getElementById('theme-selector').value).toBe(theme)
      
      // Check icon (only dark theme has different icon)
      const expectedIcon = theme === 'dark' ? '☀️' : '🌙'
      expect(document.getElementById('theme-icon').textContent).toBe(expectedIcon)
    })
  })

  test('handles missing DOM elements gracefully', () => {
    // Remove some elements
    document.getElementById('theme-link').remove()
    document.getElementById('theme-icon').remove()
    document.getElementById('theme-selector').remove()
    
    // Should not throw errors
    expect(() => applyTheme('dark')).not.toThrow()
    
    // Should still update document attribute and localStorage
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(localStorage.setItem).toHaveBeenCalledWith('cws-theme', 'dark')
  })

  test('theme persistence across page reloads', () => {
    // Apply a theme
    applyTheme('academic')
    expect(localStorage.setItem).toHaveBeenCalledWith('cws-theme', 'academic')
    
    // Simulate page reload by clearing DOM state
    document.documentElement.removeAttribute('data-theme')
    
    // Mock localStorage returning the saved theme
    localStorage.getItem.mockReturnValue('academic')
    
    // Initialize theme (simulating page load)
    initializeTheme()
    
    // Theme should be restored
    expect(document.documentElement.getAttribute('data-theme')).toBe('academic')
  })

  test('theme selector synchronization', () => {
    const selector = document.getElementById('theme-selector')
    
    // Apply theme programmatically
    applyTheme('minimal')
    expect(selector.value).toBe('minimal')
    
    // Apply different theme
    applyTheme('custom')
    expect(selector.value).toBe('custom')
  })

  test('theme icon updates correctly', () => {
    const icon = document.getElementById('theme-icon')
    
    // Test dark theme icon
    applyTheme('dark')
    expect(icon.textContent).toBe('☀️')
    
    // Test non-dark themes icon
    const nonDarkThemes = ['core', 'academic', 'minimal', 'custom']
    nonDarkThemes.forEach(theme => {
      applyTheme(theme)
      expect(icon.textContent).toBe('🌙')
    })
  })

  test('localStorage interactions', () => {
    // Test saving theme
    applyTheme('academic')
    expect(localStorage.setItem).toHaveBeenCalledWith('cws-theme', 'academic')
    
    // Test retrieving saved theme
    localStorage.getItem.mockReturnValue('minimal')
    initializeTheme()
    expect(localStorage.getItem).toHaveBeenCalledWith('cws-theme')
  })

  test('CSS link href updates', () => {
    const themeLink = document.getElementById('theme-link')
    
    const themes = ['core', 'dark', 'academic', 'minimal', 'custom']
    themes.forEach(theme => {
      applyTheme(theme)
      expect(themeLink.href).toContain(`/themes/${theme}.css`)
    })
  })
})