import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  base: '/', 
  plugins: [react(), tailwindcss()],
  server: {
    host: '0.0.0.0', 
    port: 5173,
    strictPort: true, 
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.ts',
    css: true,
  },
})   