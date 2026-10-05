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
  // 必须探测到 App 方法表，且命名空间 = Go 包名（Wails v2 以绑定结构体所在包名
  // 作为 window.go 下的命名空间——本项目绑定在 package main，故为 window.go.main.App；
  // wailsjs 生成物 frontend/wailsjs/go/main/App.js 同口径）。
  // runtime.js（assetserver 注入页首、先于 module 脚本执行）从内联 window.wailsbindings
  // 同步 SetBindings，故首次解析时必然就绪；纯浏览器 dev:web（vite 直连，无 wails
  // devserver）与 Node 测试环境无 window.go，按 mock 处理（与 Electron 版 dev:web 一致）。
  // 【勘误 M2：曾探测 go.app.App——恒 undefined，桌面窗口静默回落 mock，
  // 存档/设置/剪贴板全部走内存 mock（KNOWN_ISSUES F2）】
  const w = globalThis as unknown as { window?: { go?: { main?: { App?: unknown } } } }
  return w?.window?.go?.main?.App !== undefined
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
