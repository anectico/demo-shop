import { defineConfig } from 'vite'

// In development the API (port 8090) serves /api; the built storefront is
// served by the API itself from DEMO_STATIC_DIR, so production has one origin.
export default defineConfig({
  server: { port: 5190, proxy: { '/api': 'http://localhost:8090' } },
  build: { sourcemap: true },
})
