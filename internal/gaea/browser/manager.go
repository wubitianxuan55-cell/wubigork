package browser

// manager.go — 浏览器 Manager 核心生命周期（批 26 GA5-08 文件拆分：原 1068 行
// 五职责单文件按既有分节注释分家，本文件留「核心」——Options/NewManager/
// Ensure 连接保障/attach/teardown/Shutdown/ClosePage/空闲看门狗。多标签页见
// manager_tabs.go，页面操作见 manager_actions.go，Runtime.evaluate 与 JS 片段
// 见 manager_evaluate.go。均为同包原位搬移，零逻辑改动）。

import (
	"context"
	"errors"
	"fmt"
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
