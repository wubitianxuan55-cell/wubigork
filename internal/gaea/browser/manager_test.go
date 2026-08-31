package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// ── 造假 DevTools 服务（HTTP 元端点 + fake CDP WebSocket）────────────────

// fakeDevtools 记录服务端观察到的行为，供测试断言；pages 维护当前 page
// target 状态（/json/list 真源，/json/new 追加、/json/close/<id> 移除），
// 支撑多标签测试。
type fakeDevtools struct {
	mu          sync.Mutex
	upgrades    int              // ws 升级次数（Ensure 幂等性证据）
	navigations int              // Page.navigate 次数
	silentEval  bool             // 置真后 Runtime.evaluate 不回包（超时路径）
	pages       []map[string]any // 当前 page targets
	newSeq      int              // /json/new 序号（生成唯一 page id）
}

// wsBase 由 httptest.Server 地址推导（http→ws）。
func wsBase(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// newFakeDevtools 起 fake 服务：/json/version、/json/list、/json/new、
// /json/close、/devtools/page/<id>（CDP ws）。初始含一个 page-1 标签。
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
			f.mu.Lock()
			list := make([]map[string]any, len(f.pages))
			copy(list, f.pages)
			f.mu.Unlock()
			writeJSON(list)
		case r.URL.Path == "/json/new":
			f.mu.Lock()
			f.newSeq++
			id := fmt.Sprintf("page-new-%d", f.newSeq)
			tgt := map[string]any{
				"id": id, "type": "page", "title": "new",
				"url":                  r.URL.Query().Get("url"),
				"webSocketDebuggerUrl": wsBase(srv) + "/devtools/page/" + id,
			}
			f.pages = append(f.pages, tgt)
			f.mu.Unlock()
			writeJSON(tgt)
		case strings.HasPrefix(r.URL.Path, "/json/close/"):
			id := strings.TrimPrefix(r.URL.Path, "/json/close/")
			f.mu.Lock()
			kept := f.pages[:0]
			for _, p := range f.pages {
				if p["id"] != id {
					kept = append(kept, p)
				}
			}
			f.pages = kept
			f.mu.Unlock()
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
	f.mu.Lock()
	f.pages = append(f.pages, map[string]any{
		"id": "page-1", "type": "page", "title": "fake", "url": "about:blank",
		"webSocketDebuggerUrl": wsBase(srv) + "/devtools/page/page-1",
	})
	f.mu.Unlock()
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
	if m.activePageID != "page-1" {
		t.Fatalf("activePageID = %q, want page-1", m.activePageID)
	}
	if len(m.tabs) != 1 {
		t.Fatalf("tabs = %d, want 1（attach 应登记 active 会话）", len(m.tabs))
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
	if len(m.tabs) != 0 || m.activePageID != "" || m.httpBase != "" {
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

// ── 空闲 TTL 自动关停 ───────────────────────────────────────────────────

// TestIdleTTLAutoShutdown TTL=250ms（<4s 时 watcher 间隔钳到 1s）：活动期
// （持续 Ensure 触摸 lastActive）不被回收；停止活动后空闲超时自动 teardown；
// 回收后任意调用经 Ensure 自动重拉（天然闭环）。
func TestIdleTTLAutoShutdown(t *testing.T) {
	srv, f := newFakeDevtools(t)
	m := NewManager(Options{InjectHTTPBase: srv.URL, IdleTTL: 250 * time.Millisecond})
	defer m.Shutdown()
	ctx := context.Background()
	if err := m.Ensure(ctx); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	// 活动期：持续 Ensure 越过首个 watcher tick，不应被回收。
	for i := 0; i < 12; i++ {
		if err := m.Ensure(ctx); err != nil {
			t.Fatalf("活动期 Ensure: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	m.mu.Lock()
	alive := len(m.tabs) > 0
	m.mu.Unlock()
	if !alive {
		t.Fatal("活动期内被错误回收")
	}
	// 停止活动：空闲超过 TTL 后自动 teardown。
	deadline := time.Now().Add(3 * time.Second)
	for {
		m.mu.Lock()
		reaped := len(m.tabs) == 0 && m.httpBase == ""
		m.mu.Unlock()
		if reaped {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("空闲超时后未自动回收")
		}
		time.Sleep(50 * time.Millisecond)
	}
	// 回收后任意调用经 Ensure 自动重拉。
	if err := m.Ensure(ctx); err != nil {
		t.Fatalf("回收后 Ensure: %v", err)
	}
	f.mu.Lock()
	ups := f.upgrades
	f.mu.Unlock()
	if ups != 2 {
		t.Fatalf("upgrades = %d, want 2（回收后应重新拨号）", ups)
	}
}

// TestIdleTTLZeroDisabled IdleTTL=0（禁用）：不建 idleStop、不起 watcher，
// 长睡后会话仍在（保持 MVP 常驻行为）。
func TestIdleTTLZeroDisabled(t *testing.T) {
	srv, _ := newFakeDevtools(t)
	m := NewManager(Options{InjectHTTPBase: srv.URL}) // TTL 0 = 禁用
	defer m.Shutdown()
	if err := m.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if m.idleStop != nil {
		t.Fatal("IdleTTL=0 不应创建 idleStop")
	}
	time.Sleep(300 * time.Millisecond)
	m.mu.Lock()
	alive := len(m.tabs) > 0
	m.mu.Unlock()
	if !alive {
		t.Fatal("TTL=0（禁用）时不应自动回收")
	}
}

// TestIdleTTLFromEnv 环境变量解析表驱动：>0 覆盖默认；未设置/非法/非正回默认。
func TestIdleTTLFromEnv(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want time.Duration
	}{
		{"未设置回默认", "", defaultIdleTTL},
		{"非法字符串回默认", "abc", defaultIdleTTL},
		{"零值回默认", "0", defaultIdleTTL},
		{"负值回默认", "-5", defaultIdleTTL},
		{"秒数覆盖默认", "30", 30 * time.Second},
		{"大秒数", "3600", time.Hour},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := idleTTLFromEnv(func(string) string { return tc.env })
			if got != tc.want {
				t.Fatalf("idleTTLFromEnv(%q) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

// ── 多标签页 ────────────────────────────────────────────────────────────

// TestListTabs 列出 /json/list 真源的全部 page 与当前 active。
func TestListTabs(t *testing.T) {
	m, _ := newFakeManager(t)
	if _, err := m.NewTab(context.Background(), "http://example.local/x"); err != nil {
		t.Fatal(err)
	}
	infos, active, err := m.ListTabs(context.Background())
	if err != nil {
		t.Fatalf("ListTabs: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("tabs = %d, want 2", len(infos))
	}
	if active != "page-new-1" {
		t.Fatalf("active = %q, want page-new-1（NewTab 后应切换为新建页）", active)
	}
	ids := map[string]bool{}
	for _, it := range infos {
		ids[it.ID] = true
		if it.ID == "page-new-1" && it.URL != "http://example.local/x" {
			t.Fatalf("新页 URL = %q, want http://example.local/x", it.URL)
		}
	}
	if !ids["page-1"] || !ids["page-new-1"] {
		t.Fatalf("infos 缺页: %+v", infos)
	}
}

// TestNewTabSwitchesActive NewTab 建页、拨号并切换为 active；空 URL 走
// about:blank；非法 URL 在动作前被拒。
func TestNewTabSwitchesActive(t *testing.T) {
	m, f := newFakeManager(t)
	if m.activePageID != "page-1" {
		t.Fatalf("active = %q, want page-1", m.activePageID)
	}
	tab, err := m.NewTab(context.Background(), "http://example.local/x")
	if err != nil {
		t.Fatalf("NewTab: %v", err)
	}
	if tab.ID != "page-new-1" || tab.URL != "http://example.local/x" {
		t.Fatalf("tab = %+v", tab)
	}
	if m.activePageID != "page-new-1" {
		t.Fatalf("active = %q, want 新建页", m.activePageID)
	}
	if len(m.tabs) != 2 {
		t.Fatalf("tabs = %d, want 2", len(m.tabs))
	}
	f.mu.Lock()
	ups := f.upgrades
	f.mu.Unlock()
	if ups != 2 {
		t.Fatalf("upgrades = %d, want 2（新页应拨号）", ups)
	}
	// 空 URL → about:blank。
	tab2, err := m.NewTab(context.Background(), "")
	if err != nil {
		t.Fatalf("NewTab(空): %v", err)
	}
	if tab2.URL != "about:blank" {
		t.Fatalf("空 URL 新页 = %q, want about:blank", tab2.URL)
	}
	// 非法 URL → ErrInvalidInput（任何浏览器动作之前被拒）。
	if _, err := m.NewTab(context.Background(), "javascript:alert(1)"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("NewTab(javascript:) err = %v, want ErrInvalidInput", err)
	}
}

// TestSwitchTabInvalidatesRefs 切换标签后旧 refs 诚实失效（epoch 清零 →
// stale_refs）；未知 tab 报 ErrInvalidInput。
func TestSwitchTabInvalidatesRefs(t *testing.T) {
	m, _ := newFakeManager(t)
	ctx := context.Background()
	if _, err := m.NewTab(ctx, "http://example.local/x"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Snapshot(ctx); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if m.epoch == 0 {
		t.Fatal("snapshot 应登记 epoch")
	}
	tab, err := m.SwitchTab(ctx, "page-1")
	if err != nil {
		t.Fatalf("SwitchTab: %v", err)
	}
	if tab.ID != "page-1" {
		t.Fatalf("tab.ID = %q, want page-1", tab.ID)
	}
	if m.activePageID != "page-1" || m.epoch != 0 {
		t.Fatalf("切换后 active=%q epoch=%d, want page-1/0", m.activePageID, m.epoch)
	}
	// 旧 ref（新页上 snapshot 的）在切回后必须判 stale。
	if _, err := m.Click(ctx, 1, ""); !errors.Is(err, ErrRefsStale) {
		t.Fatalf("切换后 Click err = %v, want ErrRefsStale", err)
	}
	// 未知 tab → ErrInvalidInput。
	if _, err := m.SwitchTab(ctx, "no-such-tab"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("SwitchTab(未知) err = %v, want ErrInvalidInput", err)
	}
}

// TestCloseTabSwitchesRemaining 关闭 active 后切到剩余 tab；关非 active 不动
// active；未知 id 报 ErrInvalidInput。
func TestCloseTabSwitchesRemaining(t *testing.T) {
	m, _ := newFakeManager(t)
	ctx := context.Background()
	if _, err := m.NewTab(ctx, "http://a.local/1"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.NewTab(ctx, "http://b.local/2"); err != nil {
		t.Fatal(err)
	}
	if m.activePageID != "page-new-2" {
		t.Fatalf("active = %q, want page-new-2", m.activePageID)
	}
	// 关 active（page-new-2）：应切到剩余之一。
	if err := m.CloseTab(ctx, "page-new-2"); err != nil {
		t.Fatalf("CloseTab(active): %v", err)
	}
	m.mu.Lock()
	tabCount := len(m.tabs)
	active := m.activePageID
	closedGone := m.tabs["page-new-2"] == nil
	activeAlive := m.tabs[active] != nil
	m.mu.Unlock()
	if tabCount != 2 {
		t.Fatalf("tabs = %d, want 2", tabCount)
	}
	if !closedGone || !activeAlive {
		t.Fatalf("关闭后状态异常: tabs=%d active=%q", tabCount, active)
	}
	if active == "page-new-2" {
		t.Fatal("active 不应指向已关闭的 tab")
	}
	// 关非 active：从剩余里找一个非 active 的 id（map 序不确定），关闭后
	// active 必须不变。
	var nonActive string
	for tid := range m.tabs {
		if tid != active {
			nonActive = tid
			break
		}
	}
	if nonActive == "" {
		t.Fatal("剩余应至少有一个非 active tab")
	}
	if err := m.CloseTab(ctx, nonActive); err != nil {
		t.Fatalf("CloseTab(非 active): %v", err)
	}
	if m.activePageID != active {
		t.Fatalf("关非 active 后 active = %q, want %q", m.activePageID, active)
	}
	// 未知 id → ErrInvalidInput。
	if err := m.CloseTab(ctx, "no-such-tab"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CloseTab(未知) err = %v, want ErrInvalidInput", err)
	}
}

// TestCloseLastTabShutsDown 关闭最后一个标签 → 整体 teardown；随后 Ensure
// 重新拉起（fake 的 page 已被 /json/close 移除 → 走新建路径）。
func TestCloseLastTabShutsDown(t *testing.T) {
	srv, f := newFakeDevtools(t)
	m := NewManager(Options{InjectHTTPBase: srv.URL})
	defer m.Shutdown()
	ctx := context.Background()
	if err := m.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.CloseTab(ctx, "page-1"); err != nil {
		t.Fatalf("CloseTab: %v", err)
	}
	if len(m.tabs) != 0 || m.httpBase != "" || m.edge != nil {
		t.Fatal("最后一个标签关闭后应整体 teardown")
	}
	if err := m.Ensure(ctx); err != nil {
		t.Fatalf("teardown 后 Ensure: %v", err)
	}
	f.mu.Lock()
	ups := f.upgrades
	f.mu.Unlock()
	if ups != 2 {
		t.Fatalf("upgrades = %d, want 2", ups)
	}
}
