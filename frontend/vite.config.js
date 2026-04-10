import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: '../web/static',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/channel': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
      '/vapid-public-key': 'http://localhost:8080',
    },
  },
})
