# ChineseChessUltra Go

中国象棋桌面应用——Flutter 版（ChineseChessUltra，五期全部交付）的 **Go 语言全栈重写**。
前一版 Electron 实现见 `jxsword/chinese_chess_electron`（已交付 v1.0），其设计资产与金标准直接复用。

> 当前状态：**v1.0 代码完成，验收中**——M0~M6 已全部通过用户手测验收，M7（评估 CLI + 三平台打包 + E2E）代码完成、质量门全绿，剩余真实端点评估与首个 tag CI 产包验证。全套设计文档见 [design_docs/](design_docs/)，逐里程碑进度与验收记录见 [docs/PROGRESS.md](docs/PROGRESS.md)。

## 功能总览（对齐 Flutter/Electron 版全部功能）

- **对弈模式 ×5**：双人对弈（计时器/悔棋/保存分享）、人机对战（内置 AI 五档难度、可执黑）、人机对战（大模型）、大模型对战（红黑双 LLM 自动对局）、残局闯关（从语料进入对战）
- **规则内核**：7 棋种完整规则（马腿/象眼/炮架/将帅照面/过河兵），将死/困毙判定，中文纵线记法；**重复局面裁决**（长将判负/判和/强制和——Go 版在前期里程碑即实现，见下"两大设计要点"）
- **内置 AI 引擎**：Negamax + Alpha-Beta + 迭代加深 + 静态搜索（纯 Go，独立 goroutine），Zobrist（uint64）增量局面键内置于棋盘核心
- **大模型对弈**：OpenAI 兼容端点（智谱 GLM/DeepSeek/Kimi/OpenRouter/OpenAI），SSE 流式 + 空闲超时，回复五层过滤白名单校验；**引擎参谋制**（候选/护航两模式 + 棋力旋钮）；**思维链强制关闭**；API Key 仅 keyring 落盘、UI/日志全程掩码
- **残局求解**：迭代加深 AND/OR 杀棋搜索（证明式，可枚举多解/证明无解）；残局工作室（摆盘/FEN 导入/LLM 求解辅助/视觉大模型识图 + 人工校正/求解入库）
- **棋谱生态**：XQF（含解密）/PGN（含中文记谱解析）导入，14 万局级语料库浏览与下载，棋谱库（重放/导出 PGN/进入对战联动）
- **能力评估**：MatchRunner CLI（`go run ./cmd/eval`）——4 profile（基线/P0 提示词/候选/护航）红黑换边对抗内置 AI，JSON 报告落盘
- **打包发布**：Wails build + GitHub Actions 三平台矩阵，tag `v*` 触发构建并上传二进制安装包到 Release

## 技术栈

| 层 | 选型 |
|---|---|
| 桌面框架 | Wails v2（Go 后端 + 原生 WebView：Win=WebView2 / Linux=WebKitGTK / mac=WKWebView） |
| 前端 | React 18 + TypeScript(strict) + Vite + Zustand（自 Electron 版移植，api 适配层换 Wails 绑定） |
| 语言 | Go 1.26+（go.mod 口径；领域层 `internal/` 纯 Go，零 UI/运行时依赖） |
| 数据库 | modernc.org/sqlite（纯 Go 零 CGO） |
| LLM | net/http SSE；思维链强制关闭（按预设映射：Qwen→enable_thinking:false / GLM-4.5+→thinking:disabled） |
| 凭据 | OS keyring + 明文 0600 回退 |
| 测试 | go test + 跨语言金标准对拍 + Vitest（前端）+ Playwright（浏览器 mock E2E） |
| 打包 | wails build + GitHub Actions 三平台矩阵 + Release 上传 |

选型决策记录：[design_docs/decision_log.md](design_docs/decision_log.md)（DR-001~011，含弃用选项与理由）。

## 两大设计要点（相对 Electron 版的针对性变化）

1. **重复变招治理前置**（Electron 版为 M6 后修复，改动波及面大）：Zobrist（uint64，固定种子保证可复现快照）内置于棋盘核心；L1 搜索内重复检测 + L2 根节点历史回避随引擎首版（M3）落地，L3 规则裁决纯函数随规则首版（M1）落地——不等最后修复（DR-006；方案参数逐值搬移自 Electron 版 `built-in-ai-move-repetition-optimization-design-final.md`）。
2. **思维链强制关闭**：下棋用不到思维链且影响交互响应速度——所有 LLM 请求在请求构造层恒发关闭参数（按端点预设映射），UI/配置不提供开关（DR-005）。

## 仓库结构

```text
chinese_chess_go/
├── AGENTS.md            # AI 编码代理常驻指令（铁律/规范/命令）
├── design_docs/         # 设计文档集（Go 版唯一事实源）
├── docs/                # 开发进度与已知问题（PROGRESS.md / KNOWN_ISSUES.md）
├── frontend/            # 移植的 React 前端（src/api 层适配 Wails 绑定；e2e/ 为 Playwright 冒烟）
├── internal/            # 纯 Go 领域层：rules/engine/solver/llm/parsers/storage
├── cmd/eval/            # MatchRunner 能力评估 CLI
├── testdata/golden/     # 跨语言金标准（复制自 Electron 版 tools/golden）
├── build/               # Wails 打包资源（图标/平台配置；产物在 build/bin）
└── .github/workflows/   # ci.yml（三平台质量门）/ release.yml（tag 触发构建上传）
```

## 设计文档

| 序 | 文档 | 内容 |
|---|---|---|
| 00 | [总体架构与技术选型](design_docs/00-总体架构与技术选型.md) | 进程/分层模型、Wails 绑定与事件协议、目录结构 |
| 01 | [需求规格说明书](design_docs/01-需求规格说明书.md) | 功能需求 E-F01~F42（验收标准 + Flutter 原版锚点） |
| 02 | [规则内核设计](design_docs/02-规则内核设计.md) | 走法生成/胜负判定/中文记法/**L3 重复裁决** |
| 03 | [内置AI引擎设计](design_docs/03-内置AI引擎设计.md) | 搜索算法/参谋接口/**L0 Zobrist + L1/L2 重复治理**/goroutine 模型 |
| 04 | [残局求解器设计](design_docs/04-残局求解器设计.md) | AND/OR 搜索/多解枚举/context 取消 |
| 05 | [大模型集成设计](design_docs/05-大模型集成设计.md) | 提示词协议/五层过滤/参谋制/**思维链强制关闭** |
| 06 | [语料库与棋谱解析设计](design_docs/06-语料库与棋谱解析设计.md) | XQF 解密/PGN 解析/下载安全 |
| 07 | [持久化与状态管理设计](design_docs/07-持久化与状态管理设计.md) | SQLite/自动保存/fenHistory 四收口/凭据 |
| 08 | [UI与交互设计](design_docs/08-UI与交互设计.md) | 页面/动画/交互防错清单/移植说明 |
| 09 | [测试方案](design_docs/09-测试方案.md) | 金标准对拍/mock SSE/双测试栈/质量门 |
| 10 | [实施路线图](design_docs/10-实施路线图.md) | M0~M7 里程碑/风险表/DoD |
| 11 | [开发执行手册](design_docs/11-开发执行手册.md) | AI 提示词/子任务拆解/**里程碑验收流** |
| — | [决策记录](design_docs/decision_log.md) | DR-001~011（含弃用选项与理由） |

## 开发（WSL ubuntu2604）

```bash
# 依赖：Go 1.26+、Node 20.19+/22+（vite 7 要求）、wails v2 CLI、WebKitGTK
#       Ubuntu 24.04+ 装 libgtk-3-dev libwebkit2gtk-4.1-dev（webkit2gtk-4.0 已从源移除）
npm install            # frontend 依赖（wails dev 自动调用）
wails dev -tags webkit2_41   # 桌面开发模式（WSLg；Ubuntu 24.04+ 需该标签，Win/mac 免标签）
npm run dev:web        # 前端浏览器模式（mock api 适配层）
go test ./...          # Go 全量测试
npm run test:fe        # 前端测试
npm run test:e2e       # Playwright 浏览器 mock E2E 冒烟（frontend/ 内）
go run ./cmd/eval -- --suite   # 能力评估 CLI（LLM_BASE_URL/LLM_MODEL/LLM_API_KEY 环境变量；
                               # stderr 逐手进度，JSON 报告 stdout + tmp/ 落盘）
```

开发流程与 AI 协作方式见 [design_docs/11-开发执行手册.md](design_docs/11-开发执行手册.md)。

## 打包与发布

推送 `v*` tag 触发 [.github/workflows/release.yml](.github/workflows/release.yml)：三平台 runner 各跑质量门 + `wails build` 打包，产物汇总上传 GitHub Draft Release——Linux（deb / AppImage / 裸二进制 tar.gz 兜底）、Windows（NSIS Setup.exe）、macOS（universal dmg）；构建缓存（go-build / 模块 / wails·nfpm 工具链 / NSIS）按 lockfile 与版本失效。

## 路线图

| 里程碑 | 内容 | 状态 |
|---|---|---|
| M0 | 工程骨架（go.mod + Wails + CI 三平台） | ✅ 用户验收通过 |
| M1 | 规则内核 + L3 重复裁决（金标准对拍） | ✅ 用户验收通过 |
| M2 | 对战页 + 存储（fenHistory 四收口 + UI 裁决接线） | ✅ 用户验收通过 |
| M3 | 引擎 + L0/L1/L2 重复治理（金标准对拍） | ✅ 用户验收通过 |
| M4 | LLM 全链路（思维链强制关闭） | ✅ 用户验收通过 |
| M5 | 语料 + 棋谱 | ✅ 用户验收通过 |
| M6 | 工作室 + 求解器 + 识图 | ✅ 用户验收通过 |
| M7 | 评估（MatchRunner）+ 三平台打包发布 | 🔵 代码完成、质量门全绿，验收中（余：真实端点 `--suite` 评估 + 首个 `v*` tag CI 三平台产包） |

## 许可

待定（原版基于 GPL-2.0/GPL-3.0 生态素材、非商业定位）。
