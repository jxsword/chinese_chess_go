# 开发进度（PROGRESS.md）

> 与 AGENTS.md §开发规范联动：每个子任务/里程碑收尾更新本文件；**禁止代码领先文档**。
> 状态图例：⬜ 未开始 ｜ 🔵 进行中 ｜ ✅ 完成（测试全绿）｜ ✅+用户 已通过用户手测验收

## 当前状态

**M2（对战页 + 存储）代码完成，待用户复测**——T2.1~T2.4 全部提交、两轮复审完成（P1×1 已修复，P2/P3 记 KNOWN_ISSUES K12~K16）、质量门全绿；首轮手测发现 P0 缺陷 F2（绑定命名空间错误致桌面窗口静默走 mock），已修复并端到端取证，等待复测。

## 里程碑总览

| 里程碑 | 状态 | tag | 备注 |
|---|---|---|---|
| 文档集（首次提交） | ✅ | — | 00~11 全套 + AGENTS.md + DR-001~006 |
| M0 工程骨架 | ✅+用户 | — | go.mod + Wails + frontend 移植 + CI（用户以启动 M1 验收通过） |
| M1 规则内核 + L3 | ✅ | — | 金标准对拍 56 案例全绿 + L3 三类环裁决（T1.1~T1.5；用户以启动 M2 验收通过） |
| M2 对战页 + 存储 | 🔵 代码完成，待手测 | — | sqlite DAO + 设置/凭据 + fenHistory 四收口 + 双人页裁决接线（T2.1~T2.4） |
| M3 引擎 + L0/L1/L2 | ⬜ | | engine.json 对拍 → 开层 |
| M4 LLM 全链路 | ⬜ | | 恒关思维链 + 真实端点手测 |
| M5 语料 + 棋谱 | ⬜ | | |
| M6 工作室 + 求解器 + 识图 | ⬜ | | |
| M7 评估 + 打包发布 | ⬜ | | MatchRunner + 三平台 Release |

## 变更日志

### 2026-10-05 M2 对战页 + 存储 交付（T2.1~T2.4，待手测）

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

预期已知行为（非 bug）：人机/LLM 对战页 AI 应手提示"尚未接入"（M3/M4 占位）；棋谱库列表为空（M5）；语料/求解入口同前占位（M5/M6）；凭据槽位在 WSL 无 Secret Service 时保存会提示"已用未加密本地文件存储"（DR-011 如实回报，文件 0600）。

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
