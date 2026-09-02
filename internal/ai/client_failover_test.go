package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// ─── 引擎故障转移 v0（C 刀）──────────────────────────────────

// failoverServers 一对测试引擎服务：bad（失败引擎）/ good（转移目标）。
// 两个服务的 /models 都返回可用列表（供 RefreshModels 置 Connected）。
type failoverServers struct {
	badSrv, goodSrv    *httptest.Server
	badID, goodID      string
	mgr                *modelengine.Manager
	goodChatHits       atomic.Int32
	goodChatLastModel  string
	goodChatLastEngine string
	mu                 sync.Mutex
}

// newFailoverSetup 构造带引擎管理器的 Client 与两个自定义引擎：
//   - bad 引擎指向 badSrv（各测试决定失败方式，或 deadURL 断连）；
//   - good 引擎指向 goodSrv（/chat/completions 恒成功）。
//
// 默认两个引擎均已 RefreshModels（Status.Connected=true）。
func newFailoverSetup(t *testing.T, badChatHandler http.HandlerFunc) *failoverServers {
	t.Helper()

	writeModels := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"m1"}]}`))
	}
	f := &failoverServers{}
	goodSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			writeModels(w)
		case "/chat/completions":
			var req ChatRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			f.mu.Lock()
			f.goodChatLastModel = req.Model
			f.mu.Unlock()
			f.goodChatHits.Add(1)
			if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
				// 流式：SSE 一帧正文 + 结束帧
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"转移成功\"}}]}\n\n"))
				_, _ = w.Write([]byte("data: [DONE]\n\n"))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"c1","model":"m-good","choices":[{"index":0,"message":{"role":"assistant","content":"转移成功"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(goodSrv.Close)
	f.goodSrv = goodSrv

	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			writeModels(w)
		case "/chat/completions":
			if badChatHandler != nil {
				badChatHandler(w, r)
				return
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(badSrv.Close)
	f.badSrv = badSrv

	mgr := modelengine.NewManager("", "")
	var err error
	if f.badID, err = mgr.AddCustomEngine("bad", badSrv.URL, "sk-bad"); err != nil {
		t.Fatalf("AddCustomEngine(bad): %v", err)
	}
	if err := mgr.SetDefaultModel(f.badID, "m-bad"); err != nil {
		t.Fatalf("SetDefaultModel(bad): %v", err)
	}
	if f.goodID, err = mgr.AddCustomEngine("good", goodSrv.URL, ""); err != nil {
		t.Fatalf("AddCustomEngine(good): %v", err)
	}
	if err := mgr.SetDefaultModel(f.goodID, "m-good"); err != nil {
		t.Fatalf("SetDefaultModel(good): %v", err)
	}
	// RefreshModels 成功 → Status.Connected=true（候选资格）
	if _, err := mgr.RefreshModels(context.Background(), f.badID); err != nil {
		t.Fatalf("RefreshModels(bad): %v", err)
	}
	if _, err := mgr.RefreshModels(context.Background(), f.goodID); err != nil {
		t.Fatalf("RefreshModels(good): %v", err)
	}
	f.mgr = mgr
	return f
}

// newFailoverClient 构造绑定 mgr 的 Client（退避注入短间隔，测试不拖慢）。
func (f *failoverServers) newFailoverClient(t *testing.T) (*Client, *[]string) {
	t.Helper()
	c := &Client{
		cfg:                &config.Config{Model: "grok-4.20"},
		httpClient:         f.badSrv.Client(),
		sem:                make(chan struct{}, 4),
		chatRetryBackoff:   []time.Duration{time.Millisecond},
		streamRetryBackoff: []time.Duration{time.Millisecond},
		streamIdleTimeout:  2 * time.Second,
	}
	c.engineMgr = f.mgr
	events := &[]string{}
	c.OnFailover = func(from, to, model string) {
		f.mu.Lock()
		*events = append(*events, from+"->"+to+":"+model)
		f.mu.Unlock()
	}
	return c, events
}

// deadURL 返回一个已关闭服务的地址（连接必然被拒绝：网络类错误的最稳定形态）。
func deadURL(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NewServeMux())
	srv.Close()
	return srv.URL
}

func failoverChatReq(engineID string) *ChatRequest {
	return &ChatRequest{
		Model:    "m-bad",
		EngineID: engineID,
		Messages: []ChatMessage{{Role: "user", Content: "你好"}},
	}
}

// TestFailover_Disabled_RegressionLock 开关关（默认）：失败路径行为不变——
// 返回原错误、目标引擎零命中、无转移事件（回归锁）。
func TestFailover_Disabled_RegressionLock(t *testing.T) {
	f := newFailoverSetup(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	})
	c, events := f.newFailoverClient(t)
	// 不调用 SetEngineFailoverFunc：nil = 关闭

	_, err := c.Chat(context.Background(), failoverChatReq(f.badID))
	if err == nil {
		t.Fatal("503 应报错")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("错误 = %q, want 含 503", err.Error())
	}
	if n := f.goodChatHits.Load(); n != 0 {
		t.Errorf("开关关时目标引擎被命中 %d 次, want 0", n)
	}
	if len(*events) != 0 {
		t.Errorf("开关关不应有转移事件: %v", *events)
	}
}

// TestFailover_NetworkError_TransfersOnce 开关开 + 网络类错误（连接拒绝）：
// 换候选引擎用其 default_model 重试一次成功；转移事件触发；记账按实际
// 引擎/模型逐笔（失败引擎 1 笔失败、目标引擎 1 笔成功）。
func TestFailover_NetworkError_TransfersOnce(t *testing.T) {
	dead := deadURL(t)
	f := newFailoverSetup(t, nil)
	// bad 引擎改指已关闭服务（RefreshModels 已把 Status 置连通，不影响转移判定：
	// 转移条件看的是请求错误而非 Status）
	if err := f.mgr.UpdateCustomEngine(f.badID, "bad", dead, "sk-bad"); err != nil {
		t.Fatalf("UpdateCustomEngine: %v", err)
	}
	c, events := f.newFailoverClient(t)
	c.SetEngineFailoverFunc(func() bool { return true })

	resp, err := c.Chat(context.Background(), failoverChatReq(f.badID))
	if err != nil {
		t.Fatalf("转移重试应成功: %v", err)
	}
	if got := resp.Choices[0].Message.Content; got != "转移成功" {
		t.Errorf("回复 = %q, want 转移成功", got)
	}
	f.mu.Lock()
	lastModel := f.goodChatLastModel
	evs := append([]string(nil), *events...)
	f.mu.Unlock()
	if lastModel != "m-good" {
		t.Errorf("重试模型 = %q, want m-good（候选 default_model）", lastModel)
	}
	if len(evs) != 1 || evs[0] != f.badID+"->"+f.goodID+":m-good" {
		t.Fatalf("转移事件 = %v, want 恰 1 次 %s->%s:m-good", evs, f.badID, f.goodID)
	}

	// 记账：bad/m-bad 至少 1 笔失败；good/m-good 1 笔成功。
	sum := f.mgr.GetModelCallStats()
	var badFail, goodOK bool
	for _, pm := range sum.PerModel {
		if pm.EngineID == f.badID && pm.Model == "m-bad" && pm.FailCount >= 1 {
			badFail = true
		}
		if pm.EngineID == f.goodID && pm.Model == "m-good" && pm.SuccessCount == 1 {
			goodOK = true
		}
	}
	if !badFail || !goodOK {
		t.Errorf("记账不符: badFail=%v goodOK=%v, per-model=%+v", badFail, goodOK, sum.PerModel)
	}
}

// TestFailover_401_ConfigError_NoTransfer 开关开 + 401（配置类错误）：不转移。
func TestFailover_401_ConfigError_NoTransfer(t *testing.T) {
	f := newFailoverSetup(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("bad key"))
	})
	c, events := f.newFailoverClient(t)
	c.SetEngineFailoverFunc(func() bool { return true })

	_, err := c.Chat(context.Background(), failoverChatReq(f.badID))
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("401 应原样报错, got %v", err)
	}
	if n := f.goodChatHits.Load(); n != 0 {
		t.Errorf("401 不应转移，目标引擎被命中 %d 次", n)
	}
	if len(*events) != 0 {
		t.Errorf("401 不应产生转移事件: %v", *events)
	}
}

// TestFailover_NoCandidates_OriginalError 候选全不可用（目标引擎未连通）：
// 返回原错误，无转移。
func TestFailover_NoCandidates_OriginalError(t *testing.T) {
	dead := deadURL(t)
	f := newFailoverSetup(t, nil)
	if err := f.mgr.UpdateCustomEngine(f.badID, "bad", dead, "sk-bad"); err != nil {
		t.Fatalf("UpdateCustomEngine: %v", err)
	}
	// good 引擎置为停用 → 候选池空
	if err := f.mgr.SaveEngine(mapToEngineConfig(f.goodID)); err != nil {
		t.Fatal(err)
	}
	c, events := f.newFailoverClient(t)
	c.SetEngineFailoverFunc(func() bool { return true })

	_, err := c.Chat(context.Background(), failoverChatReq(f.badID))
	if err == nil {
		t.Fatal("无候选时应返回原错误")
	}
	if !strings.Contains(err.Error(), "dial tcp") {
		t.Errorf("错误 = %q, want 原网络错误（dial tcp）", err.Error())
	}
	if len(*events) != 0 {
		t.Errorf("无候选不应有转移事件: %v", *events)
	}
	if n := f.goodChatHits.Load(); n != 0 {
		t.Errorf("停用引擎被命中 %d 次, want 0", n)
	}
}

// mapToEngineConfig 停用引擎配置（SaveEngine 只认存在的 ID）。
func mapToEngineConfig(id string) modelengine.EngineConfig {
	return modelengine.EngineConfig{ID: id, Enabled: false}
}

// TestFailoverStream_NetworkError_Transfers 流式：首字节前失败（连接拒绝）→
// 转移到候选引擎成功出 token。
func TestFailoverStream_NetworkError_Transfers(t *testing.T) {
	dead := deadURL(t)
	f := newFailoverSetup(t, nil)
	if err := f.mgr.UpdateCustomEngine(f.badID, "bad", dead, "sk-bad"); err != nil {
		t.Fatalf("UpdateCustomEngine: %v", err)
	}
	c, events := f.newFailoverClient(t)
	c.SetEngineFailoverFunc(func() bool { return true })

	chunks, err := c.ChatStream(context.Background(), failoverChatReq(f.badID))
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var content string
	var gotErr string
	for ch := range chunks {
		if ch.Content != "" {
			content += ch.Content
		}
		if ch.Error != "" {
			gotErr += ch.Error
		}
	}
	if content != "转移成功" || gotErr != "" {
		t.Errorf("流式转移结果 content=%q err=%q, want 转移成功/空", content, gotErr)
	}
	f.mu.Lock()
	evs := append([]string(nil), *events...)
	f.mu.Unlock()
	if len(evs) != 1 {
		t.Fatalf("流式转移事件 = %v, want 恰 1 次", evs)
	}
}

// TestFailoverStream_AfterFirstByte_NoTransfer 流式已开始吐 token（200 已返回）
// 后连接被重置：错误照常透出，但不转移（目标引擎零命中、无事件）。
func TestFailoverStream_AfterFirstByte_NoTransfer(t *testing.T) {
	f := newFailoverSetup(t, func(w http.ResponseWriter, _ *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("server 不支持 hijack")
			return
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		buf.WriteString("HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\n\r\n")
		buf.WriteString("data: {\"choices\":[{\"delta\":{\"content\":\"首个\"}}]}\n\n")
		buf.Flush()
		time.Sleep(50 * time.Millisecond)
		if tc, ok := conn.(*net.TCPConn); ok {
			_ = tc.SetLinger(0) // RST 关闭：客户端读侧得到连接重置错误
		}
		_ = conn.Close()
	})
	c, events := f.newFailoverClient(t)
	c.SetEngineFailoverFunc(func() bool { return true })

	chunks, err := c.ChatStream(context.Background(), failoverChatReq(f.badID))
	if err != nil {
		t.Fatalf("200 已返回，ChatStream 不应前置失败: %v", err)
	}
	sawContent, sawErr := false, false
	for ch := range chunks {
		if ch.Content != "" {
			sawContent = true
		}
		if ch.Error != "" {
			sawErr = true
		}
	}
	if !sawContent || !sawErr {
		t.Errorf("应先收到 token 再收到错误分块, content=%v err=%v", sawContent, sawErr)
	}
	if n := f.goodChatHits.Load(); n != 0 {
		t.Errorf("首字节后不应转移，目标引擎被命中 %d 次", n)
	}
	if len(*events) != 0 {
		t.Errorf("首字节后不应有转移事件: %v", *events)
	}
}

// TestFailoverStream_BothFail_AtMostOnceTransfer 双引擎皆失败：恰转移一次
// （failoverDone 防二次转移），返回错误。
func TestFailoverStream_BothFail_AtMostOnceTransfer(t *testing.T) {
	dead := deadURL(t)
	f := newFailoverSetup(t, nil)
	if err := f.mgr.UpdateCustomEngine(f.badID, "bad", dead, "sk-bad"); err != nil {
		t.Fatal(err)
	}
	if err := f.mgr.UpdateCustomEngine(f.goodID, "good", dead, ""); err != nil {
		t.Fatal(err)
	}
	c, events := f.newFailoverClient(t)
	c.SetEngineFailoverFunc(func() bool { return true })

	_, err := c.ChatStream(context.Background(), failoverChatReq(f.badID))
	if err == nil {
		t.Fatal("双引擎皆失败应报错")
	}
	f.mu.Lock()
	n := len(*events)
	f.mu.Unlock()
	if n != 1 {
		t.Errorf("转移事件次数 = %d, want 1（最多一次转移，不递归）", n)
	}
}

// TestIsTransferableChatError 可转移错误分类表。
func TestIsTransferableChatError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{fmt.Errorf("API 错误 (HTTP 401): bad key"), false},
		{fmt.Errorf("API 错误 (HTTP 403): forbidden"), false},
		{fmt.Errorf("API 错误 (HTTP 400): invalid params"), false},
		{fmt.Errorf("API 错误 (HTTP 404): no such model"), false},
		{fmt.Errorf("API 错误 (HTTP 408): timeout"), true},
		{fmt.Errorf("API 错误 (HTTP 429): rate limited"), true},
		{fmt.Errorf("API 错误 (HTTP 500): oops"), true},
		{fmt.Errorf("API 错误 (HTTP 503): down"), true},
		{fmt.Errorf("API 请求失败: %w", errors.New(`Post "http://x": dial tcp 1.2.3.4:443: connection refused`)), true},
		{fmt.Errorf("流式请求失败: %w", errors.New(`Post "http://x": dial tcp: lookup api.example.com: no such host`)), true},
		{fmt.Errorf("API 请求失败: %w", errors.New("context deadline exceeded")), true},
		{fmt.Errorf("API 请求失败: %w", errors.New("net/http: TLS handshake timeout")), true},
		{fmt.Errorf("API 请求失败: %w", errors.New("read: connection reset by peer")), true},
		{fmt.Errorf("API 错误 (HTTP 200)?: odd"), false},
	}
	for i, tc := range cases {
		if got := isTransferableChatError(tc.err); got != tc.want {
			t.Errorf("case %d (%v) = %v, want %v", i, tc.err, got, tc.want)
		}
	}
}
