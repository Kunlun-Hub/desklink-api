import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  base: './',
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:21114',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../resources/admin',
    emptyOutDir: true,
    target: 'es2022',
  },
})
