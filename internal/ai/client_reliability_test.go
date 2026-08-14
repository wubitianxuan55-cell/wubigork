package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/auth"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/netclient"
)

// ─── T6-1.1 SSE 长行 / 超时重试 / 代理 可靠性 ─────────────────────

// collectStream 消费流式通道并返回拼接正文、是否收到 Done、首个错误。
func collectStream(t *testing.T, chunks <-chan SSEChunk) (string, bool, string) {
	t.Helper()
	var sb strings.Builder
	done := false
	for ch := range chunks {
		if ch.Error != "" {
			return sb.String(), done, ch.Error
		}
		if ch.Done {
			done = true
			continue
		}
		sb.WriteString(ch.Content)
	}
	return sb.String(), done, ""
}

// TestChatStream_LongSSELine 超长 SSE 单行（>1MB，远超 Scanner 64KB 上限）
// 必须不断流：完整内容逐字透传，无 ErrTooLong 错误。
func TestChatStream_LongSSELine(t *testing.T) {
	big := strings.Repeat("a", 1_200_000) // 1.2MB 单行
	sse := "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"" + big + "\"}}]}\n\n" +
		"data: [DONE]\n\n"

	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sse))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))

	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	got, done, streamErr := collectStream(t, chunks)
	if streamErr != "" {
		t.Fatalf("超长行不应触发流错误: %s", streamErr)
	}
	if !done {
		t.Error("未收到 Done 标记")
	}
	if len(got) != len(big) {
		t.Fatalf("内容长度 = %d, want %d（超长行被截断）", len(got), len(big))
	}
	if got != big {
		t.Error("超长行内容与发送不一致（数据丢失）")
	}
}

// TestChatStream_RetryConnectionFailureThenSuccess 连接建立失败应退避重试，
// 重试后成功读取流（首个请求被掐断，第二个请求正常返回 SSE）。
func TestChatStream_RetryConnectionFailureThenSuccess(t *testing.T) {
	var mu sync.Mutex
	attempts := 0
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		isFirst := attempts == 1
		mu.Unlock()
		if isFirst {
			// 首个请求：劫持连接后直接关闭，模拟连接建立阶段失败
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Errorf("server does not support hijack")
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			_ = conn.Close()
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"重试成功\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	c.streamRetryBackoff = []time.Duration{20 * time.Millisecond, 20 * time.Millisecond}

	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	got, done, streamErr := collectStream(t, chunks)
	if streamErr != "" {
		t.Fatalf("stream error: %s", streamErr)
	}
	if got != "重试成功" {
		t.Errorf("正文 = %q, want 重试成功", got)
	}
	if !done {
		t.Error("未收到 Done 标记")
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Errorf("请求次数 = %d, want 2（1 次失败 + 1 次重试成功）", attempts)
	}
}

// TestChatStream_Retry5xxThenSuccess 5xx 响应应退避重试，重试后成功读取流。
func TestChatStream_Retry5xxThenSuccess(t *testing.T) {
	var mu sync.Mutex
	attempts := 0
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		isFirst := attempts == 1
		mu.Unlock()
		if isFirst {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("temporarily down"))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"五零三后成功\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	c.streamRetryBackoff = []time.Duration{20 * time.Millisecond, 20 * time.Millisecond}

	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	got, done, streamErr := collectStream(t, chunks)
	if streamErr != "" {
		t.Fatalf("stream error: %s", streamErr)
	}
	if got != "五零三后成功" {
		t.Errorf("正文 = %q, want 五零三后成功", got)
	}
	if !done {
		t.Error("未收到 Done 标记")
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Errorf("请求次数 = %d, want 2", attempts)
	}
}

// TestChatStream_Retry5xxExhausted 持续 5xx 时重试耗尽（默认 2 次）后返回错误。
func TestChatStream_Retry5xxExhausted(t *testing.T) {
	var mu sync.Mutex
	attempts := 0
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("still down"))
	}))
	c.streamRetryBackoff = []time.Duration{20 * time.Millisecond, 20 * time.Millisecond}

	_, err := c.ChatStream(context.Background(), reqForTest())
	if err == nil {
		t.Fatal("持续 5xx 应报错")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("错误信息 = %q, want 含 503", err.Error())
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 3 {
		t.Errorf("请求次数 = %d, want 3（1 次初始 + 2 次重试）", attempts)
	}
}

// TestChatStream_IdleTimeout 收到响应头后长时间无任何数据应触发流空闲超时，
// 通过错误分块返回（而非永久挂起）。
func TestChatStream_IdleTimeout(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// 此后不再写任何数据；请求上下文被取消时（空闲超时触发）立即返回
		<-r.Context().Done()
	}))
	c.streamIdleTimeout = 200 * time.Millisecond

	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	_, _, streamErr := collectStream(t, chunks)
	if streamErr == "" {
		t.Fatal("应收到空闲超时错误分块")
	}
	if !strings.Contains(streamErr, "空闲超时") {
		t.Errorf("错误信息 = %q, want 含 空闲超时", streamErr)
	}
}

// newProxyTestClient 构造走 buildHTTPClient（按代理配置构建）的测试客户端。
func newProxyTestClient(t *testing.T, baseURL string, spec netclient.ProxySpec) *Client {
	t.Helper()
	cfg := &config.Config{XaiAPIBaseURL: baseURL, Model: "grok-4.20"}
	c := &Client{
		cfg:               cfg,
		tokenStore:        auth.NewTokenStore(t.TempDir() + "/token.json"),
		sem:               make(chan struct{}, 4),
		proxySpecOverride: &spec,
	}
	c.token = &auth.Token{AccessToken: "test-token", ObtainedAt: time.Now(), ExpiresIn: 7200}
	c.httpClient = c.buildHTTPClient()
	return c
}

// TestChatStream_ProxyBypassForLocalhost 本地引擎（127.0.0.1）必须直连，
// 即使配置了代理也不能把请求发往代理。
func TestChatStream_ProxyBypassForLocalhost(t *testing.T) {
	var mu sync.Mutex
	proxyHits := 0
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		proxyHits++
		mu.Unlock()
		w.WriteHeader(http.StatusBadGateway) // 若被代理访问则流必失败
	}))
	defer proxySrv.Close()

	targetSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"本地直连\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer targetSrv.Close()

	c := newProxyTestClient(t, targetSrv.URL, netclient.ProxySpec{
		Mode: netclient.ModeCustom,
		URL:  proxySrv.URL,
	})
	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	got, done, streamErr := collectStream(t, chunks)
	if streamErr != "" {
		t.Fatalf("stream error: %s", streamErr)
	}
	if got != "本地直连" || !done {
		t.Errorf("本地直连失败: got=%q done=%v", got, done)
	}
	mu.Lock()
	defer mu.Unlock()
	if proxyHits != 0 {
		t.Errorf("本地引擎不应走代理，代理命中 %d 次", proxyHits)
	}
}

// TestChatStream_ProxyForCloudEngine 云端引擎（外部域名）必须走配置的代理，
// 请求应到达代理服务器。
func TestChatStream_ProxyForCloudEngine(t *testing.T) {
	var mu sync.Mutex
	proxyHits := 0
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		proxyHits++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"云端走代理\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer proxySrv.Close()

	// 虚构的云端域名（不可解析）：只有经过代理才能连通
	c := newProxyTestClient(t, "http://api.deepseek.example/v1", netclient.ProxySpec{
		Mode: netclient.ModeCustom,
		URL:  proxySrv.URL,
	})
	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	got, done, streamErr := collectStream(t, chunks)
	if streamErr != "" {
		t.Fatalf("stream error: %s", streamErr)
	}
	if got != "云端走代理" || !done {
		t.Errorf("云端走代理失败: got=%q done=%v", got, done)
	}
	mu.Lock()
	defer mu.Unlock()
	if proxyHits != 1 {
		t.Errorf("云端引擎应走代理，代理命中 %d 次, want 1", proxyHits)
	}
}

// TestStreamProxySpec_LoopbackBypass 代理 spec 解析：回环地址（localhost/
// 127.0.0.1/::1）直连，外部域名走代理。
func TestStreamProxySpec_LoopbackBypass(t *testing.T) {
	c := &Client{proxySpecOverride: &netclient.ProxySpec{
		Mode: netclient.ModeCustom,
		URL:  "http://proxy.example:8080",
	}}
	spec := c.proxySpec()
	pf, err := netclient.ProxyFunc(spec)
	if err != nil {
		t.Fatalf("ProxyFunc: %v", err)
	}
	for _, u := range []string{
		"http://127.0.0.1:11434/api/tags",
		"http://localhost:8080/api/tags",
		"http://[::1]:8080/api/tags",
	} {
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			t.Fatalf("NewRequest(%s): %v", u, err)
		}
		got, err := pf(req)
		if err != nil {
			t.Fatalf("lookup(%s): %v", u, err)
		}
		if got != nil {
			t.Errorf("回环地址 %s 应直连，got %v", u, got)
		}
	}
	req, err := http.NewRequest("GET", "https://api.deepseek.com/v1/chat/completions", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	got, err := pf(req)
	if err != nil {
		t.Fatalf("cloud lookup: %v", err)
	}
	if got == nil || got.Host != "proxy.example:8080" {
		t.Errorf("云端应走代理，got %v", got)
	}
}
