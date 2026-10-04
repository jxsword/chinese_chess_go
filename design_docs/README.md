# design_docs — 设计文档集导航（Go 版）

> 本目录是 **Go 版唯一事实源**。Electron 版设计文档（`/home/ssy/proj/chinese_chess_electron/design_docs/`）
> 是"继承锚点"：领域细节（协议原文、算法逐行、金标准口径）以 Electron 版为补充事实源，
> 两处冲突时以本目录为准（Go 侧差异均在各文档"与 Electron 版差异"节中显式列出）。

## 文档索引与阅读顺序

| 序 | 文档 | 承载 | 与 Electron 版关系 |
|---|---|---|---|
| 00 | 总体架构与技术选型 | 分层模型、Wails 绑定与事件协议、目录 | 重写（进程模型→单进程分层） |
| 01 | 需求规格说明书 | 功能需求 + Flutter 锚点 | 继承（范围与验收标准不变） |
| 02 | 规则内核设计 | 走法/胜负/记法 + **L3 重复裁决** | 继承 + 新增 §7 |
| 03 | 内置AI引擎设计 | 搜索/参谋/**L0+L1+L2**/goroutine | 继承 + DR-018 三层并入正文 |
| 04 | 残局求解器设计 | AND/OR 搜索/多解枚举 | 继承 + context 取消 |
| 05 | 大模型集成设计 | 提示词协议/过滤管线/参谋制/**强制关思维链** | 协议继承 + 传输层重写 |
| 06 | 语料库与棋谱解析设计 | XQF/PGN/下载安全 | 继承 + IO 边界改写 |
| 07 | 持久化与状态管理设计 | SQLite/自动保存/**fenHistory 四收口**/凭据 | Schema 继承 + 凭据重裁 |
| 08 | UI与交互设计 | 页面/动画/防错清单 | 继承（前端为移植的 React） |
| 09 | 测试方案 | 金标准对拍/mock SSE/双测试栈 | 口径继承 + 工具链改写 |
| 10 | 实施路线图 | M0~M7/**重复治理前置**/风险/DoD | 重排 |
| 11 | 开发执行手册 | 子任务模板/**M0~M7 启动提示词**/**里程碑验收流** | 命令 Go 化 + 新增验收流 |
| — | decision_log.md | DR-001~006 | 新编（引用 electron-DR） |

## 角色化阅读路径

- **架构师**：00 → 10 → decision_log → 07
- **规则/引擎实现者**：02 → 03 → 09 §2.1/2.2 →（深入时）Electron 版 02/03 与 `src/packages/{rules,engine}`
- **LLM 实现者**：05 → 00 §3 → 09 §2.3 →（协议逐字）Electron 版 05 §2 与 `src/packages/llm/prompt.ts`
- **前端/移植**：08 → 07 §6 → 00 §3 →（组件源码）Electron 版 `src/renderer/`
- **测试/CI**：09 → 11 → 10

## 术语表

| 术语 | 含义 |
|---|---|
| 厘兵 | 评估分单位，100 厘兵 ≈ 一个兵的价值 |
| 内部坐标 | col 0-8（左→右）× row 0-9（0=黑方底线/棋盘顶部，9=红方底线） |
| ICCS | 坐标记法 `b2-e2`（列 a-i + 行 0-9），LLM 交互与棋谱存储格式 |
| LLM 坐标 | 与 LLM 交互的着法文本格式 `起点-终点`（同 ICCS） |
| 参谋制 | 引擎出带评分候选/护航（否决权），LLM 在约束下决策（Hybrid） |
| strengthBlend | 棋力旋钮 0~100：候选模式名单宽度 K=3+⌊blend/20⌋；护航否决阈值=80+3.2×blend |
| 五层过滤 | LLM 回复 → SSE 重组 → 归一化 → 提取 → 白名单精确匹配 |
| MoveSource | "棋手"抽象：内置引擎与大模型是可互换实现 |
| requestId | 异步通道关联键；取消后迟到响应按 id 丢弃（等价 Flutter `_gameSeq`） |
| fenHistory | 局面 FEN 历史（长度=手数+1），重复治理 L1/L2/L3 的数据基础 |
| 金标准 | 从 Flutter 原版 dart run 提取的期望值 JSON，跨语言逐位对拍 |

## 与 Electron 版模块映射总表

| Flutter 版 | Electron 版 | Go 版 |
|---|---|---|
| board/model | packages/rules | internal/rules |
| shared/engine | packages/engine | internal/engine |
| shared/llm | packages/llm | internal/llm |
| puzzle/solver | packages/solver | internal/solver |
| puzzle/parsers | packages/parsers | internal/parsers |
| — | packages/storage-schema + main/services/db | internal/storage |
| 主进程服务 | src/main/services | internal/*/（单进程内聚）+ app.go 绑定 |
| 渲染层 | src/renderer | frontend/（移植） |
| tool/llm_match_runner | tools/eval.ts | cmd/eval |
