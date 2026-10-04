package main

import (
	"context"
	"fmt"
)

// App 绑定层骨架（00 文档 §3.2「通道清单」→ Wails 绑定方法 + EventsEmit 事件）。
//
// M0 为占位实现，行为约定：
//   - 读路径返回空值（空对象/空数组/null），前端按"空态"渲染，等价 mock 适配器语义；
//   - 写路径与计算路径返回里程碑错误（errMilestone），前端适配层走失败分支；
//   - 各方法随里程碑替换为真实实现：M2 storage（db/store/secure）、M3 engine、
//     M4 llm/vision、M5 corpus/parser、M6 solver/vision。
//
// 装配纪律（AGENTS.md）：
//   - 本文件只做 Wails 方法/事件装配与 userData/生命周期管理，禁止领域逻辑（进 internal/*）；
//   - 异步方法一律带 requestID，取消经 context.Context（DR-003），取消后不得再发任何事件；
//   - 对外 HTTP 只在 internal/llm/transport（铁律 #4），本层不直接发起网络请求。
type App struct {
	ctx context.Context
}

// NewApp 创建绑定层实例（Wails Bind 入口）。
func NewApp() *App {
	return &App{}
}

// startup 保存 Wails 运行时上下文（EventsEmit/对话框等运行时能力依赖它）。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// errMilestone 占位方法统一错误：指明方法与计划接入里程碑。
func errMilestone(method, milestone string) error {
	return fmt.Errorf("功能尚未接入（%s 计划于里程碑 %s 实现）", method, milestone)
}

// ---------------------------------------------------------------------------
// LLM（M4 接入；思维链强制关闭见 DR-005，关闭参数在请求构造层恒发）
// ---------------------------------------------------------------------------

// LlmChat 受理流式对话请求（恒 resolve 语义 = 受理即返回）；
// 结局经事件 llm:chunk / llm:done / llm:error 回传，载荷含 requestID。
func (a *App) LlmChat(req map[string]any) error {
	_ = req // {requestId, url, headers, body, authSlot}
	return errMilestone("LLM 对话", "M4")
}

// LlmCancel 取消在途请求（context cancel；幂等）。
func (a *App) LlmCancel(requestID string) {
	_ = requestID // M4：取消对应 context，取消后不再发事件
}

// LlmTestConnection 配置卡"测试连接"（单次非流式，同步结果）。
func (a *App) LlmTestConnection(cfg map[string]any, authSlot string) (map[string]any, error) {
	_ = cfg
	_ = authSlot
	return nil, errMilestone("LLM 测试连接", "M4")
}

// VisionReadBoard 视觉识图（多模态非流式；M6 接入，识图请求恒发关闭参数）。
func (a *App) VisionReadBoard(cfg map[string]any, imageB64 string) (map[string]any, error) {
	_ = cfg
	_ = imageB64
	return nil, errMilestone("视觉识图", "M6")
}

// ---------------------------------------------------------------------------
// 数据库（M2 接入：internal/storage，modernc.org/sqlite，Schema 与 Electron 版逐字段一致）
// ---------------------------------------------------------------------------

// DbSaveGame 自动存档（每 mode 一条 upsert）。
func (a *App) DbSaveGame(req map[string]any) error {
	_ = req // {mode, fen, moves}
	return errMilestone("自动存档", "M2")
}

// DbLoadLatest 读取模式存档；无存档返回 null。
func (a *App) DbLoadLatest(mode string) (map[string]any, error) {
	_ = mode
	return nil, nil // M0 占位：空态
}

// DbDeleteForMode 删除模式存档（恢复后死局清理等场景）。
func (a *App) DbDeleteForMode(mode string) error {
	_ = mode
	return errMilestone("存档删除", "M2")
}

// DbRecordsList 棋谱库列表（摘要）。
func (a *App) DbRecordsList() ([]map[string]any, error) {
	return []map[string]any{}, nil // M0 占位：空态
}

// DbRecordsGet 读取单条棋谱；不存在返回 null。
func (a *App) DbRecordsGet(id int64) (map[string]any, error) {
	_ = id
	return nil, nil // M0 占位：空态
}

// DbRecordsSave 写入棋谱，返回新 id。
func (a *App) DbRecordsSave(record map[string]any) (int64, error) {
	_ = record
	return 0, errMilestone("棋谱保存", "M2")
}

// DbRecordsDelete 删除棋谱。
func (a *App) DbRecordsDelete(id int64) error {
	_ = id
	return errMilestone("棋谱删除", "M2")
}

// ---------------------------------------------------------------------------
// 设置存储（M2 接入：JSON 配置文件，键名沿用 llm_settings_*；越界值 clamp）
// ---------------------------------------------------------------------------

// StoreGet 读取设置键；未设置返回 null。
func (a *App) StoreGet(key string) (any, error) {
	_ = key
	return nil, nil // M0 占位：空态
}

// StoreSet 写入设置键。
func (a *App) StoreSet(key string, value any) error {
	_ = key
	_ = value
	return errMilestone("设置写入", "M2")
}

// ---------------------------------------------------------------------------
// 凭据（M2 接入：keyring 三槽位 + 0600 明文回退 + 掩码回读，DR-004/沿 electron-DR-011/013）
// ---------------------------------------------------------------------------

// SecureGet 读取槽位；apiKey 已按掩码语义（****+末4位），完整 Key 不回渲染层。
func (a *App) SecureGet(slot string) (map[string]any, error) {
	_ = slot
	return nil, nil // M0 占位：空态
}

// SecureSet 写入槽位；返回实际落盘方式（明文回退时 UI 如实提示）。
func (a *App) SecureSet(slot string, payload map[string]any) (map[string]any, error) {
	_ = slot
	_ = payload
	return nil, errMilestone("凭据写入", "M2")
}

// SecureDelete 删除槽位（keyring 与回退文件双向清理）。
func (a *App) SecureDelete(slot string) error {
	_ = slot
	return errMilestone("凭据删除", "M2")
}

// ---------------------------------------------------------------------------
// 语料库（M5 接入：下载器 SSRF/zip-slip 防护 + 流式索引，06 文档）
// ---------------------------------------------------------------------------

// CorpusDownload 语料下载；进度经事件 corpus:progress 回传（载荷含 requestID）。
func (a *App) CorpusDownload(req map[string]any) error {
	_ = req
	return errMilestone("语料下载", "M5")
}

// CorpusScan 扫描语料分类（root 为空串时按 用户设置>legacy>默认 解析）。
func (a *App) CorpusScan(root string) (map[string]any, error) {
	_ = root
	return map[string]any{"root": root, "exists": false, "categories": []any{}}, nil // M0 占位：空态
}

// CorpusListEntries 列出 XQF 分类下全部 .xqf 文件（不解析）。
func (a *App) CorpusListEntries(categoryPath, categoryName string) ([]map[string]any, error) {
	_ = categoryPath
	_ = categoryName
	return []map[string]any{}, nil // M0 占位：空态
}

// CorpusReadFiles 批量读取 .xqf 文件字节（转交解析管线）。
func (a *App) CorpusReadFiles(paths []string) ([]map[string]any, error) {
	_ = paths
	return []map[string]any{}, nil // M0 占位：空态
}

// CorpusPgnIndex 大 PGN 文件按局偏移索引（流式扫描）。
func (a *App) CorpusPgnIndex(path string, maxGames int) ([]map[string]any, error) {
	_ = path
	_ = maxGames
	return []map[string]any{}, nil // M0 占位：空态
}

// CorpusReadPgnGame 读取索引指向的单局文本。
func (a *App) CorpusReadPgnGame(path string, entry map[string]any) (string, error) {
	_ = path
	_ = entry
	return "", nil // M0 占位：空态
}

// CorpusPickDirectory 桌面端"选择其他棋谱目录"；取消返回空串。
func (a *App) CorpusPickDirectory() (string, error) {
	return "", nil // M0 占位：空态
}

// ---------------------------------------------------------------------------
// 对话框与剪贴板（M2/M5 随页面接线接入；M0 返回取消语义）
// ---------------------------------------------------------------------------

// DialogSaveFile 存文件对话框；返回所选路径，取消返回空串。
func (a *App) DialogSaveFile(req map[string]any) (string, error) {
	_ = req
	return "", nil // M0 占位：取消语义
}

// DialogReadFile 读文件对话框（导入棋谱）；取消返回 null。
func (a *App) DialogReadFile() (map[string]any, error) {
	return nil, nil // M0 占位：取消语义
}

// ClipboardWrite 写系统剪贴板。
func (a *App) ClipboardWrite(text string) error {
	_ = text
	return nil // M0 占位：静默成功（M2 换 wails runtime.ClipboardSetText）
}

// ---------------------------------------------------------------------------
// 引擎 / 求解器 / 解析器（Worker 通道迁移：00 文档 §3.2，goroutine + ctx，DR-003）
// 消息形状 {id, type, payload} / {id, ok, result|error|progress} 在内部协议保留（铁律 #7）。
// ---------------------------------------------------------------------------

// EngineFindBestMove 对局 AI 应手（M3 接入：internal/engine 逐行翻译 TS 版）。
func (a *App) EngineFindBestMove(requestID, fen string, difficulty int, historyFens []string) (any, error) {
	_ = requestID
	_ = fen
	_ = difficulty
	_ = historyFens
	return nil, errMilestone("内置引擎", "M3")
}

// EngineFindBestMoveEx 参谋报告（Top-K 真实分差；M3 接入）。
func (a *App) EngineFindBestMoveEx(requestID, fen string, depth, topK, timeLimitMs int) (any, error) {
	_ = requestID
	_ = fen
	_ = depth
	_ = topK
	_ = timeLimitMs
	return nil, errMilestone("参谋报告", "M3")
}

// EngineEvaluateMove 单着法评估（护航否决用；M3 接入）。
func (a *App) EngineEvaluateMove(requestID, fen string, move map[string]any, depth int) (any, error) {
	_ = requestID
	_ = fen
	_ = move
	_ = depth
	return nil, errMilestone("着法评估", "M3")
}

// EngineCancel 取消引擎请求（context cancel；幂等）。
func (a *App) EngineCancel(requestID string) {
	_ = requestID
}

// SolverSolve 求解残局（M6 接入：internal/solver，AND/OR 迭代加深）。
func (a *App) SolverSolve(requestID, fen string, timeLimitMs, maxPlies int) (any, error) {
	_ = requestID
	_ = fen
	_ = timeLimitMs
	_ = maxPlies
	return nil, errMilestone("残局求解", "M6")
}

// SolverIsWinningFirstMove 验证首着是否必胜（LLM 求解辅助裁判；M6 接入）。
func (a *App) SolverIsWinningFirstMove(requestID, fen string, firstMove map[string]any, plies, timeLimitMs int) (bool, error) {
	_ = requestID
	_ = fen
	_ = firstMove
	_ = plies
	_ = timeLimitMs
	return false, errMilestone("首着验证", "M6")
}

// SolverCancel 取消求解请求（context cancel；幂等）。
func (a *App) SolverCancel(requestID string) {
	_ = requestID
}

// ParserParseBatch 批量解析棋谱文件字节（M5 接入；进度经事件 parser:progress 回传，
// 分批 ≤128 与 generation 防陈旧由前端收口）。
func (a *App) ParserParseBatch(requestID string, files []map[string]any) (any, error) {
	_ = requestID
	_ = files
	return nil, errMilestone("批量解析", "M5")
}

// ParserCancel 取消解析批次（幂等）。（M5 接入）
func (a *App) ParserCancel(requestID string) {
	_ = requestID
}
