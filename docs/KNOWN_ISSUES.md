# 已知问题（KNOWN_ISSUES.md）

> 记录各里程碑复审（11 §6.1）产出的 P2/P3 项与按计划后置的冲突面；P0/P1 修复后即从本表移除。
> 每里程碑收尾更新；修复后随对应里程碑 commit 注销。

## M0（工程骨架）

| # | 级别 | 描述 | 处置计划 |
|---|---|---|---|
| K1 | P2 | **LLM 思维链 UI 开关与条件关闭参数与 DR-005 冲突**：随迁的 Electron 版 `@packages/llm/config.ts`/`vision.ts` 按 `disableThinking` 用户开关条件发送 `enable_thinking:false`，LLM 配置卡含「禁用思维链」UI 开关。M0 无真实 LLM 请求（占位绑定拒绝），无运行时暴露。 | ~~M4 T4.2/T4.5 处置~~ 已注销：`internal/llm/config.go` 构造层恒发关闭参数（DR-005 预设映射：GLM→`thinking:{type:disabled}`，其余→`enable_thinking:false` 兜底，无任何开关路径）；前端 `packages/llm/config.ts`/`vision.ts` 同映射恒发；配置卡开关删除、`LlmEndpointConfig` 改四字段 `{baseUrl,apiKey,model,preset}`；请求体快照测试双端锁定（internal/llm/config_test.go 九预设断言 + frontend dr005.spec）。 |
| K2 | P3 | 桌面 bundle 约 630KB（min+gzip）含暂不执行的 TS 领域代码（`@packages/*` 随迁，DR-007 选项 A 已知代价）。 | M3/M4/M5/M6 逐包替换为 Go 绑定后从别名与依赖中移除。 |
| K3 | P3 | 自动保存 `gameAutoSave.write()` 以 `void repo.saveGame(...)` 发即忘；wails 占位绑定 reject 时产生未处理 promise 拒绝（仅控制台噪音，UI 不受影响；Electron 版同型）。 | M2 接入真实存储后自然消解；如 M2 前需静默可在适配层吞错（不采纳——保持错误可见）。 |
| K4 | P3 | `parser:progress` 事件名是 00 §3.2 未列出的同型扩展（对齐 `corpus:progress`）。 | M5 落地时如有更名，同步 `api/parserClient.ts` 与 Go 侧 EventsEmit。 |
| K5 | P3 | WSLg 下 `xwd` 对 wails 窗口截图全黑（WebKitGPU 合成面不落 X pixmap 的取证伪影；`WEBKIT_DISABLE_DMABUF_RENDERER=1` 亦不改变 xwd 结果）。窗口存在性与 devserver 响应已取证；真实渲染效果以用户手测为准（10 §3 R7'）。 | 长期观察；若用户手测发现真渲染缺陷再立项（浏览器 mock 模式为开发兜底）。 |
| K6 | P3 | `frontend/node_modules` 内含 Go 代码（flatted 的 golang 实现），被根模块 `go vet/test ./...` 扫描（当前可编译、无测试，无害）。嵌套 go.mod 屏障方案因 `//go:embed` 跨模块限制不可用（已实测）。 | 若未来 npm 依赖引入损坏的 Go 文件破坏质量门，在 CI/本地命令加 `grep -v node_modules` 过滤。 |

## M1（规则内核 + L3）

| # | 级别 | 描述 | 处置计划 |
|---|---|---|---|
| K7 | P3 | **buildFen 接口收敛**：Electron 版 `buildFen({board,isRedTurn,halfMove?,fullMove?})` 的 halfMove/fullMove 参数在原版全代码库**零调用点**使用（grep 证实：studioValidate/EndgameStudioPage/vision/xqfParser/pgnParser 均只传 board+isRedTurn），Go 版收敛为 `BuildFen(board, isRedTurn)` 固定输出 `- - 0 1`（02 §1「恒 halfMove=0」口径）。 | 无需处置；M5 解析器如出现需要自定义回合计数的场景再扩参（预期不需要）。 |
| K8 | P3 | **Board.grid 用 BoardGrid（[][]*Piece）而非 02 §1 草图的 [10][9]*Piece 数组**：与 TS 版 BoardGrid 同构（parseBoardFen/buildFen 全链路共用一型），行为等价（定长 10×9 由解析/拼装强制）。 | 无需处置；设计草图与本实现的形态差异已在本条留档。 |
| K9 | P3 | **inPalace/inOwnHalf/inOpponentHalf 未导出**（TS 版导出）：TS 包外唯一使用方是前端 setupRules.ts（M0 已随前端 TS 移植）；Go 侧暂无调用者。 | M6 工作室如需 Go 侧摆盘校验再导出（一行改动）。 |
| K10 | P3 | **repetitionJudge 对历史中无效 FEN 的行为差**：TS 版 `Board.fromFen` 会 throw；Go 版 FromFen 返回 error，classifyCycle 对无效 FEN 记 gaveCheck=false 继续（注释已说明）。该路径不可达（fenHistory 由本方对局 ToFen 采集），且与 02 §1「引擎侧对无效 FEN 静默跳过」口径一致。 | 无需处置；留档作为跨语言审计点。 |
| K11 | P3 | **moveNotation 退化输入行为差**：from==to 时 Go 版越界 panic，TS 版产出含 undefined 的废串（两者均为坏输出）；真实走法 from≠to 恒成立，函数注释已声明前置条件。 | 无需处置；如未来出现退化调用方，在调用侧加校验（内核不加防御分支，保持逐行翻译）。 |

## M2（对战页 + 存储）

| # | 级别 | 描述 | 处置计划 |
|---|---|---|---|
| K12 | P3 | **凭据槽位契约面字段差（DR-005 消解面）**：Go `SecureGet/Set` 槽位 JSON 为 `{baseUrl, apiKey, model, preset}`（07 §4），随迁前端 `LlmEndpointConfig` 类型仍含 `disableThinking`、缺 `preset`——M2 仅接存储无人消费该面（双人页不用 secure 通道），`disableThinking` 被 json 丢弃、`SecureGet` 返回缺该字段（配置卡开关显示为关）。 | ~~M4 T4.5 配置卡接线~~ 已注销：与 K1 同批消解——前端类型改 `{baseUrl,apiKey,model,preset}` 并全部消费点改写（旧存档无 preset 字段时 presetFor 回退按 baseUrl 匹配、thinkingStyleFor 走兜底，向后兼容）；`@shared/ipc/types` 同步。 |
| K13 | P3 | **Wails v2 无窗口 minimize/blur 后端事件**：07 §2 映射表的 Electron `win.on('blur')/('minimize')` 无 Go 等价物。适配层以 WebView `window.blur` 派发 `blur` 相位补偿（最小化在主流平台伴随失焦）；`close`/`before-quit` 由 OnBeforeClose 发出（有界等待 300ms best-effort 存档）。极平台最小化不伴随失焦时漏一次自动保存。 | 有路由卸载（dispose）/close/手动保存三重兜底，可接受；Wails v3 或后续版本提供窗口事件时补齐。 |
| K14 | P3 | **parseMovesJson 非整数值丢弃**（db.ts 保留 `number[][]`，Go 侧仅收整数四元组）：TS 保留的 1.5 等行在 restore 必被跳脏跳过，净效果一致（db.go 注释留档）。 | 无需处置；跨语言审计点。 |
| K15 | P3 | **settings clamp 边界差**：load 时 clamp 仅对已存在键生效（缺省键不注入内存表，`StoreGet` 返回 null 由渲染层 fromRaw 兜底——electron-store 虚拟缺省语义）；present 数值截断取整（7.9→7，TS fromRaw 保留 7.9 后由渲染层再 clamp）。仅手改 settings.json 场景可观测。 | 无需处置；M4 Go 侧消费者（代理空闲超时）注意 nil→默认兜底。 |
| K16 | P3 | **beforeClose 有界等待 300ms**：退出前 fire-and-forget 自动存档的 best-effort 窗口（等价 Electron 同步 best-effort 语义）；极端慢盘下最后一着可能不入档。 | 可接受（与 Electron 版同级保真）；如手测出现高频丢档再改前台等待确认。 |

## M3（引擎 + L0/L1/L2 重复治理）

| # | 级别 | 描述 | 处置计划 |
|---|---|---|---|
| K17 | P3 | **Runner 同 id 并发在途注册表以后到者为准**：Submit 重写 cancels[id]，先到请求结算时 delete 会摘走后到的注册项，其后 Cancel(id) 对后到请求失效。requestId 全局唯一是 00 §3.2 调用方契约（前端 createRequestId UUID 工厂，页面 gameSeq 保证至多一个在途 AI 搜索），契约内不可达。 | 无需处置；如未来出现同 id 复用在途场景，Submit 改为拒绝重复 id（一行改动）。 |
| K18 | P3 | **随机路径（难度 1/2 与 L2 候选洗牌）跨语言伪随机源不同**：TS `Math.random` vs Go `math/rand/v2`（自动播种）。randomness>0 时难度 1/2 的具体应手、L2 阈值内多候选时的具体换着，Go 与 TS/Dart 不逐位一致——原版同分随机即不保证复现（金标准对拍仅覆盖 randomness=0 路径）。 | 无需处置；行为面（随机取一）与阈值/候选集口径逐值一致，确定性测试以注入随机源锁定（avoidance_test）。 |

## M4（LLM 全链路）

| # | 级别 | 描述 | 处置计划 |
|---|---|---|---|
| K19 | P3 | **构造器缺省语义差（TS `??` vs Go 零值兜底）**：`maxAttempts` 显式传 0 时 Go 兜底为 3，TS `?? 3` 仅对 undefined 兜底（显式 0 = 零次尝试直接降级）；`HybridLlmPlayer.strengthBlend` 反向——Go 不对 0 兜底（0 是合法"最严"档，缺省 50 由设置层 DEFAULT 保证），TS `?? 50` 同样仅 undefined 兜底。设置层 clamp（maxAttempts 1~10 / blend 0~100）使两差均不可达。 | 无需处置；M7 cmd/eval 直接构造时注意传入显式值。 |
| K20 | P3 | **excerpt 按 rune 计数 vs TS 按 UTF-16 code unit**：HTTP≠200 错误体截 160 字符，BMP 内字符（含中文）两者逐位一致；错误体含非 BMP 字符（emoji 等）时截断长度可能差 1（Go 1 rune = TS 2 unit）。 | 无需处置；错误消息为人类可读文本，长度差不影响断言（现有用例以"≤160 字符 + 省略号"口径锁定）。 |
| K21 | P3 | **流式 OnChunk 与取消结算的纳秒级 TOCTOU 窗口**：blocked 检查（锁内）与 OnChunk 回调（锁外）之间并发 Cancel 完成结算时，`Cancel()` 返回后仍可能收到一个 chunk。TS 版事件经 IPC 异步转发同型窗口天然存在；渲染层按 requestId 二次收口（00 §3.2 主语义）为设计第二道防线。 | 无需处置；锁内回调用户代码有死锁风险，不采纳收紧方案。 |
| K22 | P3 | **chunk.error 消息内 JSON 键序差**：`流式响应错误: {…}` 的 JSON 序列化 Go（map 字母序）与 TS（对象插入序）键排列可能不同——前缀与 message 内容逐字一致，整体串仅键序差。 | 无需处置；测试以"含前缀 + 含 message"口径断言，不锁键序。 |

## M5（语料 + 棋谱）

| # | 级别 | 描述 | 处置计划 |
|---|---|---|---|
| K23 | P2 | **ParserCancel 无"请求到达前"粘性记忆**：TS parser.worker 用 `cancelledIds` 记住先于 parseBatch 到达的 cancel，随后同 id 请求立即回 canceled；Go Runner（DR-003 goroutine 模型）无此粘性记忆，Cancel 先于 Submit 到达时为 no-op，批次会完整跑完。绑定层架构下该窗口实际不可达：前端 parserClient 先 post parseBatch（Submit 即注册 ctx）后才可能 cancel，且 cancel 已同步 reject pending 表——批次跑完只是浪费 ≤128 文件的解析，进度/结果按 requestId 丢弃（00 §3.2 主语义兜底），与 engine.Runner M3 先例（K17）同型。 | 无需处置；粘性集合在无界 id 下泄漏内存，不采纳。 |
| K24 | P3 | **zip 条目错误语义两处刻意偏离（标准库替换的代价，均加强安全）**：① archive/zip 读取时校验 CRC——坏条目 Go 跳过并计数，TS 手写解析不校验照写坏数据再 count++；② 加密/不支持压缩方法条目 Go 在迭代中遇到才抛错（此前条目已写入；atomic 路径 staging 会整体清理，仅 legacy 符号链接直解压路径可能留半成品），TS 在写任何文件前整体抛错。 | 无需处置；①由 TestExtractZipCorruptEntrySkippedNotAborts 锁定，②语料包场景不可能出现加密条目。 |
| K25 | P3 | **PGN 正则 `\s` 与 JS Unicode 空白差**：JS `\s` 匹配 U+00A0/U+3000/U+FEFF 等，Go（RE2 + unicode.IsSpace 全文去空白）标签行前缀/`序号.`间隔含此类字符时行为可能分叉；PGN 实际语料为 ASCII 空白。 | 无需处置；语料实测无影响。 |
| K26 | P3 | **DecodeUtf8Lossy 替换符粒度差**：Go `strings.ToValidUTF8` 连续一段非法字节产 1 个 U+FFFD，TextDecoder 按 WHATWG 规则逐非法子串产出；仅影响含坏字节的标签值/PGN 正文展示。 | 无需处置；文件名/正文为人类可读文本。 |
| K27 | P3 | **前/后/中省略列号分支多列歧义 token 的 from 选择序**：TS byCol 为 Map（列插入序确定 matches 覆盖序），Go map 随机序——两个候选 from 合法走到同一 to 的病态局面（如同方 4 车）下实际选中的 from 可能与 TS 不同；可唯一消解的常规 token 不受影响（len==1 判定）。 | 无需处置；token 本身歧义，语义无对错。 |
| K28 | P3 | **readXqfString 越界分支差**：`lenOffset+1+len > len(b)` 时 Go 返回空串，TS subarray 收缩到 EOF 后仍解码剩余字节；受 ParseXqf 最小长度（0x400+8）约束实际不可达。 | 无需处置。 |
| K29 | P3 | **语料扫描三处边缘差**：① legacy 目录判定 Go 校验 IsDir（TS existsSync 对文件也放行）且空串 legacyBasePath 直接跳过；② 语料根 readdir 失败 Go 返回 exists=true+空分类（前端归一为下载引导），TS 抛错由前端 catch 落到同一引导——UI 结果同型；③ sort.Slice 不稳定 + ReadDir 按名排序，不同子目录同名条目间相对顺序可能与 TS 漂移。 | 无需处置；①是收紧，②UI 结果一致，③排序键本身确定。 |
| K30 | P3 | **下载器四处边缘差（校验结论不同但拨号均不可达，进度/续传仅极端服务器触发）**：① 空 ETag：TS 会发 `If-Range:""` 启用续传，Go 判空不启用（更安全）；② Content-Length 非数字：Go 记 -1（进度 indeterminate），TS 传 NaN；③ WHATWG `new URL` 对 host 百分号解码/规范化（`127%2e0%2e0%2e1` 被拒），Go `url.Parse` 不解码放行——拨号同样失败；④ 并发 CorpusDownload 同 URL 临时文件竞争与 TS 同形，UI 下载按钮单飞（disabled）兜底。 | 无需处置。 |

## M6（工作室 + 求解器 + 识图）

| # | 级别 | 描述 | 处置计划 |
|---|---|---|---|
| K31 | P3 | **求解器置换表键 Zobrist（Go）vs 完整 FEN 串（TS）**：04 §2 明定的 Go 版差异——uint64 键表独立于引擎；语义等价（键含轮走方，FEN 亦含），实际行为差仅在理论哈希碰撞窗口（本包私有 64 位键表，工程上可忽略）；路径去重/表项语义逐行一致。 | 无需处置；6 验证 FEN 与确定性用例锁定行为一致。 |
| K32 | P3 | **识图错误消息两处微差**：① 非法棋子条目错误内的 `JSON.stringify(record)` 键序——Go map 序列化为字母序，TS 为插入序（中文前缀与字段值逐字一致）；② requestOnce 响应体 io.LimitReader 32MB 上限，TS 无上限——超大响应 Go 截断为"响应不是合法 JSON"进重试，实际端点响应远小于该值。 | 无需处置；测试以错误前缀口径断言，不锁键序。 |
| K33 | P3 | **识图无取消通道（对齐 Electron 版）**：VisionReadBoard 绑定阻塞至结算，应用退出（父 ctx 取消）时在途识图以"连接失败/超时"错误结算即弃；单次 120s × 2 次上界与 TS AbortController 同型。UI 侧"识别中"按钮 disabled 防重入（M0 移植），无中途取消入口与原版一致。 | 无需处置；如需中途取消先在 Electron 版立规格再对齐。 |

## M7（评估 + 打包发布）

| # | 级别 | 描述 | 处置计划 |
|---|---|---|---|
| K34 | P3 | **Windows CI 平台性测试差异（已修复）+ 一处误判纠正**：①sqlite 句柄未关致 t.TempDir() RemoveAll 失败（Windows 不可删打开中的文件，09 §3 平台性教训——newTestApp/TestLazyDaoOpenRetry 补 t.Cleanup 关闭）；②回退文件 0600 断言在 Windows 恒败（无 POSIX 权限位，Chmod 仅置只读位、Stat 恒 0666，"仅当前用户可读"由 NTFS ACL 承载——非 Windows 才断言）。另纠正：Windows runner 预装 MinGW，-race 可跑（M0 ci.yml 实证），release.yml 初版"无 CGO 工具链退化无 -race"系误判，已改回全平台 -race。 | 已修复（同 commit）；release v1.0.0-rc1 重打 tag 后全平台 -race 生效。 |
| K35 | P3 | **AppImage 体积 ~80MB**：linuxdeploy 按 AppDir 约束打包 webkit2gtk 全量传递依赖 so（webview 运行时固有体积），deb/tar.gz 仅 7MB/16MB 二进制本体。 | 无需处置；格式固有。若需瘦身需换 webkit 运行时宿主策略（超 v1.0 范围）。 |
| K36 | P3 | **MatchRunner 逐手质量评估深度差在 profile 评估中的口径提示**：eval CLI 质量评估固定 qualityDepth=4（FindBestMoveEx depth4/Top3），而 hybrid 参谋实际用 advisorDifficulty 5（depth6）出候选——候选/护航模式的 Top-3 失随统计不恒为 0（短名单深度高于评估深度）；与 TS eval.ts 逐行同构，非 Go 偏差，仅口径解读提示（DoD #4 的"失误率 ≈0%"以 Electron 版同口径跑真实端点对比为准）。 | 无需处置；跨语言逐位同构，对比结论有效。 |

## 已修复（保留记录）


| # | 级别 | 描述 | 修复 |
|---|---|---|---|
| F1 | P0 | api 适配器探测条件过宽：Wails 运行时先同步注入 `window.go = {}`（App 方法表异步经 SetBindings 填充），以 `window.go` 真值判定会在外部浏览器打开 devserver 时误选 wailsAdapter 并在创建期抛错白屏。 | ~~`client.ts` 探测改为 `window.go?.app?.App !== undefined`~~（M2 勘误：该修复基于错误命名空间假设 `app.App`，实际把缺陷转为静默回落 mock，见 F2）。 |
| K3 | P3 | 自动保存 `gameAutoSave.write()` 以 `void repo.saveGame(...)` 发即忘；wails 占位绑定 reject 时产生未处理 promise 拒绝（仅控制台噪音，UI 不受影响；Electron 版同型）。 | M2 已注销：db 通道接入真实存储（app.go DbSaveGame），占位拒绝噪音消解；存储真不可用时的 reject 仍保持错误可见（属正确行为面）。 |
| P1' | P1 | 凭据服务无互斥：Wails 绑定方法并发进入时回退文件读改写竞态可丢槽位更新。 | T2.4 复审修复：Credentials 全路径加 sync.Mutex（合并路径 getRawLocked 复用锁内调用防死锁）+ 并发用例 -race 覆盖。 |
| F2 | P0 | **绑定命名空间错误（M2 手测 B/C 缺陷根因）**：Wails v2 以绑定结构体所在 Go 包名挂载方法表（`window.wailsbindings={"main":{"App":{...}}}`），本项目为 `window.go.main.App`；M0 起探测与 wailsAdapter/engineClient/solverClient/parserClient 全部误用 `window.go.app.App`（恒 undefined）→ 桌面窗口内 client.ts 静默回落 mock——自动/手动存档不落库（无 sqlite 文件）、设置切换不写 settings.json、分享棋局剪贴板为 mock 空操作。M0 的"外部浏览器回落 mock"观察实为同一根因的表现（F1 的修复掩盖了它）。 | `client.ts` 探测、wailsAdapter、engine/solver/parser 三客户端统一改为 `main.App`（对齐 wailsjs 生成物与 runtime.js 注入事实）；documentsDir 在 ~/Documents 缺失时创建（07 §1 路径落地）。回归：新增 test/api/wailsAdapter.spec.ts 5 用例锁定命名空间契约与生命周期双源；端到端取证（devserver + 真实绑定调用）——DbSaveGame/DbLoadLatest 往返、settings.json 落盘、~/Documents/chinese_chess_ultra_go.sqlite 生成、ClipboardWrite 成功。 |
| F4 | P0 | **LLM 对局永挂（M4 手测）**：大模型对战页配置测试连接成功，但对局开始后一直"思考中"——无 chunk/done/error、无重试、永不降级。根因：Go 绑定 `LlmChat` 误实现为"受理即返回"（Wails invoke 立即 resolve），而前端 `llmTransport.chat()` 以 promise 结束反注册 llm:chunk/done/error 三事件订阅——**订阅在请求发出前即被注销**，Go 侧（早已正常结算）回发的全部事件无人接收，TS `chatOnce` 的 promise 永挂。Electron 版同链路成立的原因：`ipcMain.handle(CC.llm.chat) => proxy.chat(req, sender)` 是阻塞至结算才 resolve（llm-proxy.ts chat 为 async await 全流程），订阅存活期覆盖整个流。00 §3.2"恒 resolve 语义=方法立即返回受理"为错误表述（M0 占位注释沿用至今）。 | `App.LlmChat` 改为阻塞至结算后恒 resolve（对齐 Electron ipc handler 语义；取消经 LlmCancel → 结算返回）；00 §3.2 表述修正；绑定测试改严格断言（LlmChat 返回时事件必须已全部送达——若退回受理即返回该用例立即失败）；手测复测：千问 qwen3.8-max 连接+对局。 | 
| F3 | P1 | **存档恢复盘面被改写（M2 手测第二轮）**：恢复后黑马位置出现黑车、原位车消失、走法历史清空。根因：保存端 `serialize()` 存当前（终局）局面 FEN 而 `restore()` 以其为基准重放着法栈（Flutter `board_vm.dart:308-311` 与 `:87-132` 语义错位，原版即有、Electron 逐字继承）——起点在终局盘面恰有子的着法被误重放（手测局：黑车 h9 被当作马 h9→g7 的起点"再走"，车覆盖马）。 | DR-008 修复：serialize 存**本局起始 FEN** + 完整着法栈（GameVm `_startFen`；构造/newGame/newGameFromFen/restore 设定），restore 从起始局面重放重建整局——局面/历史/fenHistory 全部正确，跳脏语义保留。用例改写 3 处（gameVmFenHistory/autoSaveRestore/gameStore spec，注明依据）；端到端复现用户 4 步对局经真实 Go 存储：恢复后局面逐位一致、4 手历史完整。跨语言差异：Go 版恢复语义与原版不同（原版带缺陷），已在 07 §2 与 decision_log DR-008 留档。旧存档（修复前写入、fen 为终局语义）不兼容，重新保存一次即覆盖。 |
