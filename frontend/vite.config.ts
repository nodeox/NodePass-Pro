import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { securityHeadersPlugin } from './vite-plugin-security-headers'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    securityHeadersPlugin(),
  ],
  build: {
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks(id) {
          const normalizedId = id.replace(/\\/g, '/')
          if (normalizedId.includes('/node_modules/.pnpm/zrender@') || normalizedId.includes('/node_modules/zrender/')) {
            return 'vendor-zrender'
          }
          return undefined
        },
      },
    },
  },
})
