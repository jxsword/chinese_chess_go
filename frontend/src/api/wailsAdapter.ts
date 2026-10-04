import type {
  Unsubscribe,
  LlmChunkEvent,
  LlmDoneEvent,
  LlmErrorEvent,
  LlmEndpointConfig,
  LlmTestConnectionResult,
  VisionReadBoardRequest,
  VisionReadBoardResult,
  SavedGame,
  AutoSaveMode,
  GameRecord,
  GameRecordSummary,
  CorpusDownloadRequest,
  CorpusProgressEvent,
  CorpusScanResult,
  CorpusEntry,
  CorpusFileBytes,
  PgnIndexEntry,
  SaveFileRequest,
  FileContent,
  AppLifecycleEvent,
  SecureSlot,
  SecureSetResult
} from '@shared/ipc/types'
import type { WindowApi } from '@shared/ipc/api'

// wailsAdapter（Go 版 08 文档 §2；通道映射 00 文档 §3.2）：
//   invoke → window.go.app.App.Xxx(...)；事件 → window.runtime.EventsOn（返回反注册函数）。
// requestId 过滤逻辑从 Electron 版 ipc/client 平移到各消费方（llmTransport / *Client），
// 本适配器只做「通道名 + 载荷」的同构映射，不实现领域逻辑。
//
// window.go / window.runtime 为 Wails v2 注入的全局（桌面 WebView 内必然存在；
// client.ts 已探测 window.go 后才构造本适配器）。这里做最小类型声明，不 import
// 生成物 frontend/wailsjs（避免类型检查依赖 wails 生成时机）。

interface WailsApp {
  // ---- engine / solver / parser（Worker 通道迁移，消息形状在客户端与载荷内保留）----
  EngineFindBestMove(requestID: string, fen: string, difficulty: number, historyFens: string[]): Promise<unknown>
  EngineFindBestMoveEx(requestID: string, fen: string, depth: number, topK: number, timeLimitMs: number): Promise<unknown>
  EngineEvaluateMove(requestID: string, fen: string, move: unknown, depth: number): Promise<unknown>
  EngineCancel(requestID: string): Promise<void>
  SolverSolve(requestID: string, fen: string, timeLimitMs: number, maxPlies: number): Promise<unknown>
  SolverIsWinningFirstMove(requestID: string, fen: string, firstMove: unknown, plies: number, timeLimitMs: number): Promise<boolean>
  SolverCancel(requestID: string): Promise<void>
  ParserParseBatch(requestID: string, files: Array<{ name: string; source: string; bytes: Uint8Array }>): Promise<unknown>
  ParserCancel(requestID: string): Promise<void>
  // ---- llm / vision ----
  LlmChat(req: { requestId: string; url: string; headers: Record<string, string>; body: string; authSlot?: string }): Promise<void>
  LlmCancel(requestID: string): Promise<void>
  LlmTestConnection(config: unknown, authSlot: string): Promise<LlmTestConnectionResult>
  VisionReadBoard(config: unknown, imageBase64: string): Promise<VisionReadBoardResult>
  // ---- db ----
  DbSaveGame(req: { mode: string; fen: string; moves: number[][] }): Promise<void>
  DbLoadLatest(mode: string): Promise<SavedGame | null>
  DbDeleteForMode(mode: string): Promise<void>
  DbRecordsList(): Promise<GameRecordSummary[]>
  DbRecordsGet(id: number): Promise<GameRecord | null>
  DbRecordsSave(record: Omit<GameRecord, 'id'>): Promise<number>
  DbRecordsDelete(id: number): Promise<void>
  // ---- store / secure ----
  StoreGet(key: string): Promise<unknown>
  StoreSet(key: string, value: unknown): Promise<void>
  SecureGet(slot: string): Promise<LlmEndpointConfig | null>
  SecureSet(slot: string, payload: LlmEndpointConfig): Promise<SecureSetResult>
  SecureDelete(slot: string): Promise<void>
  // ---- corpus ----
  CorpusDownload(req: CorpusDownloadRequest): Promise<void>
  CorpusScan(root: string): Promise<CorpusScanResult>
  CorpusListEntries(categoryPath: string, categoryName: string): Promise<CorpusEntry[]>
  CorpusReadFiles(paths: string[]): Promise<CorpusFileBytes[]>
  CorpusPgnIndex(path: string, maxGames: number): Promise<PgnIndexEntry[]>
  CorpusReadPgnGame(path: string, entry: PgnIndexEntry): Promise<string>
  CorpusPickDirectory(): Promise<string>
  // ---- dialog / clipboard ----
  DialogSaveFile(req: SaveFileRequest): Promise<string>
  DialogReadFile(): Promise<FileContent | null>
  ClipboardWrite(text: string): Promise<void>
}

interface WailsRuntime {
  /** 订阅事件，返回反注册函数（wails v2 运行时语义） */
  EventsOn(eventName: string, callback: (...data: unknown[]) => void): () => void
}

function wailsApp(): WailsApp {
  const w = globalThis as unknown as { window?: { go?: { app: { App: WailsApp } } } }
  const app = w.window?.go?.app?.App
  if (app === undefined) throw new Error('Wails 绑定不存在（window.go）——wailsAdapter 只能在桌面 WebView 内使用')
  return app
}

function wailsRuntime(): WailsRuntime | null {
  const w = globalThis as unknown as { window?: { runtime?: WailsRuntime } }
  return w.window?.runtime ?? null
}

/** 订阅 Go 侧事件（EventsEmit 单载荷 → 回调首参）；事件名口径见 00 文档 §3.2 */
function eventsOn<T>(event: string, listener: (e: T) => void): Unsubscribe {
  const rt = wailsRuntime()
  if (rt === null) return () => undefined // 无运行时（异常环境）：退化为空订阅
  return rt.EventsOn(event, (...data: unknown[]) => listener(data[0] as T))
}

/** 空串 → null（Go 占位绑定以空串表达"用户取消"） */
function emptyToNull(v: string): string | null {
  return v === '' ? null : v
}

export function createWailsApi(): WindowApi {
  const app = wailsApp()
  return {
    llm: {
      chat: (req) =>
        app.LlmChat({
          requestId: req.requestId,
          url: req.url,
          headers: req.headers,
          body: req.body,
          authSlot: req.authSlot
        }),
      cancel: (requestId) => app.LlmCancel(requestId),
      testConnection: (config, authSlot) => app.LlmTestConnection(config, authSlot ?? ''),
      onChunk: (listener) => eventsOn<LlmChunkEvent>('llm:chunk', listener),
      onDone: (listener) => eventsOn<LlmDoneEvent>('llm:done', listener),
      onError: (listener) => eventsOn<LlmErrorEvent>('llm:error', listener)
    },
    vision: {
      readBoard: (req: VisionReadBoardRequest) =>
        app.VisionReadBoard(req.config, req.imageBase64)
    },
    db: {
      saveGame: (req) => app.DbSaveGame({ mode: req.mode, fen: req.fen, moves: req.moves }),
      loadLatest: (mode: AutoSaveMode) => app.DbLoadLatest(mode),
      deleteForMode: (mode: AutoSaveMode) => app.DbDeleteForMode(mode),
      recordsList: () => app.DbRecordsList(),
      recordsGet: (id) => app.DbRecordsGet(id),
      recordsSave: (record) => app.DbRecordsSave(record),
      recordsDelete: (id) => app.DbRecordsDelete(id)
    },
    store: {
      get: <T = unknown>(key: string) => app.StoreGet(key) as Promise<T | null>,
      set: (key, value) => app.StoreSet(key, value)
    },
    secure: {
      get: (slot: SecureSlot) => app.SecureGet(slot),
      set: (slot: SecureSlot, payload): Promise<SecureSetResult> => app.SecureSet(slot, payload),
      delete: (slot: SecureSlot) => app.SecureDelete(slot)
    },
    corpus: {
      download: (req: CorpusDownloadRequest) => app.CorpusDownload(req),
      onProgress: (listener) => eventsOn<CorpusProgressEvent>('corpus:progress', listener),
      scan: (root: string) => app.CorpusScan(root),
      listEntries: (categoryPath, categoryName) => app.CorpusListEntries(categoryPath, categoryName),
      readFiles: (paths: string[]) => app.CorpusReadFiles(paths),
      pgnIndex: (path: string, maxGames?: number) => app.CorpusPgnIndex(path, maxGames ?? 0),
      readPgnGame: (path, entry) => app.CorpusReadPgnGame(path, entry),
      pickDirectory: async () => emptyToNull(await app.CorpusPickDirectory())
    },
    dialog: {
      saveFile: async (req: SaveFileRequest) => emptyToNull(await app.DialogSaveFile(req)),
      readFile: () => app.DialogReadFile()
    },
    clipboard: {
      write: (text: string) => app.ClipboardWrite(text)
    },
    app: {
      onLifecycle: (listener) => eventsOn<AppLifecycleEvent>('app:lifecycle', listener)
    }
  }
}
