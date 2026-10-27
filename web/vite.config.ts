import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// T-0170：单 SPA 单入口；dev proxy → SG/core 同源坐标（原 dev 无 proxy
// 够不着 API——开发即 build 的病灶解）。
export default defineConfig({
  plugins: [react()],
  build: { outDir: 'dist', emptyOutDir: true },
  server: {
    proxy: {
      // Connect RPC 与资产同源转发——transport baseUrl=window.location.origin 不变。
      '/o.dashboard.': { target: 'http://127.0.0.1:50051', changeOrigin: true },
    },
  },
})
