# 决策记录（Decision Records）— ChineseChessUltra Go

> 格式沿用 Electron 版约定（背景/全部选项/结论/理由/影响）。
> Go 版从 DR-001 重新编号；引用 Electron 版决策时写作 `electron-DR-xxx`。

## DR-001 桌面框架：Wails v2 + 移植 Electron 版 React 前端（2026-10-05）
- 背景：Go 生态无 Electron 等价物。UI 层选型决定 08 文档（UI 规格）能否复用与交付速度。约束：三平台桌面端、WSL 开发、行为对齐 Electron v1.0、高效准确快速交付。
- 选项：
  - A. Wails v2 + 移植 React renderer——优点：Go 后端替代主进程（绑定方法+事件，与 IPC 语义同构）；**现有 React UI 近乎 1:1 移植**（08 规格/防错清单/棋盘组件直接复用，仅 `window.api` 层换 Wails 绑定适配器）；原生 WebView（WebView2/WebKitGTK/WKWebView）安装包 ~15MB 级；v2.x GA 稳定。缺点：Linux 需 webkit2gtk 系统依赖；三平台 WebView 渲染细节有差异。代价：首次编码量最小。
  - B. Fyne（纯 Go GUI）——优点：单二进制无 WebView 依赖、纯 Go 交叉编译。缺点：棋盘/工作室/棋谱库全部 UI 从零用 Fyne widget 重写（08 文档交互规格重新实现）；文本排版/动画能力弱于 Web。代价：UI 工作量约等于新增一个里程碑集合，交付最慢。弃。
  - C. Wails v3——优点：下一代事件模型更干净。缺点：alpha 阶段 API 不稳定。代价：风险最高。弃。
  - D. 本地 HTTP 服务 + 浏览器——优点：最简单。缺点：非桌面应用（无窗口/自启/打包语义）。弃。
- 结论：A。
- 理由：交付速度与行为保真优先；UI 层是 Electron 版已验收资产，直接复用把风险集中在 Go 后端（可用金标准对拍兜底）。
- 影响：00 文档进程/分层模型、08 文档移植说明、09 测试双栈、CI Linux runner 需 webkit2gtk 依赖。

## DR-002 SQLite 驱动：modernc.org/sqlite（纯 Go）（2026-10-05）
- 背景：存储层需在三平台 CI 交叉编译。Electron 版曾因 better-sqlite3 原生模块承受 ABI/rebuild 复杂度（electron-DR-006、风险 R1）。
- 选项：
  - A. modernc.org/sqlite——优点：纯 Go 零 CGO，三平台交叉编译无痛，彻底消除原生模块类风险；单二进制；性能对本项目量级（14 万局索引）足够。缺点：比 cgo 版慢 ~2-3x（非本项目瓶颈）。代价：无。
  - B. mattn/go-sqlite3——优点：性能最好、最流行。缺点：CGO 需各平台工具链（macOS 交叉编译尤其麻烦），重蹈原生模块覆辙。代价：CI 矩阵复杂化。弃。
- 结论：A。
- 影响：internal/storage、CI 无需任何原生工具链；07 文档 Schema 仍与 Electron 版逐字段一致。

## DR-003 并发模型：goroutine + context 替代 Web Worker/IPC 取消（2026-10-05）
- 背景：Electron 版用 Web Worker + requestId IPC 取消（electron-DR-003/005/009）。Go 单进程，原生并发。
- 选项：
  - A. goroutine + context.Context——优点：取消即 `ctx.Done()`，天然同构 requestId 丢弃语义；`go test -race` 可测竞态。缺点：共享内存需注意数据竞争（测试门加 -race 兜底）。代价：无。
  - B. 保留 Web Worker 式消息进程（独立 OS 进程）——优点：隔离崩溃。缺点：Go 单二进制语义下过度设计，IPC 序列化开销。弃。
- 结论：A。消息形状仍保留 `{id, type, payload}` / `{id, ok, result|error|progress}`（测试可注入 fake、与 Electron 版协议心智一致）；迟到响应按 id 丢弃的收口逻辑照搬（调用方层）。
- 影响：03 §7 goroutine 模型、04/06 的 context 取消、09 §2.4 集成测试口径。

## DR-004 外呼收口：单进程内包边界替代主进程代理（2026-10-05）
- 背景：Electron 版铁律"对外 HTTP 仅主进程 undici 代理"（electron-DR-004），防渲染层 CORS/凭据泄漏。Go 版无进程隔离。
- 选项：
  - A. 包边界收口——`internal/llm/transport` 是唯一 import net/http 的包；frontend（WebView）禁止直接 fetch 外部 URL；Key 只在凭据包内出现。优点：编译期可查（go vet 自定义检查或评审纪律）；单进程内无需自建代理层。缺点：无运行时强隔离。代价：无。
  - B. 自建本地代理进程——优点：强隔离。缺点：为隔离而引入双进程复杂度，与单二进制目标冲突。弃。
- 结论：A。安全清单（掩码/日志不泄 Key/CSP）沿用 Electron 版 00 §5 中不依赖进程边界的条目。
- 影响：05 §3 传输层、AGENTS.md 铁律 #4。

## DR-005 LLM 思维链强制关闭：请求层恒发 + 按预设映射关闭参数（2026-10-05）
- 背景：用户硬性要求——下棋用不到思维链，且显著影响交互响应速度（Electron 版实测：识图开思维链 61–115s，关闭后 6–14s）。Electron 版 `enable_thinking:false` 是可选开关（默认对识图/助手开、对弈关），且只覆盖 DashScope 语义。
- 关键事实：`enable_thinking` 是 DashScope/Qwen3 顶层布尔参数；**智谱 GLM-4.5+ 不识别该参数**（其关闭思维链参数为 `thinking:{"type":"disabled"}`）；DeepSeek-chat/Kimi/OpenAI 系无思维链（未知参数被忽略）。
- 选项：
  - A. 沿用 Electron 版单一 `enable_thinking` 开关（UI 可选）——缺点：GLM-4.5+ 落空（不识别即保持开启）；违背"必须关闭"硬性要求。弃。
  - B. 请求构造层恒发关闭参数 + **按端点预设映射**：DashScope/Qwen 系→`"enable_thinking": false`；智谱 GLM（glm-4.5 及以上）→`"thinking": {"type": "disabled"}`；其余端点→`"enable_thinking": false` 兜底（未知参数被忽略，无害）。UI/配置**不提供任何开启思维链的开关**。优点：全预设端点语义正确；配置面简化。缺点：自定义端点若参数名不同需走"自定义"预设兜底。代价：无。
  - C. 保留 UI 开关、默认关闭——缺点：用户要求"必须"，开关即漏洞面。弃。
- 结论：B。请求 JSON 顶层字段（在 Electron 版 buildChatRequest 基础上的唯一差异）：temperature 0.3、max_tokens（v1=4096/v2=8192）、stream:true 恒定；关闭参数按预设映射恒发。识图（非流式，temperature 0.1、max_tokens 4096）同样恒发。MatchRunner 评估请求同样恒关（不保留 Electron 版 eval 的 disableThinking=false 基线口径）。存储 schema 中不再有 disableThinking 字段（07 §3）。
- 影响：05 §3.1 请求构造表、07 §3 设置 schema、AGENTS.md 铁律 #10、eval CLI。

## DR-006 重复变招治理前置：L0/L1/L2 随引擎首版、L3 随规则首版（2026-10-05）
- 背景：用户硬性要求——Electron 版该修复在 M6 后插入（electron-DR-018/019，工作量约为单做 L1+L2 的 3 倍，波及引擎/worker/VM/四对局页/存档/五处文档）。Go 版必须在前期对应里程碑实现，避免同等返工。
- 选项：
  - A. 照搬 Electron 版时序（M6 后修）——缺点：已知大返工，违背用户要求。弃。
  - B. **首版即含**（采纳）：L0 Zobrist 内置于棋盘核心（Go 用 uint64 单键，比 TS 双 32 位干净；固定种子 xorshift 保证可复现快照，键值跨语言不必一致）；**L1 搜索内检测 + L2 根节点回避随 M3 引擎首版**（`FindBestMoveOptions.HistoryFens []string` 签名第一版就有，缺省=关闭=与 Dart 口径逐位一致；参谋接口 findBestMoveEx/evaluateMove 一律不接受该参数——可复现铁律不变）；**L3 规则裁决纯函数随 M1 规则首版**（无状态 `JudgeRepetition(fenHistory)`，悔棋/读档/新局回滚问题被设计消解）；GameVm 的 fenHistory 四收口点（初始/落子 push/悔棋 pop/restore 重放采集）随 M2 状态层首版；UI 裁决接线随 M2（双人/人机）与 M4（LLM 页）出现即挂；GameResult 首版即含 draw 与 agreeDraw。优点：无返工；每层落地时其依赖恰好就绪。缺点：M1/M3 各多付一小节实现成本。代价：远低于事后返工。
  - C. 只做 L1+L2 前置、L3 后补——缺点：规则判罚缺口在 M4 LLM 对战循环中即暴露死锁风险（Electron 版 R8）。弃。
- 结论：B。M3 内部顺序固化：先"不传 HistoryFens 与金标准（engine.json）逐位一致"对拍全绿，**再**开启 L1/L2（受控偏差、默认关闭铁律，沿 Electron 版 §7 测试口径）。全部参数逐值搬移：PATH_REPEAT_PENALTY=[50,150]（第 3 次起返回 0 和棋分）、GLOBAL_REPEAT_PENALTY=100、GLOBAL_CHECK_MAX_PLY=3 且 count≥2 才罚、AVOID_THRESHOLD_BASE={1:200,2:200,3:100,4:50,5:30}、优劣势系数（>200→×0.5，<−200→×2.0）、FORCED_CHANGE_FLOOR=500、L3 裁决状态机（k=2 单方长将警告/k=3 单方判负/双方全程将军不变作和/无人将军判和（AI 自动、玩家询问）/k≥4 强制和）。**长捉与自然限着仍不实现**。
- 影响：02 §7（L3）、03 §4/§6（L0/L1/L2）、07 §2（fenHistory 四收口）、08 §3（裁决交互）、09（各层测试口径）、10 里程碑表。

## 附：沿用 Electron 版不做重裁的决策清单

| electron-DR | 主题 | Go 版沿用方式 |
|---|---|---|
| DR-007 | Int8Array(90) 一维棋盘 + 整数打包走法 | 翻译为 `[90]int8` + packed move int（逐行翻译） |
| DR-008 | MVV-LVA 排序键内嵌 packed 高位 | 逐行翻译（剪枝形态敏感，AGENTS 铁律 #9） |
| DR-011 | 安全存储不可用→明文文件 0600 + 如实回报 | keyring 失败回退路径同构 |
| DR-012 | 大模型对战空配置侧跟随对方 | 业务规则照搬 |
| DR-013 | 掩码 Key 回写合并 + onAttempt 进度 | 照搬 |
| DR-014 | 黑方配置跨页镜像 + 引擎类型可选 | 照搬 |
| DR-015 | XQF 解密验证：真实样例锚点 + 测试侧往返构造器 | 照搬（样例文件从 Electron 版 test 资产复制） |
| DR-018/019 | 重复治理三层 + Zobrist 单源 | 见 DR-006（前置化） |
| DR-016 | zip 零依赖解压 | Go 标准库 archive/zip 实现（同等安全检查） |
| electron-DR-005 | invoke 恒 resolve + 结局走事件 | 映射为 Wails 绑定 + EventsEmit（00 §3） |
