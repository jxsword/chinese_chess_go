package rules

// 中国象棋规则引擎（对应 board.dart / Electron 版 board.ts）。
//
// - 棋盘以 10 行 × 9 列矩阵存储，row 0 为黑方底线（FEN 第一行），row 9 为红方底线。
// - 两层走法：PseudoMovesFor（伪合法，供搜索与 IsCheck）→ LegalMovesFor
//   （自将过滤，供 UI/清单，T1.3 落地）。
// - 纯 Go，不依赖任何环境 API，可独立单测。

// 是否在九宫格内（board.dart:54-57）：col 3-5；红 row 7-9 / 黑 row 0-2。
func inPalace(col, row int, side Side) bool {
	if col < 3 || col > 5 {
		return false
	}
	if side == Red {
		return row >= 7 && row <= 9
	}
	return row >= 0 && row <= 2
}

// inOwnHalf 是否在自己半场（未过河，board.dart:60-62）：红 row≥5 / 黑 row≤4。
func inOwnHalf(row int, side Side) bool {
	if side == Red {
		return row >= 5
	}
	return row <= 4
}

// inOpponentHalf 是否已过河到对方半场（board.dart:65）。
func inOpponentHalf(row int, side Side) bool { return !inOwnHalf(row, side) }

// Board 棋盘（board.dart / Electron 版 board.ts Board）。
//
// DR-006：Zobrist 键不在此处（引擎内部单源，见 03 §4）——规则层保持无哈希，
// L3 用 FEN 字符串比较（02 §7），与 Electron 版口径一致。
type Board struct {
	grid    BoardGrid
	redTurn bool
}

// FromFen 从 FEN 字符串构造棋盘（board.dart:16-20）。无效 FEN 返回 error
// （引擎侧 L2 历史表对无效 FEN 静默跳过，02 文档 §1）。
func FromFen(fen string) (*Board, error) {
	grid, err := ParseBoardFen(fen)
	if err != nil {
		return nil, err
	}
	return &Board{grid: grid, redTurn: ParseTurnFen(fen)}, nil
}

// Initial 标准初始局面（board.dart:23）。FENInitial 为编译期常量、必然合法。
func Initial() *Board {
	b, err := FromFen(FENInitial)
	if err != nil {
		panic("rules: FENInitial 常量非法（不可能发生）: " + err.Error())
	}
	return b
}

// IsRedTurn 当前是否轮到红方走（board.dart:29）。
func (b *Board) IsRedTurn() bool { return b.redTurn }

// Turn 当前轮走方（board.dart:32）。
func (b *Board) Turn() Side {
	if b.redTurn {
		return Red
	}
	return Black
}

// PieceAt 取某格棋子（board.dart:35）。
func (b *Board) PieceAt(col, row int) *Piece { return b.grid[row][col] }

// PieceAtP 取某格棋子（Position 版，board.dart:38）。
func (b *Board) PieceAtP(p Position) *Piece { return b.grid[p.Row][p.Col] }

// ToFen 序列化为 FEN（board.dart:41）。
func (b *Board) ToFen() string { return BuildFen(b.grid, b.redTurn) }

// Copy 拷贝当前棋盘（深拷贝，Worker 快照传参用；board.dart:44-47）。
func (b *Board) Copy() *Board {
	grid := make(BoardGrid, 10)
	for r := 0; r < 10; r++ {
		grid[r] = make([]*Piece, 9)
		copy(grid[r], b.grid[r])
	}
	return &Board{grid: grid, redTurn: b.redTurn}
}
