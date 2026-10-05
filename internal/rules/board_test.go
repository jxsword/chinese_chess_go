package rules

// 规则引擎测试（对齐原版 board_test.dart 全部 17 用例 + 02 §2.3/§3 合法性契约；
// Electron 版 test/rules/board.spec.ts 的 Go 对应）。
// 走法生成用例走 PseudoMovesFor（这些场面均无自将干扰，pseudo == legal）；
// 自将过滤 / 照面 / 将死 / 困毙走 LegalMovesFor 与 IsCheck 系列（T1.3 增补）。
import "testing"

// pieceSpec [col, row, FEN字符] 三元组，如 {4, 5, 'R'} = 红车在 (4,5)。
type pieceSpec struct {
	col, row int
	ch       byte
}

// boardWith 在空棋盘上指定棋子构造 Board（对齐 Dart _boardWith / TS boardWith）。
func boardWith(t *testing.T, pieces []pieceSpec, redTurn bool) *Board {
	t.Helper()
	grid := make(BoardGrid, 10)
	for r := range grid {
		grid[r] = make([]*Piece, 9)
	}
	for _, s := range pieces {
		p := PieceFromFenChar(s.ch)
		if p == nil {
			t.Fatalf("非法棋子字符: %c", s.ch)
		}
		grid[s.row][s.col] = p
	}
	return &Board{grid: grid, redTurn: redTurn}
}

// pseudoTargets 取 (col,row) 格棋子的伪合法走法目标集合。
func pseudoTargets(b *Board, col, row int) []Move {
	return b.PseudoMovesFor(Pos(col, row))
}

// hasTarget 走法集合中是否存在 to == (col,row) 的走法。
func hasTarget(moves []Move, col, row int) bool {
	for _, m := range moves {
		if m.To.Col == col && m.To.Row == row {
			return true
		}
	}
	return false
}

func TestBoardInitialPieceCount(t *testing.T) {
	board := Initial()
	red, black := 0, 0
	for r := 0; r < 10; r++ {
		for c := 0; c < 9; c++ {
			p := board.PieceAt(c, r)
			if p == nil {
				continue
			}
			if p.Side == Red {
				red++
			} else {
				black++
			}
		}
	}
	if red != 16 || black != 16 {
		t.Errorf("初始局面红 %d 黑 %d, 期望各 16", red, black)
	}
}

func TestBoardInitialRedTurn(t *testing.T) {
	if !Initial().IsRedTurn() {
		t.Errorf("初始局面红方先行, IsRedTurn = false")
	}
}

func TestRookAllOpenLines(t *testing.T) {
	// 红车放在 (4,5)，红将放在九宫 (3,9)，无任何阻挡。
	board := boardWith(t, []pieceSpec{{4, 5, 'R'}, {3, 9, 'K'}}, true)
	moves := pseudoTargets(board, 4, 5)
	// 同列 9 格（不含自身），同行 8 格（不含自身），共 17。
	if len(moves) != 17 {
		t.Errorf("车空棋盘走法数 = %d, 期望 17", len(moves))
	}
}

func TestRookCannotJumpOver(t *testing.T) {
	board := boardWith(t, []pieceSpec{{4, 5, 'R'}, {4, 3, 'p'}, {4, 9, 'K'}}, true)
	moves := pseudoTargets(board, 4, 5)
	// 上方遇到黑兵在 (4,3)，可以吃，但不能跳到 (4,2)/(4,1)/(4,0)。
	if !hasTarget(moves, 4, 4) {
		t.Errorf("缺少走法 (4,4)")
	}
	if !hasTarget(moves, 4, 3) {
		t.Errorf("缺少吃兵走法 (4,3)")
	}
	if hasTarget(moves, 4, 2) || hasTarget(moves, 4, 1) || hasTarget(moves, 4, 0) {
		t.Errorf("车不应越过 (4,3) 的黑兵")
	}
}

func TestKnightEightDirections(t *testing.T) {
	board := boardWith(t, []pieceSpec{{4, 5, 'N'}, {4, 0, 'K'}}, true)
	moves := pseudoTargets(board, 4, 5)
	expected := []Position{
		Pos(5, 7), Pos(3, 7), Pos(6, 6), Pos(2, 6),
		Pos(6, 4), Pos(2, 4), Pos(5, 3), Pos(3, 3),
	}
	if len(moves) != len(expected) {
		t.Errorf("马走法数 = %d, 期望 %d", len(moves), len(expected))
	}
	for _, e := range expected {
		if !hasTarget(moves, e.Col, e.Row) {
			t.Errorf("缺少马走法 (%d,%d)", e.Col, e.Row)
		}
	}
}

func TestKnightLegBlocked(t *testing.T) {
	board := boardWith(t, []pieceSpec{{4, 5, 'N'}, {4, 4, 'p'}, {4, 9, 'K'}}, true)
	moves := pseudoTargets(board, 4, 5)
	// 马腿 (4,4) 被堵，向下不能走 (3,3) / (5,3)。
	if hasTarget(moves, 3, 3) || hasTarget(moves, 5, 3) {
		t.Errorf("马腿被堵时不应有 (3,3)/(5,3) 走法")
	}
	// 其余 6 个方向不受影响。
	if len(moves) != 6 {
		t.Errorf("马走法数 = %d, 期望 6", len(moves))
	}
}

func TestCannonScreenCapture(t *testing.T) {
	board := boardWith(t, []pieceSpec{{4, 5, 'C'}, {4, 3, 'p'}, {4, 1, 'a'}, {4, 9, 'K'}}, true)
	moves := pseudoTargets(board, 4, 5)
	// 上方可走到空格 (4,4)；(4,3) 是炮架不能吃；隔架可吃 (4,1)；(4,0) 越过目标不可达。
	if !hasTarget(moves, 4, 4) {
		t.Errorf("缺少空格走法 (4,4)")
	}
	if hasTarget(moves, 4, 3) {
		t.Errorf("(4,3) 是炮架，不可吃")
	}
	if !hasTarget(moves, 4, 1) {
		t.Errorf("缺少隔架吃子 (4,1)")
	}
	if hasTarget(moves, 4, 0) {
		t.Errorf("(4,0) 越过目标不可达")
	}
}

func TestMinisterElephantMove(t *testing.T) {
	board := boardWith(t, []pieceSpec{{4, 9, 'B'}, {4, 0, 'K'}}, true)
	moves := pseudoTargets(board, 4, 9)
	// 红相在 (4,9)，可走 (2,7)、(6,7)，不能过河（row < 5）。
	if !hasTarget(moves, 2, 7) || !hasTarget(moves, 6, 7) {
		t.Errorf("缺少象走田 (2,7)/(6,7)")
	}
	for _, m := range moves {
		if m.To.Row < 5 {
			t.Errorf("象过河走法 (%d,%d)", m.To.Col, m.To.Row)
		}
	}
}

func TestAdvisorPalaceOnly(t *testing.T) {
	board := boardWith(t, []pieceSpec{{3, 9, 'A'}, {4, 9, 'K'}}, true)
	moves := pseudoTargets(board, 3, 9)
	// 红士在 (3,9)，可斜走到 (4,8)。
	if !hasTarget(moves, 4, 8) {
		t.Errorf("缺少士斜走 (4,8)")
	}
	// 不能平走到 (3,8) 或 (2,9)（不属于斜走）。
	if hasTarget(moves, 3, 8) || hasTarget(moves, 2, 9) {
		t.Errorf("士不应有直走/横走")
	}
	if len(moves) != 1 {
		t.Errorf("士走法数 = %d, 期望 1", len(moves))
	}
}

func TestKingPalaceOnly(t *testing.T) {
	board := boardWith(t, []pieceSpec{{4, 9, 'K'}}, true)
	moves := pseudoTargets(board, 4, 9)
	if !hasTarget(moves, 4, 8) || !hasTarget(moves, 3, 9) || !hasTarget(moves, 5, 9) {
		t.Errorf("缺少九宫直走一格走法")
	}
	// 不能斜走。
	if hasTarget(moves, 3, 8) {
		t.Errorf("将不应斜走 (3,8)")
	}
	// 不能走出九宫。
	if hasTarget(moves, 4, 6) {
		t.Errorf("将不应走出九宫 (4,6)")
	}
	if len(moves) != 3 {
		t.Errorf("将走法数 = %d, 期望 3", len(moves))
	}
}

func TestPawnBeforeRiver(t *testing.T) {
	board := boardWith(t, []pieceSpec{{4, 6, 'P'}, {4, 9, 'K'}}, true)
	moves := pseudoTargets(board, 4, 6)
	if !hasTarget(moves, 4, 5) {
		t.Errorf("缺少兵前进 (4,5)")
	}
	if hasTarget(moves, 3, 6) || hasTarget(moves, 5, 6) {
		t.Errorf("未过河兵不应横走")
	}
	if len(moves) != 1 {
		t.Errorf("兵走法数 = %d, 期望 1", len(moves))
	}
}

func TestPawnAfterRiver(t *testing.T) {
	board := boardWith(t, []pieceSpec{{4, 4, 'P'}, {4, 9, 'K'}}, true)
	moves := pseudoTargets(board, 4, 4)
	if !hasTarget(moves, 4, 3) || !hasTarget(moves, 3, 4) || !hasTarget(moves, 5, 4) {
		t.Errorf("过河兵缺少前进/横走")
	}
	if hasTarget(moves, 4, 5) {
		t.Errorf("兵不能后退 (4,5)")
	}
	if len(moves) != 3 {
		t.Errorf("兵走法数 = %d, 期望 3", len(moves))
	}
}

func TestApplyMoveUndoMoveInverse(t *testing.T) {
	fen0 := Initial().ToFen()
	board, err := FromFen(fen0)
	if err != nil {
		t.Fatalf("FromFen 报错: %v", err)
	}
	move := Move{From: Pos(1, 7), To: Pos(2, 7)}
	snapshot := board.ApplyMove(move)
	if board.ToFen() == fen0 {
		t.Errorf("走子后 FEN 不应等于原 FEN")
	}
	if board.IsRedTurn() {
		t.Errorf("走子后应轮到黑方")
	}
	board.UndoMove(snapshot)
	if board.ToFen() != fen0 {
		t.Errorf("悔棋后 FEN = %q, 期望 %q", board.ToFen(), fen0)
	}
	if !board.IsRedTurn() {
		t.Errorf("悔棋后应轮回红方")
	}
}
