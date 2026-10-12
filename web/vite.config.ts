import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const outDir = resolve(__dirname, '../backend/internal/platform/webui/dist')

export default defineConfig({
  plugins: [
    react(),
    // Keep the placeholder so `go build` works on a fresh clone before any npm build.
    { name: 'keep-gitkeep', closeBundle: () => writeFileSync(resolve(outDir, '.gitkeep'), '') },
  ],
  build: {
    outDir,
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: { '/api': 'http://localhost:8080' },
  },
})
