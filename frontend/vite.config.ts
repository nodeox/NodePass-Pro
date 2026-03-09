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
  },
})
