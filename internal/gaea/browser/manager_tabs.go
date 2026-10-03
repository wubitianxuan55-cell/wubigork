package browser

// manager_tabs.go — 多标签页（批 26 GA5-08 文件拆分，自 manager.go 原位搬移，
// 零逻辑改动）：ListTabs/NewTab/SwitchTab/CloseTab + findPageTarget/ValidateURL。
// Manager 生命周期见 manager.go，页面操作见 manager_actions.go，Runtime.evaluate
// 与 JS 片段见 manager_evaluate.go。

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

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
	if _, ok := m.tabs[id]; !ok { // 已列举但未拨号（如页面内 window.open 的目标）：按需 dial
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
