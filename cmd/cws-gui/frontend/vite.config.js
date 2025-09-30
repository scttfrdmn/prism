import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    assetsDir: 'assets',
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      input: {
        main: './index.html'
      },
      output: {
        manualChunks: {
          // Split Cloudscape components into separate chunk for better caching
          'cloudscape': ['@cloudscape-design/components', '@cloudscape-design/design-tokens', '@cloudscape-design/global-styles'],
          // React core libs
          'react-vendor': ['react', 'react-dom'],
        }
      }
    }
  },
  server: {
    port: 3000,
    strictPort: true
  },
  publicDir: 'public'
})