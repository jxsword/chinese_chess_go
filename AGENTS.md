除非显式指定读取当前工作区 tmp/ 目录下的文件，否则默认不读取该目录的任何文件。

# ChineseChessUltra Go — 项目常驻指令（AGENTS.md）

> 本文件位于项目根目录，AI 编码代理每次会话自动读取。它是所有会话共享的"系统提示词"。
> 各里程碑/子任务提示词（见 design_docs/11-开发执行手册.md）在此基础上叠加，**不重复本文件内容**。

## 项目定位

将中国象棋应用用 **Go 语言全栈重写**为桌面应用。行为事实源与对照实现：

- **领域行为锚点**：Flutter 原版 `/home/ssy/proj/ChineseChessUltra`（五期交付，需求规格含源码锚点）；
- **直接翻译源**：Electron 版 `/home/ssy/proj/chinese_chess_electron`（已与 Dart 金标准对拍全绿，含重复变招治理 DR-018/019 与全部协议实现）；
- **本仓库设计文档是 Go 版唯一事实源**，位于 `design_docs/`。两处引用冲突时，协议原文/算法参数以 Electron 版为逐字事实源、架构与工程以本仓库文档为准。

## 技术栈（已定稿，见 design_docs/decision_log.md DR-001~006，不得擅自更换）

- 桌面框架：**Wails v2**（Go 后端 + 原生 WebView；前端移植 Electron 版 React 18 + TS + Vite + Zustand）
- 语言：**Go 1.23+**；领域层 `internal/` 为**纯 Go**（零 Wails/零 UI/零网络依赖的包不 import 任何运行时符号）
- 数据库：**modernc.org/sqlite**（纯 Go 零 CGO）
- LLM：net/http + SSE 流式；**思维链强制关闭**（请求层恒发关闭参数，无 UI 开关，DR-005）
- 凭据：OS keyring（zalando/go-keyring）+ 不可用时明文文件 0600 回退（沿 Electron 版 DR-011）
- 测试：`go test` + **跨语言金标准对拍**（复用 Electron 版 tools/golden 的 fen/moves/notation/engine 四份 JSON）+ Vitest（前端）+ Playwright（浏览器 mock 模式 E2E）
- 依赖白名单（新依赖须走说明理由+确认流程）：wails/v2、modernc.org/sqlite、zalando/go-keyring 及其传递依赖；前端沿用 Electron 版 package.json。

## 架构铁律（违反即返工）

1. `internal/*`（rules/engine/solver/llm/parsers/storage）是**纯 Go 包**：禁止 import Wails / net/http / frontend / 任何 GUI 或运行时绑定符号——它们必须可被 `go test`、CLI（cmd/eval）、Wails 后端三端直接调用。
2. Electron 版 05 文档 §2 的提示词文本、§4 的归一化/正则/过滤管线是**调优过的协议**，必须逐字搬运，禁止意译改写、"优化"措辞；prompt 快照以 Electron 版既有快照为基准。
3. 本地规则内核是着法合法性的**唯一事实源**：LLM 回复必须与本地生成的白名单精确匹配（字符串精确比对）；`playMove` 始终保留最终校验。
4. 对外 HTTP 一律收口在 `internal/llm/transport` 包；frontend 禁止直接 fetch 外部 URL（浏览器 mock 模式除外，且 mock 不触网）。
5. 所有异步通道带 `requestId`；取消一律走 `context.Context`；新局/悔棋/离开页面必须 cancel 且**迟到响应按 id 丢弃**（等价原版 `_gameSeq`）；输入锁在失败/取消/dispose 时必须解锁。
6. 对局状态 store **每局一实例**（Zustand 工厂），禁止全局单例。
7. CPU 密集计算（搜索/求解/批量解析）只在独立 goroutine 中执行，调用方经 context 取消；Worker 协议形状统一 `{id, type, payload}` / `{id, ok, result|error|progress}`。
8. API Key 只经 keyring 加密落盘（回退文件 0600）；日志、异常消息、UI 一律掩码（`****`+末4位）。
9. **引擎翻译纪律**：internal/engine 以 Electron 版 `src/packages/engine`（TS）为逐行翻译源——搜索/评估/排序细节（含 MVV-LVA 排序键内嵌，Electron 版 DR-008）决定剪枝形态，**禁止重写优化**；一切偏离先过金标准对拍。
10. **思维链强制关闭**：任何 LLM 请求不得携带开启思维链的参数；关闭参数按预设映射恒发（05 文档 §3.1），不得在 UI 或配置中提供开启开关。

## 开发规范

- 每个子任务一个 commit，格式 `feat(m1): 描述` / `test(m1): 描述` / `fix(m1): 描述` / `docs: 描述`；涉及技术决策的改动在提交信息末尾加 `Decision: DR-编号`，新决策先追加 `design_docs/decision_log.md`（Go 版从 DR-001 起重新编号，引用 Electron 版 DR 时注明 `electron-DR-xxx`）。
- **文档与代码同步**：任何行为变更先改 design_docs/ 对应章节再动代码（同一 commit 或文档先行）；每里程碑收尾更新 `docs/PROGRESS.md`；**禁止代码领先文档多个提交**。
- **测试先行**：协议面代码（提示词/PGN 导出/JSON 报告）先写**快照测试**再实现；规则/引擎/求解器先写**金标准对拍测试**（09 §2）再实现；金标准文件从 Electron 版 tools/golden 复制，改金标准必须给出原版源码依据并单独 commit。
- 里程碑验收流：自动化测试通过 → commit → **暂停等待用户手动验收** → 有 bug 则修复并回归 → 验证通过再继续下一里程碑；进度记入 docs/PROGRESS.md。
- 迁移对照：行为不确定时，先查 01 文档的 Flutter 源码锚点（文件:行号），再读原版源码确认；**禁止凭直觉发明行为**。
- 规则缺口口径：长将判负与重复局面判和**已按前置方案实现**（DR-006，见 02 §7 / 03 §6）；**长捉与自然限着（60 回合无吃子）不实现**，保持与原版一致。

## 常用命令

```bash
wails dev              # Wails 开发模式（WSLg 窗口）
npm run dev:web        # 前端浏览器模式（mock api 适配层，E2E 用）
go test ./...          # Go 全量测试（CI 加 -race）
npm run test:fe        # 前端 Vitest
gofmt -l . && go vet ./...   # 质量门 lint 部分
go run ./cmd/eval      # MatchRunner 能力评估（LLM_BASE_URL/LLM_MODEL 环境变量）
wails build            # 生产构建（产物 build/bin）
```

## 质量门（每个任务收尾前必须全绿）

`gofmt -l` 空输出 + `go vet ./...` 0 问题 + `go test ./... -race` 全量通过 + `npm run test:fe` 通过；UI 任务另加：dev 模式手测对应交互（对照 08 文档防错清单）+ 用户手动验收。
