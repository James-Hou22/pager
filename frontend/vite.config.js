import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    // Rewrite /event/* to /attendee.html in dev so the attendee SPA is served
    // correctly. In production the Go server handles this routing.
    {
      name: 'attendee-spa-rewrite',
      configureServer(server) {
        server.middlewares.use((req, _res, next) => {
          if (req.url.startsWith('/event/')) {
            req.url = '/attendee.html'
          } else if (req.url === '/' || req.url === '') {
            req.url = '/landing-page.html'
          }
          next()
        })
      },
    },
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: '../web/static',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: path.resolve(__dirname, 'index.html'),
        attendee: path.resolve(__dirname, 'attendee.html'),
        landing: path.resolve(__dirname, 'landing-page.html'),
      },
    },
  },
  server: {
    // Node 17+ can resolve "localhost" to ::1 while curl/browsers try
    // 127.0.0.1 first, causing ECONNREFUSED even though Vite is running.
    // Binding explicitly to 127.0.0.1 avoids the mismatch.
    host: '127.0.0.1',
    proxy: {
      '/channel': 'http://localhost:8080',
      '/attendee/events': 'http://localhost:8080',
      '/attendee/channel': 'http://localhost:8080',
      '/attendee/verify': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
      '/waitlist': 'http://localhost:8080',
      '/vapid-public-key': 'http://localhost:8080',
      '/auth': 'http://localhost:8080',
      '/events': 'http://localhost:8080',
    },
  },
})
