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
  // 必须探测到 App 方法表（go.app.App），而非仅 window.go：
  // Wails 运行时先同步注入 window.go = {}，App 方法表经 SetBindings 稍后填充；
  // 桌面 WebView 内 window.wailsbindings 内联于页面、SetBindings 在模块求值前同步完成，
  // 故首次解析时必然就绪；而 wails devserver 在外部浏览器打开时只有空 {}（无 wailsbindings），
  // 按 mock 处理（与 Electron 版 dev:web 行为一致）。
  const w = globalThis as unknown as { window?: { go?: { app?: { App?: unknown } } } }
  return w?.window?.go?.app?.App !== undefined
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
