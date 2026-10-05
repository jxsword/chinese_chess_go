/**
 * 解析客户端（Go 版 00 文档 §3.2：Worker 通道迁移为 App.Parser* 绑定）：
 * requestId 匹配 + 批次进度 + 迟到响应丢弃。
 *
 * - 传输后端二选一（构造注入或按 window.go 探测）：'wails' Go 侧 goroutine 分批解析，
 *   进度经事件 parser:progress {requestId, done, total} 回投；
 *   'mock' 复用协议核心在渲染线程同步计算（dev:web/测试，不触网）。
 * - 迟到响应按 id 匹配，不在 pending 表即丢弃（00 §3.2 主语义）；
 * - parseBatch 的 progress 事件按 id 回调（语料库页进度条）。
 */
import type { ParsedPuzzle } from '@packages/parsers'
import { createRequestId } from '@renderer/api/client'
import {
  CANCELED_ERROR,
  createParserWorkerCore,
  type ParseBatchPayload,
  type ParseBatchResult,
  type ParserRequestMsg,
  type ParserRequestType,
  type ParserResponseMsg
} from './parserProtocol'

/** 传输后端接口（结构同原 Worker：请求 postMessage，响应/进度经 onmessage 回投）。 */
export interface ParserTransport {
  postMessage(data: ParserRequestMsg): void
  terminate(): void
  onmessage: ((e: { data: ParserResponseMsg }) => void) | null
  onerror: ((e: unknown) => void) | null
}

interface PendingEntry {
  resolve: (v: unknown) => void
  reject: (e: Error) => void
  onProgress?: (done: number, total: number) => void
}

interface WailsWindow {
  window?: {
    go?: { main: { App: Record<string, (...args: unknown[]) => Promise<unknown>> } }
    runtime?: { EventsOn(name: string, cb: (...data: unknown[]) => void): () => void }
  }
}

/** Wails 绑定传输：请求映射 App.ParserParseBatch，进度经 parser:progress 事件回投。 */
function createWailsParserTransport(): ParserTransport {
  const t = {} as ParserTransport
  const rt = (globalThis as unknown as WailsWindow).window?.runtime
  if (rt !== undefined) {
    rt.EventsOn('parser:progress', (...data: unknown[]) => {
      const e = data[0] as { requestId?: string; done: number; total: number }
      if (e !== null && typeof e === 'object' && typeof e.requestId === 'string') {
        t.onmessage?.({ data: { id: e.requestId, ok: true, progress: { done: e.done, total: e.total } } })
      }
    })
  }
  const call = async (msg: ParserRequestMsg): Promise<void> => {
    const bindings = (globalThis as unknown as WailsWindow).window?.go?.main?.App
    if (bindings === undefined) {
      t.onmessage?.({ data: { id: msg.id, ok: false, error: 'Wails 绑定不存在（window.go）' } })
      return
    }
    if (msg.type === 'cancel') {
      await bindings.ParserCancel(msg.id)
      return
    }
    if (msg.type !== 'parseBatch') {
      t.onmessage?.({ data: { id: msg.id, ok: false, error: `Unknown parser request type: ${String(msg.type)}` } })
      return
    }
    try {
      const result = await bindings.ParserParseBatch(msg.id, (msg.payload as ParseBatchPayload | undefined)?.files ?? [])
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

/** mock 传输（dev:web/测试）：协议核心 microtask 结算，进度行为与 Worker 一致。 */
function createMockParserTransport(): ParserTransport {
  const t = {} as ParserTransport
  const core = createParserWorkerCore()
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

function resolveParserTransport(): { transport: ParserTransport; backend: 'wails' | 'mock' } {
  const hasWails = (globalThis as unknown as WailsWindow).window?.go !== undefined
  return hasWails
    ? { transport: createWailsParserTransport(), backend: 'wails' }
    : { transport: createMockParserTransport(), backend: 'mock' }
}

export class ParserClient {
  private readonly transport: ParserTransport
  private readonly pending = new Map<string, PendingEntry>()
  private disposed = false

  constructor(
    createTransport?: () => ParserTransport,
    /** requestId 工厂（测试注入可预测 id；默认 UUID，00 §3.2）。 */
    private readonly newId: () => string = createRequestId
  ) {
    if (createTransport !== undefined) {
      try {
        this.transport = createTransport()
      } catch {
        // 注入工厂抛错：回退默认后端选择（与 engine/solver 客户端一致）。
        this.transport = resolveParserTransport().transport
      }
    } else {
      this.transport = resolveParserTransport().transport
    }
    this.transport.onmessage = (e) => {
      this.handleResponse(e.data)
    }
    this.transport.onerror = () => undefined
  }

  /**
   * 批量解析：files 顺序与结果一一对应；损坏/无可演示走法的位为 null。
   * [onProgress] 每解析完一个文件回调 (done, total)。
   */
  async parseBatch(
    files: ParseBatchPayload['files'],
    onProgress?: (done: number, total: number) => void
  ): Promise<ParseBatchResult> {
    const result = (await this.request('parseBatch', { files }, onProgress)) as ParseBatchResult
    return result
  }

  /** 取消在途批次：立即以 canceled 结算并移出 pending（迟到响应丢弃）。 */
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

  private request(
    type: ParserRequestType,
    payload: unknown,
    onProgress?: (done: number, total: number) => void
  ): Promise<unknown> {
    const id = this.newId()
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject, onProgress })
      if (this.disposed) {
        this.pending.delete(id)
        reject(new Error('parser client disposed'))
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

  private handleResponse(resp: ParserResponseMsg): void {
    const entry = this.pending.get(resp.id)
    if (entry === undefined) return // 迟到响应：按 id 丢弃（00 §3.2）
    if (resp.progress !== undefined) {
      entry.onProgress?.(resp.progress.done, resp.progress.total)
      // 进度不是终态：等 result/error。
      return
    }
    this.pending.delete(resp.id)
    if (resp.ok) entry.resolve(resp.result)
    else entry.reject(new Error(resp.error ?? 'parser error'))
  }
}

export type { ParsedPuzzle }
