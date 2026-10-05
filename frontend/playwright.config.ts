import { defineConfig } from '@playwright/test'

// Playwright 浏览器 mock 模式 E2E（09 §2.5 / T7.3）：
//   - 冒烟链路 = Electron 版 e2e/smoke.spec.mjs 的浏览器移植（主页 7 入口 →
//     双人走一着 → 悔棋 → 新游戏 → 棋谱库打开）；
//   - webServer 起 vite dev（npm run dev:web，端口 5173 strictPort）——window.go
//     探测失败自动落 mock 适配器，全程不触网、不依赖 Go 后端；
//   - Wails 桌面窗口不做自动化驱动（WebKitGTK 无 CDP），仅浏览器模式入 CI 口径外
//     的本地验收。
export default defineConfig({
  testDir: './e2e',
  timeout: 120_000,
  forbidOnly: !!process.env.CI,
  use: {
    baseURL: 'http://127.0.0.1:5173',
    viewport: { width: 1280, height: 800 }
  },
  webServer: {
    command: 'npm run dev:web',
    url: 'http://127.0.0.1:5173',
    reuseExistingServer: true,
    timeout: 60_000
  },
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }]
})
