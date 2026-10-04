import { fileURLToPath, URL } from 'node:url'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// Wails dev（wails.json frontend:dev:watcher 启动）与浏览器 mock 模式（npm run dev:web）
// 共用一份配置；wails/mock 适配器在运行时按 window.go 探测选择（Go 版 08 文档 §2）。
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@shared': fileURLToPath(new URL('./src/shared', import.meta.url)),
      '@renderer': fileURLToPath(new URL('./src', import.meta.url)),
      '@packages': fileURLToPath(new URL('./src/packages', import.meta.url)),
      // 浏览器包图内以 TextDecoder shim 替代 iconv-lite（其依赖 node:buffer 会白屏崩溃，
      // 见 src/shims/iconv-lite.ts 头注；vitest/Node 侧不受影响）
      'iconv-lite': fileURLToPath(new URL('./src/shims/iconv-lite.ts', import.meta.url))
    }
  },
  server: {
    port: 5173,
    strictPort: true
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
