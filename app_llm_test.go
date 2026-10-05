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

	// 渲染层 wailsAdapter 载荷形状：{requestId, url, headers, body, authSlot}
	err := app.LlmChat(LlmChatRequest{
		RequestID: "id-chat-1",
		URL:       server.srv.URL + "/v1/chat/completions",
		Headers:   map[string]string{"Content-Type": "application/json", "Accept": "text/event-stream"},
		Body:      `{"model":"glm-4-flash","stream":true}`,
	})
	if err != nil {
		t.Fatalf("LlmChat 应受理即返回：%v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_, done, errs := collector.snapshot()
		if len(done) > 0 || len(errs) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
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
		t.Fatalf("事件流不符：chunks=%q done=%v", text, done)
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

	if err := app.LlmChat(LlmChatRequest{
		RequestID: "id-cancel",
		URL:       server.srv.URL,
		Headers:   map[string]string{"Content-Type": "application/json"},
		Body:      "{}",
	}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(120 * time.Millisecond) // 首块到达
	app.LlmCancel("id-cancel")         // 幂等：再取消一次
	app.LlmCancel("id-cancel")
	time.Sleep(200 * time.Millisecond)

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

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_, done, _ := collector.snapshot()
		if len(done) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if seenAuth != "Bearer sk-real-black-9876" {
		t.Fatalf("应按槽位注入真实鉴权头：%q", seenAuth)
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
