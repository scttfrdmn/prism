import { expect, afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';
import * as matchers from '@testing-library/jest-dom/matchers';

// Extend Vitest's expect with jest-dom matchers
expect.extend(matchers);

// Cleanup after each test case (e.g. clearing jsdom)
afterEach(() => {
  cleanup();
});

// Mock window.matchMedia for Cloudscape components
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => {},
  }),
});

// Mock ResizeObserver for Cloudscape components
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// Mock CSS.supports for Cloudscape
Object.defineProperty(window, 'CSS', {
  value: {
    supports: () => false,
  },
  writable: true
});

// Mock window.getComputedStyle
Object.defineProperty(window, 'getComputedStyle', {
  value: () => ({
    getPropertyValue: () => '',
  }),
});

// Mock scroll APIs
Object.defineProperty(Element.prototype, 'scrollIntoView', {
  value: () => {},
});

// Suppress CSS parsing errors in test environment
const originalConsoleError = console.error;
console.error = (...args) => {
  // Suppress specific CSS parsing errors that don't affect functionality
  if (
    args.length > 0 &&
    typeof args[0] === 'string' &&
    (args[0].includes('\\8 and \\9 are not allowed in strict mode') ||
     args[0].includes('CSS parsing error') ||
     args[0].includes('nwsapi'))
  ) {
    return;
  }
  originalConsoleError.apply(console, args);
};