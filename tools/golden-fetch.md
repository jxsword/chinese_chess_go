# 金标准来源与再生成说明（09 §3）

跨语言金标准落盘于 `testdata/golden/`，**与 TS 版共用 Dart 提取的期望值**（同一 JSON，逐位可比）。
本目录文件**只读**：改动必须给出 Electron/原版源码依据并单独 commit（AGENTS.md 测试先行纪律）。

## 现有文件

| 文件 | 用途 | 对拍测试 | 条目 |
|---|---|---|---|
| `fen.json` | FEN 往返（fromFen→toFen 恒等）+ isValidFen 粗校 | `internal/rules/golden_test.go` | 19 FEN |
| `moves.json` | `AllLegalMoves(side)` 集合对拍 + 终局 status | `internal/rules/golden_test.go` | 16 案例 |
| `notation.json` | 中文纵线记法逐字对拍 | `internal/rules/golden_test.go` | 21 案例 |
| `engine.json` | findBestMoveEx 逐层 best/bestCp/topK | `internal/engine/golden_test.go`（M3 引入） | — |

## 来源与再生成

- 提取脚本（Dart 金标准 → JSON）：Electron 版仓库 `/home/ssy/proj/chinese_chess_electron/tools/`
  （提取自 Flutter 原版 `test/` 的 fen_test/board_test 局面与 M1 手工核验构造局面）。
- 再生成方式：在 Electron 版仓库重跑提取脚本，产出后**逐字节复制**到本目录
  （当前哈希基线见 git 历史 `test(m1)` 基建 commit）。
- 同分着法间排序不参与对拍（Dart sort 不稳定，09 §2.1）。
