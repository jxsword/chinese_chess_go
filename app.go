package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jxsword/chinese_chess_go/internal/storage"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 绑定层（00 文档 §3.2「通道清单」→ Wails 绑定方法 + EventsEmit 事件）。
//
// M2 接入 db/store/secure/clipboard 通道与生命周期事件；engine/solver/parser/llm
// 等通道仍为占位，随 M3+ 里程碑替换（占位行为见 errMilestone）。
//
// 装配纪律（AGENTS.md）：
//   - 本文件只做 Wails 方法/事件装配与 userData/生命周期管理，禁止领域逻辑（进 internal/*）；
//   - 异步方法一律带 requestID，取消经 context.Context（DR-003），取消后不得再发任何事件；
//   - 对外 HTTP 只在 internal/llm/transport（铁律 #4），本层不直接发起网络请求。
type App struct {
	ctx context.Context

	mu          sync.Mutex
	dao         *storage.ChessDao
	daoPath     string // <Documents>/chinese_chess_ultra_go.sqlite（07 §1）
	settings    *storage.Settings
	credentials *storage.Credentials
}

// NewApp 创建绑定层实例（Wails Bind 入口）。
func NewApp() *App {
	return &App{}
}

// errMilestone 占位方法统一错误：指明方法与计划接入里程碑。
func errMilestone(method, milestone string) error {
	return fmt.Errorf("功能尚未接入（%s 计划于里程碑 %s 实现）", method, milestone)
}

// errStorageUnavailable 存储不可用（磁盘/权限异常）：invoke 拒绝，
// 渲染层按"本地存储不可用"降级、新局兜底（07 §1 懒打开语义）。
var errStorageUnavailable = fmt.Errorf("本地存储不可用")

// userDataDirName 配置目录名（<UserConfigDir>/chinese_chess_ultra_go）。
const userDataDirName = "chinese_chess_ultra_go"

// databaseFilename SQLite 库文件名（07 §1；M2 实现期勘误记录见 design_docs/07 §1）。
const databaseFilename = "chinese_chess_ultra_go.sqlite"

// startup 保存 Wails 运行时上下文并初始化 M2 存储（懒打开仅指 SQLite；
// 设置/凭据服务在启动时装配，文件 IO 延迟到首次读写）。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	configDir, err := os.UserConfigDir()
	if err != nil {
		// 无 XDG/Home 环境（异常部署）：设置/凭据退化为不可用，invoke 拒绝
		return
	}
	userData := filepath.Join(configDir, userDataDirName)
	a.initServices(userData, filepath.Join(documentsDir(), databaseFilename), storage.OSKeyring{})
}

// initServices 装配设置/凭据/DAO 路径（测试注入临时目录与伪 keyring）。
func (a *App) initServices(userDataDir, daoPath string, kr storage.Keyring) {
	a.daoPath = daoPath
	if s, err := storage.OpenSettings(userDataDir); err == nil {
		a.settings = s
	}
	a.credentials = storage.NewCredentials(kr, filepath.Join(userDataDir, storage.CredentialsFallbackFilename))
}

// documentsDir 文档目录（对齐 Electron app.getPath('documents') 语义）；
// 无 Documents 目录时退回主目录。
func documentsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	docs := filepath.Join(home, "Documents")
	if info, statErr := os.Stat(docs); statErr == nil && info.IsDir() {
		return docs
	}
	return home
}

// getDao 懒打开数据库（07 §1）：首次调用建库；失败不缓存，下次调用重试
// （等价 Electron registerDbIpc 的 `if (dao === null) dao = openDatabase()`）。
func (a *App) getDao() (*storage.ChessDao, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.dao != nil {
		return a.dao, nil
	}
	if a.daoPath == "" {
		return nil, errStorageUnavailable
	}
	dao, err := storage.OpenDao(a.daoPath)
	if err != nil {
		return nil, errStorageUnavailable
	}
	a.dao = dao
	return dao, nil
}

// beforeClose 生命周期收口（07 §2 映射表）：窗口关闭/进程退出前广播 close 与
// before-quit 相位，驱动渲染层 GameAutoSave 的离开保存；给 fire-and-forget 存档
// 一个有界等待（等价 Electron 版"同步 best-effort 写入"），随后放行退出。
func (a *App) beforeClose(ctx context.Context) bool {
	if a.ctx == nil {
		a.ctx = ctx
	}
	wailsruntime.EventsEmit(a.ctx, "app:lifecycle", map[string]string{"phase": "close"})
	time.Sleep(300 * time.Millisecond)
	wailsruntime.EventsEmit(a.ctx, "app:lifecycle", map[string]string{"phase": "before-quit"})
	time.Sleep(100 * time.Millisecond)
	return false // 放行退出
}

// ---------------------------------------------------------------------------
// 数据库（internal/storage：modernc.org/sqlite，Schema 与 Electron 版逐字段一致）
// ---------------------------------------------------------------------------

// SaveGameRequest cc:db:saveGame 载荷（00 §3.1；moves 为裸四元组，07 §1.1/§1.3）。
type SaveGameRequest struct {
	Mode  string  `json:"mode"`
	Fen   string  `json:"fen"`
	Moves [][]int `json:"moves"`
}

// DbSaveGame 自动存档（每 mode 一条 upsert）。
func (a *App) DbSaveGame(req SaveGameRequest) error {
	dao, err := a.getDao()
	if err != nil {
		return err
	}
	_, err = dao.UpsertForMode(req.Mode, req.Fen, req.Moves)
	return err
}

// DbLoadLatest 读取模式存档；无存档返回 null。
func (a *App) DbLoadLatest(mode string) (*storage.SavedGame, error) {
	dao, err := a.getDao()
	if err != nil {
		return nil, err
	}
	return dao.LatestForMode(mode)
}

// DbDeleteForMode 删除模式存档（恢复后死局清理等场景，07 §2）。
func (a *App) DbDeleteForMode(mode string) error {
	dao, err := a.getDao()
	if err != nil {
		return err
	}
	return dao.DeleteForMode(mode)
}

// DbRecordsList 棋谱库列表（摘要）。
func (a *App) DbRecordsList() ([]storage.GameRecordSummary, error) {
	dao, err := a.getDao()
	if err != nil {
		return nil, err
	}
	return dao.RecordSummaries()
}

// DbRecordsGet 读取单条棋谱；不存在返回 null。
func (a *App) DbRecordsGet(id int64) (*storage.GameRecord, error) {
	dao, err := a.getDao()
	if err != nil {
		return nil, err
	}
	return dao.RecordByID(id)
}

// DbRecordsSave 写入棋谱，返回新 id。
func (a *App) DbRecordsSave(record storage.GameRecord) (int64, error) {
	dao, err := a.getDao()
	if err != nil {
		return 0, err
	}
	return dao.InsertRecord(&record)
}

// DbRecordsDelete 删除棋谱。
func (a *App) DbRecordsDelete(id int64) error {
	dao, err := a.getDao()
	if err != nil {
		return err
	}
	return dao.DeleteRecord(id)
}

// ---------------------------------------------------------------------------
// 设置存储（internal/storage Settings：JSON 配置文件，键名沿用原版前缀；越界 clamp）
// ---------------------------------------------------------------------------

// StoreGet 读取设置键；未设置返回 null；global_auto_save 缺省 true。
func (a *App) StoreGet(key string) (any, error) {
	if a.settings == nil {
		return nil, errStorageUnavailable
	}
	return a.settings.Get(key), nil
}

// StoreSet 写入设置键（原子写 settings.json）。
func (a *App) StoreSet(key string, value any) error {
	if a.settings == nil {
		return errStorageUnavailable
	}
	return a.settings.Set(key, value)
}

// ---------------------------------------------------------------------------
// 凭据（keyring 三槽位 + 0600 明文回退 + 掩码回读，DR-004/沿 electron-DR-011/013）
// ---------------------------------------------------------------------------

// SecureSlotPayload 凭据槽位载荷（07 §4：{baseUrl, apiKey, model, preset}；
// 无 disableThinking 字段——DR-005，关闭参数在请求构造层恒发）。
type SecureSlotPayload struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
	Model   string `json:"model"`
	Preset  string `json:"preset"`
}

// SecureGet 读取槽位；apiKey 已按掩码语义（****+末 4 位），完整 Key 不回渲染层。
// 未配置/损坏返回 null。
func (a *App) SecureGet(slot string) (map[string]any, error) {
	if a.credentials == nil {
		return nil, errStorageUnavailable
	}
	cfg := a.credentials.Get(slot)
	if cfg == nil {
		return nil, nil
	}
	return map[string]any{
		"baseUrl": cfg.BaseURL,
		"apiKey":  cfg.APIKey,
		"model":   cfg.Model,
		"preset":  cfg.Preset,
	}, nil
}

// SecureSet 写入槽位；返回实际落盘方式（plainFallback 时 UI 如实提示
// "已用未加密本地文件存储"，沿 electron-DR-011）。
func (a *App) SecureSet(slot string, payload SecureSlotPayload) (storage.SecureSetResult, error) {
	if a.credentials == nil {
		return storage.SecureSetResult{}, errStorageUnavailable
	}
	return a.credentials.Set(slot, storage.SlotConfig{
		BaseURL: payload.BaseURL,
		APIKey:  payload.APIKey,
		Model:   payload.Model,
		Preset:  payload.Preset,
	})
}

// SecureDelete 删除槽位（keyring 与回退文件双向清理；幂等）。
func (a *App) SecureDelete(slot string) error {
	if a.credentials == nil {
		return errStorageUnavailable
	}
	return a.credentials.Delete(slot)
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
// 对话框与剪贴板（M5 随棋谱导出接入对话框；剪贴板 M2 已接 wails runtime）
// ---------------------------------------------------------------------------

// DialogSaveFile 存文件对话框；返回所选路径，取消返回空串。
func (a *App) DialogSaveFile(req map[string]any) (string, error) {
	_ = req
	return "", nil // 占位：取消语义（M5 换 wails runtime.SaveFileDialog）
}

// DialogReadFile 读文件对话框（导入棋谱）；取消返回 null。
func (a *App) DialogReadFile() (map[string]any, error) {
	return nil, nil // 占位：取消语义（M5）
}

// ClipboardWrite 写系统剪贴板。
func (a *App) ClipboardWrite(text string) error {
	return wailsruntime.ClipboardSetText(a.ctx, text)
}

// ---------------------------------------------------------------------------
// 引擎 / 求解器 / 解析器 / LLM / 视觉（Worker 通道迁移：00 文档 §3.2，goroutine + ctx，DR-003）
// 消息形状 {id, type, payload} / {id, ok, result|error|progress} 在内部协议保留（铁律 #7）。
// ---------------------------------------------------------------------------

// LlmChat 受理流式对话请求（恒 resolve 语义 = 受理即返回）；
// 结局经事件 llm:chunk / llm:done / llm:error 回传，载荷含 requestID。（M4 接入）
func (a *App) LlmChat(req map[string]any) error {
	_ = req // {requestId, url, headers, body, authSlot}
	return errMilestone("LLM 对话", "M4")
}

// LlmCancel 取消在途请求（context cancel；幂等）。（M4 接入）
func (a *App) LlmCancel(requestID string) {
	_ = requestID
}

// LlmTestConnection 配置卡"测试连接"（单次非流式，同步结果）。（M4 接入）
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
