package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/auth"
	"github.com/gaea/gaea/internal/config"
)

// ─── T7-1.4 非流式 Chat 重试 / 401 刷新 / 状态并发 ─────────────────────

// writeChatOK 写入一份最小可解析的非流式 chat completions 200 响应。
func writeChatOK(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":    "chat-1",
		"model": "grok-4.20",
		"choices": []map[string]any{{
			"index":   0,
			"message": map[string]any{"role": "assistant", "content": content},
		}},
	})
}

// newOAuthRefreshStub 构造 OIDC discovery + token 端点桩，使 auth.RefreshAccessToken
// 离线完成刷新（返回 access_token=new-acc），返回 discovery URL 与 token 端点命中计数。
func newOAuthRefreshStub(t *testing.T) (discoveryURL string, refreshHits *atomic.Int32) {
	t.Helper()
	hits := &atomic.Int32{}
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-acc",
			"refresh_token": "new-ref",
			"token_type":    "Bearer",
			"expires_in":    21600,
		})
	}))
	t.Cleanup(tokenSrv.Close)
	discSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": "https://auth.example/authorize",
			"token_endpoint":         tokenSrv.URL,
		})
	}))
	t.Cleanup(discSrv.Close)
	return discSrv.URL, hits
}

// TestChat_RetryConnectionFailureThenSuccess 非流式 Chat：首个请求连接建立失败
// （劫持后直接断开）应退避重试，重试后成功返回完整回复。
func TestChat_RetryConnectionFailureThenSuccess(t *testing.T) {
	var mu sync.Mutex
	attempts := 0
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		isFirst := attempts == 1
		mu.Unlock()
		if isFirst {
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
		writeChatOK(w, "连接重试成功")
	}))

	resp, err := c.Chat(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if got := resp.Choices[0].Message.Content; got != "连接重试成功" {
		t.Errorf("回复 = %q, want 连接重试成功", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Errorf("请求次数 = %d, want 2（1 次失败 + 1 次重试成功）", attempts)
	}
}

// TestChat_Retry5xxThenSuccess 非流式 Chat 收到 5xx 应退避重试，重试后成功。
func TestChat_Retry5xxThenSuccess(t *testing.T) {
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
		writeChatOK(w, "五零三后成功")
	}))

	resp, err := c.Chat(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if got := resp.Choices[0].Message.Content; got != "五零三后成功" {
		t.Errorf("回复 = %q, want 五零三后成功", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Errorf("请求次数 = %d, want 2", attempts)
	}
}

// TestChat_Retry5xxExhausted 持续 5xx 时重试耗尽（默认 2 次）后返回错误。
func TestChat_Retry5xxExhausted(t *testing.T) {
	var mu sync.Mutex
	attempts := 0
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("still down"))
	}))

	_, err := c.Chat(context.Background(), reqForTest())
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

// TestChat_401RefreshThenRetrySuccess 非流式 Chat 收到 401 应刷新 token 后
// 在同一函数内重发（新 token 生效），最终返回成功响应。
func TestChat_401RefreshThenRetrySuccess(t *testing.T) {
	discURL, _ := newOAuthRefreshStub(t)
	var mu sync.Mutex
	var auths []string
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		auths = append(auths, r.Header.Get("Authorization"))
		n := len(auths)
		mu.Unlock()
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}
		writeChatOK(w, "刷新后成功")
	}))
	c.cfg.OIDCDiscoveryURL = discURL
	c.token.RefreshToken = "old-refresh"

	resp, err := c.Chat(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if got := resp.Choices[0].Message.Content; got != "刷新后成功" {
		t.Errorf("回复 = %q, want 刷新后成功", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(auths) != 2 {
		t.Fatalf("请求次数 = %d, want 2（401 + 重试成功）", len(auths))
	}
	if auths[0] != "Bearer test-token" {
		t.Errorf("首次 Authorization = %q, want Bearer test-token", auths[0])
	}
	if auths[1] != "Bearer new-acc" {
		t.Errorf("重试 Authorization = %q, want Bearer new-acc（刷新后 token）", auths[1])
	}
}

// TestChat_401RefreshNoDoubleSlot 401 刷新重试不得递归占用第二个信号量槽：
// 信号量容量为 1 时，若 401 走递归重试会因槽满而阻塞；此处用超时 ctx 兜底，
// 修复后应在单槽内完成并成功返回。
func TestChat_401RefreshNoDoubleSlot(t *testing.T) {
	discURL, _ := newOAuthRefreshStub(t)
	var mu sync.Mutex
	attempts := 0
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		isFirst := attempts == 1
		mu.Unlock()
		if isFirst {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}
		writeChatOK(w, "单槽成功")
	}))
	c.cfg.OIDCDiscoveryURL = discURL
	c.token.RefreshToken = "old-refresh"
	c.sem = make(chan struct{}, 1) // 容量 1：递归重试会因槽满死锁

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := c.Chat(ctx, reqForTest())
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if got := resp.Choices[0].Message.Content; got != "单槽成功" {
		t.Errorf("回复 = %q, want 单槽成功", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Errorf("请求次数 = %d, want 2", attempts)
	}
}

// TestClientState_ConcurrentAccess 并发读写引擎/图片后端/token 状态不得数据竞争
// （-race 验证）。tryRefreshToken 无 refresh token 时快速返回，用于覆盖写锁路径。
func TestClientState_ConcurrentAccess(t *testing.T) {
	c := &Client{
		tokenStore: auth.NewTokenStore(t.TempDir() + "/token.json"),
		sem:        make(chan struct{}, 4),
	}
	c.token = &auth.Token{AccessToken: "concurrent-token", ObtainedAt: time.Now(), ExpiresIn: 7200}

	ids := []string{"xai", "ollama", "comfyui"}
	types := []string{"", "xai", "comfyui", "herdsman"}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.SetActiveEngine(ids[n%len(ids)])
			_ = c.ActiveEngineID()
			c.SetImageBackend(nil, types[n%len(types)])
			_ = c.GetImageBackend()
			_ = c.GetImageBackendType()
			if tok, err := c.GetToken(); err != nil || tok != "concurrent-token" {
				t.Errorf("GetToken = %q, %v", tok, err)
			}
			_ = c.tryRefreshToken() // 无 refresh token → 快速返回错误，覆盖写锁路径
		}(i)
	}
	wg.Wait()

	if got := c.ActiveEngineID(); got == "" {
		t.Error("ActiveEngineID 不应为空")
	}
	if got := c.GetImageBackendType(); got == "" {
		t.Error("GetImageBackendType 不应为空")
	}
}

// TestGetToken_SingleFlightRefresh 并发 GetToken 触发刷新时只能打一次刷新请求
// （single-flight）：首个调用方持写锁刷新，其余等待后复用刷新结果。
func TestGetToken_SingleFlightRefresh(t *testing.T) {
	discURL, hits := newOAuthRefreshStub(t)

	store := auth.NewTokenStore(t.TempDir() + "/token.json")
	// 落盘一个已过期、带 refresh token 的 token：GetToken 慢路径会读到它并触发刷新
	if err := store.Save(&auth.Token{
		AccessToken:  "expired-acc",
		RefreshToken: "old-refresh",
		ObtainedAt:   time.Now().Add(-2 * time.Hour),
		ExpiresIn:    3600,
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	c := &Client{
		cfg:        &config.Config{XaiClientID: "test-client", OIDCDiscoveryURL: discURL},
		tokenStore: store,
		sem:        make(chan struct{}, 4),
	}

	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tok, err := c.GetToken()
			if err != nil {
				t.Errorf("GetToken: %v", err)
				return
			}
			if tok != "new-acc" {
				t.Errorf("token = %q, want new-acc", tok)
			}
		}()
	}
	wg.Wait()

	if got := hits.Load(); got != 1 {
		t.Errorf("刷新请求次数 = %d, want 1（single-flight）", got)
	}
}
