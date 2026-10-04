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

## 已修复（保留记录）

| # | 级别 | 描述 | 修复 |
|---|---|---|---|
| F1 | P0 | api 适配器探测条件过宽：Wails 运行时先同步注入 `window.go = {}`（App 方法表异步经 SetBindings 填充），以 `window.go` 真值判定会在外部浏览器打开 devserver 时误选 wailsAdapter 并在创建期抛错白屏。 | `client.ts` 探测改为 `window.go?.app?.App !== undefined`；实测两路径：外部浏览器回落 mock 正常渲染；注入 fake 绑定后 wailsAdapter 全链路（DbLoadLatest/DbSaveGame 载荷形状）验证通过。 |
