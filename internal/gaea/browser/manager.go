package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// 语义化错误：builtin 层用 errors.Is 映射 envelope code。
var (
	// ErrInvalidInput 参数/URL 校验失败。
	ErrInvalidInput = errors.New("invalid input")
	// ErrRefsStale snapshot 的 ref 已失效（页面跳转或尚未 snapshot）。
	ErrRefsStale = errors.New("refs stale")
	// ErrElementNotFound 页面上未找到目标元素。
	ErrElementNotFound = errors.New("element not found")
)

const (
	// defaultNavTimeout 页面加载等待默认上限。
	defaultNavTimeout = 20 * time.Second
	// defaultReadChars Read 默认截断字符数。
	defaultReadChars = 6000
	// maxReadChars Read 截断上限（防超长页面灌爆上下文）。
	maxReadChars = 50000
	// defaultScrollPx 滚动默认像素。
	defaultScrollPx = 800
	// maxScrollPx 单次滚动上限。
	maxScrollPx = 10000
	// snapshotMaxRefs 单页最多标记的可交互元素数。
	snapshotMaxRefs = 200
	// defaultIdleTTL 空闲自动关停默认时长（Default() 单例生效；可用
	// GAEA_BROWSER_IDLE_TTL（秒）覆盖，>0 才生效）。
	defaultIdleTTL = 10 * time.Minute
)

// Options 管理器选项。
type Options struct {
	// Headless 无头启动（测试/CI 用；默认有头）。
	Headless bool
	// InjectHTTPBase 非空时跳过 Edge 启动与探活，直接把该地址当 DevTools
	// HTTP 端点（fake CDP 测试注入）。
	InjectHTTPBase string
	// ProbeTimeout DevTools 探活上限（默认 20s）。
	ProbeTimeout time.Duration
	// IdleTTL 空闲自动关停时长：超过该时长无任何 browser_* 调用且仍有活动
	// 会话/标签时自动 teardown（0=禁用，保持常驻；Default() 单例默认 10m）。
	IdleTTL time.Duration
}

// Manager 会话管理器：幂等 Ensure（互斥防双拉起）+ 多标签页操作 + Shutdown。
// 多标签 MVP：tabs 维护每个已拨号 page target 的 CDP 会话，activePageID 指
// 向当前操作页；空闲超时由 watcher goroutine 自动回收（回收后任意调用经
// Ensure 自动重拉）。
type Manager struct {
	opts Options

	mu           sync.Mutex
	tabs         map[string]*Conn // targetID → 已拨号的 page CDP 会话
	activePageID string           // 当前 active 的 targetID（必有对应 conn）
	edge         *EdgeProcess
	httpBase     string
	port         int
	epoch        int64 // 最近一次 snapshot 的 refs 代数；0 = 无有效 refs

	// 空闲自动关停：lastActive 最近活跃时刻（Ensure 成功路径刷新）；watcher
	// 每 TTL/4 轮询一次，超时即 teardown；Shutdown 经 idleStop 停 watcher。
	lastActive   atomic.Int64 // UnixNano；0 = 从未活跃
	idleStop     chan struct{}
	idleStopOnce sync.Once
	idleOnce     sync.Once
}

var (
	defaultMu      sync.Mutex
	defaultManager *Manager
)

// Default 返回进程级单例（懒创建；工具层每次 Execute 调用）。单例默认带
// defaultIdleTTL 空闲关停（GAEA_BROWSER_IDLE_TTL 秒数可覆盖）。
func Default() *Manager {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultManager == nil {
		defaultManager = NewManager(Options{IdleTTL: idleTTLFromEnv(os.Getenv)})
	}
	return defaultManager
}

// SetForTest 替换单例（返回旧实例），供测试注入 fake 端点。
func SetForTest(m *Manager) *Manager {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	prev := defaultManager
	defaultManager = m
	return prev
}

// NewManager 构造管理器（不做任何 I/O）。IdleTTL>0 时预建 watcher 停止通道
// （避免 Shutdown 与 watcher 启动之间的竞态）。
func NewManager(opts Options) *Manager {
	m := &Manager{opts: opts}
	if opts.IdleTTL > 0 {
		m.idleStop = make(chan struct{})
	}
	return m
}

// idleTTLFromEnv 从 GAEA_BROWSER_IDLE_TTL（秒）解析空闲 TTL：>0 覆盖默认；
// 未设置或非法值忽略回默认。注入 getenv 便于测试。
func idleTTLFromEnv(getenv func(string) string) time.Duration {
	raw := strings.TrimSpace(getenv("GAEA_BROWSER_IDLE_TTL"))
	if raw == "" {
		return defaultIdleTTL
	}
	secs, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || secs <= 0 {
		return defaultIdleTTL
	}
	return time.Duration(secs) * time.Second
}

// activeConn 返回当前 active tab 的会话（未拨号则 nil）。须持 mu 调用。
func (m *Manager) activeConn() *Conn {
	if m.activePageID == "" {
		return nil
	}
	return m.tabs[m.activePageID]
}

// Ensure 幂等确保浏览器就绪：定位 → 启动 → 探活 → 取 page target → dial →
// Page.enable。已就绪时健康探测通过即复用（并刷新 lastActive）；失联则整体
// 重拉。全程持有 mu（仿 tts ensure），天然防双拉起。
func (m *Manager) Ensure(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conn := m.activeConn(); conn != nil && conn.healthy(ctx) {
		m.lastActive.Store(time.Now().UnixNano())
		return nil
	}
	m.teardownLocked()
	var err error
	if m.opts.InjectHTTPBase != "" {
		m.httpBase = m.opts.InjectHTTPBase
		err = m.attachLocked(ctx)
	} else {
		err = m.ensureRealLocked(ctx)
	}
	if err != nil {
		return err
	}
	m.lastActive.Store(time.Now().UnixNano())
	m.startIdleWatcher()
	return nil
}

// ensureRealLocked 定位并启动 Edge，等 DevTools 端口就绪。
func (m *Manager) ensureRealLocked(ctx context.Context) error {
	exe, err := FindEdge()
	if err != nil {
		return err
	}
	port, err := freePort()
	if err != nil {
		return fmt.Errorf("browser: 取空闲端口失败: %w", err)
	}
	ep, err := Launch(ctx, exe, port, LaunchOptions{Headless: m.opts.Headless})
	if err != nil {
		return err
	}
	m.edge = ep
	m.port = port
	m.httpBase = fmt.Sprintf("http://127.0.0.1:%d", port)
	probeTO := m.opts.ProbeTimeout
	if probeTO <= 0 {
		probeTO = defaultNavTimeout
	}
	if err := waitDevtools(ctx, m.httpBase, probeTO); err != nil {
		proc.KillTracked(ep.Cmd, ep.Job)
		_ = os.RemoveAll(ep.ProfileDir)
		m.edge = nil
		m.port = 0
		m.httpBase = ""
		return fmt.Errorf("browser: Edge 探活失败: %w", err)
	}
	return m.attachLocked(ctx)
}

// attachLocked 列举全部 page target（无则 PUT /json/new 建 about:blank），
// 取第一个为 active 并 dial（其余已列举的 page 不拨号，切换时按需 dial）。
// 须持 mu。
func (m *Manager) attachLocked(ctx context.Context) error {
	targets, err := listTargets(ctx, m.httpBase)
	if err != nil {
		return fmt.Errorf("browser: 列举目标失败: %w", err)
	}
	var pages []Target
	for i := range targets {
		if targets[i].Type == "page" {
			pages = append(pages, targets[i])
		}
	}
	if len(pages) == 0 {
		t, err := newPageTarget(ctx, m.httpBase, "about:blank")
		if err != nil {
			return err
		}
		pages = append(pages, *t)
	}
	first := pages[0]
	if first.WSURL == "" {
		return fmt.Errorf("browser: page target %s 缺少 webSocketDebuggerUrl", first.ID)
	}
	conn, err := Dial(ctx, first.WSURL)
	if err != nil {
		return err
	}
	if err := conn.Call(ctx, "Page.enable", map[string]any{}, nil); err != nil {
		_ = conn.Close()
		return fmt.Errorf("browser: Page.enable 失败: %w", err)
	}
	if m.tabs == nil {
		m.tabs = map[string]*Conn{}
	}
	if old, ok := m.tabs[first.ID]; ok { // 重复 attach 时先收旧会话
		_ = old.Close()
	}
	m.tabs[first.ID] = conn
	m.activePageID = first.ID
	m.epoch = 0
	return nil
}

// teardownLocked 收割子进程与临时 profile、断开全部会话（须持 mu）。
func (m *Manager) teardownLocked() {
	for id, conn := range m.tabs {
		_ = conn.Close()
		delete(m.tabs, id)
	}
	if m.edge != nil {
		proc.KillTracked(m.edge.Cmd, m.edge.Job)
		_ = os.RemoveAll(m.edge.ProfileDir)
		m.edge = nil
	}
	m.httpBase = ""
	m.port = 0
	m.activePageID = ""
	m.epoch = 0
}

// Shutdown 终止受控浏览器并清理临时 profile（幂等）。之后任意 browser_*
// 调用会重新拉起。同时停掉空闲 watcher（进程级单例收尾）。
func (m *Manager) Shutdown() {
	m.idleStopOnce.Do(func() {
		if m.idleStop != nil {
			close(m.idleStop)
		}
	})
	m.mu.Lock()
	defer m.mu.Unlock()
	m.teardownLocked()
}

// ClosePage 关闭当前页面并 Shutdown 整个浏览器（browser_close 语义）。
func (m *Manager) ClosePage(ctx context.Context) error {
	m.mu.Lock()
	activeID, httpBase := m.activePageID, m.httpBase
	m.mu.Unlock()
	if activeID == "" {
		return nil // 未启动：视为已关
	}
	_ = closePageTarget(ctx, httpBase, activeID)
	m.Shutdown()
	return nil
}

// startIdleWatcher 启动空闲回收 watcher（幂等，idleOnce 守护）：ticker 间隔
// = TTL/4（TTL<4s 用 1s）；每轮持 mu 检查：有活动会话且空闲超过 TTL →
// teardownLocked（回收后任意调用经 Ensure 自动重拉，天然闭环）。Shutdown
// 关闭 idleStop 即停；goroutine 自带 recover 防异常退出泄漏。
func (m *Manager) startIdleWatcher() {
	if m.opts.IdleTTL <= 0 {
		return
	}
	m.idleOnce.Do(func() {
		interval := m.opts.IdleTTL / 4
		if interval < time.Second {
			interval = time.Second
		}
		go func() {
			defer func() { _ = recover() }()
			t := time.NewTicker(interval)
			defer t.Stop()
			for {
				select {
				case <-m.idleStop:
					return
				case <-t.C:
					m.mu.Lock()
					active := m.edge != nil || len(m.tabs) > 0
					if active && time.Since(time.Unix(0, m.lastActive.Load())) > m.opts.IdleTTL {
						m.teardownLocked()
					}
					m.mu.Unlock()
				}
			}
		}()
	})
}

// ── 多标签页 ────────────────────────────────────────────────────────────

// TabInfo 一个浏览器标签页的概要信息。
type TabInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ListTabs 列出全部 page 标签（/json/list 真源，含未拨号的 window.open
// 目标）与当前 active tab id。
func (m *Manager) ListTabs(ctx context.Context) ([]TabInfo, string, error) {
	if err := m.Ensure(ctx); err != nil {
		return nil, "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	targets, err := listTargets(ctx, m.httpBase)
	if err != nil {
		return nil, "", fmt.Errorf("browser: 列举目标失败: %w", err)
	}
	infos := make([]TabInfo, 0, len(targets))
	for i := range targets {
		if targets[i].Type != "page" {
			continue
		}
		infos = append(infos, TabInfo{ID: targets[i].ID, Title: targets[i].Title, URL: targets[i].URL})
	}
	return infos, m.activePageID, nil
}

// NewTab 新建一个标签页并切换为 active：URL 空 → about:blank；非空须过
// ValidateURL。新建后 dial 并 Page.enable；epoch 清零（新页无有效 refs）。
func (m *Manager) NewTab(ctx context.Context, rawURL string) (TabInfo, error) {
	pageURL := strings.TrimSpace(rawURL)
	if pageURL == "" {
		pageURL = "about:blank"
	} else if err := ValidateURL(pageURL); err != nil {
		return TabInfo{}, err
	}
	if err := m.Ensure(ctx); err != nil {
		return TabInfo{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	target, err := newPageTarget(ctx, m.httpBase, pageURL)
	if err != nil {
		return TabInfo{}, err
	}
	if target.WSURL == "" {
		return TabInfo{}, fmt.Errorf("browser: page target %s 缺少 webSocketDebuggerUrl", target.ID)
	}
	conn, err := Dial(ctx, target.WSURL)
	if err != nil {
		return TabInfo{}, err
	}
	if err := conn.Call(ctx, "Page.enable", map[string]any{}, nil); err != nil {
		_ = conn.Close()
		return TabInfo{}, fmt.Errorf("browser: Page.enable 失败: %w", err)
	}
	if m.tabs == nil {
		m.tabs = map[string]*Conn{}
	}
	m.tabs[target.ID] = conn
	m.activePageID = target.ID
	m.epoch = 0
	return TabInfo{ID: target.ID, Title: target.Title, URL: target.URL}, nil
}

// SwitchTab 切换 active 到指定 tab：复用已有会话或按 /json/list 真源现拨；
// 切换后 epoch 清零（旧页 refs 诚实失效，需重新 snapshot）。未知 id →
// ErrInvalidInput。
func (m *Manager) SwitchTab(ctx context.Context, id string) (TabInfo, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return TabInfo{}, fmt.Errorf("%w: tab_id 必填", ErrInvalidInput)
	}
	if err := m.Ensure(ctx); err != nil {
		return TabInfo{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	target, err := findPageTarget(ctx, m.httpBase, id)
	if err != nil {
		return TabInfo{}, err
	}
	if target == nil {
		return TabInfo{}, fmt.Errorf("%w: 未知 tab %q（用 browser_tabs 查看当前标签）", ErrInvalidInput, id)
	}
	conn, ok := m.tabs[id]
	if !ok { // 已列举但未拨号（如页面内 window.open 的目标）：按需 dial
		if target.WSURL == "" {
			return TabInfo{}, fmt.Errorf("browser: page target %s 缺少 webSocketDebuggerUrl", target.ID)
		}
		conn, err = Dial(ctx, target.WSURL)
		if err != nil {
			return TabInfo{}, err
		}
		if err := conn.Call(ctx, "Page.enable", map[string]any{}, nil); err != nil {
			_ = conn.Close()
			return TabInfo{}, fmt.Errorf("browser: Page.enable 失败: %w", err)
		}
		m.tabs[id] = conn
	}
	m.activePageID = id
	m.epoch = 0
	return TabInfo{ID: target.ID, Title: target.Title, URL: target.URL}, nil
}

// CloseTab 关闭指定标签页：closePageTarget → 摘表关 conn；关的是 active →
// 切到剩余第一个 tab（epoch 清零）；最后一个 → 整体 teardownLocked。未知
// id → ErrInvalidInput（已整体关闭时幂等返回 nil）。
func (m *Manager) CloseTab(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: tab_id 必填", ErrInvalidInput)
	}
	if err := m.Ensure(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.tabs) == 0 && m.httpBase == "" {
		return nil // 浏览器整体已关：幂等
	}
	if _, ok := m.tabs[id]; !ok {
		// 未拨号的目标（如页面内 window.open 弹出的页）：按 /json/list 真源
		// 校验存在性，仍可经 HTTP 关闭。
		target, err := findPageTarget(ctx, m.httpBase, id)
		if err != nil {
			return err
		}
		if target == nil {
			return fmt.Errorf("%w: 未知 tab %q（用 browser_tabs 查看当前标签）", ErrInvalidInput, id)
		}
	}
	_ = closePageTarget(ctx, m.httpBase, id)
	if conn, ok := m.tabs[id]; ok {
		_ = conn.Close()
		delete(m.tabs, id)
	}
	if id == m.activePageID {
		m.activePageID = ""
		m.epoch = 0
		for tid := range m.tabs { // 切到剩余第一个（map 序，仅需任一个）
			m.activePageID = tid
			break
		}
	}
	if len(m.tabs) == 0 {
		m.teardownLocked()
	}
	return nil
}

// findPageTarget 在 /json/list 中按 id 查找 page target（找不到 → nil, nil）。
func findPageTarget(ctx context.Context, httpBase, id string) (*Target, error) {
	targets, err := listTargets(ctx, httpBase)
	if err != nil {
		return nil, fmt.Errorf("browser: 列举目标失败: %w", err)
	}
	for i := range targets {
		if targets[i].Type == "page" && targets[i].ID == id {
			return &targets[i], nil
		}
	}
	return nil, nil
}

// ValidateURL 导航 URL 白名单：只接受绝对 http/https 地址（拒绝 file:
// /javascript:/data:/about: 等scheme，报错说明）；127.0.0.1/localhost 天然放行。
func ValidateURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%w: %q 不是合法的绝对 URL", ErrInvalidInput, raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: 仅支持 http/https，拒绝 %s: 地址", ErrInvalidInput, u.Scheme)
	}
	return nil
}

// ── 页面操作 ────────────────────────────────────────────────────────────

// NavigateResult 导航结果（含落点上下文）。
type NavigateResult struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// Navigate 打开 URL：Page.navigate → 等 Page.loadEventFired（超时兜底轮询
// document.readyState）。导航令 snapshot refs 失效（epoch 清零）。作用于
// active tab。
func (m *Manager) Navigate(ctx context.Context, rawURL string, timeoutSecs int) (NavigateResult, error) {
	if err := ValidateURL(rawURL); err != nil {
		return NavigateResult{}, err
	}
	if err := m.Ensure(ctx); err != nil {
		return NavigateResult{}, err
	}
	wait := navTimeout(timeoutSecs)

	m.mu.Lock()
	conn := m.tabs[m.activePageID]
	m.mu.Unlock()
	if conn == nil {
		return NavigateResult{}, errors.New("browser: 会话未建立")
	}

	// 清残留事件，防上一次导航的 loadEventFired 误配本次等待。
	conn.drainEvents()
	var nav struct {
		ErrorText string `json:"errorText"`
	}
	if err := conn.Call(ctx, "Page.navigate", map[string]any{"url": rawURL}, &nav); err != nil {
		return NavigateResult{}, fmt.Errorf("browser: Page.navigate 失败: %w", err)
	}
	if nav.ErrorText != "" {
		return NavigateResult{}, fmt.Errorf("browser: 页面导航失败: %s", nav.ErrorText)
	}

	m.mu.Lock()
	m.epoch = 0 // 导航后旧 refs 一律失效
	m.mu.Unlock()

	if !conn.waitEvent(ctx, "Page.loadEventFired", wait) {
		// 事件兜底：轮询 readyState（SPA/already-loaded 等不发 load 的情况）
		if err := waitReadyState(ctx, conn, wait); err != nil {
			return NavigateResult{}, err
		}
	}
	var meta struct {
		okField
		Title string `json:"title"`
		URL   string `json:"url"`
	}
	if err := m.evaluate(ctx, jsMeta, &meta); err != nil {
		return NavigateResult{}, err
	}
	if !meta.OK {
		return NavigateResult{}, errors.New(meta.Error)
	}
	res := NavigateResult{URL: meta.URL, Title: meta.Title}
	if res.URL == "" {
		res.URL = rawURL
	}
	return res, nil
}

// navTimeout 归一化导航等待时长：0 取默认，钳制 [5s, 120s]。
func navTimeout(secs int) time.Duration {
	if secs <= 0 {
		return defaultNavTimeout
	}
	d := time.Duration(secs) * time.Second
	if d < 5*time.Second {
		return 5 * time.Second
	}
	if d > 120*time.Second {
		return 120 * time.Second
	}
	return d
}

// waitReadyState 轮询 document.readyState==complete（200ms 间隔，deadline 兜底）。
func waitReadyState(ctx context.Context, conn *Conn, wait time.Duration) error {
	deadline := time.Now().Add(wait)
	for {
		var resp struct {
			Result struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"result"`
		}
		_ = conn.Call(ctx, "Runtime.evaluate", map[string]any{
			"expression": "document.readyState", "returnByValue": true,
		}, &resp) // 解析失败按未就绪继续轮询
		if resp.Result.Value == "complete" {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w: 页面加载超时（%.0fs 内未完成）", context.DeadlineExceeded, wait.Seconds())
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// ReadResult 页面文本。
type ReadResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Text  string `json:"text"`
}

// Read 读取 active tab 页面文本（selector 为空读全文，否则读该元素；
// maxChars 截断）。frame 非空时改在指定 iframe 内读取（frame = iframe 的
// frame URL 子串或 CSS 选择器）。
func (m *Manager) Read(ctx context.Context, selector string, maxChars int, frame string) (ReadResult, error) {
	if err := m.Ensure(ctx); err != nil {
		return ReadResult{}, err
	}
	if maxChars <= 0 {
		maxChars = defaultReadChars
	}
	if maxChars > maxReadChars {
		maxChars = maxReadChars
	}
	contextID, err := m.frameContextID(ctx, frame)
	if err != nil {
		return ReadResult{}, err
	}
	var res struct {
		okField
		Title string `json:"title"`
		URL   string `json:"url"`
		Text  string `json:"text"`
	}
	if err := m.evaluateIn(ctx, fmt.Sprintf(jsRead, maxChars, jsString(selector)), contextID, &res); err != nil {
		return ReadResult{}, err
	}
	if !res.OK {
		return ReadResult{}, jsErr(res.Error)
	}
	return ReadResult{Title: res.Title, URL: res.URL, Text: res.Text}, nil
}

// RefItem snapshot 标记出的一个可交互元素。
type RefItem struct {
	Ref  int    `json:"ref"`
	Tag  string `json:"tag"`
	Text string `json:"text"`
	Path string `json:"path"` // CSS 选择器路径（ref 失效时的兜底定位）
}

// SnapshotResult 交互元素清单 + 页面上下文。
type SnapshotResult struct {
	Title string    `json:"title"`
	URL   string    `json:"url"`
	Items []RefItem `json:"items"`
}

// Snapshot 给 active tab 的可交互元素标 data-gaea-ref 并返回紧凑清单；refs
// 跨调用持久，navigate/切页后由 epoch 机制判失效。
func (m *Manager) Snapshot(ctx context.Context) (SnapshotResult, error) {
	if err := m.Ensure(ctx); err != nil {
		return SnapshotResult{}, err
	}
	var res struct {
		okField
		Title string    `json:"title"`
		URL   string    `json:"url"`
		Epoch int64     `json:"epoch"`
		Items []RefItem `json:"items"`
	}
	if err := m.evaluate(ctx, fmt.Sprintf(jsSnapshot, snapshotMaxRefs), &res); err != nil {
		return SnapshotResult{}, err
	}
	if !res.OK {
		return SnapshotResult{}, jsErr(res.Error)
	}
	m.mu.Lock()
	m.epoch = res.Epoch
	m.mu.Unlock()
	if res.Items == nil {
		res.Items = []RefItem{}
	}
	return SnapshotResult{Title: res.Title, URL: res.URL, Items: res.Items}, nil
}

// ActionResult 点击/输入/滚动的通用结果。
type ActionResult struct {
	Text string `json:"text"` // 元素文本（click）或输入值（type）或落点（scroll）
}

// Click 点击 active tab 元素：ref（snapshot 返回，优先）或 CSS selector，
// 二选一。frame 非空时改在指定 iframe 内点击（仅支持 selector——ref 属主
// 文档，跨文档会撞号；iframe 隔离世界无 __gaeaEpoch，不做 ref 失效守门）。
func (m *Manager) Click(ctx context.Context, ref int, selector, frame string) (ActionResult, error) {
	if strings.TrimSpace(frame) != "" {
		return m.clickInFrame(ctx, ref, selector, frame)
	}
	target, err := m.resolveTarget(ctx, ref, selector)
	if err != nil {
		return ActionResult{}, err
	}
	var res struct {
		okField
		Text string `json:"text"`
	}
	if err := m.evaluate(ctx, fmt.Sprintf(jsClick, jsString(target)), &res); err != nil {
		return ActionResult{}, err
	}
	if !res.OK {
		return ActionResult{}, jsErr(res.Error)
	}
	return ActionResult{Text: res.Text}, nil
}

// clickInFrame 在 iframe 内点击：仅 selector 定位；校验先于 Ensure（任何浏览
// 器动作之前被拒）。
func (m *Manager) clickInFrame(ctx context.Context, ref int, selector, frame string) (ActionResult, error) {
	if ref > 0 {
		return ActionResult{}, fmt.Errorf("%w: frame 模式仅支持 selector 定位（ref 属主文档，跨文档会撞号）", ErrInvalidInput)
	}
	if strings.TrimSpace(selector) == "" {
		return ActionResult{}, fmt.Errorf("%w: frame 模式需 selector 定位元素", ErrInvalidInput)
	}
	if err := m.Ensure(ctx); err != nil {
		return ActionResult{}, err
	}
	contextID, err := m.frameContextID(ctx, frame)
	if err != nil {
		return ActionResult{}, err
	}
	var res struct {
		okField
		Text string `json:"text"`
	}
	if err := m.evaluateIn(ctx, fmt.Sprintf(jsClick, jsString(selector)), contextID, &res); err != nil {
		return ActionResult{}, err
	}
	if !res.OK {
		return ActionResult{}, jsErr(res.Error)
	}
	return ActionResult{Text: res.Text}, nil
}

// Type 向 active tab 的输入元素输入文本：ref 或 selector 定位；原生 setter
// + input/change 事件（React 兼容）；submit 时请求提交所在表单。frame 非空
// 时改在指定 iframe 内输入（仅支持 selector，同 Click 的 frame 语义）。
func (m *Manager) Type(ctx context.Context, ref int, selector, text string, submit bool, frame string) (ActionResult, error) {
	if text == "" {
		return ActionResult{}, fmt.Errorf("%w: text 必填", ErrInvalidInput)
	}
	if strings.TrimSpace(frame) != "" {
		return m.typeInFrame(ctx, ref, selector, text, submit, frame)
	}
	target, err := m.resolveTarget(ctx, ref, selector)
	if err != nil {
		return ActionResult{}, err
	}
	var res struct {
		okField
		Text string `json:"text"`
	}
	expr := fmt.Sprintf(jsType, jsString(target), jsString(text), boolJS(submit))
	if err := m.evaluate(ctx, expr, &res); err != nil {
		return ActionResult{}, err
	}
	if !res.OK {
		return ActionResult{}, jsErr(res.Error)
	}
	return ActionResult{Text: res.Text}, nil
}

// typeInFrame 在 iframe 内输入：仅 selector 定位；校验先于 Ensure。
func (m *Manager) typeInFrame(ctx context.Context, ref int, selector, text string, submit bool, frame string) (ActionResult, error) {
	if ref > 0 {
		return ActionResult{}, fmt.Errorf("%w: frame 模式仅支持 selector 定位（ref 属主文档，跨文档会撞号）", ErrInvalidInput)
	}
	if strings.TrimSpace(selector) == "" {
		return ActionResult{}, fmt.Errorf("%w: frame 模式需 selector 定位元素", ErrInvalidInput)
	}
	if err := m.Ensure(ctx); err != nil {
		return ActionResult{}, err
	}
	contextID, err := m.frameContextID(ctx, frame)
	if err != nil {
		return ActionResult{}, err
	}
	var res struct {
		okField
		Text string `json:"text"`
	}
	expr := fmt.Sprintf(jsType, jsString(selector), jsString(text), boolJS(submit))
	if err := m.evaluateIn(ctx, expr, contextID, &res); err != nil {
		return ActionResult{}, err
	}
	if !res.OK {
		return ActionResult{}, jsErr(res.Error)
	}
	return ActionResult{Text: res.Text}, nil
}

// Scroll 滚动 active tab 页面：direction=up/down，amount 像素；selector
// 限定滚动容器。
func (m *Manager) Scroll(ctx context.Context, direction string, amount int, containerSelector string) (ActionResult, error) {
	switch direction {
	case "up", "down":
	case "":
		direction = "down"
	default:
		return ActionResult{}, fmt.Errorf("%w: direction 仅支持 up/down，得到 %q", ErrInvalidInput, direction)
	}
	if amount <= 0 {
		amount = defaultScrollPx
	}
	if amount > maxScrollPx {
		amount = maxScrollPx
	}
	var res struct {
		okField
		Text string `json:"top"`
	}
	expr := fmt.Sprintf(jsScroll, jsString(direction), amount, jsString(containerSelector))
	if err := m.evaluate(ctx, expr, &res); err != nil {
		return ActionResult{}, err
	}
	if !res.OK {
		return ActionResult{}, jsErr(res.Error)
	}
	return ActionResult{Text: res.Text}, nil
}

// resolveTarget 归一化元素定位：ref 优先（拼 data-gaea-ref 选择器），否则用
// selector；先校验 epoch 防跨导航/跨页的失效 ref。
func (m *Manager) resolveTarget(ctx context.Context, ref int, selector string) (string, error) {
	if ref <= 0 && strings.TrimSpace(selector) == "" {
		return "", fmt.Errorf("%w: ref 与 selector 至少提供一个（先 browser_snapshot 获取 ref）", ErrInvalidInput)
	}
	if err := m.Ensure(ctx); err != nil {
		return "", err
	}
	if err := m.guardEpoch(ctx); err != nil {
		return "", err
	}
	if ref > 0 {
		return fmt.Sprintf(`[data-gaea-ref="%d"]`, ref), nil
	}
	return strings.TrimSpace(selector), nil
}

// guardEpoch 校验 snapshot refs 仍有效：页内 __gaeaEpoch 必须等于管理器记录值
// （0 表示从未 snapshot；页面跳转/切换标签后页内变量被清空或变化 → 强制重新
// snapshot）。
func (m *Manager) guardEpoch(ctx context.Context) error {
	m.mu.Lock()
	want := m.epoch
	m.mu.Unlock()
	var got float64
	if err := m.evaluate(ctx, "window.__gaeaEpoch||0", &got); err != nil {
		return err
	}
	if want == 0 || int64(got) != want {
		return fmt.Errorf("%w: 页面已跳转或尚未 snapshot，请重新 browser_snapshot 获取 ref", ErrRefsStale)
	}
	return nil
}

// ── Runtime.evaluate 封装与 JS 片段 ─────────────────────────────────────

// okField 每段页面 JS 返回值的公共头：JS 内 try/catch，失败带 error 文本。
type okField struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// evaluate 在 active tab 的默认执行上下文执行一段 JS 并把返回值解进 out
// （returnByValue + awaitPromise）。
func (m *Manager) evaluate(ctx context.Context, expression string, out any) error {
	return m.evaluateIn(ctx, expression, 0, out)
}

// evaluateIn 同 evaluate，但可指定 executionContextId（>0 = iframe 隔离世界；
// 0 = 主文档默认上下文）。
func (m *Manager) evaluateIn(ctx context.Context, expression string, contextID int, out any) error {
	m.mu.Lock()
	conn := m.tabs[m.activePageID]
	m.mu.Unlock()
	if conn == nil {
		return errors.New("browser: 会话未建立")
	}
	var resp struct {
		Result struct {
			Type        string          `json:"type"`
			Value       json.RawMessage `json:"value"`
			Description string          `json:"description"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	params := map[string]any{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
	}
	if contextID > 0 {
		params["contextId"] = contextID
	}
	if err := conn.Call(ctx, "Runtime.evaluate", params, &resp); err != nil {
		return err
	}
	if resp.ExceptionDetails != nil {
		d := resp.ExceptionDetails.Exception.Description
		if d == "" {
			d = resp.ExceptionDetails.Text
		}
		return fmt.Errorf("browser: JS 执行失败: %s", d)
	}
	if out != nil && len(resp.Result.Value) > 0 {
		if err := json.Unmarshal(resp.Result.Value, out); err != nil {
			return fmt.Errorf("browser: JS 返回值解析失败: %w", err)
		}
	}
	return nil
}

// jsErr 把页面 JS 的 {ok:false,error} 转成 Go 错误（未找到元素 → 语义化哨兵）。
func jsErr(msg string) error {
	if strings.Contains(msg, "未找到元素") {
		return fmt.Errorf("%w: %s", ErrElementNotFound, msg)
	}
	return errors.New(msg)
}

// jsString 把 Go 字符串编码成 JS 字符串字面量（json.Marshal 转义即安全注入）。
func jsString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

// boolJS 布尔值注入。
func boolJS(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// jsMeta 取标题与落点 URL（导航后调用）。__gaeaMeta 为 fake CDP 的匹配 token。
const jsMeta = `(function(){try{var t='__gaeaMeta';return {ok:true,title:document.title||"",url:location.href};}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`

// jsRead 读全文或 selector 局部文本（limit 参数 %d，selector 字面量 %s）。
const jsRead = `(function(){try{
var limit=%d; var sel=%s; var text;
if(sel){var el=document.querySelector(sel);if(!el)return {ok:false,error:"未找到元素："+sel};text=el.innerText||"";}
else{text=document.body?document.body.innerText:"";}
if(text.length>limit)text=text.slice(0,limit)+"…[已截断]";
return {ok:true,title:document.title||"",url:location.href,text:text};
}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`

// jsSnapshot 给可交互元素标 data-gaea-ref，登记 window.__gaeaRefs 与
// __gaeaEpoch（refs 代数，跨调用持久、导航后随页面重置）。
const jsSnapshot = `(function(){try{
var MAXREF=%d;
function cssPath(el){
 if(el.id)return '#'+el.id;
 var parts=[],cur=el,depth=0;
 while(cur&&cur.nodeType===1&&depth<4){
  depth++;
  if(cur.id){parts.unshift('#'+cur.id);break;}
  var seg=(cur.tagName||'').toLowerCase();
  var p=cur.parentNode;
  if(p&&p.children){
   var same=0,idx=0;
   for(var i=0;i<p.children.length;i++){
    if(p.children[i].tagName===cur.tagName){same++;if(p.children[i]===cur)idx=same;}
   }
   if(same>1&&idx>0)seg+=':nth-of-type('+idx+')';
  }
  parts.unshift(seg);
  cur=cur.parentNode;
 }
 return parts.join(' > ');
}
var nodes=document.querySelectorAll('a,button,input,textarea,select,[onclick],[role=button]');
var items=[];
for(var i=0;i<nodes.length&&items.length<MAXREF;i++){
 var el=nodes[i];
 if(el.disabled)continue;
 var r=el.getBoundingClientRect();
 if(r.width===0&&r.height===0)continue;
 var ref=items.length+1;
 el.setAttribute('data-gaea-ref',String(ref));
 var tag=(el.tagName||'').toLowerCase();
 var text=(tag==='input'||tag==='textarea')?(el.value||el.placeholder||''):(el.innerText||el.getAttribute('aria-label')||'');
 text=String(text).replace(/\s+/g,' ').trim().slice(0,80);
 items.push({ref:ref,tag:tag,text:text,path:cssPath(el)});
}
window.__gaeaRefs=items;
window.__gaeaEpoch=(window.__gaeaEpoch||0)+1;
return {ok:true,title:document.title||"",url:location.href,epoch:window.__gaeaEpoch,items:items};
}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`

// jsClick 按目标选择器点击（先滚到视野中央），返回元素文本确认。
const jsClick = `(function(){try{
var op='gaeaClick'; var target=%s;
var el=document.querySelector(target);
if(!el)return {ok:false,error:"未找到元素："+target};
el.scrollIntoView({block:'center'});
el.click();
var t=(el.innerText||el.value||'').trim().slice(0,120);
return {ok:true,text:t};
}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`

// jsType 聚焦并写入值：input/textarea 用原生 setter + input/change 事件
// （React 受控组件兼容）；submit 时提交所在表单或退化点击。
const jsType = `(function(){try{
var op='gaeaType'; var target=%s; var value=%s; var doSubmit=%s;
var el=document.querySelector(target);
if(!el)return {ok:false,error:"未找到元素："+target};
el.scrollIntoView({block:'center'});
el.focus();
var tag=(el.tagName||'').toLowerCase();
if(tag==='input'||tag==='textarea'){
 var proto=tag==='input'?window.HTMLInputElement.prototype:window.HTMLTextAreaElement.prototype;
 var desc=Object.getOwnPropertyDescriptor(proto,'value');
 if(desc&&desc.set){desc.set.call(el,value);}else{el.value=value;}
 el.dispatchEvent(new Event('input',{bubbles:true}));
 el.dispatchEvent(new Event('change',{bubbles:true}));
}else if(el.isContentEditable){
 el.textContent=value;
 el.dispatchEvent(new Event('input',{bubbles:true}));
}else{
 el.value=value;
 el.dispatchEvent(new Event('change',{bubbles:true}));
}
if(doSubmit){
 var f=el.form||el.closest('form');
 if(f){if(f.requestSubmit){f.requestSubmit();}else{f.submit();}}
 else{el.click();}
}
return {ok:true,text:String(value).slice(0,120)};
}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`

// jsScroll 窗口滚动或容器内滚动。
const jsScroll = `(function(){try{
var op='gaeaScroll'; var dir=%s; var amt=%d; var container=%s;
var delta=dir==='up'?-amt:amt;
if(container){
 var c=document.querySelector(container);
 if(!c)return {ok:false,error:"未找到元素："+container};
 c.scrollBy(0,delta);
 return {ok:true,top:String(c.scrollTop)};
}
window.scrollBy(0,delta);
return {ok:true,top:String(window.scrollY)};
}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`
