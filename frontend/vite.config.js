import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
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
