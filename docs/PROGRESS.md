# 开发进度（PROGRESS.md）

> 与 AGENTS.md §开发规范联动：每个子任务/里程碑收尾更新本文件；**禁止代码领先文档**。
> 状态图例：⬜ 未开始 ｜ 🔵 进行中 ｜ ✅ 完成（测试全绿）｜ ✅+用户 已通过用户手测验收

## 当前状态

**M0（工程骨架）待用户手测验收**——T0.1/T0.2/T0.3 全部完成、质量门全绿（见下方 M0 交付记录）；按 11 §5 暂停等待手动验收。

## 里程碑总览

| 里程碑 | 状态 | tag | 备注 |
|---|---|---|---|
| 文档集（首次提交） | ✅ | — | 00~11 全套 + AGENTS.md + DR-001~006 |
| M0 工程骨架 | 🔵 代码完成，待手测 | — | go.mod + Wails + frontend 移植 + CI（验证门取证见下） |
| M1 规则内核 + L3 | ⬜ | | 金标准对拍 |
| M2 对战页 + 存储 | ⬜ | | fenHistory 四收口 + 裁决接线 |
| M3 引擎 + L0/L1/L2 | ⬜ | | engine.json 对拍 → 开层 |
| M4 LLM 全链路 | ⬜ | | 恒关思维链 + 真实端点手测 |
| M5 语料 + 棋谱 | ⬜ | | |
| M6 工作室 + 求解器 + 识图 | ⬜ | | |
| M7 评估 + 打包发布 | ⬜ | | MatchRunner + 三平台 Release |

## 变更日志

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

## 手测指引（M0 待验收）

1. **桌面窗口（wails dev）**：在仓库根目录执行 `wails dev -tags webkit2_41` → WSLg 弹出「中国象棋 Ultra」窗口，主页 7 入口可见可点击。
2. **双人页对局**：主页 →「双人对弈」→ 点选红炮（h 炮）→ 点击 e 线目标（炮二平五）→ 棋子动画落位、回合变黑方、步数 +1；「悔棋」回退、「新游戏」有确认框。
3. **其余页面空态**：人机对战页可进（AI 应手会提示引擎未接入——M3 前占位，属预期）；人机对战（大模型）/大模型对战页配置卡正常显示；残局选关显示「未找到本地棋谱语料 + 下载引导」（mock）；残局工作室三 Tab 可切换；棋谱库空列表。
4. **浏览器 mock 模式**：`npm --prefix frontend run dev:web` → http://localhost:5173 重复 2~3 的导航。
5. **全局设置**：主页右上 ⚙ 打开弹窗，开关可切换、关闭后不报错。
6. 如发现 bug：列出现象（页面 + 操作步骤），按 11 §5/§6.2 修复回归后再验收。

预期已知行为（非 bug）：AI/LLM/语料/保存相关动作提示"尚未接入"（占位绑定，M2~M6 逐个落地）；桌面窗口内的控制台可能打印自动保存未处理拒绝（K3）。
