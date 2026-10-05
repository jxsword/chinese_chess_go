# 已知问题（KNOWN_ISSUES.md）

> 记录各里程碑复审（11 §6.1）产出的 P2/P3 项与按计划后置的冲突面；P0/P1 修复后即从本表移除。
> 每里程碑收尾更新；修复后随对应里程碑 commit 注销。

## M0（工程骨架）

| # | 级别 | 描述 | 处置计划 |
|---|---|---|---|
| K1 | P2 | **LLM 思维链 UI 开关与条件关闭参数与 DR-005 冲突**：随迁的 Electron 版 `@packages/llm/config.ts`/`vision.ts` 按 `disableThinking` 用户开关条件发送 `enable_thinking:false`，LLM 配置卡含「禁用思维链」UI 开关。M0 无真实 LLM 请求（占位绑定拒绝），无运行时暴露。 | M4 T4.2/T4.5：请求构造移入 `internal/llm/config.go` 恒发关闭参数（DR-005 预设映射），删除配置卡开关与 `disableThinking` 字段（07 §4 槽位 JSON 无该字段）。 |
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
| K12 | P3 | **凭据槽位契约面字段差（DR-005 消解面）**：Go `SecureGet/Set` 槽位 JSON 为 `{baseUrl, apiKey, model, preset}`（07 §4），随迁前端 `LlmEndpointConfig` 类型仍含 `disableThinking`、缺 `preset`——M2 仅接存储无人消费该面（双人页不用 secure 通道），`disableThinking` 被 json 丢弃、`SecureGet` 返回缺该字段（配置卡开关显示为关）。 | M4 T4.5 配置卡接线：删除思维链开关与 `disableThinking` 字段（铁律 #10/K1 同源）、引入 preset 选择；`@shared/ipc/types` 同步。 |
| K13 | P3 | **Wails v2 无窗口 minimize/blur 后端事件**：07 §2 映射表的 Electron `win.on('blur')/('minimize')` 无 Go 等价物。适配层以 WebView `window.blur` 派发 `blur` 相位补偿（最小化在主流平台伴随失焦）；`close`/`before-quit` 由 OnBeforeClose 发出（有界等待 300ms best-effort 存档）。极平台最小化不伴随失焦时漏一次自动保存。 | 有路由卸载（dispose）/close/手动保存三重兜底，可接受；Wails v3 或后续版本提供窗口事件时补齐。 |
| K14 | P3 | **parseMovesJson 非整数值丢弃**（db.ts 保留 `number[][]`，Go 侧仅收整数四元组）：TS 保留的 1.5 等行在 restore 必被跳脏跳过，净效果一致（db.go 注释留档）。 | 无需处置；跨语言审计点。 |
| K15 | P3 | **settings clamp 边界差**：load 时 clamp 仅对已存在键生效（缺省键不注入内存表，`StoreGet` 返回 null 由渲染层 fromRaw 兜底——electron-store 虚拟缺省语义）；present 数值截断取整（7.9→7，TS fromRaw 保留 7.9 后由渲染层再 clamp）。仅手改 settings.json 场景可观测。 | 无需处置；M4 Go 侧消费者（代理空闲超时）注意 nil→默认兜底。 |
| K16 | P3 | **beforeClose 有界等待 300ms**：退出前 fire-and-forget 自动存档的 best-effort 窗口（等价 Electron 同步 best-effort 语义）；极端慢盘下最后一着可能不入档。 | 可接受（与 Electron 版同级保真）；如手测出现高频丢档再改前台等待确认。 |

## M3（引擎 + L0/L1/L2 重复治理）

| # | 级别 | 描述 | 处置计划 |
|---|---|---|---|
| K17 | P3 | **Runner 同 id 并发在途注册表以后到者为准**：Submit 重写 cancels[id]，先到请求结算时 delete 会摘走后到的注册项，其后 Cancel(id) 对后到请求失效。requestId 全局唯一是 00 §3.2 调用方契约（前端 createRequestId UUID 工厂，页面 gameSeq 保证至多一个在途 AI 搜索），契约内不可达。 | 无需处置；如未来出现同 id 复用在途场景，Submit 改为拒绝重复 id（一行改动）。 |
| K18 | P3 | **随机路径（难度 1/2 与 L2 候选洗牌）跨语言伪随机源不同**：TS `Math.random` vs Go `math/rand/v2`（自动播种）。randomness>0 时难度 1/2 的具体应手、L2 阈值内多候选时的具体换着，Go 与 TS/Dart 不逐位一致——原版同分随机即不保证复现（金标准对拍仅覆盖 randomness=0 路径）。 | 无需处置；行为面（随机取一）与阈值/候选集口径逐值一致，确定性测试以注入随机源锁定（avoidance_test）。 |

## 已修复（保留记录）

| # | 级别 | 描述 | 修复 |
|---|---|---|---|
| F1 | P0 | api 适配器探测条件过宽：Wails 运行时先同步注入 `window.go = {}`（App 方法表异步经 SetBindings 填充），以 `window.go` 真值判定会在外部浏览器打开 devserver 时误选 wailsAdapter 并在创建期抛错白屏。 | ~~`client.ts` 探测改为 `window.go?.app?.App !== undefined`~~（M2 勘误：该修复基于错误命名空间假设 `app.App`，实际把缺陷转为静默回落 mock，见 F2）。 |
| K3 | P3 | 自动保存 `gameAutoSave.write()` 以 `void repo.saveGame(...)` 发即忘；wails 占位绑定 reject 时产生未处理 promise 拒绝（仅控制台噪音，UI 不受影响；Electron 版同型）。 | M2 已注销：db 通道接入真实存储（app.go DbSaveGame），占位拒绝噪音消解；存储真不可用时的 reject 仍保持错误可见（属正确行为面）。 |
| P1' | P1 | 凭据服务无互斥：Wails 绑定方法并发进入时回退文件读改写竞态可丢槽位更新。 | T2.4 复审修复：Credentials 全路径加 sync.Mutex（合并路径 getRawLocked 复用锁内调用防死锁）+ 并发用例 -race 覆盖。 |
| F2 | P0 | **绑定命名空间错误（M2 手测 B/C 缺陷根因）**：Wails v2 以绑定结构体所在 Go 包名挂载方法表（`window.wailsbindings={"main":{"App":{...}}}`），本项目为 `window.go.main.App`；M0 起探测与 wailsAdapter/engineClient/solverClient/parserClient 全部误用 `window.go.app.App`（恒 undefined）→ 桌面窗口内 client.ts 静默回落 mock——自动/手动存档不落库（无 sqlite 文件）、设置切换不写 settings.json、分享棋局剪贴板为 mock 空操作。M0 的"外部浏览器回落 mock"观察实为同一根因的表现（F1 的修复掩盖了它）。 | `client.ts` 探测、wailsAdapter、engine/solver/parser 三客户端统一改为 `main.App`（对齐 wailsjs 生成物与 runtime.js 注入事实）；documentsDir 在 ~/Documents 缺失时创建（07 §1 路径落地）。回归：新增 test/api/wailsAdapter.spec.ts 5 用例锁定命名空间契约与生命周期双源；端到端取证（devserver + 真实绑定调用）——DbSaveGame/DbLoadLatest 往返、settings.json 落盘、~/Documents/chinese_chess_ultra_go.sqlite 生成、ClipboardWrite 成功。 |
| F3 | P1 | **存档恢复盘面被改写（M2 手测第二轮）**：恢复后黑马位置出现黑车、原位车消失、走法历史清空。根因：保存端 `serialize()` 存当前（终局）局面 FEN 而 `restore()` 以其为基准重放着法栈（Flutter `board_vm.dart:308-311` 与 `:87-132` 语义错位，原版即有、Electron 逐字继承）——起点在终局盘面恰有子的着法被误重放（手测局：黑车 h9 被当作马 h9→g7 的起点"再走"，车覆盖马）。 | DR-008 修复：serialize 存**本局起始 FEN** + 完整着法栈（GameVm `_startFen`；构造/newGame/newGameFromFen/restore 设定），restore 从起始局面重放重建整局——局面/历史/fenHistory 全部正确，跳脏语义保留。用例改写 3 处（gameVmFenHistory/autoSaveRestore/gameStore spec，注明依据）；端到端复现用户 4 步对局经真实 Go 存储：恢复后局面逐位一致、4 手历史完整。跨语言差异：Go 版恢复语义与原版不同（原版带缺陷），已在 07 §2 与 decision_log DR-008 留档。旧存档（修复前写入、fen 为终局语义）不兼容，重新保存一次即覆盖。 |
