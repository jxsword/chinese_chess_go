import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'

// 前端单测运行于 Node 环境（与 Electron 版 vitest 配置同构；08 §1 移植清单随迁用例）
// UI 组件测试在各自文件以 `// @vitest-environment jsdom` 声明浏览器环境。
// 领域包（rules/engine/llm/parsers/solver/storage）测试由 Go 侧 internal/* 对拍承接，
// 不在本配置范围（见 09 文档）。
export default defineConfig({
  esbuild: {
    // test/*.tsx 不在任何 tsconfig include 内的旧结构残留：显式指定自动 JSX 运行时
    jsx: 'automatic'
  },
  resolve: {
    alias: {
      '@shared': fileURLToPath(new URL('./src/shared', import.meta.url)),
      '@renderer': fileURLToPath(new URL('./src', import.meta.url)),
      '@packages': fileURLToPath(new URL('./src/packages', import.meta.url))
    }
  },
  test: {
    include: ['test/**/*.spec.{ts,tsx}'],
    environment: 'node',
    passWithNoTests: true
  }
})
