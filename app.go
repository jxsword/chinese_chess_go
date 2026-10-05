package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jxsword/chinese_chess_go/internal/engine"
	"github.com/jxsword/chinese_chess_go/internal/llm"
	"github.com/jxsword/chinese_chess_go/internal/parsers"
	"github.com/jxsword/chinese_chess_go/internal/solver"
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

	// engineRunner 引擎通道（M3）：每请求独立 goroutine + ctx 取消注册表
	//（DR-003/03 §7）。
	engineRunner *engine.Runner

	// parserRunner 解析通道（M5）：与 engine.Runner 同型（goroutine 分批 +
	// ctx 取消，electron-DR-016 的 goroutine 映射，06 §6）。
	parserRunner *parsers.Runner

	// solverRunner 求解通道（M6）：与 engine.Runner 同型（goroutine + ctx 取消，
	// DR-003，04 §5）。
	solverRunner *solver.Runner

	// llmProxy LLM 传输代理（M4，05 §3.2）：受理即返回 + 事件回发 + authSlot 注入。
	llmProxy *llm.Proxy
	llmOnce  sync.Once
	// llmSenderOverride 测试注入的事件收集器（生产走 EventsEmit）。
	llmSenderOverride llm.ProxySender

	// corpusDocumentsOverride 测试注入的文档目录基路径（生产走 documentsDir()）。
	corpusDocumentsOverride string
	// parserProgressOverride 测试注入的解析进度收集器（生产走 EventsEmit）。
	parserProgressOverride func(done, total int)
	// corpusProgressOverride 测试注入的下载进度收集器（生产走 EventsEmit）。
	corpusProgressOverride func(received, total int64)
}

// NewApp 创建绑定层实例（Wails Bind 入口）。
func NewApp() *App {
	return &App{engineRunner: engine.NewRunner(), parserRunner: parsers.NewRunner(), solverRunner: solver.NewRunner()}
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

// documentsDir 文档目录（对齐 Electron app.getPath('documents') 语义，07 §1：
// 存档落在 <Documents>/chinese_chess_ultra_go.sqlite）。WSL 默认无 ~/Documents，
// 此处创建之（用户可见的标准位置）；创建失败退回主目录。
func documentsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	docs := filepath.Join(home, "Documents")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		return home
	}
	return docs
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
// 语料库（M5：扫描/条目/字节/PGN 索引，06 文档 §1/§4；下载器见 T5.5）
// ---------------------------------------------------------------------------

// CorpusDownload 语料下载；进度经事件 corpus:progress 回传（载荷含 requestId，
// 06 §5）。targetDir 为空时按 用户设置>legacy>默认 解析；阻塞至下载解压结算
// （对齐 Electron 版 ipc handler await 语义），失败整体清理。
func (a *App) CorpusDownload(req map[string]any) error {
	requestID, _ := req["requestId"].(string)
	downloadURL, _ := req["url"].(string)
	targetDir, _ := req["targetDir"].(string)
	if targetDir == "" {
		targetDir = a.corpusRoot("")
	}
	_, err := storage.DownloadCorpus(storage.DownloadCorpusOptions{
		URL:       downloadURL,
		TargetDir: targetDir,
		OnProgress: func(received, total int64) {
			if a.corpusProgressOverride != nil {
				a.corpusProgressOverride(received, total)
				return
			}
			if a.ctx == nil {
				return
			}
			wailsruntime.EventsEmit(a.ctx, "corpus:progress", map[string]any{
				"requestId": requestID, "received": received, "total": total,
			})
		},
	})
	return err
}

// corpusUserPathKey electron-store 的语料目录键（07 文档 §3 corpus.userPath）。
const corpusUserPathKey = "corpus.userPath"

// corpusRoot 解析当前生效语料目录：入参非空直用；否则按
// 用户设置 > legacy 相对目录 > 平台默认（corpus_paths.dart:150-172）。
func (a *App) corpusRoot(root string) string {
	if root != "" {
		return root
	}
	user := ""
	if a.settings != nil {
		if v, ok := a.settings.Get(corpusUserPathKey).(string); ok {
			user = v
		}
	}
	legacy := ""
	if wd, err := os.Getwd(); err == nil {
		legacy = wd
	}
	documents := documentsDir()
	if a.corpusDocumentsOverride != "" {
		documents = a.corpusDocumentsOverride
	}
	return storage.ResolveCorpusDir(storage.CorpusDirOptions{
		UserSetting:    user,
		DocumentsPath:  documents,
		LegacyBasePath: legacy,
	})
}

// CorpusScan 扫描语料分类（root 为空串时按 用户设置>legacy>默认 解析）。
func (a *App) CorpusScan(root string) (storage.CorpusScanResult, error) {
	return storage.ScanCorpus(a.corpusRoot(root)), nil
}

// CorpusListEntries 列出 XQF 分类下全部 .xqf 文件（不解析）。
func (a *App) CorpusListEntries(categoryPath, categoryName string) ([]storage.CorpusEntry, error) {
	return storage.ListXqfEntries(categoryPath, categoryName), nil
}

// CorpusReadFiles 批量读取棋谱文件字节（转交解析管线）。
// Bytes 走 JSON base64 承载（Wails 通道无结构化克隆，前端适配层解码）。
func (a *App) CorpusReadFiles(paths []string) ([]storage.CorpusFileBytes, error) {
	return storage.ReadCorpusFiles(paths), nil
}

// CorpusPgnIndex 大 PGN 文件按局偏移索引（流式扫描；maxGames ≤ 0 表示不限）。
func (a *App) CorpusPgnIndex(path string, maxGames int) ([]parsers.PgnGameIndex, error) {
	return storage.ScanPgnIndex(path, maxGames)
}

// CorpusReadPgnGame 读取索引指向的单局文本。
func (a *App) CorpusReadPgnGame(path string, entry map[string]any) (string, error) {
	raw, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	var idx parsers.PgnGameIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		return "", err
	}
	return storage.ReadPgnGameText(path, idx)
}

// CorpusPickDirectory 桌面端"选择其他棋谱目录"；取消返回空串。
func (a *App) CorpusPickDirectory() (string, error) {
	if a.ctx == nil {
		return "", nil
	}
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:                "选择语料目录",
		CanCreateDirectories: true,
	})
}

// ---------------------------------------------------------------------------
// 对话框与剪贴板（M5：导出 PGN / 导入棋谱对话框；剪贴板 M2 已接 wails runtime）
// ---------------------------------------------------------------------------

// DialogSaveFile 存文件对话框；返回所选路径，取消返回空串。
// 所选路径的内容写盘在此收口（对齐 Electron 版 dialog ipc：原生语义）。
func (a *App) DialogSaveFile(req map[string]any) (string, error) {
	defaultName, _ := req["defaultName"].(string)
	content, _ := req["content"].(string)
	if a.ctx == nil {
		return "", nil
	}
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "导出文件",
		DefaultFilename: defaultName,
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil // 取消
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// DialogReadFile 读文件对话框（导入棋谱）；取消返回 null。
func (a *App) DialogReadFile() (map[string]any, error) {
	if a.ctx == nil {
		return nil, nil
	}
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "导入棋谱",
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil // 取消
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return map[string]any{"path": path, "content": string(content)}, nil
}

// ClipboardWrite 写系统剪贴板。
func (a *App) ClipboardWrite(text string) error {
	return wailsruntime.ClipboardSetText(a.ctx, text)
}

// ---------------------------------------------------------------------------
// 引擎 / 求解器 / 解析器 / LLM / 视觉（Worker 通道迁移：00 文档 §3.2，goroutine + ctx，DR-003）
// 消息形状 {id, type, payload} / {id, ok, result|error|progress} 在内部协议保留（铁律 #7）。
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// LLM 传输（M4：internal/llm Proxy + 事件回发，00 文档 §3.2 通道映射；铁律 #4）
// ---------------------------------------------------------------------------

// llmEventSender EventsEmit 事件回发（载荷含 requestID；ctx 未就绪时静默丢弃）。
type llmEventSender struct {
	app *App
}

func (s llmEventSender) SendChunk(requestID string, delta llm.Delta) {
	if s.app.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(s.app.ctx, "llm:chunk", map[string]any{"requestId": requestID, "delta": delta})
}

func (s llmEventSender) SendDone(requestID string, text string) {
	if s.app.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(s.app.ctx, "llm:done", map[string]any{"requestId": requestID, "text": text})
}

func (s llmEventSender) SendError(requestID string, message string) {
	if s.app.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(s.app.ctx, "llm:error", map[string]any{"requestId": requestID, "message": message})
}

// llmTimeoutSeconds 空闲超时秒数来源（llm_settings_timeoutSeconds；未配置回落默认并 clamp）。
func (a *App) llmTimeoutSeconds() int {
	if a.settings == nil {
		return llm.ResolveTimeoutSeconds(nil)
	}
	return llm.ResolveTimeoutSeconds(a.settings.Get(llm.SettingKeyTimeoutSeconds))
}

// resolveApiKey 凭据槽位 → 完整 API Key（authSlot 注入用；掩码 Key 不回渲染层的
// 配对出口，DR-010 对应）。无/未配置返回空串。
func (a *App) resolveApiKey(slot string) string {
	if a.credentials == nil {
		return ""
	}
	if cfg := a.credentials.GetRaw(slot); cfg != nil {
		return cfg.APIKey
	}
	return ""
}

// llmSender 事件回发出口（测试注入收集器；生产走 EventsEmit）。
func (a *App) llmSender() llm.ProxySender {
	if a.llmSenderOverride != nil {
		return a.llmSenderOverride
	}
	return llmEventSender{app: a}
}

// getLlmProxy 懒装配 LLM 代理（受理由 llmOnce 保证单例；测试可先 initServices）。
func (a *App) getLlmProxy() *llm.Proxy {
	a.llmOnce.Do(func() {
		a.llmProxy = llm.NewProxy(llm.ProxyOptions{
			GetTimeoutSeconds: a.llmTimeoutSeconds,
			ResolveAPIKey:     a.resolveApiKey,
		})
	})
	return a.llmProxy
}

// LlmChatRequest cc:llm:chat 载荷（00 §3.1；url/body/headers 由渲染层 packages/llm 组装）。
type LlmChatRequest struct {
	RequestID string            `json:"requestId"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	AuthSlot  string            `json:"authSlot"`
}

// LlmChat 受理流式对话请求，阻塞至结算后恒 resolve（对齐 Electron 版 ipc/llm.ts
// 的 `return proxy.chat(req, safeSender)`：invoke promise 在结算后才 resolve——
// 前端 llmTransport 以 promise 结束反注册事件订阅，依赖此语义；受理即返回会
// 使事件在订阅注销后才回发、前端永挂，M4 手测 F4）。结局经事件
// llm:chunk / llm:done / llm:error 回传，载荷含 requestId。
// 取消经 LlmCancel；取消后不再有任何事件（迟到丢弃在渲染层按 requestId 收口）。
func (a *App) LlmChat(req LlmChatRequest) error {
	if req.RequestID == "" || req.URL == "" {
		return fmt.Errorf("llm chat 载荷不完整（需 requestId 与 url）")
	}
	finished := a.getLlmProxy().Chat(llm.ChatRequest{
		RequestID: req.RequestID,
		URL:       req.URL,
		Headers:   req.Headers,
		Body:      req.Body,
		AuthSlot:  req.AuthSlot,
	}, a.llmSender())
	<-finished // 阻塞至结算（正常/出错/取消），期间事件订阅存活
	return nil
}

// LlmCancel 取消在途请求（context cancel；幂等）。
func (a *App) LlmCancel(requestID string) {
	a.getLlmProxy().Cancel(requestID)
}

// LlmTestConnection 配置卡"测试连接"（单次最小流式请求，同步收集结局）。
// cfg 为渲染层 LlmEndpointConfig（掩码 Key + authSlot 由传输代理注入真实鉴权）。
func (a *App) LlmTestConnection(cfg map[string]any, authSlot string) (map[string]any, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var config llm.LlmEndpointConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	result := a.getLlmProxy().TestConnection(config, authSlot)
	return map[string]any{"ok": result.OK, "message": result.Message}, nil
}

// VisionReadBoard 视觉识图（多模态非流式，M6 接入，05 §7；识图请求恒发关闭
// 参数按预设映射——dashscope→enable_thinking:false、glm-4.5v→thinking:disabled，
// DR-005）。cfg 为渲染层 LlmEndpointConfig（掩码 Key），掩码时经 authSlot 注入
// 真实鉴权（DR-010 同机制）；失败以 error reject（已重试 N 次仍失败：…）。
func (a *App) VisionReadBoard(cfg map[string]any, imageB64, mime, authSlot string) (map[string]any, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var config llm.LlmEndpointConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reader := llm.NewVisionReader(llm.VisionReaderOptions{ResolveAPIKey: a.resolveApiKey})
	result, err := reader.ReadBoard(ctx, llm.VisionReadBoardRequest{
		Config:      config,
		ImageBase64: imageB64,
		Mime:        mime,
		AuthSlot:    authSlot,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"fen": result.Fen}, nil
}

// EngineFindBestMove 对局 AI 应手（M3：internal/engine 逐行翻译 TS 版；
// goroutine + ctx 取消，03 §7——绑定阻塞至结算，取消经 EngineCancel）。
func (a *App) EngineFindBestMove(requestID, fen string, difficulty int, historyFens []string) (any, error) {
	return a.engineCall(requestID, engine.ReqFindBestMove, engine.FindBestMovePayload{
		Fen:         fen,
		Difficulty:  difficulty, // 0=未设 → 缺省 3（03 §7 wire 缺省约定）
		HistoryFens: historyFens,
	})
}

// EngineFindBestMoveEx 参谋报告（Top-K 真实分差；M3 接入）。
func (a *App) EngineFindBestMoveEx(requestID, fen string, depth, topK, timeLimitMs int) (any, error) {
	return a.engineCall(requestID, engine.ReqFindBestMoveEx, engine.FindBestMoveExPayload{
		Fen:         fen,
		Depth:       depth,       // 0=未设 → 6
		TopK:        topK,        // 0=未设 → 5
		TimeLimitMs: timeLimitMs, // 0=未设 → 5000
	})
}

// EngineEvaluateMove 单着法评估（护航否决用；M3 接入）。move 为渲染层
// 原始 JSON 对象（{from:{col,row}, to:{col,row}}），协议层解为 WireMove。
func (a *App) EngineEvaluateMove(requestID, fen string, move map[string]any, depth int) (any, error) {
	rawMove, err := json.Marshal(move)
	if err != nil {
		return nil, err
	}
	return a.engineCall(requestID, engine.ReqEvaluateMove, engine.EvaluateMovePayload{
		Fen:   fen,
		Move:  rawMove,
		Depth: depth, // 0=未设 → 4
	})
}

// EngineCancel 取消引擎请求（context cancel；幂等）。取消后结算的迟到结果
// 由前端按 requestId 丢弃（00 §3.2 主语义）。
func (a *App) EngineCancel(requestID string) {
	a.engineRunner.Cancel(requestID)
}

// engineCall 提交请求并阻塞等待结算；失败以 error 返回（Wails invoke reject，
// 前端 engineClient 统一映射为 {ok:false, error} 响应）。
func (a *App) engineCall(requestID string, reqType engine.RequestType, payload any) (any, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp := <-a.engineRunner.Submit(engine.Request{ID: requestID, Type: reqType, Payload: raw})
	if !resp.OK {
		return nil, errors.New(resp.Error)
	}
	return resp.Result, nil
}

// SolverSolve 求解残局（M6：internal/solver，AND/OR 迭代加深；goroutine + ctx
// 取消，04 §2/§5——绑定阻塞至结算，取消经 SolverCancel）。
func (a *App) SolverSolve(requestID, fen string, timeLimitMs, maxPlies int) (any, error) {
	return a.solverCall(requestID, solver.ReqSolve, solver.SolvePayload{
		Fen:         fen,
		TimeLimitMs: timeLimitMs, // 0=未设 → 缺省 30s（04 §2 wire 缺省约定）
		MaxPlies:    maxPlies,    // 0=未设 → 缺省 9
	})
}

// SolverIsWinningFirstMove 验证首着是否必胜（LLM 求解辅助裁判，04 §3/05 §6）。
// firstMove 为渲染层原始 JSON 对象（{from:{col,row}, to:{col,row}}），
// 协议层解为 WireMove。
func (a *App) SolverIsWinningFirstMove(requestID, fen string, firstMove map[string]any, plies, timeLimitMs int) (bool, error) {
	rawMove, err := json.Marshal(firstMove)
	if err != nil {
		return false, err
	}
	result, err := a.solverCall(requestID, solver.ReqIsWinningFirstMove, solver.IsWinningFirstMovePayload{
		Fen:         fen,
		FirstMove:   rawMove,
		Plies:       plies,       // 0=未设 → 缺省 9（04 §2 wire 缺省约定）
		TimeLimitMs: timeLimitMs, // 0=未设 → 缺省 30s
	})
	if err != nil {
		return false, err
	}
	win, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("首着验证结果形状异常: %T", result)
	}
	return win, nil
}

// SolverCancel 取消求解请求（context cancel；幂等）。取消后结算的迟到结果
// 由前端按 requestId 丢弃（00 §3.2 主语义）。
func (a *App) SolverCancel(requestID string) {
	a.solverRunner.Cancel(requestID)
}

// solverCall 提交请求并阻塞等待结算；失败以 error 返回（Wails invoke reject，
// 前端 solverClient 统一映射为 {ok:false, error} 响应）。
func (a *App) solverCall(requestID string, reqType solver.RequestType, payload any) (any, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp := <-a.solverRunner.Submit(solver.Request{ID: requestID, Type: reqType, Payload: raw})
	if !resp.OK {
		return nil, errors.New(resp.Error)
	}
	return resp.Result, nil
}

// ParserParseBatch 批量解析棋谱文件字节（M5：internal/parsers 协议层，goroutine
// 分批 + ctx 取消，06 §6）。files 的 bytes 走 JSON base64 承载（Wails 通道无
// 结构化克隆，前端适配层编码）；进度经事件 parser:progress {requestId, done, total}
// 回传，分批 ≤128 与 generation 防陈旧由前端收口。
func (a *App) ParserParseBatch(requestID string, files []map[string]any) (any, error) {
	// files 为绑定入参（数组）；补齐协议载荷形状 {files: [...]}，bytes 以
	// JSON base64 解码进 []byte（前端 api/binary.ts 对端契约）。
	raw, err := json.Marshal(map[string]any{"files": files})
	if err != nil {
		return nil, err
	}
	resp := <-a.parserRunner.Submit(parsers.Request{
		ID:      requestID,
		Type:    parsers.ReqParseBatch,
		Payload: raw,
	}, a.parserProgressSender(requestID))
	if !resp.OK {
		return nil, errors.New(resp.Error)
	}
	return resp.Result, nil
}

// ParserCancel 取消解析批次（context cancel；幂等）。取消后结算的迟到结果
// 由前端按 requestId 丢弃（00 §3.2 主语义）。
func (a *App) ParserCancel(requestID string) {
	a.parserRunner.Cancel(requestID)
}

// parserProgressSender parser:progress 事件发送器（ctx 未就绪时静默丢弃；
// 测试注入收集器，沿 llmSenderOverride 同型）。
func (a *App) parserProgressSender(requestID string) func(done, total int) {
	if a.parserProgressOverride != nil {
		return a.parserProgressOverride
	}
	return func(done, total int) {
		if a.ctx == nil {
			return
		}
		wailsruntime.EventsEmit(a.ctx, "parser:progress", map[string]any{
			"requestId": requestID, "done": done, "total": total,
		})
	}
}
