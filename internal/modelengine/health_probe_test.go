package modelengine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ─── 健康巡检（C 刀 v0）──────────────────────────────────────

// newProbeTestManager 构造巡检测试 Manager：状态文件落 TempDir，deepseek
// 引擎（云端、启用）指向 srvURL；其余内置引擎一律停用（不探真实云端、
// 事件隔离），需要的引擎由各测试显式再启用。
func newProbeTestManager(t *testing.T, srvURL string) *Manager {
	t.Helper()
	m := NewManager("", "")
	if err := m.LoadState(filepath.Join(t.TempDir(), "engines.json")); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	for _, e := range m.GetEngines() {
		if e.ID != "deepseek" {
			if err := m.SaveEngine(EngineConfig{ID: e.ID, Enabled: false}); err != nil {
				t.Fatalf("SaveEngine(%s): %v", e.ID, err)
			}
		}
	}
	if err := m.SaveEngine(EngineConfig{ID: "deepseek", BaseURL: srvURL, Enabled: true}); err != nil {
		t.Fatalf("SaveEngine(deepseek): %v", err)
	}
	return m
}

func statusOf(t *testing.T, m *Manager, id string) EngineStatus {
	t.Helper()
	e, ok := m.GetEngine(id)
	if !ok {
		t.Fatalf("引擎 %s 缺失", id)
	}
	return e.Status
}

// TestHealthProbe_2xx 成功探测：GET /models 携带引擎鉴权头，2xx → Connected +
// ModelCount 解析 + LastChecked/LatencyMs 落 Status。
func TestHealthProbe_2xx(t *testing.T) {
	var mu sync.Mutex
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"m1"},{"id":"m2"}]}`))
	}))
	defer srv.Close()

	m := newProbeTestManager(t, srv.URL)
	m.UpdateDeepseekKey("sk-secret-abc")
	m.probeRound(context.Background())

	st := statusOf(t, m, "deepseek")
	if !st.Connected {
		t.Error("2xx 后 Status.Connected 应为 true")
	}
	if st.ModelCount != 2 {
		t.Errorf("Status.ModelCount = %d, want 2", st.ModelCount)
	}
	if st.Error != "" {
		t.Errorf("成功后 Status.Error 应为空，实际 %q", st.Error)
	}
	if st.LastChecked == "" || st.ID != "deepseek" {
		t.Errorf("Status 未落 LastChecked/ID: %+v", st)
	}
	mu.Lock()
	defer mu.Unlock()
	if gotPath != "/models" {
		t.Errorf("探测路径 = %q, want /models", gotPath)
	}
	if gotAuth != "Bearer sk-secret-abc" {
		t.Errorf("探测鉴权头 = %q, want Bearer sk-secret-abc", gotAuth)
	}
}

// TestHealthProbe_5xx_ConsecutiveFailPrefix 5xx → Connected=false；连续失败
// 计数：前 2 次无前缀，第 3 次起 Error 带「连续 N 次探测失败」前缀；成功清零。
func TestHealthProbe_5xx_ConsecutiveFailPrefix(t *testing.T) {
	ok := &atomic.Bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if ok.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	m := newProbeTestManager(t, srv.URL)
	m.probeRound(context.Background())
	st := statusOf(t, m, "deepseek")
	if st.Connected || !strings.Contains(st.Error, "HTTP 500") {
		t.Fatalf("首次失败 Status = %+v, want Connected=false 且 Error 含 HTTP 500", st)
	}
	if strings.Contains(st.Error, "连续") {
		t.Errorf("首次失败不应有连续前缀: %q", st.Error)
	}

	m.probeRound(context.Background())
	if st = statusOf(t, m, "deepseek"); strings.Contains(st.Error, "连续") {
		t.Errorf("第 2 次失败不应有连续前缀: %q", st.Error)
	}

	m.probeRound(context.Background())
	st = statusOf(t, m, "deepseek")
	if !strings.HasPrefix(st.Error, "连续 3 次探测失败") {
		t.Errorf("第 3 次失败 Error = %q, want 「连续 3 次探测失败」前缀", st.Error)
	}

	// 成功即清零：Error 清空，随后再次失败从无前缀重新计数。
	ok.Store(true)
	m.probeRound(context.Background())
	if st = statusOf(t, m, "deepseek"); !st.Connected || st.Error != "" {
		t.Fatalf("成功后 Status = %+v, want Connected=true 且 Error 空", st)
	}
	ok.Store(false)
	m.probeRound(context.Background())
	if st = statusOf(t, m, "deepseek"); strings.Contains(st.Error, "连续") {
		t.Errorf("清零后首次失败不应有连续前缀: %q", st.Error)
	}
}

// TestHealthProbe_NotifyOnlyOnChange 状态相对上次变化才回调：失败首轮回调
// false，错误串不变的第二轮不回调，加前缀的第三轮回调，恢复成功回调 true。
func TestHealthProbe_NotifyOnlyOnChange(t *testing.T) {
	ok := &atomic.Bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if ok.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	m := newProbeTestManager(t, srv.URL)
	var mu sync.Mutex
	var events []string
	m.SetHealthNotifyFunc(func(id string, connected bool) {
		mu.Lock()
		events = append(events, fmt.Sprintf("%s=%v", id, connected))
		mu.Unlock()
	})

	for i := 0; i < 3; i++ { // 失败 1（回调 false）→ 失败 2（错误串不变，不回调）→ 失败 3（加前缀，回调 false）
		m.probeRound(context.Background())
	}
	ok.Store(true)
	m.probeRound(context.Background())

	mu.Lock()
	defer mu.Unlock()
	want := []string{"deepseek=false", "deepseek=false", "deepseek=true"}
	if len(events) != len(want) {
		t.Fatalf("回调事件 = %v, want %v（仅状态变化轮回调）", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("回调事件 = %v, want %v", events, want)
		}
	}
}

// TestHealthProbe_SkipsLocalAndDisabled 本地引擎（ollama，保活机制管理）与
// 停用引擎不探测。
func TestHealthProbe_SkipsLocalAndDisabled(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := newProbeTestManager(t, srv.URL)
	// ollama 本地 + 启用 → 跳过；deepseek 云端 + 停用 → 跳过
	if err := m.SaveEngine(EngineConfig{ID: "ollama", BaseURL: srv.URL, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := m.SaveEngine(EngineConfig{ID: "deepseek", BaseURL: srv.URL, Enabled: false}); err != nil {
		t.Fatal(err)
	}
	m.probeRound(context.Background())
	if n := hits.Load(); n != 0 {
		t.Errorf("本地/停用引擎被探测 %d 次, want 0", n)
	}
}

// TestHealthProbe_BadJSONStillConnected 2xx + 坏 JSON：连通性为 true（2xx
// 口径），ModelCount 保持 0。
func TestHealthProbe_BadJSONStillConnected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html>not json</html>`))
	}))
	defer srv.Close()

	m := newProbeTestManager(t, srv.URL)
	m.probeRound(context.Background())
	st := statusOf(t, m, "deepseek")
	if !st.Connected {
		t.Errorf("2xx 坏 JSON 应判连通，Status = %+v", st)
	}
	if st.ModelCount != 0 {
		t.Errorf("坏 JSON ModelCount = %d, want 0", st.ModelCount)
	}
}

// TestHealthProbe_Timeout 探测超时返回失败（不 panic、错误可读）。
func TestHealthProbe_Timeout(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-block // 挂起直到测试结束
	}))
	t.Cleanup(func() {
		close(block)
		srv.Close()
	})

	m := newProbeTestManager(t, srv.URL)
	e, ok := m.GetEngine("deepseek")
	if !ok {
		t.Fatal("deepseek 缺失")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	connected, _, err := m.probeEndpoint(ctx, e)
	if connected || err == nil {
		t.Fatalf("超时探测 connected=%v err=%v, want false+error", connected, err)
	}
}

// TestHealthProbe_ErrorSanitized 无 Key 泄漏：请求确实带上了 Key（server 侧
// 收到 Bearer 头），但错误路径产出的 Status.Error 不含 Key/Authorization 内容
// （含服务端把 Key 原样回显进响应体的对抗场景）。
func TestHealthProbe_ErrorSanitized(t *testing.T) {
	const secret = "sk-secret-abc123"
	var mu sync.Mutex
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()
		// 对抗：服务端把收到的鉴权头原样回显进 500 响应体。
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("echo: " + gotAuth))
	}))
	defer srv.Close()

	m := newProbeTestManager(t, srv.URL)
	m.UpdateDeepseekKey(secret)
	m.probeRound(context.Background())

	mu.Lock()
	auth := gotAuth
	mu.Unlock()
	if auth != "Bearer "+secret {
		t.Fatalf("请求应携带 Key，实际头 = %q", auth)
	}
	if st := statusOf(t, m, "deepseek"); strings.Contains(st.Error, secret) {
		t.Errorf("Status.Error 泄漏 Key: %q", st.Error)
	}

	// sanitize 单元口径：含 Key 的任意错误串一律打码。
	if got := sanitizeProbeError(secret, "boom Bearer "+secret+" boom"); strings.Contains(got, secret) {
		t.Errorf("sanitizeProbeError 未打码: %q", got)
	}
}

// TestHealthProbe_GLMUsesChatPing GLM 无 /models 端点：巡检走与 TestConnection
// 同源的 chat ping（POST /chat/completions），连通即 Connected。
func TestHealthProbe_GLMUsesChatPing(t *testing.T) {
	var method, path string
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		method, path = r.Method, r.URL.Path
		mu.Unlock()
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	m := newProbeTestManager(t, srv.URL)
	if err := m.SaveEngine(EngineConfig{ID: "glm", BaseURL: srv.URL, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	m.UpdateGLMKey("glm-key")
	m.probeRound(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if path != "/chat/completions" || method != http.MethodPost {
		t.Errorf("GLM 探测应 POST /chat/completions，实际 %s %s", method, path)
	}
	if st := statusOf(t, m, "glm"); !st.Connected {
		t.Errorf("GLM ping 成功应判连通，Status = %+v", st)
	}
}

// TestHealthProbe_StartStopIdempotent Start 幂等（重启旧循环）、Stop 幂等
// （含未启动时直接 Stop）。
func TestHealthProbe_StartStopIdempotent(t *testing.T) {
	m := NewManager("", "")
	m.StartHealthProbe(context.Background())
	m.StartHealthProbe(context.Background()) // 重复 Start：先停旧循环，不 panic
	m.StopHealthProbe()
	m.StopHealthProbe() // 重复 Stop：空操作
	m.StopHealthProbe()
}

// ─── 故障转移候选（C 刀 v0）──────────────────────────────────

// TestFailoverCandidates 候选口径：enabled + 最近 Connected + default_model
// 判型 llm；排除失败引擎；顺序按 m.order；副本不带 API Key。
func TestFailoverCandidates(t *testing.T) {
	m := NewManager("", "")
	setConnected := func(id string, connected bool) {
		m.mu.Lock()
		m.engines[id].Status.Connected = connected
		m.mu.Unlock()
	}
	// order: xai, ollama, herdsman, deepseek, glm, cosyvoice, opencode-go, opencode-zen
	setConnected("xai", true)       // 候选（grok-4.20 → llm）
	setConnected("deepseek", true)  // 候选
	setConnected("ollama", true)    // 本地但 default_model 空 → 排除
	setConnected("cosyvoice", true) // default_model 判型 tts → 排除
	setConnected("glm", false)      // 未连通 → 排除
	if err := m.SaveEngine(EngineConfig{ID: "opencode-go", Enabled: false}); err != nil {
		t.Fatal(err)
	}
	setConnected("opencode-go", true) // 停用 → 排除

	got := m.FailoverCandidates("")
	if len(got) != 2 || got[0].ID != "xai" || got[1].ID != "deepseek" {
		t.Fatalf("候选 = %v, want [xai deepseek]（按 order 序）", idsOf(got))
	}
	got = m.FailoverCandidates("xai")
	if len(got) != 1 || got[0].ID != "deepseek" {
		t.Fatalf("排除失败引擎后候选 = %v, want [deepseek]", idsOf(got))
	}
	for _, c := range got {
		if c.APIKey != "" {
			t.Errorf("候选 %s 泄漏 APIKey", c.ID)
		}
	}
}

// TestHealthProbe_StateChangePersisted 巡检结果随 saveState 落盘。
func TestHealthProbe_StateChangePersisted(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "engines.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[{"id":"m1"}]}`))
	}))
	defer srv.Close()

	m := NewManager("", "")
	if err := m.LoadState(statePath); err != nil {
		t.Fatal(err)
	}
	if err := m.SaveEngine(EngineConfig{ID: "deepseek", BaseURL: srv.URL, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	m.probeRound(context.Background())

	m2 := NewManager("", "")
	if err := m2.LoadState(statePath); err != nil {
		t.Fatal(err)
	}
	if st := statusOf(t, m2, "deepseek"); !st.Connected {
		t.Errorf("巡检状态未持久化: %+v", st)
	}
}
