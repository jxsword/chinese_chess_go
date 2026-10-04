import { createMockApi } from './mockAdapter'
import { createWailsApi } from './wailsAdapter'
import type { WindowApi } from '@shared/ipc/api'

// api 适配层入口（Go 版 08 文档 §2）：桌面 Wails 环境注入 window.go（Wails 绑定 +
// 运行时事件），实现为 wailsAdapter；纯浏览器（dev:web）与 Node 测试环境无 window.go，
// 切换到 mockAdapter，保证 UI 开发/E2E 不依赖后端（与 Electron 版 window.api 探测同构）。

declare global {
  interface Window {
    /** Wails v2 注入的绑定命名空间（仅桌面 WebView 存在） */
    readonly go?: unknown
  }
}

function hasWailsBindings(): boolean {
  // 经 globalThis 取窗体引用：本文件会被 Node 侧测试间接引入（无 DOM 库）
  const w = (globalThis as unknown as { window?: { readonly go?: unknown } }).window
  return w?.go !== undefined
}

function resolveApi(): WindowApi {
  if (hasWailsBindings()) return createWailsApi()
  return createMockApi()
}

export const api: WindowApi = resolveApi()

/** requestId 工厂（00 文档 §3.2）：所有异步操作（LLM/下载/引擎/求解/解析）统一使用 */
export function createRequestId(): string {
  return crypto.randomUUID()
}
