package rules

// 金标准对拍测试（09 §2.1，Electron 版 test/rules/golden.spec.ts 的 Go 对应）：
// 1. fen.json      —— FEN 逐条往返 FromFen(f).ToFen() == f（T1.1）；
// 2. moves.json    —— AllLegalMoves 输出与期望走法集合相等，顺序无关（T1.2）；
// 3. notation.json —— 中文记法逐字一致（T1.4）。
// 数据落盘于 testdata/golden/，与 TS 版共用 Dart 提取的期望值，跨语言逐位可比。
import (
	"encoding/json"
	"os"
	"testing"
)

func loadGolden(t *testing.T, name string, v any) {
	t.Helper()
	data, err := os.ReadFile("../../testdata/golden/" + name)
	if err != nil {
		t.Fatalf("读取金标准 %s 失败: %v", name, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("解析金标准 %s 失败: %v", name, err)
	}
}

type goldenFenFile struct {
	Comment string   `json:"comment"`
	Fens    []string `json:"fens"`
}

// 金标准 FEN 往返（tools/golden/fen.json）：isValidFen 全通过 + fromFen→toFen 逐条恒等。
func TestGoldenFenRoundTrip(t *testing.T) {
	var g goldenFenFile
	loadGolden(t, "fen.json", &g)
	for _, fen := range g.Fens {
		if !IsValidFen(fen) {
			t.Errorf("IsValidFen(%q) = false, 期望 true", fen)
			continue
		}
		b, err := FromFen(fen)
		if err != nil {
			t.Errorf("FromFen(%q) 报错: %v", fen, err)
			continue
		}
		if got := b.ToFen(); got != fen {
			t.Errorf("FEN 往返不一致: got %q, want %q", got, fen)
		}
	}
}
