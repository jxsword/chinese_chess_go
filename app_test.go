package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jxsword/chinese_chess_go/internal/storage"
)

// T2.4 绑定面验证（09 §2.4）：wailsAdapter 载荷形状（JSON 语义）与 Go 绑定往返，
// 掩码不泄 Key（铁律 #8），存储不可用降级（07 §1 懒打开语义）。

// fakeKeyring 伪 keyring（broken=模拟系统安全存储不可用）。
type fakeKeyring struct {
	broken bool
	store  map[string]string
}

func newFakeKeyring(broken bool) *fakeKeyring {
	return &fakeKeyring{broken: broken, store: map[string]string{}}
}

func (f *fakeKeyring) Get(service, user string) (string, error) {
	if f.broken {
		return "", errors.New("keyring unavailable")
	}
	v, ok := f.store[service+"/"+user]
	if !ok {
		return "", errors.New("secret not found")
	}
	return v, nil
}

func (f *fakeKeyring) Set(service, user, secret string) error {
	if f.broken {
		return errors.New("keyring unavailable")
	}
	f.store[service+"/"+user] = secret
	return nil
}

func (f *fakeKeyring) Delete(service, user string) error {
	if f.broken {
		return errors.New("keyring unavailable")
	}
	delete(f.store, service+"/"+user)
	return nil
}

func newTestApp(t *testing.T, kr storage.Keyring) *App {
	t.Helper()
	dir := t.TempDir()
	app := NewApp()
	app.initServices(dir, filepath.Join(dir, "test.sqlite"), kr)
	return app
}

// 双人页存档恢复链路：saveGame（适配器载荷）→ loadLatest → restore 载荷形状。
func TestDbSaveGameLoadLatestRoundTrip(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	// wailsAdapter: app.DbSaveGame({mode, fen, moves}) —— JSON 载荷语义
	payload := `{"mode":"humanVsHuman","fen":"rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1","moves":[[1,7,4,7],[1,0,4,0]]}`
	var req SaveGameRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := app.DbSaveGame(req); err != nil {
		t.Fatalf("DbSaveGame: %v", err)
	}
	saved, err := app.DbLoadLatest("humanVsHuman")
	if err != nil {
		t.Fatalf("DbLoadLatest: %v", err)
	}
	if saved == nil {
		t.Fatal("DbLoadLatest = nil, want row")
	}
	// 契约形状：{id, mode, fen, moves, createdAt, updatedAt}
	out, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	var shape map[string]any
	if err := json.Unmarshal(out, &shape); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "mode", "fen", "moves", "createdAt", "updatedAt"} {
		if _, ok := shape[key]; !ok {
			t.Fatalf("SavedGame 缺少键 %q: %s", key, out)
		}
	}
	if shape["mode"] != "humanVsHuman" {
		t.Fatalf("mode = %v", shape["mode"])
	}
	moves, ok := shape["moves"].([]any)
	if !ok || len(moves) != 2 {
		t.Fatalf("moves = %v", shape["moves"])
	}
}

func TestDbLoadLatestEmptyReturnsNull(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	saved, err := app.DbLoadLatest("humanVsAi")
	if err != nil || saved != nil {
		t.Fatalf("空库应返回 (nil, nil)，got (%v, %v)", saved, err)
	}
}

// 恢复死局清理（07 §2）：restore 终局后 deleteForMode 的绑定面可用。
func TestDbDeleteForMode(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	var req SaveGameRequest
	if err := json.Unmarshal([]byte(`{"mode":"humanVsHuman","fen":"fen","moves":[]}`), &req); err != nil {
		t.Fatal(err)
	}
	if err := app.DbSaveGame(req); err != nil {
		t.Fatal(err)
	}
	if err := app.DbDeleteForMode("humanVsHuman"); err != nil {
		t.Fatalf("DbDeleteForMode: %v", err)
	}
	saved, err := app.DbLoadLatest("humanVsHuman")
	if err != nil || saved != nil {
		t.Fatalf("删除后应 (nil, nil)，got (%v, %v)", saved, err)
	}
}

func TestDbRecordsSaveGetListDelete(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	// wailsAdapter: app.DbRecordsSave(record) —— Omit<GameRecord,'id'> 形状
	recordJSON := `{
		"title": "测试棋谱", "mode": "humanVsHuman",
		"initialFen": "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1",
		"moves": [{"f":[7,7],"t":[4,7],"p":"C","x":null},{"f":[7,0],"t":[6,2],"p":"n","x":"C"}],
		"result": null, "solveStatus": "none", "solutions": [], "llmNote": null, "note": null,
		"createdAt": 1791321600000
	}`
	var record storage.GameRecord
	if err := json.Unmarshal([]byte(recordJSON), &record); err != nil {
		t.Fatalf("unmarshal record: %v", err)
	}
	id, err := app.DbRecordsSave(record)
	if err != nil {
		t.Fatalf("DbRecordsSave: %v", err)
	}
	if id <= 0 {
		t.Fatalf("id = %d", id)
	}
	got, err := app.DbRecordsGet(id)
	if err != nil || got == nil {
		t.Fatalf("DbRecordsGet: %v, %v", got, err)
	}
	if got.Title != "测试棋谱" || len(got.Moves) != 2 {
		t.Fatalf("got = %+v", got)
	}
	if got.Moves[0].P != "C" || got.Moves[0].X != nil || got.Moves[1].X == nil || *got.Moves[1].X != "C" {
		t.Fatalf("moves 往返失真: %+v", got.Moves)
	}
	summaries, err := app.DbRecordsList()
	if err != nil || len(summaries) != 1 {
		t.Fatalf("DbRecordsList: %v, %v", summaries, err)
	}
	if err := app.DbRecordsDelete(id); err != nil {
		t.Fatal(err)
	}
	got, err = app.DbRecordsGet(id)
	if err != nil || got != nil {
		t.Fatalf("删除后应 (nil, nil)，got (%v, %v)", got, err)
	}
}

func TestStoreSetGetAndAutoSaveDefault(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	// global_auto_save 缺省 true（07 §3）
	v, err := app.StoreGet("global_auto_save")
	if err != nil || v != true {
		t.Fatalf("StoreGet(global_auto_save) = (%v, %v), want true", v, err)
	}
	// set/get 往返（GlobalSettings.setAutoSave 路径）
	if err := app.StoreSet("global_auto_save", false); err != nil {
		t.Fatalf("StoreSet: %v", err)
	}
	v, err = app.StoreGet("global_auto_save")
	if err != nil || v != false {
		t.Fatalf("StoreSet 后 = (%v, %v), want false", v, err)
	}
	// 未设置键返回 null（契约面 nil ↔ JS null）
	v, err = app.StoreGet("corpus.userPath")
	if err != nil || v != nil {
		t.Fatalf("StoreGet(corpus.userPath) = (%v, %v), want null", v, err)
	}
}

func TestStoreClampOnReopen(t *testing.T) {
	dir := t.TempDir()
	app := NewApp()
	app.initServices(dir, filepath.Join(dir, "test.sqlite"), newFakeKeyring(false))
	// 渲染层保存前已 clamp；此处直接写越界值模拟手改文件后重开的 load 兜底
	if err := app.StoreSet("llm_settings_timeoutSeconds", 0); err != nil {
		t.Fatal(err)
	}
	app2 := NewApp()
	app2.initServices(dir, filepath.Join(dir, "test.sqlite"), newFakeKeyring(false))
	v, err := app2.StoreGet("llm_settings_timeoutSeconds")
	if err != nil || v != 5 {
		t.Fatalf("重开 load 后 timeoutSeconds = (%v, %v), want 5（clamp 下界）", v, err)
	}
}

func TestSecureSetGetMaskingAndMerge(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	// SecureSet 载荷（渲染层 LlmEndpointConfig 形状；disableThinking 字段被 json 丢弃——DR-005）
	payload := `{"baseUrl":"https://open.bigmodel.cn/api/paas/v4","apiKey":"sk-secret-1234567890abcd","model":"glm-4-flash","disableThinking":true}`
	var p SecureSlotPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		t.Fatal(err)
	}
	res, err := app.SecureSet(storage.SlotRed, p)
	if err != nil {
		t.Fatalf("SecureSet: %v", err)
	}
	if res.Stored != "encrypted" {
		t.Fatalf("stored = %s, want encrypted", res.Stored)
	}
	got, err := app.SecureGet(storage.SlotRed)
	if err != nil || got == nil {
		t.Fatalf("SecureGet: %v, %v", got, err)
	}
	// 掩码 + 不泄 Key（铁律 #8）
	if got["apiKey"] != "****abcd" {
		t.Fatalf("apiKey = %v, want ****abcd", got["apiKey"])
	}
	raw, _ := json.Marshal(got)
	if strings.Contains(string(raw), "sk-secret") {
		t.Fatalf("SecureGet 泄漏完整 Key: %s", raw)
	}
	// 掩码回写合并不覆盖真实 Key（electron-DR-013）
	mergePayload := `{"baseUrl":"https://open.bigmodel.cn/api/paas/v4","apiKey":"****abcd","model":"renamed-model"}`
	var p2 SecureSlotPayload
	if err := json.Unmarshal([]byte(mergePayload), &p2); err != nil {
		t.Fatal(err)
	}
	if _, err := app.SecureSet(storage.SlotRed, p2); err != nil {
		t.Fatal(err)
	}
	got, err = app.SecureGet(storage.SlotRed)
	if err != nil {
		t.Fatal(err)
	}
	if got["model"] != "renamed-model" {
		t.Fatalf("model = %v, want renamed-model", got["model"])
	}
	// getRaw（后端内部）真实 Key 仍在
	rawCfg := app.credentials.GetRaw(storage.SlotRed)
	if rawCfg == nil || rawCfg.APIKey != "sk-secret-1234567890abcd" {
		t.Fatalf("真实 Key 丢失: %+v", rawCfg)
	}
	if err := app.SecureDelete(storage.SlotRed); err != nil {
		t.Fatal(err)
	}
	got, err = app.SecureGet(storage.SlotRed)
	if err != nil || got != nil {
		t.Fatalf("删除后应 (null, nil)，got (%v, %v)", got, err)
	}
}

func TestSecurePlainFallbackResult(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(true)) // keyring 不可用（如 WSL 无 Secret Service）
	var p SecureSlotPayload
	if err := json.Unmarshal([]byte(`{"baseUrl":"https://api.deepseek.com/v1","apiKey":"sk-x","model":"deepseek-chat"}`), &p); err != nil {
		t.Fatal(err)
	}
	res, err := app.SecureSet(storage.SlotBlack, p)
	if err != nil {
		t.Fatalf("SecureSet: %v", err)
	}
	// UI 如实回报（DR-011）：stored=plainFallback → "已用未加密本地文件存储"
	if res.Stored != "plainFallback" {
		t.Fatalf("stored = %s, want plainFallback", res.Stored)
	}
}

func TestStorageUnavailableDegradation(t *testing.T) {
	// 未 initServices（startup 前异常环境）：invoke 拒绝，前端按"存储不可用"降级
	app := NewApp()
	if _, err := app.DbLoadLatest("humanVsHuman"); !errors.Is(err, errStorageUnavailable) {
		t.Fatalf("DbLoadLatest err = %v", err)
	}
	if err := app.DbSaveGame(SaveGameRequest{}); !errors.Is(err, errStorageUnavailable) {
		t.Fatalf("DbSaveGame err = %v", err)
	}
	if err := app.StoreSet("global_auto_save", true); !errors.Is(err, errStorageUnavailable) {
		t.Fatalf("StoreSet err = %v", err)
	}
	if _, err := app.SecureGet(storage.SlotRed); !errors.Is(err, errStorageUnavailable) {
		t.Fatalf("SecureGet err = %v", err)
	}
}

// 懒打开失败不缓存：目录不可写时 invoke 拒绝，修复后重试成功。
func TestLazyDaoOpenRetry(t *testing.T) {
	dir := t.TempDir()
	app := NewApp()
	badPath := filepath.Join(dir, "no-such-dir", "test.sqlite")
	app.daoPath = badPath // 未 initServices：仅设置 daoPath，模拟打开失败路径
	if _, err := app.getDao(); !errors.Is(err, errStorageUnavailable) {
		t.Fatalf("坏路径应拒绝: %v", err)
	}
	app.daoPath = filepath.Join(dir, "test.sqlite")
	dao, err := app.getDao()
	if err != nil || dao == nil {
		t.Fatalf("修复后重试应成功: %v, %v", dao, err)
	}
	if !reflect.DeepEqual(app.dao, dao) {
		t.Fatal("应复用已打开连接")
	}
}
