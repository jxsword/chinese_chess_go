# 开发进度（PROGRESS.md）

> 与 AGENTS.md §开发规范联动：每个子任务/里程碑收尾更新本文件；**禁止代码领先文档**。
> 状态图例：⬜ 未开始 ｜ 🔵 进行中 ｜ ✅ 完成（测试全绿）｜ ✅+用户 已通过用户手测验收

## 当前状态

**M6（工作室 + 求解器 + 识图）代码完成，待用户手测**——T6.1~T6.5 全部落地（internal/solver AND/OR 迭代加深 + uint64 Zobrist 置换表 + ctx 取消、6 验证 FEN 金标准全绿；internal/llm/vision 识图协议逐字 + VisionReader + DR-005 预设恒发关闭参数；app.go Solver*/VisionReadBoard 绑定真接线 + wailsAdapter mime/authSlot 透传；前端工作室三 Tab/演示播放器为 M0 移植资产，本里程碑验证 43+10 用例）、两轮复审完成（语义一致性：design_docs/04 §2/§5 文档同步补丁；缺陷扫描：铁律 grep 全过，Go-TS 边缘差 K31~K33 留档）、质量门全绿。手测清单见下方 M6 交付段。

## 里程碑总览

| 里程碑 | 状态 | tag | 备注 |
|---|---|---|---|
| 文档集（首次提交） | ✅ | — | 00~11 全套 + AGENTS.md + DR-001~006 |
| M0 工程骨架 | ✅+用户 | — | go.mod + Wails + frontend 移植 + CI（用户以启动 M1 验收通过） |
| M1 规则内核 + L3 | ✅ | — | 金标准对拍 56 案例全绿 + L3 三类环裁决（T1.1~T1.5；用户以启动 M2 验收通过） |
| M2 对战页 + 存储 | ✅ | — | sqlite DAO + 设置/凭据 + fenHistory 四收口 + 双人页裁决接线（T2.1~T2.4；用户以启动 M3 验收通过） |
| M3 引擎 + L0/L1/L2 | ✅ | | engine.json 对拍全绿（含慢速集）→ L1/L2 → 人机页 Go 引擎接线（T3.1~T3.5；用户以启动 M4 验收通过） |
| M4 LLM 全链路 | 🔵 代码完成，待手测 | | 恒关思维链（DR-005）+ mock SSE 全场景 + 真实端点手测（T4.1~T4.5） |
| M5 语料 + 棋谱 | 🔵 代码完成，待手测 | | ICCS/PGN/XQF 解析 + 大文件流式索引 + 语料库页 + 下载器 + 棋谱库（T5.1~T5.6） |
| M6 工作室 + 求解器 + 识图 | 🔵 代码完成，待手测 | | AND/OR 求解器 + 6 验证 FEN + 工作室三 Tab + 视觉识图（DR-005）+ 求解辅助 + 演示播放器（T6.1~T6.5） |
| M7 评估 + 打包发布 | ⬜ | | MatchRunner + 三平台 Release |

## 变更日志

### 2026-10-05 M6 工作室 + 求解器 + 识图 交付（T6.1~T6.5，待手测）

**完成清单**（internal/solver 2 文件 + 2 测试；internal/llm/vision 2 文件；app.go 绑定接线；wailsAdapter 识图透传；Go 47 新用例 + 前端 241 用例全绿）
- T6.1（`feat(m6)` e12a207）：`internal/solver/endgameSolver.go`——AND/OR 迭代加深杀棋搜索逐行翻译（OR 节点存在一着必胜/AND 节点全防着皆败/将杀困毙判胜/路径去重近似长将/多解枚举 _enumerate+_extendLine/_findWinningReply/着法排序将军>吃子子力>其他且**稳定排序保 TS 枚举顺序**）；置换表按 04 §2 用 **uint64 Zobrist 本包私有键表**（独立于引擎搜索键，语义等价 TS FEN 串键含轮走方）；超时/取消以哨兵错误沿递归传播（等价 TS SearchTimeout 异常流），ctx 取消探针每 512 节点（04 §5）；`protocol.go` Runner/Handle 同 engine 包 DR-003 口径（goroutine+ctx 注册表、入口/出口双复查取消），wire 对齐前端 solverProtocol.ts（firstMove 以 RawMessage 承接沿 M3 先例）；app.go 三绑定替换占位。测试：**6 验证 FEN 金标准**（多解/无解/超时/已将死/非法 FEN/缺王 + isWinningFirstMove 防御路径，Electron solver.spec.ts 逐项移植）+ ctx 取消 + 两次求解确定性 + 协议 roundtrip/取消链路 + 绑定端到端 3 用例。
- T6.2（M0 移植资产验证，无新代码）：工作室三 Tab（摆盘/FEN 导入/图片识图）+ 摆盘规则集（setupRules 九宫/士象斜线田字/兵卒底线/数量上限）+ 整体校验五条（studioValidate）与 Electron 版逐位一致（仅 import 路径差）；test/studio 三 spec 22 用例 + solverClient 10 用例全绿；mock 后端（solverProtocol 核心）浏览器模式可用；桌面链路由 T6.1 绑定接通。
- T6.3（`feat(m6)` d35d36c）：`internal/llm/vision.go`——VISION 协议逐字（system 提示词/JSON 模板提示词/围栏剥除+首 { 末 } 提取/坐标 a-i 行 0-9/未知棋子拦截/**双王硬校验**/turn 大小写不敏感黑方判定/PNG 魔数判 MIME/160 字符截断）；BuildVisionRequest **非流式**多模态（temperature 0.1、max_tokens 4096、无 stream 字段）；VisionReader（单次 120s ctx 截止 × 最多 2 次，HTTP≠200 截体、choices/content 链解、超时文案逐字、耗尽抛 LlmApiError；掩码 Key 经 authSlot 注入真实鉴权 DR-010）；**【DR-005】识图关闭参数按预设恒发无开关**——dashscope→enable_thinking:false、glm-4.5v→thinking:{type:disabled}、其余兜底（与对弈通道同源 ThinkingStyleFor）；app.go VisionReadBoard 真接线（cfg/imageB64/mime/authSlot 四参 → {fen}）+ wailsAdapter 透传 mime+authSlot（M6 唯一前端改动面）。测试：vision 15 用例等价集逐项移植（⑮ 按 DR-005 恒发语义改写）+ Reader httptest mock 7 场景。
- T6.4（`test(m6)` 1bc510c）：前端 solveAssist 10 用例等价集自 Electron 版移植（SOLVE_ASSIST_SYSTEM 协议面逐字快照/三行格式解析全角冒号容错/首着不在清单追加失败原因重试 2 次/onError/未配置与无着短路；成功用例附断言请求体恒发 enable_thinking:false）。链路全貌：proposeSolveFirstMove（前端 M0 移植）→ App.LlmChat 流式 → 提议 → App.SolverIsWinningFirstMove 裁判（T6.1）→ llmNote 入库；入库 wire 形状（solutions 二维 ICCS/llmNote/solveStatus）M2 存储往返已锁。
- T6.5（M0 移植资产验证，无新代码）：演示播放器状态机（idle→playing→paused→completed/速度 0.5x/1x/2x/自定义间隔 200-4000ms/循环）puzzleDemo.ts 与 Electron 版**逐字节一致**（diff 全等）+ 11 store 用例；PuzzleDetailView/RecordDetailPage 重放器逐位一致（仅 import 路径差）；语料残局详情与棋谱详情两处演示入口可用。
- 收尾（`docs(m6)`）：design_docs/04 §2/§5 文档同步补丁（Result 形状沿 Electron TS 契约定稿、Zobrist 键与测试资产口径）；两轮复审（第一轮语义一致性：04/05 §6/§7/08 §4 逐节对拍 + 工作室/播放器 diff 逐位；第二轮缺陷扫描：铁律 grep、-race 全绿、超时/取消用例 5 次重复无 flaky）；质量门全量（gofmt 0 / vet 0 / go test -race 全绿 / tsc 0 错 / npm run lint 0 错 3 条 M3~M4 遗留警告 / vitest 241 用例）。
- 追加（`feat(m6)` DR-009，验收期用户需求）：**研究助手配置运行时借用**——助手槽三字段全空时，识图/求解辅助按 黑→红 优先级在内存中借用对战配置（authSlot 随来源槽走 DR-010 掩码注入闭环），**不写入助手槽**；前端 `features/studio/assistantConfig.ts` 纯函数 + EndgameStudioPage 两调用点（识图消息区注记来源与"可能不支持识图"风险、求解辅助 toast 来源），Go 后端零改动；弃案 B 绑定层回退/C 保存时复制/D 全局凭据重构/E 弹窗导入按钮（decision_log DR-009 全矩阵留档）；测试 assistantConfig 8 用例（前端 249 全绿），05 §6/§7、08 §4 口径同步。
- 追加（GUI 优化，验收期用户请求，08 §3.6 留档）：①求解结果页关闭按钮置顶——sticky 头部右上角 ×（testid 随迁），解法多时无需滚底关闭，底部按钮保留；②求解进度改**非阻塞悬浮条**（原模态遮罩挡棋盘）——pointer-events 关闭、棋盘完整可见可操作，无关闭入口（TC-SOL-007 不可误关语义保留）+ startSolve 重入守卫；③两 LLM 对战页"立即保存"**粘性置底**（sticky 挂 cc-panel-area 滚动容器，滚动恒可见；备选思路页头/移除按钮对比留档 08 §3.6）。组件逻辑零改动，前端 249 用例回归全绿。

**性能实测**（WSL2，debug 构建）：FEN-A 双车闷杀 maxPlies=3 <5ms；FEN-B 裸王无解证明 maxPlies=5 <50ms；初始局面 1ms 限时稳定触发 timeout（探针粒度 512 节点）。真实识图耗时依赖端点（qwen-vl-max 关思维链实测 6~14s，沿 Electron 版口径）。

**手测指引（M6 待验收）**（桌面 `wails dev`；浏览器 mock 模式识图按钮按设计报"浏览器 mock 不提供识图能力"）：
- 摆盘/FEN：三 Tab 切换；选子放置（非法位置 toast 拦截）、橡皮/清空/初始局面/行棋方；整体校验——缺王或双王局面点"AI 求破解/保存棋局"被拦（"双方必须各有一个将/帅"）；FEN 导入 `3k5/9/9/9/R8/8R/9/9/9/4K4 w`（双车闷杀，可省行棋方）与乱串（"FEN 无效"）两向。
- 数量校验（FEN 导入只校格式，能载入显示——与 Electron 一致；拦截发生在求解/保存时）：导入 `R2k4R/R8/9/9/3R5/9/9/9/9/4K4 w`（4 红车，Electron 求解器测试夹具）→ 点"AI 求破解"toast"红方车最多 2 枚（当前 4 枚）"。
- 求解链路：双车闷杀 FEN → 求解设置（限时/深度）→ 求解中弹窗不可误关（TC-SOL-007）→ BottomSheet"已破解（2 条破解走法）"+ 中文记谱 + 自动入库（棋谱库"已解"筛选可见）；裸王 `3k5/9/9/9/9/9/9/9/9/4K4 w` →"无解（N 半着内已证明）"入库；**"对方已被将死"局面经工作室不可达**（五条校验第 4 条"轮走方行棋前对方已被将军，局面非法"拦截——Electron 同构，0 步解分支仅存在于求解器层，金标准已覆盖）。
- 大模型辅助：未勾选/未配置 → 静默跳过无 llmNote；配置研究助手后勾选 → llmNote"大模型首选 xx（已验证为必胜着法）…"或"未通过求解器验证，已忽略"。
- 视觉识图：研究助手配置选视觉预设（通义千问 VL/智谱 GLM-4.5V）→ 选棋盘截图 → 计时 → 结果载入 + "请人工核对后再求解" → 摆盘 Tab 点击纠错（08 §4 校正流）→ 再求解；错误 Key →"识图失败：已重试 2 次仍失败：HTTP 401: …"；未配置 →"请先配置研究助手模型（需视觉模型）"。
- **配置运行时借用（DR-009，手测补充项）**：清空助手配置（配置弹窗删空三字段保存）+ 人机 LLM 页存有黑方对战配置 → 工作室勾选大模型辅助求解出现 toast"研究助手未配置，已临时借用黑方对战配置"且功能可用；识图 Tab 出现注记"…已临时借用黑方对战配置（不写入研究助手配置）——对战配置可能不支持识图"（借对话模型识图会失败，错误信息含"请改用视觉理解模型"提示）；黑槽也清空则借用红方；三槽全空维持"请先配置"；**重新打开研究助手配置弹窗确认仍为空**（未落盘）。
- **GUI 优化手测项**：①多解局面求解 → 结果页右上角 × 恒可见（不滚动即可关闭），底部"关闭"仍在；②求解期间（选 3 分钟限时更易观察）棋盘完整可见、悬浮进度条不挡任何操作、重复点"AI 求破解"不弹设置框（守卫）；③人机 LLM/LLM vs LLM 页侧板不滚动时"立即保存"在内容末尾，滚动时恒钉在面板底部可见。
- 棋谱库/播放器联动：SolveStatus 筛选；详情页线路下拉（主变/解法 1..n）+ 步进 + llmNote 元信息；语料残局详情演示播放器（播放/暂停/停止/速度/自定义间隔/循环）。
- 回归抽查：双人/人机/LLM 对战页走子悔棋、存档恢复（M6 未触碰这些链路）。

### 2026-10-05 M5 语料 + 棋谱 交付（T5.1~T5.6，待手测）

**完成清单**（internal/parsers 6 文件实现 + 5 测试文件 + testdata 样例；internal/storage 3 文件 + 3 测试；app.go 绑定接线；前端适配层 base64 契约；Go 43 新用例 + 前端 231 用例全绿）
- T5.1（`feat(m5)`）：`internal/parsers/iccs.go`（宽松正则/行镜像 row=9−rank/黑底线 10 兼容）+ `pgnParser.go`（双着法格式：ICCS 与中文纵线记谱并存；中文记谱消解——前后中修饰/列号/全角数字/唯一合法匹配；注释/变着/NAG/结果剔除；多局切分；**ScanGameOffsets 大文件流式按局索引**：1MB 块/8MB 单行截断 P2-5 语义/UTF-8 跨块安全）+ `parsedPuzzle.go`（难度分档/残局判定关键词序）。测试：ICCS 7 + PGN 12（含 **99813 局大文件索引+分页验收基准**、9MB 超长行截断）+ 数据模型 5。
- T5.2（`feat(m5)`，沿 electron-DR-015）：`internal/parsers/xqfParser.go`——Dong Shiwei 解密逐行翻译（版本 0x0A 分界/formula 链乘密钥/f32 32 字节表/版本≥12 布局位置置换/走子主线 0x18 0x20 偏移+keyXYf/keyXYt/注解 keyRmkSize）；GB18030 经 `golang.org/x/text`（**wails 传递依赖升直接，白名单内**）；真样例锚点 `testdata/xqf/sample_xqf.xqf`（v0x0D 实文件）+ 测试侧往返构造器（旧格式/加密 v0x0C/置换 v0x12/让子 0xFF/黑先行 v0x0B）。真样例 77 着全量重放合法。
- T5.3（`feat(m5)`）：`internal/parsers/puzzleParser.go` 门面——扩展名分发/重放校验非法着截断（至少 1 着否则弃局）/id `#` 后缀去重/ShouldStreamImport 8MB 阈值。测试 7 用例。
- T5.4（`feat(m5)`）：`internal/storage/corpus.go`（ResolveCorpusDir 三级优先/ScanCorpus 一级子目录聚合+`_`忽略+CGLemon-PGN 每文件一分类/ListXqfEntries 递归+source 前两级去 gamebooks/ReadCorpusFiles 扩展名白名单/ScanPgnIndex+ReadPgnGameText）+ `internal/parsers/protocol.go`（worker 协议形状保留铁律 #7，parseBatch 逐文件进度回调，ctx 文件边界取消，Runner 与 engine 同型）+ app.go 绑定（CorpusScan/ListEntries/ReadFiles/PgnIndex/ReadPgnGame/PickDirectory + ParserParseBatch/ParserCancel + `parser:progress` 事件）+ **前端 base64 字节契约**（Wails invoke JSON 序列化 Uint8Array 退化索引对象——`api/binary.ts` 收口编解码，wailsAdapter readFiles 解码/parserClient 编码，mock 路径不变；双端测试锁定；06 文档 §0 留档）。
- T5.5（`feat(m5)`）：`internal/storage/corpusDownloader.go` + `corpusZip.go`（archive/zip 承载）——SSRF 白名单全表逐条（仅 https/localhost/.local/.internal/环回/私有/链路本地/保留段/IPv6 环回 fe80 fc00 fd/mapped IPv4 还原判段/十进制与 0x 整数 IP/八进制分段/溢出大数拒绝）；重定向手动跟随 ≤5 跳逐跳校验；15s 连接超时 + 30s 块间停滞超时经 ctx cancel 中止阻塞 Read；**Range 续传**（If-Range + ETag sidecar，206 续写/200 失败安全回退整体重下，完成清 sidecar）；zip 魔数 + 1MB~512MB 校验；corpus.tmp-<ts> staging 原子 rename 替换失败整体清理；临时目录清理带退避重试（09 §2.4 教训）；zip-slip 全表（../绝对路径/盘符/反斜杠归一/嵌套 ../UNC/符号链接/保留名含扩展形式/尾随点空格/同名冲突不中断/坏 CRC 条目跳过）；app.go CorpusDownload（`corpus:progress` 事件）。
- T5.6（`feat(m5)`）：app.go DialogSaveFile/DialogReadFile（wails 原生对话框 + 内容写盘/读取绑定层收口；WSLg 原生对话框可用性列手测项——Electron 版因 portal 冻结改自绘，Go 版待实测）；前端 `test/storage/pgnWriter.spec.ts` + 快照自 Electron 基准复制**一次通过**（导出协议面逐位一致）；棋谱库重放（puzzleDemo store M0 已移植）/进入对战（RecordLauncherDialog）页面级用例既有全绿。

**两轮复审（11 §6）**
- 第一轮·语义一致性（独立代理逐行对照 TS 事实源，8 组）：P1×1（zip 单条目解压失败语义相反：TS 跳过继续/Go 整体中止）+ P2×2（目录条目不落盘计数虚高；ParserCancel 无"请求到达前"粘性记忆）+ P3×14。处置：P1+P2 目录条目当场修复并加回归用例（坏 CRC 跳过 + 目录落盘两条）；P2 粘性记忆经架构评估为不可达窗口（绑定层调用序 + 前端 pending 表先行拒绝，engine.Runner K17 同型）留档 K23；P3 留档 K24~K30（含整数 host 溢出拒绝与 PGN 越界文案两条当场修复）。
- 第二轮·缺陷扫描（自查）：竞态（Runner mutex/下载 UI 单飞/进度回调 goroutine 安全）、取消链（ParserCancel ctx 全覆盖/下载无取消与 Electron 同）、兜底三态（parseBatch null 位/语料缺失引导）、错误消息无 Key 参与、`-race` 全绿。

**回归护栏（11 §6.3）**：铁律 grep 自检——#1 internal/parsers 零运行时依赖（go list 验证）；net/http 仅在 internal/llm/transport 与 internal/storage/corpusDownloader（**06 §5 补口径说明**：网络/IO 边缘包同 transport 先例，领域包 rules/engine/solver/parsers 仍纯）；#3 解析走法与规则内核合法走法求交；#4 前端零 fetch 出口；#5 事件载荷全含 requestId；#8 M5 无 Key 参与。

**验证门**：`gofmt -l` 空 + `go vet ./...` 0 + `go test ./... -race` 全绿（parsers 32 函数/storage 21 函数/绑定 7 函数）；前端 tsc 0/eslint 0/vitest **25 文件 231 用例**全绿。大文件索引/分页行为与 Electron 版一致（99813 局基准用例）；PGN 导出快照与 Electron 基准逐位一致。

**新增依赖**：`golang.org/x/text v0.39.0`（GB18030 解码；wails 传递依赖升直接，白名单内，T5.2 commit 说明）。

### 2026-10-05 M4 手测缺陷修复（F4 LlmChat 事件订阅生命周期，待复测）

- **现象**（用户手测，千问 qwen3.8-max）：配置卡"测试连接"成功并保存；大模型对战页开局后红方一直"思考中…"（93 秒+），无重试进度、空闲超时 60 秒后也未降级内置 AI。
- **根因（P0）**：Go 绑定 `LlmChat` 被实现为"受理即返回"（invoke 立即 resolve），而前端 `llmTransport.chat()` 在 promise 结束的 `.finally()` 里反注册 llm:chunk/done/error 三事件订阅——**订阅在请求发出前即被注销**，Go 侧回发的全部事件（含 Go 侧早已正常完成的结果）无人接收，TS `chatOnce` 永挂。Electron 版同链路成立的原因：`ipcMain.handle(CC.llm.chat) => proxy.chat(req, sender)` 阻塞至结算才 resolve，订阅存活期覆盖整个流。00 §3.2 的"恒 resolve=方法立即返回受理"表述自 M0 占位起即为错误。
- **修复**：`App.LlmChat` 阻塞至结算后恒 resolve（取消经 LlmCancel → 结算返回，对齐 Electron ipc handler 语义）；00 §3.2 表述修正（文档先行）；绑定测试改严格断言（LlmChat 返回时事件必须已全部送达，防语义回退）；取消用例改 goroutine 调用。质量门全绿（go test -race / tsc / vitest 214）。
- 已知行为澄清（非缺陷）：空闲超时/总上限的降级发生在**模型真失败**场景——每次重试独立计时（3 次重试 × 最长 240s 总上限），"等待降级"期间状态栏经 onAttempt 显示"第 N/M 次尝试"；本次现象中 Go 侧请求实际已成功，只是事件被丢弃。

### 2026-10-05 M4 LLM 全链路 交付（T4.1~T4.5，待手测）

**完成清单**（internal/llm 共 9 文件实现 + 8 测试文件，63 Go 用例 + 前端 214 用例全绿）
- T4.1（commit `feat(m4)` e08ceb7，Decision: DR-005）：`internal/llm/config.go` 请求构造——**思维链关闭参数恒发、无任何开关路径**，按端点预设映射（智谱 GLM→`thinking:{type:"disabled"}`；DashScope/Qwen 与其余端点→`enable_thinking:false` 兜底；`ThinkingStyleFor` 按预设名解析，未知/自定义走兜底）；`LlmEndpointConfig` 四字段 `{baseUrl,apiKey,model,preset}`（无 disableThinking——07 §4）；requestUrl 去尾斜杠自动补 /chat/completions、掩码 Key 三态（空 Key 不带头/`****` 前缀走 AuthSlot/完整 Key 内联 Bearer）、测试连接最小请求、DR-012 空侧镜像。`transport.go` 替代 Electron main 进程 LlmProxy：net/http 流式 POST + 逐块读取自切行（半行跨 TCP 分包重组、行长无上限）、**空闲超时=块间最大间隔**（每收到一块重置，默认 60s clamp 5~600）、**总上限=空闲×4** 独立计时器、取消经 context（Cancel(requestID) 幂等、此后零事件）、HTTP≠200 截 160 字符（rune 计）+ AnnotateModelHint、五条错误文案逐字（空闲超时/总耗时/连接失败/连接中断/流式响应错误）+ Proxy（受理即返回/事件回发/authSlot 经凭据槽位注入真实鉴权）。测试：请求体快照（含九预设关闭参数断言）、SSE 重组逐行（双字段名 reasoning_content??reasoning）、mock SSE httptest 服务器（写已断开连接守卫）、传输层 20 场景（跨分包含中文多字节/计时器逐块重置/半行慢流不误判/EOF 残留丢弃/cancel 无迟到/HTTP 4xx5xx/并发隔离/authSlot 注入/测试连接）。
- T4.2（commit `feat(m4)` 6b28256，铁律 #2）：prompt.go（systemV1/userV1/retryFeedback/systemV2/userV2/retryFeedbackV2/vetoFeedback/looksLikeRepetition/historyTextV2）/parser.go（MOVE_PATTERN/LABELED_PATTERN/normalizeReply 围栏剥离→零宽删除→小写→全角 U+FF01–FF5E 逐码位 −0xFEE0；多坐标对「着法:」标记后优先）/annotation.go（annotateMove/scoreBucket 30/100/250/600/annotatedWithBucket/asciiBoard）逐字翻译；**测试：30 条解析用例 + 提示词 v1/v2 全文逐字快照 + 初始局面 ASCII 全字快照**（快照基准=Electron 版既有快照，第一轮复审经脚本机械比对逐字一致）。
- T4.3（commit `feat(m4)` 31ade69）：LlmPlayer 实现 engine.MoveSource——LlmChatClient（ChatOnce 取消注册表/掩码 authSlot/CancelCurrent 本地结算+通知传输）；五层过滤管线（SSE 重组→归一化→提取→decodeCell→**白名单精确字符串匹配**，铁律 #3）+ user 末尾追加反馈重试（无状态两消息协议整段重发）+ 降级链 builtinAi（note=…已由内置 AI 兜底走子+fromFallback）/resign（…按判负处理）；取消 ErrCanceled 原样上抛。测试：失败模式表 25 场景全绿。
- T4.4（commit `feat(m4)` af727fd）：HybridLlmPlayer 参谋制——off（惰性单例委托纯 Prompt v2，delegateMu 互斥）/candidate（Top-K 短名单+分档清单）/gate（全量清单+否决权）；旋钮逐值 K=min(8,max(3,3+⌊blend/20⌋))、否决阈值=80+⌊3.2×clamp(blend,0,100)⌋、复评深度=depth−1、timeLimitMs 5000；**否决-再问-代走**：vetoFeedback 带分桶厘兵重问一次→二次通过/违抗→report.best 代走（fromFallback，**不走 resign 降级分支**——参谋职责）。测试：三模式 7 场景+否决两分支+旋钮边界+防御路径。
- T4.5（commit `feat(m4)` 36eb471）：app.go LlmChat/LlmCancel/LlmTestConnection 替换占位（llm.Proxy 受理即返回 + llm:chunk/done/error 事件回发含 requestId + authSlot 注入 + 空闲超时接 llm_settings_timeoutSeconds）；前端 DR-005 预设映射（preset 字段引入、disableThinking 全删、LlmConfigCard 开关 UI 删除、vision 同映射恒发——K1/K12 注销）；绑定端到端 5 用例（mock SSE 全链路/取消无迟到/authSlot 注入/测试连接三态）+ dr005.spec 6 用例。

**两轮复审（11 §6）**
- 第一轮·语义一致性：PASS（P0/P1=0）——提示词中文字面量经脚本机械比对逐字一致；9 个检查面（prompt/parser/annotation/config/sse/transport/llmplayer/hybridplayer/前端映射）全部通过。缺陷：P2×2（HybridLlmPlayer 引擎参谋 findBestMoveEx/evaluateMove 非取消错误被吞为 noLegalMove/兜底链，TS 语义是异常传播→页面 onSideFailed）当场修复并加回归用例；P3×3（off 委托多发 OnAttempt 已修、空闲计时器按行重置已改按块、RE2 \s 缺 Unicode 空白已补显式字符类「着法〈全角空格〉:」回归）。
- 第二轮·缺陷扫描：PASS（P0/P1=0）——P2×1（Proxy.ActiveCount 无锁读 active map）当场修复；P3×1（测试 sleep 等首块 CI 慢载假失败）改 deadline 轮询；P3×2（OnChunk TOCTOU 纳秒窗口/safeReadBody 无上限）前者留档 K21（TS 同型+渲染层收口），后者加 2MB LimitReader 防御。修复过程中连带发现并修复读取循环两处实现缺陷（ReadSlice 无换行不返回致重置失效；行切片与缓冲同底层数组被挪移覆盖）——半行慢流与 EOF 残留两用例锁定。
- P3 留档：K19（构造器 `??` vs 零值兜底语义差，设置层 clamp 内不可达）/K20（excerpt rune vs UTF-16 计数，BMP 内一致）/K21/K22（chunk.error JSON 键序差）。

**回归护栏（11 §6.3）**：铁律 grep 自检全过——#1 net/http 仅在 internal/llm/transport.go（铁律 #4 明文收口点，engine/rules/storage 零依赖）、#3 白名单精确匹配+页面 playMove 双保险、#4 对外 HTTP 零新增出口、#5 事件载荷全含 requestId、#8 Key 仅注入路径无消息拼接、#10 `enable_thinking` 全库仅 false 恒发、`disableThinking` 仅存于注释。

**验证门**：`gofmt -l` 空 + `go vet ./...` 0 + `go test ./... -race` 全绿（internal/llm 63 用例含 mock SSE 计时器/取消全场景）+ 前端 tsc 0/eslint 0/vitest 214 全绿（含 dr005.spec 6 用例）。

**下一里程碑**：M5（语料 + 棋谱）——本里程碑手测（含真实端点 GLM/DeepSeek 各一整局）通过后，新会话逐字粘贴 11 §4.6 启动提示词。

### 2026-10-05 M3 引擎 + L0/L1/L2 重复治理 交付（T3.1~T3.5，已验收）

**完成清单**（每子任务一 commit；①②对拍门先行、全绿后才进 ③④——顺序铁律 ✓）
- 文档先行（随 T3.1 commit）：03 §4 勘误——键表形状 `[2][15][90]` 为首版起草残留的 TS 双表形状，按同句「uint64 单键」口径勘误为 `[15][90]uint64` + 补 PRNG 移位（Marsaglia 13/7/17）；§6.2 补「Difficulty 0=未设（Go 零值）即缺省 3」；§7 补 wire 缺省约定（前端 `?? 0` → Go 按 0=未设取缺省档）与协议层取消收口；§8 补 ChessAiPlayer fenHistory 引擎侧单源推导口径。
- T3.1（`feat(m3)` edb63a6，Decision: DR-006）：`internal/engine/engineboard.go` + `zobrist.go`——Electron 版 engineBoard.ts 逐行翻译：`[90]int8` 一维盘（−7..7 带符号编码）、packed 位段走法（`[rank|captured|to|from]`，MVV-LVA 排序键内嵌高位，等级表/PSQ 修正表同式构建）、apply/undo 栈式回退 + kings 缓存、IsCheck 四判定（照面/车炮直线/马位反查/兵）、GenerateMoves(For) 与 rules.Board 语义 1:1（对拍测试锁定盘面/走法集合/captured 位段/apply-undo 往返）、评估固定分 9 例（过河卒 172/沉底车 910/居中炮 466 等）；Zobrist L0 内置 EngineBoard：uint64 单键、固定种子 xorshift64 惰性 sync.Once、FromFen 全量重建 + apply/undo 对称异或严格互逆，五项测试（初始键快照 4 FEN 十六进制定值/重建一致/固定种子 LCG 随机对局 120 步增量===重建+undo 复原/轮走方参与键/吃子改键）。
- T3.2（`feat(m3)` f61592f，Decision: DR-006）：`internal/engine/search.go` + `chessai.go` + 金标准对拍——search.ts 逐行翻译：negamax+alpha-beta（fail-hard）/qsearch 只延伸吃子（ply<8）/被将军强制全应将（ply<16）/迭代加深超时返回上层完整结果/根节点全窗口（runScored 强制、randomness>0 启用）/每 64 节点 deadline+取消探针；中断按 03 §5 改 **error sentinel** 上抛 iterate 捕获（等价 Dart `_TimeUp`/TS SearchAbort，盘面停留中途状态随实例废弃，TS catch+finally 的不回退语义逐行对应）；**金标准 gate：engine.json 自 Electron tools/golden 逐字节复制（sha256 一致），不传 HistoryFens 时 findBestMoveEx best/bestCp/topK 分数序列逐位一致——快速集 32 子测试全绿后，RUN_SLOW=1 慢速集（initial/midgame d5/d6）8 子测试亦全绿**；chessAi 三接口（FindBestMove/Ex/EvaluateMove）同 commit 落地并过 evaluateMove 金标准条目。
- T3.3（`test(m3)` cc8531b）：L1 搜索内检测测试（检测代码随 TS 源逐行翻译已在 T3.2 落地，缺省 nil 整段关闭零开销）——缺省两次运行 best/nodeCount 逐位一致；全局历史 count=2/3 惩罚精确值断言（落子后静态评估 −100/−200，ply≤3 命中即剪枝无搜索噪声）；空表路径检测确定性冒烟（摇摆局面）；性能门开/关 nodeCount 差 ≤5%（实测中炮局 4061374 vs 4059333 = **+0.05%**；显式 60s 大超时防 CI 负载假超时——Electron 版教训）。
- T3.4（`test(m3)` 7524927）：L2 根节点回避测试——`PickAvoidanceMove` 纯函数 5 场景（注入随机源：阈值内随机取一/优劣势系数 0.5·2.0/长将强制变着底线 −500/底线内无候选保留原着/闲着重复交 L3；`AVOID_THRESHOLD_BASE={1:200,2:200,3:100,4:50,5:30}`、`FORCED_CHANGE_FLOOR=500` 逐值搬移）；FindBestMove 集成：不传 HistoryFens 时 best===FindBestMoveEx best（向后兼容铁律）、命中历史返回落子后非重复着法、一步杀优先于回避；三接口行为等价用例（Dart ai_engine_test/ex_test 用例集 15 例）。
- T3.5（`feat(m3)` 0e3d4f6，Decision: DR-003）：`internal/engine/protocol.go` + `movesource.go` + `app.go` 绑定接线——协议层保留 Worker 消息形状 `{id,type,payload}/{id,ok,result|error}`（铁律 #7），wire 载荷对齐前端 engineProtocol.ts（Move `{from,to,captured?}` / topK `[move,cp]` 二元组 MarshalJSON）；`Runner` 每请求独立 goroutine + ctx 取消注册表（DR-003；无 FIFO 队列——对局页至多一个在途搜索），取消后以 `canceled` 结算（文案对齐前端 CANCELED_ERROR），迟到部分结果不作有效应答；`ChessAiPlayer`（MoveSource 实现）经 `HistoryFensFromBoard` 从 board+history 逐手 UndoMove 回放推导 fenHistory——与页面重放口径逐项等价（测试断言 5 手对局逐项一致 + 两次应手一致）；`app.go` 四 Engine* 绑定替换占位（difficulty 0=未设→缺省档），绑定端到端 13 用例（三类型 roundtrip/取消链路/同 id 重提/无效 FEN/幂等 cancel/被将死 null/wire 形状）。

**顺序铁律执行记录**：T3.1/T3.2 翻译完成 → 先跑金标准对拍（快速集+慢速集全绿，commit f61592f）→ 才提交 T3.3/T3.4（L1/L2 测试）与 T3.5（接线）。金标准对拍期望零改动（未改 testdata/golden/engine.json 一个字节）。

**质量门（全绿）**：`gofmt -l` 空输出；`go vet ./...` 0 问题；`go test ./... -race` 全绿（internal/engine 59 测试函数/57 断言通过项，默认 4 项慢速对拍子测试 SKIP 需 RUN_SLOW=1）；`npm run test:fe` 22 文件 207 用例通过；tsc --noEmit 0 error；eslint 0 error（3 warning 均为 M0 随迁文件历史项）。
**新增依赖**：无（仅 stdlib：math/rand/v2、slices、sync 等）。

**两轮复审（11 §6.1）**：第一轮语义一致——search.ts/chessAi.ts/engineBoard.ts/moveSource.ts 四文件逐行对照（含 TS catch+finally 的盘面不回退语义、根节点 try/catch 中断语义、`Math.max(remaining,100)` 剩余时限、`AVOID_THRESHOLD_BASE[difficulty] ?? 100` 回退）；翻译缺陷 2 处**当场由测试捕获并修复**（IsCheck 敌王缓存取反——TS `kings[isRed?1:0]` 写反致根节点全过滤；negamax/quiescence/evasions/根四处递归调用漏负号——TS `-this.negamax(...)`，均属 T3.1/T3.2 commit 内修复）。第二轮缺陷扫描——Handle 补显式 ctx 入口检查（取消即回 canceled 不空算）；铁律 #1 import 面机器检查 PASS（internal/engine 仅 stdlib+internal/rules）；`-race` 报告 0。P3×2 记 docs/KNOWN_ISSUES.md（K17 Runner 同 id 并发在途后到者为准/契约内不可达、K18 随机路径跨语言伪随机源不同/randomness=0 对拍面不受影响）。

**性能实测（Go 引擎，WSL2 同机）**：
| 项目 | Go 实测 | 门限/参照 |
|---|---|---|
| 金标准 initial d6 全窗口 | **5.05s** | Dart 80.7s（金标准 note）、TS 版 3.6~25s 量级 |
| midgame d6 全窗口 | 1.90s | Dart 25.5s |
| **难度 5（depth 6）初始局面应答** | **1.18s** | 门 ≤7.5s（剪枝模式，09 §2.2） |
| L1 检测 nodeCount 开销 | +0.05% | 门 ≤5% |

**下一里程碑**：M4（LLM 全链路，恒关思维链）——本里程碑手测验收通过后，新会话逐字粘贴 11 §4.4 启动提示词。

### 2026-10-05 M2 对战页 + 存储 交付（T2.1~T2.4，已验收）

**完成清单**（每子任务一 commit）
- T2.1（`feat(m2)`）：`internal/storage/db.go`——sqlite DAO 逐字段对照 Electron 版 `src/main/services/db.ts`：07 §1.1 两表 DDL **逐字段一致**（脚本比对 IDENTICAL）、迁移 V1（PRAGMA table_info → ALTER ADD COLUMN DEFAULT 'legacy'）、裸 upsert 标 'legacy'、按模式分桶 upsert、`ORDER BY updated_at DESC`/`created_at DESC, id DESC`、parseResult 非法串回 'draw'/parseSolveStatus 回 'none'、旧库 TEXT 时间戳兼容解析、game_records 解码链（decode→过滤脏行→re-encode，`p: r.p ?? ''` 保真）含 x:"" 存量修复；**gameDao.spec 全量 18 用例**移植为 Go 表驱动（saved_games 5 + 分模式 4 + 迁移 1 + 棋谱库 6 + 回归 2）。附 07 §1 文件名勘误：`chinese_chess_ultra_go.sqlite`（原文复制自 Electron 版，两应用共存不得共写同一库）。
- T2.2（`feat(m2)`，Decision: DR-005）：`internal/storage/settings.go`——`<UserConfig>/chinese_chess_ultra_go/settings.json` 替代 electron-store（原子写 tmp+rename）；`global_auto_save` 缺省 true；`llm_settings_*` **load 时越界 clamp**（05 §8 范围表；枚举 index 越界回落默认、红黑强度缺省回落 strengthBlend 原始值再 clamp，语义对照 llmSettingsFromRaw）+ `credentials.go`——keyring 三槽位（service=`chinese-chess-ultra-go`，user=槽位名），槽位 JSON `{baseUrl, apiKey, model, preset}` **无 disableThinking**（DR-005）；keyring 不可用 → `credentials.enc` 明文回退 **0600** 原子写、SecureSetResult 如实回报 `plainFallback`（沿 electron-DR-011）；掩码 `****`+末 4 位、**掩码回写合并不覆盖真实 Key**（沿 electron-DR-013）；读失败/损坏一律按未配置。settings 7 用例 + credentials 12 用例 + MaskApiKey 表驱动。
- T2.3（`docs(m2)`）：GameVm 为 M0 整体移植资产——fenHistory **四收口点**齐备（初始/newGame/newGameFromFen 重置 `gameVm.ts:71/146/274/283`、executeMove push `:203`、undoOnceInternal pop `:264`、restore 重放逐手采集含跳脏不 push `:156`），长度恒=手数+1；`agreeDraw`/`resign(draw)` 首版即有；前端 21 文件 **202 用例**全绿（gameVmFenHistory 7 + autoSaveRestore 12 + repetitionJudge 接线 3 等）。【DR-006 检查点：四收口 ✓】
- T2.4（`feat(m2)` + `fix(m2)` 复审修复）：`app.go` 绑定层 M2 通道真实实现——db 五方法（懒打开失败不缓存可重试、失败 invoke 拒绝→前端"本地存储不可用"降级新局兜底）、StoreGet/StoreSet、SecureGet/SecureSet/SecureDelete（掩码不回渲染层，铁律 #8）、ClipboardWrite（wails runtime）；`main.go` OnBeforeClose 生命周期收口（close→有界等待→before-quit，驱动 GameAutoSave，07 §2 映射表）；`wailsAdapter.ts` **blur 相位补偿**（Wails v2 无窗口失焦事件，08 §2 文档先行同 commit）；`app_test.go` 绑定面 10 用例（适配器 JSON 载荷形状往返/存储不可用降级/懒打开重试/掩码不泄 Key/掩码合并/明文回退如实回报）。`fix(m2)`：复审 P1 修复——Credentials 加互斥（并发绑定进入时回退文件竞态）+ 并发用例 -race 覆盖；OpenDao busy_timeout(5000) 对齐 better-sqlite3 默认。

**质量门（全绿）**：`gofmt -l` 空输出；`go vet ./...` 0 问题；`go test ./... -race` 通过（internal/rules 84 + internal/storage 49 + 绑定面 10）；`npm run test:fe` 21 文件 202 用例通过；tsc --noEmit / eslint 0 error（wailsAdapter 改动后回归）。
**新增依赖**：modernc.org/sqlite v1.60.1、zalando/go-keyring v0.2.8（均在白名单；godbus 传递升级 v5.2.2）。
**两轮复审（11 §6.1）**：第一轮语义一致——DDL 逐字段脚本比对 IDENTICAL；迁移 V1/裸 upsert legacy/排序键/枚举回退语义逐条对照 db.ts；凭据读序（keyring 命中解析失败→未配置即停；缺失/不可用→回退文件）映射 readEncrypted/readPlain；useRepetitionJudge 与 Electron 原版 **diff 全等**。第二轮缺陷扫描——竞态：Credentials 无互斥（**P1 已修复**）；并发写 busy_timeout（**P2 已修复**）；铁律 grep 自检 #1/#4/#6/#8 全 PASS；错误消息无 Key 泄漏。P3×5 记 docs/KNOWN_ISSUES.md（K12~K16），K3 注销（真实存储消解占位噪音）。
**性能实测**：无算法面；SQLite 内存库 18 用例 <0.05s（modernc 纯 Go 驱动，07 §1 量级足够）；引擎耗时自 M3 起记录。

**下一里程碑**：M3（引擎 + L0/L1/L2）——本里程碑手测验收通过后，新会话逐字粘贴 11 §4.3 启动提示词。

### 2026-10-05 M2 手测缺陷修复（F2 绑定命名空间，待复测）

**手测反馈**：B 分享棋局剪贴板无记谱、自动/手动存档未落库（无 `~/Documents/chinese_chess_ultra_go.sqlite`）；C 设置切换后 `settings.json` 不存在（目录存在）。

**根因（P0，F2）**：Wails v2 以绑定结构体所在 **Go 包名**挂载方法表（`window.wailsbindings={"main":{"App":{...}}}` → `window.go.main.App`）；M0 起探测与适配器/三客户端误用 `window.go.app.App`——恒 undefined，桌面窗口内 client.ts **静默回落 mock**：存档/设置/剪贴板全部走内存 mock（配置目录存在是 Go startup 建的，与前端走 mock 不矛盾）。F1（M0）的"外部浏览器回落 mock"观察与本次缺陷同根因，其修复（改探测 `.app.App`）实际把崩溃缺陷转为静默 mock，掩盖至今。

**修复**（`fix(m2)`）：
- `client.ts` 探测、`wailsAdapter.ts`、`engineClient.ts`、`solverClient.ts`、`parserClient.ts` 五处统一改为 `window.go.main.App`（对齐 wailsjs 生成物与 runtime.js 注入事实；08 §2 勘误同步）；
- `app.go documentsDir` 在 `~/Documents` 缺失时创建（07 §1 存档路径落地，不再静默退回 $HOME）；
- 回归用例：新增 `frontend/test/api/wailsAdapter.spec.ts` **5 用例**（main.App 命名空间下 db/store/secure/clipboard 载荷透传、旧 `app.App` 形态拒绝、无 window.go 抛错、生命周期双源接线）；Go 侧 `TestDocumentsDirCreatesMissingDocuments`。

**端到端取证**（wails dev devserver + headless Chrome 真实绑定调用，与桌面窗口同管线）：命名空间 `["main"]` 命中；DbSaveGame → sqlite 落盘 `~/Documents/chinese_chess_ultra_go.sqlite` → DbLoadLatest 读回完整行；StoreSet → `~/.config/chinese_chess_ultra_go/settings.json` 写入；ClipboardWrite 成功（err=null）。测试数据已清理（存档行删除、开关复位）。

**质量门**：`go test ./... -race` 全绿；前端 22 文件 **207 用例**（+5）全绿；tsc 0 error；eslint 0 error（3 warning 均为 M0 随迁文件历史项）。

**复测指引**：重开 `wails dev -tags webkit2_41` → 双人页走数着后手动保存（toast"棋局已保存"）→ 确认 `~/Documents/chinese_chess_ultra_go.sqlite` 已生成 → 关闭重进恢复局面；设置弹窗切换自动保存 → `~/.config/chinese_chess_ultra_go/settings.json` 出现；分享棋局 → 剪贴板含中文记谱。其余 M2 手测清单（PROGRESS §手测指引 A 组）请一并复测。

### 2026-10-05 M2 手测缺陷修复第二轮（F3 存档恢复语义，DR-008，待复测）

**手测反馈**：保存成功，但恢复后盘面被改写——黑马位置出现黑车、原位车消失（对照图：保存时黑车在 display-h9/黑马在 g7；恢复后 g7 变车、h9 空）。

**根因（P1，F3）**：保存端 `serialize()` 存**当前（终局）局面 FEN**（Flutter `board_vm.dart:308-311` 原版即如此，Electron 逐字继承，Go M0 逐字继承），而恢复端 `restore(fen, moves)` 以该 FEN 为基准重放着法栈（`board_vm.dart:87-132`，其注释将"源格有子"视为常态）——两端语义错位：正常情形着法全部被"源格无子"跳过（恢复后历史必清空）；本局第 4 步"车 i9→h9"后 h9 恰有车，重放"马 h9→g7"时源格有车 → 车被误走到 g7 覆盖马。即原版潜伏缺陷，Go 版逐字继承后首次被手测暴露。

**修复（Decision: DR-008）**：serialize 存**本局起始 FEN** + 完整着法栈（gameVm.ts 新增 `_startFen`：构造/newGame/newGameFromFen/restore 设定；serialize 与 restore 语义对齐）——恢复即重放重建整局：局面正确、走法历史与 fenHistory 完整重建（07 §3 restore 逐手采集与 DR-006 四收口真正成立）、跳脏语义保留、恢复后悔棋/续存自洽。文档先行：07 §2/§3 修订 + decision_log DR-008（决策矩阵：A 保持 1:1 带病 / B 起始 FEN 语义 / C 忽略着法直接终局开局——选 B）。

**用例改写（3 处，均注明 DR-008 依据）**：gameVmFenHistory.spec（restore 重建断言）、autoSaveRestore.spec（存档 fen=起始 FEN + 恢复后 2 手历史 + 悔棋跨恢复点）、gameStore.spec（serialize 返回起始 FEN）。**端到端复现取证**：wails dev + 真实页面环境 + 真实 Go 存储，逐手复现手测 4 步（炮二平五/黑马8进7/马二进三/车9平8）→ 保存 → 恢复：**恢复后局面与保存时逐位一致、4 手历史完整、fenHistory 5 项**。

**质量门**：Go `-race` 全绿；前端 22 文件 **207 用例**全绿；tsc/eslint 0 error。

**复测指引**：重开 `wails dev -tags webkit2_41` → 双人页走 4+ 步（含车/马移动）→ 关闭或手动保存 → 重进：局面与保存时完全一致、步数不变、悔棋可一路退回开局。⚠ 修复前写入的旧存档（fen 为终局语义）不兼容，首次进入若见异常盘面，点"新游戏"后再保存一次即覆盖为新语义。

## 手测指引（M3 待验收）

### A. 桌面人机完整对局（验证门主链路）

`wails dev -tags webkit2_41` 启动（WSLg 弹窗）→ 主页进入「人机对战」：

1. **AI 应手（Go 引擎）**：执红走「炮二平五」→ AI 思考中（spinner）→ AI 应手合法落子、动画正常。高级档（难度 3）应答应在 ~2s 内。
2. **难度切换**：下拉切「初级」→ AI 应手明显变快且带随机性（同局面重复开局不走法完全一致属正常——低难度随机窗口）；切「大师」→ 应手变慢（≤5s，超时有 deadline 兜底）。
3. **执黑（AI 先行）**：执方下拉切「执黑」→ 新局由 AI 执红先行；玩家走黑棋，全程无卡死。
4. **被将军时 AI 解将**：构造对 AI 的将军（如车照将）→ AI 应手应解除将军而非无视。
5. **状态栏四态**：对局结束/AI 思考中/被将军/等待玩家，随局面正确切换。
6. **AI 失败防软死锁**（防御面，正常不出现）：若 AI 应手被拒仅 toast 提示，输入不锁死。

### B. 重复治理（DR-006 主链路，人机页）

1. **L2 回避**：构造重复——与 AI 来回互返着法（如「炮二平五/炮5平2」「炮2平5/炮8平5」再「炮8平5/炮2平5」类，使局面第二次出现）→ AI 倾向换着（阈值内非重复着法随机取一），不无限重复拉环；若 AI 最佳为长将形态且历史命中 → 应见其强制变着（底线 −500 内）。
2. **L3 裁决接线**：若真走出三次重复（k=3，AI 侧自动接受和棋）→ 人机页弹「三次重复局面」确认框（玩家执方弹框；AI 侧不弹）→ 接受=和棋横幅 / 变着继续=关框；长将环 → toast 警告后第 3 次判负；k≥4 强制判和 toast。
3. **悔棋后重置**：出现重复警告后悔棋 → 警告状态重置。

### C. 自动保存/恢复回归（M3 引擎不触存储，抽测即可）

人机对局数着 → 关窗重进 → 局面/手数恢复正确；将死后重进 → 死局清理开新局。

### D. 浏览器 mock 模式回归（开发兜底）

`npm --prefix frontend run dev:web` → 人机页走子 → AI 应手正常（mock 模式走 TS 引擎内核，与桌面 Go 引擎行为同口径；难度 3 下应手风格可能因随机源不同略有差异——K18 留档，非缺陷）。

### E. 命令验证

```bash
go test ./... -race -count=1                  # 全绿（含 internal/engine 59 测试函数）
RUN_SLOW=1 go test ./internal/engine/ -v      # +慢速金标准对拍 8 子测试 + 性能门（≈10s）
npm --prefix frontend run test:fe             # 207 用例
gofmt -l . | grep -v node_modules             # 空输出
go vet ./...                                  # 0 问题
```

预期已知行为（非 bug）：难度 1/2 的具体应手跨语言不逐位一致（随机窗口，K18）；LLM 对战页仍提示"尚未接入"（M4 占位）；棋谱库/语料/求解入口同前占位（M5/M6）。

## 手测指引（M2 待验收）

### A. 桌面双人完整对局（验证门主链路）

`wails dev -tags webkit2_41` 启动（WSLg 弹窗）→ 主页进入「双人对弈」：

1. **防错 #1/#2（动画窗口）**：快速连点棋盘——动画 220ms 期间点击被忽略；动画结束才真正落子（不出现鬼子/穿子）。
2. 走数着（如炮二平五、马八进七）：回合切换/步数/用时累计/将军标记正常。
3. **悔棋**：悔一着回到前一局面，步数回退；悔棋后重复裁决状态重置（再走不会立即误报重复）。
4. **防错 #11/#12（重复裁决）**：构造重复——连续「炮二平五/炮5平2」「炮2平5/炮8平5」类互返着法（或任意三次回到同局面）：第 3 次同局面且无将军环 → 弹「三次重复局面」确认框（接受和棋=终局横幅"和棋"/变着继续=关框）；若环内含连续将军 → 先见非阻塞 toast「红/黑方连续将军重复，再次将判负」，第 3 次 → 判负横幅（复用终局结算）。拒绝一次后第 4 次重复 → toast「再次重复局面，强制判和」。
5. **终局后禁手（防错 #6）**：将死/认输/和棋后棋盘点击无响应。
6. **新游戏确认框**：对局中点「新游戏」弹确认，确认后回初始局面。

### B. 自动保存与恢复（验证门）

1. 对局数着后**关闭窗口**（X）：重新 `wails dev` 进双人页 → 应恢复到关闭前局面（重放）。
2. 对局数着后**最小化/切走焦点再切回**：不报错；关闭应用重进应见这些着法（blur 相位保存）。
3. **死局清理**：把一局走到将死 → 关闭重进：存档已判死局 → 自动删除并开新局（07 §2）。
4. **手动「保存棋局」**：toast「棋局已保存」；重进恢复到保存点。
5. **分享棋局**：剪贴板含中文记谱文本（`wl-v` 等 wails 剪贴板）。
6. **存档库位置**：`~/Documents/chinese_chess_ultra_go.sqlite` 生成（07 §1）。

### C. 设置存储（非敏感，冒烟）

1. 主页「设置」弹窗切换自动保存开关 → 关闭重开应用，开关状态保持（`~/.config/chinese_chess_ultra_go/settings.json`）。

### D. 浏览器 mock 模式回归（开发兜底）

`npm --prefix frontend run dev:web` → 双人页走子/悔棋/新游戏正常（mock 存储）；控制台无意外报错。

### E. 命令验证

```bash
go test ./... -race -count=1        # 全绿（含 storage 49 用例 + 绑定面 10 用例）
npm --prefix frontend run test:fe   # 202 用例
gofmt -l . | grep -v node_modules   # 空输出
go vet ./...                        # 0 问题
```

预期已知行为（非 bug）：LLM 对战页 AI 应手提示"尚未接入"（M4 占位）；棋谱库列表为空（M5）；语料/求解入口同前占位（M5/M6）；凭据槽位在 WSL 无 Secret Service 时保存会提示"已用未加密本地文件存储"（DR-011 如实回报，文件 0600）。人机对战页 AI 应手自 M3 起由 Go 引擎供手（占位提示已消解，手测指引见上方「手测指引（M3 待验收）」）。

### 2026-10-05 M1 规则内核 + L3 交付（T1.1~T1.5，待手测）

**完成清单**（每子任务一 commit，红→绿）
- 测试基建（`test(m1)` 0cf7ec0）：金标准 `fen.json`(19)/`moves.json`(16 案例)/`notation.json`(21 案例) 自 Electron 版 tools/golden **逐字节复制**至 `testdata/golden/`（sha256 一致）+ `tools/golden-fetch.md` 来源说明（09 §3；engine.json 随 M3 引入）。
- T1.1（`feat(m1)` c44979f）：position/piece/move/fen 逐行翻译 Electron 版 piece.ts/position.ts/move.ts/fen.ts + Board 构造/序列化基座；fen.json 往返对拍全绿 + fen.spec 6 用例。
- T1.2（`feat(m1)` 9e0bd89）：board.ts 剩余部分逐行翻译——七棋种伪合法走法（马腿/象眼/炮架/兵过河/九宫）、willBeInCheckAfter 原地模拟自将过滤、IsCheck（含将帅照面）、checkmate/stalemate、apply/undo 互逆；moves.json **集合对拍 16 案例全绿**（含初始局面 44 着法/照面负例/困毙 status）+ board.spec 走法生成 13 用例。
- T1.3（`test(m1)` ed788bf）：负例用例——被牵制车自将过滤/非轮走方空表/照面无遮蔽过滤与有遮蔽合法/单车将死/炮牵制困毙（board.spec 对应段 9 用例）。
- T1.4（`feat(m1)` 6ecdfe9）：中文纵线记法 moveNotation.ts 逐行翻译（红汉字/黑阿拉伯数字、平/进退/斜走三分支）；notation.json **逐字对拍 21 案例全绿** + moveNotation.spec 7 用例。
- T1.5（`feat(m1)` af054d3，**Decision: DR-006**）：L3 重复裁决 JudgeRepetition——repetitionJudge.ts 逐行翻译（02 §7 全规格：无状态纯函数/classifyCycle 将军归责+结果缓存/k=2 单方警告且闲着环·照面环不警告/k=3 判负·不变作和·判和/k≥4 强制和）；SWING/QUIET/FACING **三类环 9 用例与 Electron 版 repetitionJudge.spec.ts 期望逐条一致**（跨语言同 FEN 同裁决）+ 悔棋截断回滚用例。【DR-006 检查点：L3 已在本里程碑落地 ✓】

**质量门（全绿）**：`gofmt -l` 空输出；`go vet ./...` 0 问题；`go test ./... -race` 通过；`npm run test:fe` 21 文件 202 用例通过（M1 未触前端，回归确认）。
**Go 侧用例**：internal/rules 84 用例全绿（含金标准 56 案例：19 FEN 往返 + 16 走法集合 + 21 记法逐字；单测 28：fen 6 + board 22 + 记法 7 + L3 9）。
**两轮复审（11 §6.1）**：第一轮逐文件对照 TS 版函数面（33 导出符）与行为分支——语义一致，无 P0/P1；第二轮缺陷扫描——纯函数库无竞态/取消面，Board 非并发安全已加注释（跨 goroutine 用 Copy 快照），铁律 #1 import 面机器检查=仅 stdlib（fmt/strconv/strings）。P3×5 记 docs/KNOWN_ISSUES.md（K7~K11：buildFen 接口收敛依据、BoardGrid 形态、辅助函数未导出、L3 无效 FEN 不可达分支、记法退化输入）。
**性能实测**：无（M1 无算法面；L3 为 FEN 串比较纯函数，n≤数百微秒级，03 引擎 L0/L1/L2 自 M3 起记录）。

**下一里程碑**：M2（对战页 + 存储，fenHistory 四收口 + UI 裁决接线）——本里程碑手测验收通过后，新会话逐字粘贴 11 §4.3 启动提示词。

### 2026-10-05 M0 工程骨架交付（T0.1/T0.2/T0.3，待手测）

**完成清单**
- T0.1（commit `feat(m0)`）：go.mod（module github.com/jxsword/chinese_chess_go，Go 1.23）+ Wails v2.16.0 初始化 + main.go（embed frontend/dist、ldflags 版本注入口径）+ app.go 绑定骨架（00 §3.2 全通道占位：读路径空态、写/计算路径里程碑错误；requestID+context 取消占位）+ build/ 打包资源 + `.gitignore`（frontend/dist 仅保留 .gitkeep，build 脚本跨平台自恢复）。依赖仅白名单内 wails/v2（传递依赖随框架）；modernc/sqlite、go-keyring 随 M2 引入。
- T0.2（commit `feat(m0)`，Decision: DR-007）：frontend 移植——src/renderer 整体复制（17 文件 import 路径机械改写 ipc→api、workers→api）+ src/shared、src/packages 随迁（DR-007 选项 A）+ 新增 src/api/ 九件（client 探测、wailsAdapter 00 §3.2 通道映射、mockAdapter 移植、engine/solver/parser 客户端+协议：传输后端注入 wails 绑定/协议核心 mock，{id,type,payload} 形状与迟到丢弃/取消语义逐行保留，铁律 #7）+ 删除 workers/*.worker.ts 薄壳（桌面引擎计算改走 App.Engine* 占位绑定）+ iconv-lite 浏览器 shim（TextDecoder gb18030；实测 Electron 版 dev:web 同样白屏，见 DR-007 附带决策）。
- T0.3（commit `ci(m0)`）：.github/workflows/ci.yml——三平台矩阵 gofmt/vet/go test -race（Linux 装 libwebkit2gtk-4.1-dev + webkit2_41 标签）+ 前端 job（npm ci → tsc → eslint → vitest → vite build）+ 缓存（setup-go：go-build 与 ~/go/pkg/mod，key 随 go.sum；setup-node：npm）。
- CI 修复一轮：Windows runner CRLF 检出致 gofmt 误报 → `.gitattributes` 强制 `*.go` LF。**最终 CI 三平台全绿**（run 37232020881：Frontend 41s / Go ubuntu 1m04s / Go macos 24s / Go windows 1m17s；注解仅 Node20→24 的 actions 弃用提示，不影响）。

**测试随迁（10 R9）**：21 文件 202 用例全绿——stores(5)/pages(8)/ui(1)/studio(3)/api 客户端(3)/contract(1)。领域包用例（rules/engine/llm/parsers/solver/storage）不随迁，由 M1/M3/M4/M5 Go 侧金标准对拍承接；worker 崩溃降级用例按 DR-003 语义等价改写为「绑定错误响应收口」用例（用例数不减）。

**验证门取证**
- `wails dev`（WSLg）：窗口「中国象棋 Ultra」1280×800 出现（xwininfo 取证）；devserver :34115 HTTP 200；日志无 panic/fatal。⚠ Ubuntu 26.04 需 `-tags webkit2_41`（webkit2gtk-4.0 已从源移除，10 §3 R1' 缓解——CI 同口径）。窗口内容渲染效果待用户手测确认（xwd 截图全黑为 WSLg 取证伪影，KNOWN_ISSUES K5）。
- 浏览器 mock 模式（dev:web）：headless Chrome 点击式导航 7 入口全部 PASS + 双人页「炮二平五」真实走子 PASS（回合切换/步数/落子动画，全程 console 0 错误）；vite build 产物 630KB。
- **wailsAdapter 桌面链路**：注入 fake 绑定后全链路验证（DbLoadLatest/DbSaveGame 载荷形状精确匹配 SaveGameRequest）；修复探测 P0（window.go={} 真值误判，F1）。外部浏览器打开 devserver 自动回落 mock 模式。

**两轮复审（11 §6）**：P0×1 已修复（F1）；P2×1 + P3×5 记 docs/KNOWN_ISSUES.md（K1~K6；K1=DR-005 冲突面按计划 M4 消解）。铁律 grep 自检：net/http=0、frontend fetch()=0、协议形状 3/3 保留、internal/ 未建（M1 起，铁律 #1/#3 逐里程碑自检）。

**性能实测**：无（M0 无算法面）。引擎/求解耗时自 M3 起记录。

**命令口径**：npm 脚本位于 `frontend/`——`npm --prefix frontend run dev:web / test:fe`（或在 frontend/ 内执行）；AGENTS.md 常用命令中的 `npm run dev:web`、`npm run test:fe` 均按此理解。本地桌面开发：`wails dev -tags webkit2_41`。

**已知问题**：见 docs/KNOWN_ISSUES.md（K1~K6）。

**下一里程碑**：M1（规则内核 + L3）——用户新会话逐字粘贴 11 §4.2 启动提示词；本里程碑验收通过后再启动（11 §5 纪律）。

### 2026-10-05 补充：11 手册 M0~M7 完整启动提示词
- 新增 11 §4（8 条自包含启动提示词，全新会话逐字粘贴即可开工；每条含收尾=复审两轮→质量门→PROGRESS→暂停等手测）；
- 硬性要求逐条注入：DR-006（M1 L3 / M2 fenHistory 四收口+接线 / M3 先对拍后开 L1/L2）、DR-005（M4/M6/M7 恒发关闭参数+请求体快照断言）；
- 章节顺延（验收流 §5 / 复审 §6 / 跑偏表 §7 / 同步纪律 §8），全部内部引用已同步。

### 2026-10-05 设计文档集交付（首次提交）
- 交付：AGENTS.md（10 条铁律，Go 化）、design_docs/00~11 全套 + README 导航、decision_log.md（DR-001~006）、.gitignore 修正（build/bin）、本文件。
- 两大前置落位：
  - **DR-006 重复治理前置**：L3 → M1；fenHistory 四收口 + 双人页裁决接线 → M2；L0/L1/L2 → M3（先对拍后开层）。参数逐值搬移自 electron-DR-018/019 final 设计。
  - **DR-005 思维链强制关闭**：请求构造层恒发、按预设映射（Qwen→enable_thinking:false / GLM-4.5+→thinking:disabled / 其余兜底）、无 UI 开关；eval 同样恒关。
- 关键资产复用：金标准 testdata/golden（自 electron tools/golden 复制）、引擎逐行翻译源=electron TS 版、React 前端整体移植。
- 已知问题：无。
- 下一步：M0（在新会话逐字粘贴 11 §4.1 M0 启动提示词启动；完成后按 11 §5 验收流暂停等手测）。

## 开发环境（跨会话备忘，2026-10-05 盘点）

- 工具链已齐：Go 1.27.1、Node 24.15（nvm）、wails CLI v2.16.0、git、gh（已认证）；Wails Linux 依赖 libgtk-3-dev 与 libwebkit2gtk-4.1-dev 已装（webkit2gtk-4.0 已从 Ubuntu 26.04 源移除，一律用 `wails dev/build -tags webkit2_41`，CI 同口径）。
- 测试取证：google-chrome（headless 冒烟）、xwd（窗口取证）；playwright-core 借用 `/home/ssy/proj/chinese_chess_electron/node_modules`（M7 E2E 落地时再入依赖）。
- **安装权限约定（用户指示）**：开发/构建缺包时代理应自行安装；本机 `sudo` 需密码，**缺包时代理给出确切命令通知用户手动执行**（go/npm 用户态操作不受影响）。M7 打包工具（nfpm/appimagetool/nsis）届时按此办理。

## 手测指引（M1，已随用户启动 M2 验收通过，留档）

M1 为纯 Go 规则内核（internal/rules），**无 UI 面**——走子/悔棋/照面禁手/重复裁决的界面级手测在 M2 双人页落地后进行。本里程碑手测以命令验证为主：

1. **全量规则测试**：`go test ./internal/rules/ -v -count=1` → 84 用例全绿（金标准 56 案例与 Electron/Vitest 版共用同一 JSON 期望值，跨语言逐位可比）。
2. **带竞态检测的全仓测试**：`go test ./... -race` → 全绿。
3. **前端回归**（M1 未触前端，确认无回归）：`npm --prefix frontend run test:fe` → 202 用例通过。
4. **金标准来源复核**（可选）：`sha256sum testdata/golden/*.json` 与 Electron 版 `/home/ssy/proj/chinese_chess_electron/tools/golden/` 一致（基建 commit 留档）。
5. **L3 三类环裁决抽查**（可选）：`go test ./internal/rules/ -run TestJudgeRepetition -v` → k=2 单方长将警告/照面环不警告、k=3 判负·不变作和·判和、k=4 强制和、悔棋截断回到 k=2。
6. 如发现问题：列出现象与复现命令，按 11 §5/§6.2 处理（禁止删/跳用例变绿）。

预期已知行为（非 bug）：AI/LLM/语料/保存相关动作提示"尚未接入"（占位绑定，M2~M6 逐个落地）；桌面窗口内的控制台可能打印自动保存未处理拒绝（K3）。
