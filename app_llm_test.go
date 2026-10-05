package main

// T4.5 LLM 绑定端到端（09 §2.4）：wailsAdapter 载荷形状（JSON 语义）与 Go 绑定往返，
// mock SSE 端点全链路（chat 事件流 / cancel / authSlot 注入 / 测试连接）。
// DR-005 关闭参数断言在 internal/llm config 用例（绑定层只透传渲染层构造的 body）。

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jxsword/chinese_chess_go/internal/llm"
	"github.com/jxsword/chinese_chess_go/internal/storage"
)

// llmCollector 事件收集器（llm.ProxySender 实现）。
type llmCollector struct {
	mu     sync.Mutex
	chunks []llm.Delta
	done   []string
	errors []string
}

func (c *llmCollector) SendChunk(_ string, delta llm.Delta) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.chunks = append(c.chunks, delta)
}

func (c *llmCollector) SendDone(_ string, text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.done = append(c.done, text)
}

func (c *llmCollector) SendError(_ string, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = append(c.errors, message)
}

func (c *llmCollector) snapshot() (chunks []llm.Delta, done, errs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]llm.Delta(nil), c.chunks...), append([]string(nil), c.done...), append([]string(nil), c.errors...)
}

// mockLlmSSE 最小 SSE mock：按脚本逐块推送（自动 flush；响应头懒发送，
// 非流式脚本用 respond 自定状态码）。
type mockLlmSSE struct {
	srv      *httptest.Server
	mu       sync.Mutex
	requests []http.Header
}

// mockLlmWriter 脚本写出 API（respond 用于 HTTP 4xx/5xx 场景）。
type mockLlmWriter struct {
	write   func(string)
	respond func(status int)
}

func startMockLlmSSE(t *testing.T, script func(r *http.Request, w mockLlmWriter)) *mockLlmSSE {
	t.Helper()
	m := &mockLlmSSE{}
	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		m.mu.Lock()
		m.requests = append(m.requests, r.Header.Clone())
		m.mu.Unlock()

		responded := false
		ensureHead := func() {
			if !responded {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				responded = true
			}
		}
		flush := func() {
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		writer := mockLlmWriter{
			write: func(text string) {
				ensureHead()
				_, _ = w.Write([]byte(text))
				flush()
			},
			respond: func(status int) {
				w.WriteHeader(status)
				responded = true
			},
		}
		func() {
			defer func() { _ = recover() }() // 客户端断开（cancel 场景）静默收尾
			script(r, writer)
		}()
	}))
	t.Cleanup(m.srv.Close)
	return m
}

func sseLine(payload string) string { return "data: " + payload + "\n\n" }

func newLlmTestApp(t *testing.T) (*App, *llmCollector) {
	t.Helper()
	app := newTestApp(t, newFakeKeyring(false))
	collector := &llmCollector{}
	app.llmSenderOverride = collector
	return app, collector
}

func TestLlmChatBindingEndToEnd(t *testing.T) {
	app, collector := newLlmTestApp(t)
	server := startMockLlmSSE(t, func(_ *http.Request, w mockLlmWriter) {
		w.write(sseLine(`{"choices":[{"delta":{"content":"着法"}}]}`))
		w.write(sseLine(`{"choices":[{"delta":{"content":": b2-e2"}}]}`))
		w.write(sseLine("[DONE]"))
	})

	// 渲染层 wailsAdapter 载荷形状：{requestId, url, headers, body, authSlot}；
	// LlmChat 阻塞至结算（F4 语义：事件在订阅存活期内回发完毕）。
	err := app.LlmChat(LlmChatRequest{
		RequestID: "id-chat-1",
		URL:       server.srv.URL + "/v1/chat/completions",
		Headers:   map[string]string{"Content-Type": "application/json", "Accept": "text/event-stream"},
		Body:      `{"model":"glm-4-flash","stream":true}`,
	})
	if err != nil {
		t.Fatalf("LlmChat 应恒 resolve nil：%v", err)
	}

	// F4 防回归：LlmChat 返回时事件必须已全部回发（若退回"受理即返回"，
	// 事件在订阅注销后才到达，此处 done 即为空）。
	chunks, done, errs := collector.snapshot()
	if len(errs) != 0 {
		t.Fatalf("不应有错误：%v", errs)
	}
	var text string
	for _, d := range chunks {
		if d.Content != nil {
			text += *d.Content
		}
	}
	if text != "着法: b2-e2" || len(done) != 1 || done[0] != "着法: b2-e2" {
		t.Fatalf("LlmChat 返回时事件应已全部送达：chunks=%q done=%v", text, done)
	}

	// 载荷校验：缺 requestId / url 拒绝。
	if err := app.LlmChat(LlmChatRequest{URL: server.srv.URL}); err == nil {
		t.Fatal("缺 requestId 应拒绝")
	}
	if err := app.LlmChat(LlmChatRequest{RequestID: "x"}); err == nil {
		t.Fatal("缺 url 应拒绝")
	}
}

func TestLlmChatBindingCancel(t *testing.T) {
	app, collector := newLlmTestApp(t)
	release := make(chan struct{})
	server := startMockLlmSSE(t, func(_ *http.Request, w mockLlmWriter) {
		w.write(sseLine(`{"choices":[{"delta":{"content":"首块"}}]}`))
		<-release // 挂住直至测试结束（cancel 由客户端侧收口）
		w.write(sseLine("[DONE]"))
	})
	defer close(release)

	// LlmChat 阻塞至结算（对齐 Electron ipc handler 语义，F4）：放 goroutine，取消后返回。
	chatReturned := make(chan error, 1)
	go func() {
		chatReturned <- app.LlmChat(LlmChatRequest{
			RequestID: "id-cancel",
			URL:       server.srv.URL,
			Headers:   map[string]string{"Content-Type": "application/json"},
			Body:      "{}",
		})
	}()
	// 等首块到达（deadline 轮询，防 CI 慢载假失败）
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if chunks, _, _ := collector.snapshot(); len(chunks) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	app.LlmCancel("id-cancel") // 幂等：再取消一次
	app.LlmCancel("id-cancel")
	select {
	case err := <-chatReturned:
		if err != nil {
			t.Fatalf("取消结算应恒 resolve nil：%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("取消后 LlmChat 未在 3s 内返回")
	}
	time.Sleep(100 * time.Millisecond)

	chunks, done, errs := collector.snapshot()
	if len(done) != 0 || len(errs) != 0 {
		t.Fatalf("取消后不应有任何结局：done=%v errors=%v", done, errs)
	}
	if len(chunks) == 0 {
		t.Fatal("取消前应收到首块")
	}
}

func TestLlmChatBindingAuthSlot(t *testing.T) {
	app := newTestApp(t, newFakeKeyring(false))
	collector := &llmCollector{}
	app.llmSenderOverride = collector

	// 预置黑方槽位完整 Key（掩码回读 → authSlot 注入链路）。
	if _, err := app.SecureSet("llm_config_black", SecureSlotPayload{
		BaseURL: "https://a.com/v1", APIKey: "sk-real-black-9876", Model: "m", Preset: "",
	}); err != nil {
		t.Fatal(err)
	}

	var seenAuth string
	server := startMockLlmSSE(t, func(r *http.Request, w mockLlmWriter) {
		seenAuth = r.Header.Get("Authorization")
		w.write(sseLine("[DONE]"))
	})

	if err := app.LlmChat(LlmChatRequest{
		RequestID: "id-auth",
		URL:       server.srv.URL,
		Headers:   map[string]string{"Content-Type": "application/json", "Authorization": "Bearer ****9876"},
		AuthSlot:  "llm_config_black",
	}); err != nil {
		t.Fatal(err)
	}

	if seenAuth != "Bearer sk-real-black-9876" {
		t.Fatalf("应按槽位注入真实鉴权头：%q", seenAuth)
	}
	if _, done, _ := collector.snapshot(); len(done) != 1 {
		t.Fatalf("LlmChat 返回时 done 应已送达：%v", done)
	}
}

func TestLlmTestConnectionBinding(t *testing.T) {
	app, _ := newLlmTestApp(t)

	// 成功：固定成功消息。
	server := startMockLlmSSE(t, func(_ *http.Request, w mockLlmWriter) {
		w.write(sseLine(`{"choices":[{"delta":{"content":"ok"}}]}`))
		w.write(sseLine("[DONE]"))
	})
	res, err := app.LlmTestConnection(map[string]any{
		"baseUrl": server.srv.URL, "apiKey": "", "model": "test-model", "preset": "",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if res["ok"] != true || res["message"] != "连接成功，模型 test-model 响应正常" {
		t.Fatalf("测试连接结果不符：%v", res)
	}

	// 失败：HTTP 400 + 翻译模型误用提示。
	badServer := startMockLlmSSE(t, func(_ *http.Request, w mockLlmWriter) {
		w.respond(http.StatusBadRequest)
		w.write("Streaming translation is not supported")
	})
	res2, err := app.LlmTestConnection(map[string]any{
		"baseUrl": badServer.srv.URL, "apiKey": "", "model": "m", "preset": "",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if res2["ok"] != false {
		t.Fatalf("应失败：%v", res2)
	}
	msg, _ := res2["message"].(string)
	if !strings.Contains(msg, "连接失败：") || !strings.Contains(msg, "翻译模型") {
		t.Fatalf("失败消息不符：%q", msg)
	}

	// 未配置：构建期错误直接作为失败消息。
	res3, err := app.LlmTestConnection(map[string]any{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if res3["ok"] != false {
		t.Fatalf("应失败：%v", res3)
	}
	msg3, _ := res3["message"].(string)
	if !strings.Contains(msg3, "模型端点未配置") {
		t.Fatalf("未配置消息不符：%q", msg3)
	}
}

// 渲染层 LlmEndpointConfig 形状经 LlmTestConnection 往返（preset 字段 DR-005 schema）。
func TestLlmTestConnectionConfigShape(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"baseUrl": "https://open.bigmodel.cn/api/paas/v4",
		"apiKey":  "****abcd",
		"model":   "glm-4-flash",
		"preset":  "智谱 GLM",
	})
	if err != nil {
		t.Fatal(err)
	}
	var config llm.LlmEndpointConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	if config.Preset != "智谱 GLM" || config.Model != "glm-4-flash" {
		t.Fatalf("配置形状不符：%+v", config)
	}
}

// ---------------------------------------------------------------------------
// T6.3 视觉识图绑定（05 §7）：mock 多模态端点全链路——wire 形状 {fen}、
// 掩码 Key 经 authSlot 注入（DR-010）、恒发关闭参数断言（DR-005，绑定层透传）。
// ---------------------------------------------------------------------------

func TestVisionReadBoardBindingEndToEnd(t *testing.T) {
	var authHeader string
	var bodyMap map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&bodyMap)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{
				"message": map[string]any{
					"content": `{"turn":"red","pieces":[{"col":"d","row":"0","piece":"k"},{"col":"e","row":"9","piece":"K"}]}`,
				},
			}},
		})
	}))
	defer server.Close()

	app := NewApp()
	app.initServices(t.TempDir(), t.TempDir()+"/dao.sqlite", newFakeKeyring(false))
	app.credentials.Set("llm_config_assistant", storage.SlotConfig{
		BaseURL: server.URL, APIKey: "sk-real-vision", Model: "qwen-vl-max", Preset: "通义千问 VL（阿里云百炼）",
	})

	result, err := app.VisionReadBoard(map[string]any{
		"baseUrl": server.URL,
		"apiKey":  "****real", // 掩码 Key（secure.get 回读语义）
		"model":   "qwen-vl-max",
		"preset":  "通义千问 VL（阿里云百炼）",
	}, "QUJD", "image/png", "llm_config_assistant")
	if err != nil {
		t.Fatalf("VisionReadBoard 报错: %v", err)
	}
	fen, ok := result["fen"].(string)
	if !ok || !strings.HasPrefix(fen, "3k5/9/9/9/9/9/9/9/9/4K4") {
		t.Fatalf("结果应含 {fen}: %v", result)
	}
	if authHeader != "Bearer sk-real-vision" {
		t.Errorf("掩码 Key 应由槽位注入真实鉴权: %q", authHeader)
	}
	if bodyMap["enable_thinking"] != false {
		t.Errorf("识图请求应恒发 enable_thinking:false（dashscope 预设）: %v", bodyMap["enable_thinking"])
	}
	if _, hasStream := bodyMap["stream"]; hasStream {
		t.Errorf("识图请求应非流式")
	}
	// 存储不可用/未知槽位：注入空 Key（本地网关语义），请求仍发出但不带鉴权。
	app2 := NewApp()
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("空 Key 不应带鉴权头")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": `{"pieces":[]}`}}},
		})
	}))
	defer server2.Close()
	// 未配置解析器（resolveApiKey 恒空）：掩码 Key → 空鉴权。
	if _, err := app2.VisionReadBoard(map[string]any{
		"baseUrl": server2.URL, "apiKey": "****xxxx", "model": "m", "preset": "",
	}, "QUJD", "image/jpeg", "llm_config_assistant"); err != nil && !strings.Contains(err.Error(), "双方王数量异常") {
		t.Fatalf("识别管线错误应如实上抛: %v", err)
	}
}
