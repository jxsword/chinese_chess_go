/**
 * 求解客户端（Go 版 00 文档 §3.2：Worker 通道迁移为 App.Solver* 绑定）：
 * requestId 匹配 + 取消 + 迟到响应丢弃。
 *
 * - 传输后端二选一（构造注入或按 window.go 探测）：'wails' Go 侧 goroutine 计算；
 *   'mock' 复用协议核心在渲染线程同步计算（dev:web/测试，不触网）。
 * - 迟到响应按 id 匹配，不在 pending 表即丢弃（00 §3.2 主语义）；
 * - cancel：立即以 canceled 结算对应请求并从 pending 删除（后续迟到响应丢弃），
 *   同时通知后端（防排队请求空算；在途求解由 timeLimitMs 兜底收口）。
 */
import type { Move } from '@packages/rules'
import type { SolveResult } from '@packages/solver'
import { createRequestId } from '@renderer/api/client'
import {
  SOLVER_CANCELED_ERROR,
  createSolverWorkerCore,
  type IsWinningFirstMovePayload,
  type SolvePayload,
  type SolverRequestMsg,
  type SolverRequestType,
  type SolverResponseMsg
} from './solverProtocol'

/** 传输后端接口（结构同原 Worker：请求 postMessage，响应经 onmessage 回投）。 */
export interface SolverTransport {
  postMessage(data: SolverRequestMsg): void
  terminate(): void
  onmessage: ((e: { data: SolverResponseMsg }) => void) | null
  onerror: ((e: unknown) => void) | null
}

export type SolverBackend = 'wails' | 'mock'

interface WailsWindow {
  window?: { go?: { app: { App: Record<string, (...args: unknown[]) => Promise<unknown>> } } }
}

function wailsApp(): Record<string, (...args: unknown[]) => Promise<unknown>> {
  const app = (globalThis as unknown as WailsWindow).window?.go?.app?.App
  if (app === undefined) throw new Error('Wails 绑定不存在（window.go）')
  return app
}

/** Wails 绑定传输：{id, type, payload} 映射到 App.Solver* 绑定。 */
function createWailsSolverTransport(): SolverTransport {
  const t = {} as SolverTransport
  const call = async (msg: SolverRequestMsg): Promise<void> => {
    const bindings = wailsApp()
    const p = (msg.payload ?? {}) as SolvePayload & IsWinningFirstMovePayload
    try {
      let result: unknown
      switch (msg.type) {
        case 'solve':
          result = await bindings.SolverSolve(msg.id, p.fen, p.timeLimitMs ?? 0, p.maxPlies ?? 0)
          break
        case 'isWinningFirstMove':
          result = await bindings.SolverIsWinningFirstMove(msg.id, p.fen, p.firstMove, p.plies ?? 0, p.timeLimitMs ?? 0)
          break
        case 'cancel':
          await bindings.SolverCancel(msg.id)
          return
        default:
          throw new Error(`Unknown solver request type: ${String(msg.type)}`)
      }
      t.onmessage?.({ data: { id: msg.id, ok: true, result } })
    } catch (e) {
      t.onmessage?.({ data: { id: msg.id, ok: false, error: e instanceof Error ? e.message : String(e) } })
    }
  }
  t.postMessage = (msg) => void call(msg)
  t.terminate = () => undefined
  t.onmessage = null
  t.onerror = null
  return t
}

/** mock 传输（dev:web/测试）：协议核心 microtask 结算。 */
function createMockSolverTransport(): SolverTransport {
  const t = {} as SolverTransport
  const core = createSolverWorkerCore()
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

function resolveSolverTransport(): { transport: SolverTransport; backend: SolverBackend } {
  const hasWails = (globalThis as unknown as WailsWindow).window?.go !== undefined
  return hasWails
    ? { transport: createWailsSolverTransport(), backend: 'wails' }
    : { transport: createMockSolverTransport(), backend: 'mock' }
}

export interface SolveRequestOptions {
  timeLimitMs?: number
  maxPlies?: number
}

export class SolverClient {
  private readonly transport: SolverTransport
  private readonly pending = new Map<string, { resolve: (v: unknown) => void; reject: (e: Error) => void }>()
  private disposed = false
  private readonly _backend: SolverBackend

  /** 当前后端：'wails'（Go 绑定）或 'mock'（渲染线程内核/注入 fake）。 */
  get backend(): SolverBackend {
    return this._backend
  }

  constructor(
    createTransport?: () => SolverTransport,
    /** requestId 工厂（测试注入可预测 id；默认 UUID，00 §3.2）。 */
    private readonly newId: () => string = createRequestId
  ) {
    if (createTransport !== undefined) {
      try {
        this.transport = createTransport()
        this._backend = 'mock'
      } catch {
        // 注入工厂抛错（等价原"Worker 初始化失败"）：回退默认后端选择。
        const resolved = resolveSolverTransport()
        this.transport = resolved.transport
        this._backend = resolved.backend
      }
    } else {
      const resolved = resolveSolverTransport()
      this.transport = resolved.transport
      this._backend = resolved.backend
    }
    this.transport.onmessage = (e) => {
      this.handleResponse(e.data)
    }
    this.transport.onerror = () => undefined
  }

  /** 求解残局（SolveResult 为纯数据；页面侧不做二次校验）。 */
  async solve(fen: string, options: SolveRequestOptions = {}): Promise<SolveResult> {
    const payload: SolvePayload = {
      fen,
      timeLimitMs: options.timeLimitMs,
      maxPlies: options.maxPlies
    }
    return (await this.request('solve', payload)) as SolveResult
  }

  /** 验证首着是否必胜（LLM 求解辅助的裁判，04 文档 §6）。 */
  async isWinningFirstMove(
    fen: string,
    firstMove: Move,
    options: { plies?: number; timeLimitMs?: number } = {}
  ): Promise<boolean> {
    const payload: IsWinningFirstMovePayload = {
      fen,
      firstMove,
      plies: options.plies,
      timeLimitMs: options.timeLimitMs
    }
    return (await this.request('isWinningFirstMove', payload)) as boolean
  }

  /**
   * 取消请求：立即以 canceled 结算并移出 pending（其迟到响应将被丢弃），
   * 同时通知后端。id 不在 pending 时仅通知（幂等）。
   */
  cancel(id: string): void {
    const entry = this.pending.get(id)
    if (entry !== undefined) {
      this.pending.delete(id)
      entry.reject(new Error(SOLVER_CANCELED_ERROR))
    }
    this.transport.postMessage({ id, type: 'cancel' })
  }

  /** 终止后端传输（页面销毁时调用）；此后请求以 disposed 拒绝。 */
  dispose(): void {
    this.disposed = true
    this.transport.terminate()
  }

  private request(type: SolverRequestType, payload: unknown): Promise<unknown> {
    const id = this.newId()
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject })
      if (this.disposed) {
        this.pending.delete(id)
        reject(new Error('solver client disposed'))
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

  private handleResponse(resp: SolverResponseMsg): void {
    const entry = this.pending.get(resp.id)
    if (entry === undefined) return // 迟到响应：按 id 丢弃（00 §3.2）
    this.pending.delete(resp.id)
    if (resp.ok) entry.resolve(resp.result)
    else entry.reject(new Error(resp.error ?? 'solver error'))
  }
}
