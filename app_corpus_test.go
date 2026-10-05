package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jxsword/chinese_chess_go/internal/storage"
)

// M5 语料绑定面验证（09 §2.4）：Corpus* 绑定的目录解析与 ParserParseBatch 的
// base64 字节契约（Wails invoke JSON 序列化，前端 api/binary.ts 对端）。

func TestCorpusScanResolvesRootPriority(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	docs := t.TempDir()
	// 显式 root 直用。
	scan, err := app.CorpusScan(docs)
	if err != nil {
		t.Fatal(err)
	}
	if scan.Root != docs || !scan.Exists {
		t.Fatalf("scan = %+v", scan)
	}
}

func TestCorpusScanUsesStoredUserPath(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	docs := t.TempDir()
	userRoot := t.TempDir()
	mkdirCorpusCategory(t, userRoot, "XQF测试谱", "残局", "a.xqf")
	if err := app.StoreSet(corpusUserPathKey, userRoot); err != nil {
		t.Fatal(err)
	}
	// 临时切 documents 基路径（documentsDir 依赖环境，测试注入校验解析结果）。
	app.corpusDocumentsOverride = docs
	scan, err := app.CorpusScan("")
	if err != nil {
		t.Fatal(err)
	}
	if scan.Root != userRoot || len(scan.Categories) != 1 || scan.Categories[0].Name != "XQF测试谱" {
		t.Fatalf("scan = %+v", scan)
	}
}

func TestCorpusScanEmptyCorpusFallsBackToDocumentsDefault(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	docs := t.TempDir()
	app.corpusDocumentsOverride = docs
	scan, err := app.CorpusScan("")
	if err != nil {
		t.Fatal(err)
	}
	if scan.Root != filepath.Join(docs, "ChineseChessUltra", "corpus") {
		t.Fatalf("root = %q", scan.Root)
	}
	if scan.Exists {
		t.Fatal("默认目录尚未创建应 exists=false")
	}
}

func TestCorpusListEntriesAndReadFilesBase64(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	root := t.TempDir()
	filePath := mkdirCorpusCategory(t, root, "XQF测试谱", "残局/适情雅趣", "第一局.xqf")
	_ = root

	entries, err := app.CorpusListEntries(filepath.Join(root, "XQF测试谱"), "XQF测试谱")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].DisplayName != "第一局" || entries[0].Source != "残局/适情雅趣" {
		t.Fatalf("entries = %+v", entries)
	}

	// CorpusReadFiles → JSON 线上 bytes 为 base64（Wails 通道契约）。
	files, err := app.CorpusReadFiles([]string{filePath, strings.Replace(filePath, ".xqf", ".txt", 1)})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != filePath {
		t.Fatalf("files = %+v", files)
	}
	wire, err := json.Marshal(files)
	if err != nil {
		t.Fatal(err)
	}
	var wireFiles []struct {
		Path  string `json:"path"`
		Bytes string `json:"bytes"`
	}
	if err := json.Unmarshal(wire, &wireFiles); err != nil {
		t.Fatal(err)
	}
	if len(wireFiles) != 1 || wireFiles[0].Bytes != base64.StdEncoding.EncodeToString([]byte("XQ")) {
		t.Fatalf("wire = %+v", wireFiles)
	}
}

func TestParserParseBatchBindingBase64AndProgress(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	// 一个合法 PGN 文件（base64 承载）+ 一个损坏文件位。
	pgn := base64.StdEncoding.EncodeToString([]byte("1. 炮二平五 马8进7\n"))
	files := []map[string]any{
		{"name": "a.pgn", "source": "全局", "bytes": pgn},
		{"name": "bad.xqf", "source": "残局", "bytes": base64.StdEncoding.EncodeToString([]byte("garbage"))},
	}
	var progressEvents []map[string]any
	app.parserProgressOverride = func(done, total int) {
		progressEvents = append(progressEvents, map[string]any{"done": done, "total": total})
	}
	result, err := app.ParserParseBatch("req-1", files)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Puzzles []struct {
			ID            string   `json:"id"`
			SolutionMoves []string `json:"solutionMoves"`
			Format        string   `json:"format"`
			Title         *string  `json:"title"`
		} `json:"puzzles"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Puzzles) != 2 {
		t.Fatalf("puzzles = %d", len(out.Puzzles))
	}
	if out.Puzzles[0].Format != "pgn" || len(out.Puzzles[0].SolutionMoves) != 2 {
		t.Fatalf("puzzles[0] = %+v", out.Puzzles[0])
	}
	if out.Puzzles[1].ID != "" {
		t.Fatal("损坏文件位应为 null")
	}
	// 进度 n/m 经发送器回报（生产为 EventsEmit，测试注入收集）。
	if len(progressEvents) != 2 || progressEvents[1]["done"] != 2 || progressEvents[1]["total"] != 2 {
		t.Fatalf("progress = %+v", progressEvents)
	}
}

func TestParserParseBatchRejectsNonBase64Bytes(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	// 契约外形状（Uint8Array 的 JSON 索引对象）应拒绝而非静默误解。
	files := []map[string]any{{"name": "a.pgn", "source": "s", "bytes": map[string]any{"0": 104}}}
	if _, err := app.ParserParseBatch("req-2", files); err == nil {
		t.Fatal("非 base64 bytes 应报错")
	}
}

func TestCorpusPgnIndexAndReadPgnGameBindings(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	root := t.TempDir()
	pgn := filepath.Join(root, "multi.pgn")
	if err := os.WriteFile(pgn, []byte("[Event \"甲局\"]\n\n1. 炮二平五 马8进7\n\n[Event \"乙局\"]\n\n1. 兵七进一 卒7进1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	index, err := app.CorpusPgnIndex(pgn, 0) // 0 = 不限（前端 maxGames ?? 0 口径）
	if err != nil {
		t.Fatal(err)
	}
	if len(index) != 2 || index[1].Event == nil || *index[1].Event != "乙局" {
		t.Fatalf("index = %+v", index)
	}
	entry, err := json.Marshal(index[1])
	if err != nil {
		t.Fatal(err)
	}
	var entryMap map[string]any
	if err := json.Unmarshal(entry, &entryMap); err != nil {
		t.Fatal(err)
	}
	text, err := app.CorpusReadPgnGame(pgn, entryMap)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "兵七进一") {
		t.Fatalf("text = %q", text)
	}
}

// mkdirCorpusCategory 构造 <root>/<分类>/<sub>/ 文件，返回完整文件路径；
// content 固定 "XQ"（合法魔数占位，扫描/读取用例不解析）。
func mkdirCorpusCategory(t *testing.T, root, category, sub, name string) string {
	t.Helper()
	dir := filepath.Join(root, category, sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("XQ"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// storage 引用保持（newTestApp 的 keyring 类型来自 storage）。
var _ storage.Keyring = (*fakeKeyring)(nil)
