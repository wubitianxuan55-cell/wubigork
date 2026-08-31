package browser

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// ── 造假 DevTools 服务（HTTP 元端点 + fake CDP WebSocket）────────────────

// fakeDevtools 记录服务端观察到的行为，供测试断言。
type fakeDevtools struct {
	mu          sync.Mutex
	upgrades    int  // ws 升级次数（Ensure 幂等性证据）
	navigations int  // Page.navigate 次数
	silentEval  bool // 置真后 Runtime.evaluate 不回包（超时路径）
}

// wsBase 由 httptest.Server 地址推导（http→ws）。
func wsBase(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// newFakeDevtools 起 fake 服务：/json/version、/json/list、/json/new、
// /json/close、/devtools/page/<id>（CDP ws）。
func newFakeDevtools(t *testing.T) (*httptest.Server, *fakeDevtools) {
	t.Helper()
	f := &fakeDevtools{}
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON := func(v any) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(v)
		}
		switch {
		case r.URL.Path == "/json/version":
			writeJSON(map[string]any{"Browser": "FakeEdge/1.0"})
		case r.URL.Path == "/json/list":
			writeJSON([]map[string]any{{
				"id": "page-1", "type": "page", "title": "fake", "url": "about:blank",
				"webSocketDebuggerUrl": wsBase(srv) + "/devtools/page/page-1",
			}})
		case r.URL.Path == "/json/new":
			writeJSON(map[string]any{
				"id": "page-new", "type": "page", "title": "new",
				"url":                  r.URL.Query().Get("url"),
				"webSocketDebuggerUrl": wsBase(srv) + "/devtools/page/page-new",
			})
		case strings.HasPrefix(r.URL.Path, "/json/close/"):
			writeJSON(map[string]any{})
		case strings.HasPrefix(r.URL.Path, "/devtools/page/"):
			conn, err := up.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			f.mu.Lock()
			f.upgrades++
			f.mu.Unlock()
			f.serveCDP(conn)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, f
}

// serveCDP 最小 CDP 应答：id-echo；Page.navigate 应答后补发
// Page.loadEventFired；Runtime.evaluate 按表达式特征回固定 JSON。
func (f *fakeDevtools) serveCDP(conn *websocket.Conn) {
	defer conn.Close()
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var req struct {
			ID     int64           `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(raw, &req) != nil {
			continue
		}
		switch req.Method {
		case "Page.navigate":
			f.mu.Lock()
			f.navigations++
			f.mu.Unlock()
			_ = conn.WriteJSON(map[string]any{"id": req.ID, "result": map[string]any{}})
			_ = conn.WriteJSON(map[string]any{"method": "Page.loadEventFired"})
		case "Runtime.evaluate":
			if f.silentEval {
				continue
			}
			var p struct {
				Expression string `json:"expression"`
			}
			_ = json.Unmarshal(req.Params, &p)
			_ = conn.WriteJSON(map[string]any{"id": req.ID, "result": map[string]any{
				"result": map[string]any{"type": "object", "value": fakeEvalValue(p.Expression)},
			}})
		default:
			_ = conn.WriteJSON(map[string]any{"id": req.ID, "result": map[string]any{}})
		}
	}
}

// fakeEvalValue 按表达式特征返回固定值（各 JS 片段带独特 token，匹配稳定）。
func fakeEvalValue(expr string) any {
	switch {
	case strings.Contains(expr, "__gaeaMeta"):
		return map[string]any{"ok": true, "title": "测试页", "url": "http://fake.local/page"}
	case strings.Contains(expr, "document.readyState"):
		return "complete"
	case strings.Contains(expr, "__gaeaRefs"):
		return map[string]any{"ok": true, "title": "测试页", "url": "http://fake.local/page", "epoch": 1,
			"items": []map[string]any{{"ref": 1, "tag": "a", "text": "链接一", "path": "#link1"}}}
	case strings.Contains(expr, "__gaeaEpoch"):
		return 1
	case strings.Contains(expr, "gaeaClick"):
		return map[string]any{"ok": true, "text": "链接一"}
	case strings.Contains(expr, "gaeaType"):
		return map[string]any{"ok": true, "text": "hello"}
	case strings.Contains(expr, "gaeaScroll"):
		return map[string]any{"ok": true, "top": "800"}
	case strings.Contains(expr, "#missing"):
		return map[string]any{"ok": false, "error": "未找到元素：#missing"}
	case strings.Contains(expr, "innerText") || strings.Contains(expr, "document.body"):
		return map[string]any{"ok": true, "title": "测试页", "url": "http://fake.local/page", "text": "hello gaea 正文"}
	}
	return map[string]any{"ok": true}
}

// newFakeManager 经注入端点构造已 Ensure 的管理器。
func newFakeManager(t *testing.T) (*Manager, *fakeDevtools) {
	t.Helper()
	srv, f := newFakeDevtools(t)
	m := NewManager(Options{InjectHTTPBase: srv.URL})
	t.Cleanup(m.Shutdown)
	if err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	return m, f
}

// ── Ensure 全链 ─────────────────────────────────────────────────────────

// TestEnsureFakeAndReuse 全链（列举→dial→Page.enable）+ 二次 Ensure 复用会话。
func TestEnsureFakeAndReuse(t *testing.T) {
	m, f := newFakeManager(t)
	if err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("二次 Ensure: %v", err)
	}
	f.mu.Lock()
	ups := f.upgrades
	f.mu.Unlock()
	if ups != 1 {
		t.Fatalf("upgrades = %d, want 1（Ensure 应复用已就绪会话）", ups)
	}
	if m.pageID != "page-1" {
		t.Fatalf("pageID = %q, want page-1", m.pageID)
	}
}

// TestShutdownThenRelaunch Shutdown 后可重新拉起（会话重建）。
func TestShutdownThenRelaunch(t *testing.T) {
	srv, f := newFakeDevtools(t)
	m := NewManager(Options{InjectHTTPBase: srv.URL})
	if err := m.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	m.Shutdown()
	if m.conn != nil || m.httpBase != "" {
		t.Fatal("Shutdown 后状态应清零")
	}
	if err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("Shutdown 后 Ensure: %v", err)
	}
	f.mu.Lock()
	ups := f.upgrades
	f.mu.Unlock()
	if ups != 2 {
		t.Fatalf("upgrades = %d, want 2（Shutdown 后应重新拨号）", ups)
	}
}

// ── Navigate ────────────────────────────────────────────────────────────

func TestNavigateSuccess(t *testing.T) {
	m, f := newFakeManager(t)
	res, err := m.Navigate(context.Background(), "http://example.local/x", 10)
	if err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	if res.Title != "测试页" || res.URL != "http://fake.local/page" {
		t.Fatalf("res = %+v", res)
	}
	f.mu.Lock()
	navs := f.navigations
	f.mu.Unlock()
	if navs != 1 {
		t.Fatalf("navigations = %d, want 1", navs)
	}
	// 导航后 refs 失效：未重新 snapshot 直接 click 必须被拒。
	if _, err := m.Click(context.Background(), 1, ""); !errors.Is(err, ErrRefsStale) {
		t.Fatalf("导航后 Click err = %v, want ErrRefsStale", err)
	}
}

// TestNavigateTimeout 永不回包的 evaluate → 导航等待超时报错。
func TestNavigateTimeout(t *testing.T) {
	srv, f := newFakeDevtools(t)
	f.mu.Lock()
	f.silentEval = true
	f.mu.Unlock()
	m := NewManager(Options{InjectHTTPBase: srv.URL})
	defer m.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := m.Navigate(ctx, "http://example.local/x", 5) // 事件与 readyState 轮询都得不到应答
	if err == nil {
		t.Fatal("应超时报错")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "超时") {
		t.Fatalf("err = %v, want 超时语义", err)
	}
}

// ── Read / Snapshot / Click / Type / Scroll ─────────────────────────────

func TestRead(t *testing.T) {
	m, _ := newFakeManager(t)
	res, err := m.Read(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if res.Text != "hello gaea 正文" || res.Title != "测试页" {
		t.Fatalf("res = %+v", res)
	}
	// 局部读取：元素未命中走语义化错误。
	if _, err := m.Read(context.Background(), "#missing", 100); !errors.Is(err, ErrElementNotFound) {
		t.Fatalf("Read(#missing) err = %v, want ErrElementNotFound", err)
	}
}

func TestSnapshotAndClick(t *testing.T) {
	m, _ := newFakeManager(t)
	snap, err := m.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snap.Items) != 1 || snap.Items[0].Ref != 1 || snap.Items[0].Text != "链接一" {
		t.Fatalf("snap = %+v", snap)
	}
	if m.epoch != 1 {
		t.Fatalf("epoch = %d, want 1（snapshot 应登记 refs 代数）", m.epoch)
	}
	act, err := m.Click(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("Click(ref): %v", err)
	}
	if act.Text != "链接一" {
		t.Fatalf("click text = %q", act.Text)
	}
	// selector 兜底路径。
	if _, err := m.Click(context.Background(), 0, "#link1"); err != nil {
		t.Fatalf("Click(selector): %v", err)
	}
	// ref/selector 都缺 → 参数错误。
	if _, err := m.Click(context.Background(), 0, ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Click(空) err = %v, want ErrInvalidInput", err)
	}
}

// TestClickWithoutSnapshot 从未 snapshot → ref 无效（epoch=0 守门）。
func TestClickWithoutSnapshot(t *testing.T) {
	m, _ := newFakeManager(t)
	if _, err := m.Click(context.Background(), 1, ""); !errors.Is(err, ErrRefsStale) {
		t.Fatalf("Click err = %v, want ErrRefsStale", err)
	}
}

func TestType(t *testing.T) {
	m, _ := newFakeManager(t)
	if _, err := m.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	act, err := m.Type(context.Background(), 1, "", "hello", true)
	if err != nil {
		t.Fatalf("Type: %v", err)
	}
	if act.Text != "hello" {
		t.Fatalf("type text = %q", act.Text)
	}
	if _, err := m.Type(context.Background(), 1, "", "", false); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Type(空文本) err = %v, want ErrInvalidInput", err)
	}
}

func TestScroll(t *testing.T) {
	m, _ := newFakeManager(t)
	act, err := m.Scroll(context.Background(), "down", 0, "")
	if err != nil {
		t.Fatalf("Scroll: %v", err)
	}
	if act.Text != "800" {
		t.Fatalf("scroll top = %q, want 默认 800px", act.Text)
	}
	if _, err := m.Scroll(context.Background(), "left", 100, ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Scroll(left) err = %v, want ErrInvalidInput", err)
	}
}

// ── Conn 层：超时与断连 ─────────────────────────────────────────────────

// TestCallTimeout 服务端永不回包 → Call 按超时失败。
func TestCallTimeout(t *testing.T) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		for { // 只读不应答
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	conn, err := Dial(context.Background(), wsBase(srv)+"/devtools/page/1")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	err = conn.call(context.Background(), 300*time.Millisecond, "Foo.bar", map[string]any{}, nil)
	if err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("err = %v, want 超时", err)
	}
}

// TestConnClosedFailsPending 服务端断开 → pending 统一失败、后续调用报错。
func TestConnClosedFailsPending(t *testing.T) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = conn.Close() // 立即断开
	}))
	defer srv.Close()

	conn, err := Dial(context.Background(), wsBase(srv)+"/devtools/page/1")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	time.Sleep(100 * time.Millisecond) // 让读循环感知断开
	if err := conn.Call(context.Background(), "Foo.bar", map[string]any{}, nil); err == nil {
		t.Fatal("断连后 Call 应报错")
	}
}

// TestCloseIdempotent Close 幂等。
func TestCloseIdempotent(t *testing.T) {
	srv, _ := newFakeDevtools(t)
	conn, err := Dial(context.Background(), wsBase(srv)+"/devtools/page/page-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("二次 Close 应幂等, got %v", err)
	}
}
