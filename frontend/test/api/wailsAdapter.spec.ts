// @vitest-environment jsdom
/**
 * wailsAdapter 绑定命名空间契约（M2 手测缺陷 F2 回归锁定，08 文档 §2）：
 *
 * Wails v2 以绑定结构体所在 Go 包名作为 window.go 下的命名空间——本项目绑定在
 * package main，方法表位于 window.go.main.App（wailsjs 生成物同口径）。
 * 曾误用 window.go.app.App：恒 undefined，桌面窗口内静默回落 mock，
 * 存档/设置/凭据/剪贴板全部内存化（B/C 手测缺陷根因）。
 *
 * 同时覆盖 T2.4 的生命周期双源接线：Go app:lifecycle 事件 + 适配层 window blur 补偿。
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { createWailsApi } from '@renderer/api/wailsAdapter'

interface RecordedCall {
  method: string
  args: unknown[]
}

interface FakeHarness {
  calls: RecordedCall[]
  /** 触发一次 Go 侧 EventsEmit（单载荷，等价 runtime.EventsOn 回调时序） */
  fireGoEvent: (name: string, payload: unknown) => void
}

function injectWailsBindings(): FakeHarness {
  const calls: RecordedCall[] = []
  const eventListeners = new Map<string, Set<(...data: unknown[]) => void>>()
  const app = new Proxy(
    {},
    {
      get(_t, method: string) {
        return (...args: unknown[]) => {
          calls.push({ method, args })
          return Promise.resolve({ ok: true })
        }
      }
    }
  )
  const runtime = {
    EventsOn: (name: string, cb: (...data: unknown[]) => void) => {
      if (!eventListeners.has(name)) eventListeners.set(name, new Set())
      eventListeners.get(name)!.add(cb)
      return () => {
        eventListeners.get(name)?.delete(cb)
      }
    }
  }
  // 注入到 jsdom 真窗口（保留 addEventListener/dispatchEvent 等原生能力）
  const w = globalThis as unknown as {
    window?: Record<string, unknown>
    go?: unknown
  }
  w.go = { main: { App: app } }
  w.window!.go = w.go
  w.window!.runtime = runtime
  return {
    calls,
    fireGoEvent: (name, payload) => {
      for (const cb of eventListeners.get(name) ?? []) cb(payload)
    }
  }
}

function clearWindow(): void {
  const w = globalThis as unknown as {
    window?: Record<string, unknown>
    go?: unknown
  }
  delete w.go
  delete w.window?.go
  delete w.window?.runtime
}

describe('wailsAdapter 绑定命名空间（window.go.main.App）', () => {
  beforeEach(clearWindow)
  afterEach(clearWindow)

  it('main.App 命名空间：db.saveGame 载荷形状精确透传（{mode,fen,moves}）', async () => {
    const harness = injectWailsBindings()
    const api = createWailsApi()
    const moves = [
      [1, 7, 4, 7],
      [1, 0, 4, 0]
    ]
    await expect(
      api.db.saveGame({ mode: 'humanVsHuman', fen: 'rnbakabnr/9/1c5c1 w', moves })
    ).resolves.toEqual({ ok: true })
    expect(harness.calls).toEqual([
      { method: 'DbSaveGame', args: [{ mode: 'humanVsHuman', fen: 'rnbakabnr/9/1c5c1 w', moves }] }
    ])
  })

  it('main.App 命名空间：store/secure/clipboard 调用映射到对应绑定方法', async () => {
    const harness = injectWailsBindings()
    const api = createWailsApi()
    await api.store.set('global_auto_save', false)
    await api.secure.set('llm_config_red', {
      baseUrl: 'https://x', apiKey: 'k', model: 'm', disableThinking: true
    } as Parameters<typeof api.secure.set>[1])
    await api.clipboard.write('记谱文本')
    expect(harness.calls.map((c) => c.method)).toEqual(['StoreSet', 'SecureSet', 'ClipboardWrite'])
    expect(harness.calls[1]!.args).toEqual([
      'llm_config_red',
      { baseUrl: 'https://x', apiKey: 'k', model: 'm', disableThinking: true }
    ])
  })

  it('错误命名空间（go.app.App，历史缺陷形态）不满足探测 → createWailsApi 抛错', () => {
    const w = globalThis as unknown as { window?: Record<string, unknown> }
    w.window!.go = { app: { App: new Proxy({}, { get: () => () => Promise.resolve() }) } }
    expect(() => createWailsApi()).toThrow(/window\.go\.main\.App/)
  })

  it('无 window.go（dev:web/Node）→ createWailsApi 抛错（client.ts 此时选 mock）', () => {
    expect(() => createWailsApi()).toThrow(/window\.go\.main\.App/)
  })
})

describe('wailsAdapter 生命周期双源（08 §2：Go 事件 + blur 补偿）', () => {
  let harness: FakeHarness

  beforeEach(() => {
    clearWindow()
    harness = injectWailsBindings()
  })
  afterEach(clearWindow)

  it('Go app:lifecycle 事件派发 phase；window blur 派发 {phase:"blur"}；反注册后两源均断开', () => {
    const api = createWailsApi()
    const seen: string[] = []
    const off = api.app.onLifecycle((e) => seen.push(e.phase))

    harness.fireGoEvent('app:lifecycle', { phase: 'close' })
    globalThis.window?.dispatchEvent(new Event('blur'))
    expect(seen).toEqual(['close', 'blur'])

    off()
    harness.fireGoEvent('app:lifecycle', { phase: 'before-quit' })
    globalThis.window?.dispatchEvent(new Event('blur'))
    expect(seen).toEqual(['close', 'blur']) // 反注册后两源均不再派发
  })
})
