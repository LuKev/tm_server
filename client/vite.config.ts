import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import { fileURLToPath } from 'url'

const projectDir = fileURLToPath(new URL('.', import.meta.url))
const backendPort = process.env.VITE_BACKEND_PORT ?? process.env.TM_PLAYWRIGHT_SERVER_PORT ?? '8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // Base path for production deployment at kezilu.com/tm
  base: process.env.VITE_BASE_PATH || '/',
  cacheDir: process.env.VITE_CACHE_DIR || 'node_modules/.vite',
  resolve: {
    alias: {
      '@': path.resolve(projectDir, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: `http://127.0.0.1:${backendPort}`,
        changeOrigin: true,
        ws: true,
      },
      '/ws': {
        target: `ws://127.0.0.1:${backendPort}`,
        ws: true,
      },
    },
  },
})
