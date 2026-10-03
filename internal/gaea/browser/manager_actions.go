package browser

// manager_actions.go — 页面操作（批 26 GA5-08 文件拆分，自 manager.go 原位
// 搬移，零逻辑改动）：Navigate/Read/Snapshot/Click/Type/Scroll（含 frame 变体
// 与 ref/selector 解析 resolveTarget、代际守卫 guardEpoch）。多标签页见
// manager_tabs.go，Runtime.evaluate 与 JS 片段见 manager_evaluate.go。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

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
