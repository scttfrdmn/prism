// Test setup file for Vitest
import '@testing-library/jest-dom'
import { beforeAll, afterAll, beforeEach, afterEach, vi } from 'vitest'
import { server } from './mocks/daemon-server.js'

// Global test setup
beforeAll(() => {
  // Start mock service worker for API mocking
  server.listen({ onUnhandledRequest: 'error' })
  
  // Mock Wails runtime
  global.wails = {
    CloudWorkstationService: {
      GetTemplates: vi.fn(),
      GetInstances: vi.fn(),
      LaunchInstance: vi.fn(),
      StopInstance: vi.fn(),
      ConnectToInstance: vi.fn()
    }
  }
  
  // Mock localStorage
  const localStorageMock = {
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn(),
  }
  vi.stubGlobal('localStorage', localStorageMock)
  
  // Mock console methods to reduce noise in tests
  vi.stubGlobal('console', {
    ...console,
    log: vi.fn(),
    info: vi.fn(),
    debug: vi.fn(),
  })
})

// Cleanup after all tests
afterAll(() => {
  server.close()
  vi.restoreAllMocks()
})

// Reset handlers after each test
afterEach(() => {
  server.resetHandlers()
  vi.clearAllMocks()
})

// Setup DOM before each test
beforeEach(() => {
  // Reset DOM to clean state
  document.body.innerHTML = ''
  document.head.innerHTML = ''
  
  // Reset localStorage mock
  localStorage.clear.mockClear()
  localStorage.getItem.mockClear()
  localStorage.setItem.mockClear()
  localStorage.removeItem.mockClear()
  
  // Reset Wails service mocks
  Object.values(global.wails.CloudWorkstationService).forEach(mock => {
    mock.mockClear()
  })
})

// Custom matchers
expect.extend({
  toBeVisible(received) {
    const pass = received && !received.classList.contains('hidden')
    return {
      message: () => `expected element to ${pass ? 'not ' : ''}be visible`,
      pass,
    }
  },
  
  toHaveTheme(received, theme) {
    const pass = received.getAttribute('data-theme') === theme
    return {
      message: () => `expected element to have theme "${theme}", got "${received.getAttribute('data-theme')}"`,
      pass,
    }
  }
})