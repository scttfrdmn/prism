import { defineConfig } from 'vite'

export default defineConfig({
  test: {
    // Use jsdom environment for DOM testing
    environment: 'jsdom',
    
    // Enable global test utilities
    globals: true,
    
    // Setup files to run before each test
    setupFiles: ['./tests/setup.js'],
    
    // Coverage configuration
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      reportsDirectory: './coverage',
      exclude: [
        'node_modules/**',
        'tests/**',
        'dist/**'
      ],
      thresholds: {
        global: {
          branches: 80,
          functions: 80,
          lines: 80,
          statements: 80
        }
      }
    },
    
    // Test file patterns - exclude E2E and visual tests
    include: [
      'tests/unit/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}'
    ],
    
    // Exclude patterns
    exclude: [
      'node_modules/**',
      'dist/**'
    ],
    
    // Timeout for tests
    testTimeout: 10000,
    
    // Watch options
    watch: {
      mode: 'smart'
    }
  }
})