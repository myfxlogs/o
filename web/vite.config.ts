import { defineConfig } from 'vite'
import { resolve } from 'path'
export default defineConfig({
  build: {
    outDir: 'dist', emptyOutDir: true,
    rollupOptions: { input: { index: resolve(__dirname, 'index.html'), trx: resolve(__dirname, 'trx.html') } },
  },
})
