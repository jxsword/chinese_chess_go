/**
 * 引擎客户端（Go 版 00 文档 §3.2：Worker 通道迁移为 App.Engine* 绑定）：
 * requestId 匹配 + 取消 + 迟到响应丢弃。
 *
 * - 传输后端二选一（构造注入或按 window.go 探测，与 api/client 探测同构）：
 *   'wails' —— 经 Wails 绑定在 Go 侧 goroutine 计算（桌面模式，DR-003）；
 *   'mock'  —— 复用协议处理核心在渲染线程同步计算（dev:web/测试/注入 fake，不触网）。
 * - 迟到响应按 id 匹配，不在 pending 表即丢弃（00 §3.2 主语义，
 *   等价 Flutter `_gameSeq` 代数机制；丢弃逻辑在渲染层收口）；
 * - cancel：立即以 canceled 结算对应请求并从 pending 删除（后续迟到响应丢弃），
 *   同时通知后端（防排队请求空算）。
 */
import type { Move } from '@packages/rules'
import type { EngineReport } from '@packages/engine'
import { createRequestId } from '@renderer/api/client'
import {
  CANCELED_ERROR,
  createEngineWorkerCore,
  type EngineRequestMsg,
  type EngineRequestType,
  type EngineResponseMsg,
  type EvaluateMovePayload,
  type FindBestMoveExPayload,
  type FindBestMovePayload
} from './engineProtocol'

/** 传输后端接口（结构同原 Worker：请求 postMessage，响应经 onmessage 回投）。 */
export interface EngineTransport {
  postMessage(data: EngineRequestMsg): void
  terminate(): void
  onmessage: ((e: { data: EngineResponseMsg }) => void) | null
  onerror: ((e: unknown) => void) | null
}

export type EngineBackend = 'wails' | 'mock'

interface WailsWindow {
  window?: { go?: { app: { App: Record<string, (...args: unknown[]) => Promise<unknown>> } } }
}

function wailsApp(): Record<string, (...args: unknown[]) => Promise<unknown>> {
  const app = (globalThis as unknown as WailsWindow).window?.go?.app?.App
  if (app === undefined) throw new Error('Wails 绑定不存在（window.go）')
  return app
}

/**
 * Wails 绑定传输：{id, type, payload} 映射到 App.Engine* 绑定（00 文档 §3.2）；
 * 绑定 reject（含 M3 前的占位错误）映射为 {ok:false, error} 响应。
 */
function createWailsEngineTransport(): EngineTransport {
  const t = {} as EngineTransport
  const call = async (msg: EngineRequestMsg): Promise<void> => {
    const bindings = wailsApp()
    const p = (msg.payload ?? {}) as FindBestMovePayload & FindBestMoveExPayload & EvaluateMovePayload
    try {
      let result: unknown
      switch (msg.type) {
        case 'findBestMove':
          result = await bindings.EngineFindBestMove(msg.id, p.fen, p.difficulty ?? 0, p.historyFens ?? [])
          break
        case 'findBestMoveEx':
          result = await bindings.EngineFindBestMoveEx(msg.id, p.fen, p.depth ?? 0, p.topK ?? 0, p.timeLimitMs ?? 0)
          break
        case 'evaluateMove':
          result = await bindings.EngineEvaluateMove(msg.id, p.fen, p.move, p.depth ?? 0)
          break
        case 'cancel':
          await bindings.EngineCancel(msg.id)
          return
        default:
          throw new Error(`Unknown engine request type: ${String(msg.type)}`)
      }
      t.onmessage?.({ data: { id: msg.id, ok: true, result } })
    } catch (e) {
      // 绑定错误（含取消）统一映射为错误响应；迟到丢弃仍由 client pending 表收口。
      t.onmessage?.({ data: { id: msg.id, ok: false, error: e instanceof Error ? e.message : String(e) } })
    }
  }
  t.postMessage = (msg) => void call(msg)
  t.terminate = () => undefined
  t.onmessage = null
  t.onerror = null
  return t
}

/**
 * mock 传输（dev:web/测试）：复用协议处理核心——与 Electron 版 dev:web 的
 * Worker 内核同代码路径（FIFO 串行 + cancel 防线），microtask 结算保持异步语义。
 */
function createMockEngineTransport(): EngineTransport {
  const t = {} as EngineTransport
  const core = createEngineWorkerCore()
  t.postMessage = (msg) => {
    queueMicrotask(() => {
      core.handleRequest(msg, (resp) => {
        t.onmessage?.({ data: resp })
      })
    })
  }
  t.terminate = () => undefined
  t.onmessage = null
  t.onerror = null
  return t
}

function hasWailsBindings(): boolean {
  return (globalThis as unknown as WailsWindow).window?.go !== undefined
}

function resolveEngineTransport(): { transport: EngineTransport; backend: EngineBackend } {
  if (hasWailsBindings()) return { transport: createWailsEngineTransport(), backend: 'wails' }
  return { transport: createMockEngineTransport(), backend: 'mock' }
}

export class EngineClient {
  private readonly transport: EngineTransport
  private readonly pending = new Map<string, { resolve: (v: unknown) => void; reject: (e: Error) => void }>()
  private disposed = false
  private readonly _backend: EngineBackend

  /** 当前后端：'wails'（Go 绑定）或 'mock'（渲染线程内核/注入 fake）。 */
  get backend(): EngineBackend {
    return this._backend
  }

  constructor(
    createTransport?: () => EngineTransport,
    /** requestId 工厂（测试注入可预测 id；默认 UUID，00 §3.2）。 */
    private readonly newId: () => string = createRequestId
  ) {
    if (createTransport !== undefined) {
      try {
        this.transport = createTransport()
        this._backend = 'mock'
      } catch {
        // 注入工厂抛错（等价原"Worker 初始化失败"）：回退默认后端选择。
        const resolved = resolveEngineTransport()
        this.transport = resolved.transport
        this._backend = resolved.backend
      }
    } else {
      const resolved = resolveEngineTransport()
      this.transport = resolved.transport
      this._backend = resolved.backend
    }
    this.transport.onmessage = (e) => {
      this.handleResponse(e.data)
    }
    // Wails 绑定无 onerror 通道：绑定错误经 reject → {ok:false} 响应收口。
    this.transport.onerror = () => undefined
  }

  /** 对局 AI 应手（Move 为纯数据，页面 playMove 仍做最终校验）。 */
  async findBestMove(
    fen: string,
    options: { difficulty?: number; historyFens?: readonly string[] } = {}
  ): Promise<Move | null> {
    const payload: FindBestMovePayload = {
      fen,
      difficulty: options.difficulty,
      historyFens: options.historyFens === undefined ? undefined : [...options.historyFens]
    }
    return (await this.request('findBestMove', payload)) as Move | null
  }

  /** 参谋报告（Top-K 真实分差）。 */
  async findBestMoveEx(
    fen: string,
    options: { depth?: number; topK?: number; timeLimitMs?: number } = {}
  ): Promise<EngineReport | null> {
    const payload: FindBestMoveExPayload = {
      fen,
      depth: options.depth,
      topK: options.topK,
      timeLimitMs: options.timeLimitMs
    }
    return (await this.request('findBestMoveEx', payload)) as EngineReport | null
  }

  /** 单着法评估（护航否决用）。 */
  async evaluateMove(
    fen: string,
    move: Move,
    options: { depth?: number } = {}
  ): Promise<number | null> {
    const payload: EvaluateMovePayload = { fen, move, depth: options.depth }
    return (await this.request('evaluateMove', payload)) as number | null
  }

  /**
   * 取消请求：立即以 canceled 结算并移出 pending（其迟到响应将被丢弃），
   * 同时通知后端。id 不在 pending 时仅通知（幂等）。
   */
  cancel(id: string): void {
    const entry = this.pending.get(id)
    if (entry !== undefined) {
      this.pending.delete(id)
      entry.reject(new Error(CANCELED_ERROR))
    }
    this.transport.postMessage({ id, type: 'cancel' })
  }

  /** 终止后端传输（页面销毁时调用）；此后请求以 disposed 拒绝。 */
  dispose(): void {
    this.disposed = true
    this.transport.terminate()
  }

  private request(type: EngineRequestType, payload: unknown): Promise<unknown> {
    const id = this.newId()
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject })
      if (this.disposed) {
        this.pending.delete(id)
        reject(new Error('engine client disposed'))
        return
      }
      try {
        this.transport.postMessage({ id, type, payload })
      } catch (e) {
        this.pending.delete(id)
        reject(e instanceof Error ? e : new Error(String(e)))
      }
    })
  }

  private handleResponse(resp: EngineResponseMsg): void {
    const entry = this.pending.get(resp.id)
    if (entry === undefined) return // 迟到响应：按 id 丢弃（00 §3.2）
    this.pending.delete(resp.id)
    if (resp.ok) entry.resolve(resp.result)
    else entry.reject(new Error(resp.error ?? 'engine error'))
  }
}
