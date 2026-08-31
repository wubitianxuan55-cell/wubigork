package builtin

// browser_tools.go — 受控浏览器自动化 MVP（browser 包薄封装）：
// 懒拉起独立 Edge（独立临时 profile，绝不碰用户主 profile），10 个工具共享
// browser.Default() 单例会话。多标签支持 browser_tabs / browser_new_tab /
// browser_switch_tab / browser_close(tab_id)；空闲 TTL 自动关停，任意
// browser_* 调用自动重拉。结果统一走 envelope 结构化返回；错误用 errors.Is
// 映射语义化 code（validation_error/not_found/stale_refs/timeout）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/browser"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() {
	tool.RegisterBuiltin(browserNavigate{})
	tool.RegisterBuiltin(browserRead{})
	tool.RegisterBuiltin(browserSnapshot{})
	tool.RegisterBuiltin(browserClick{})
	tool.RegisterBuiltin(browserType{})
	tool.RegisterBuiltin(browserScroll{})
	tool.RegisterBuiltin(browserTabs{})
	tool.RegisterBuiltin(browserNewTab{})
	tool.RegisterBuiltin(browserSwitchTab{})
	tool.RegisterBuiltin(browserClose{})
}

// browserErrCode 把 browser 包错误映射为 envelope code。
func browserErrCode(err error) string {
	switch {
	case errors.Is(err, browser.ErrInvalidInput):
		return tool.CodeValidationError
	case errors.Is(err, browser.ErrElementNotFound):
		return tool.CodeNotFound
	case errors.Is(err, browser.ErrRefsStale):
		return "stale_refs"
	case errors.Is(err, context.DeadlineExceeded):
		return tool.CodeTimeout
	default:
		return tool.CodeExecError
	}
}

// browserFail 统一错误出口（envelope 承载错误，不再走 Go error 通道）。
func browserFail(err error, data map[string]any) (string, error) {
	return tool.WrapError(browserErrCode(err), err.Error(), data), nil
}

// ── browser_navigate ────────────────────────────────────────────────────

type browserNavigate struct{}

func (browserNavigate) Name() string { return "browser_navigate" }

func (browserNavigate) Description() string {
	return "在受控 Edge 浏览器中打开 URL（仅 http/https；使用独立临时 profile，不影响你的浏览器）。首次调用会自动拉起浏览器。打开后用 browser_read 读文本、browser_snapshot 拿可交互元素清单。"
}

func (browserNavigate) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "url":{"type":"string","description":"要打开的绝对 URL，仅支持 http:// 或 https://"},
  "timeout_secs":{"type":"integer","description":"页面加载等待上限（秒，默认 20，范围 5-120）","minimum":5,"maximum":120}
},
"required":["url"]
}`)
}

func (browserNavigate) ReadOnly() bool                 { return false }
func (browserNavigate) SpaceTag() string               { return spaces.SpaceWork }
func (browserNavigate) CompactDescription() string     { return compactDesc["browser_navigate"] }
func (browserNavigate) CompactSchema() json.RawMessage { return compactSchema["browser_navigate"] }

func (browserNavigate) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		URL         string `json:"url"`
		TimeoutSecs int    `json:"timeout_secs"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.WrapError(tool.CodeValidationError, "invalid args: "+err.Error(), nil), nil
	}
	if strings.TrimSpace(p.URL) == "" {
		return tool.WrapError(tool.CodeValidationError, "url 必填（仅支持 http/https）", nil), nil
	}
	res, err := browser.Default().Navigate(ctx, p.URL, p.TimeoutSecs)
	if err != nil {
		return browserFail(err, map[string]any{"url": p.URL})
	}
	return tool.WrapResult("ok", map[string]any{
		"url":     res.URL,
		"title":   res.Title,
		"message": "已打开；用 browser_snapshot 获取可交互元素 ref，browser_read 读取文本",
	}), nil
}

// ── browser_read ────────────────────────────────────────────────────────

type browserRead struct{}

func (browserRead) Name() string { return "browser_read" }

func (browserRead) Description() string {
	return "读取受控浏览器当前页面的文本（默认全文，传 selector 只读该元素；max_chars 截断，默认 6000）。只读不改变页面。"
}

func (browserRead) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "selector":{"type":"string","description":"CSS 选择器，只读取该元素的文本（缺省读全文）"},
  "max_chars":{"type":"integer","description":"返回文本最大字符数（默认 6000）","minimum":1}
}
}`)
}

func (browserRead) ReadOnly() bool                 { return true }
func (browserRead) SpaceTag() string               { return spaces.SpaceWork }
func (browserRead) CompactDescription() string     { return compactDesc["browser_read"] }
func (browserRead) CompactSchema() json.RawMessage { return compactSchema["browser_read"] }

func (browserRead) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Selector string `json:"selector"`
		MaxChars int    `json:"max_chars"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return tool.WrapError(tool.CodeValidationError, "invalid args: "+err.Error(), nil), nil
		}
	}
	res, err := browser.Default().Read(ctx, p.Selector, p.MaxChars)
	if err != nil {
		return browserFail(err, map[string]any{"selector": p.Selector})
	}
	return tool.WrapResult("ok", map[string]any{
		"url":   res.URL,
		"title": res.Title,
		"text":  res.Text,
	}), nil
}

// ── browser_snapshot ────────────────────────────────────────────────────

type browserSnapshot struct{}

func (browserSnapshot) Name() string { return "browser_snapshot" }

func (browserSnapshot) Description() string {
	return "列出受控浏览器当前页面的可交互元素（链接/按钮/输入框等），返回 [#ref] 清单与页面标题/URL。点击或输入前必须先 snapshot 获取 ref，再用 ref 调 browser_click / browser_type；页面跳转后 ref 失效，需重新 snapshot。"
}

func (browserSnapshot) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (browserSnapshot) ReadOnly() bool                 { return true }
func (browserSnapshot) SpaceTag() string               { return spaces.SpaceWork }
func (browserSnapshot) CompactDescription() string     { return compactDesc["browser_snapshot"] }
func (browserSnapshot) CompactSchema() json.RawMessage { return compactSchema["browser_snapshot"] }

func (browserSnapshot) Execute(ctx context.Context, _ json.RawMessage) (string, error) {
	snap, err := browser.Default().Snapshot(ctx)
	if err != nil {
		return browserFail(err, nil)
	}
	return tool.WrapResult("ok", map[string]any{
		"url":    snap.URL,
		"title":  snap.Title,
		"count":  len(snap.Items),
		"list":   browserSnapshotListing(snap),
		"notice": "先 snapshot 拿 ref，再 browser_click/browser_type；页面跳转后 ref 失效",
	}), nil
}

// browserSnapshotListing 生成紧凑清单：[#1] <a> 链接文字（文本为空时给选择器路径）。
func browserSnapshotListing(snap browser.SnapshotResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "页面: %s\nURL: %s\n可交互元素 %d 个:\n", snap.Title, snap.URL, len(snap.Items))
	for _, it := range snap.Items {
		if it.Text == "" {
			fmt.Fprintf(&b, "[#%d] <%s> @ %s\n", it.Ref, it.Tag, it.Path)
			continue
		}
		fmt.Fprintf(&b, "[#%d] <%s> %s\n", it.Ref, it.Tag, it.Text)
	}
	if len(snap.Items) == 0 {
		b.WriteString("(未发现可交互元素)\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// ── browser_click ───────────────────────────────────────────────────────

type browserClick struct{}

func (browserClick) Name() string { return "browser_click" }

func (browserClick) Description() string {
	return "点击受控浏览器页面上的元素：传 browser_snapshot 返回的 ref（推荐）或 CSS selector，二选一。点击可能触发页面跳转，跳转后 ref 失效需重新 snapshot。"
}

func (browserClick) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "ref":{"type":"integer","description":"browser_snapshot 返回的元素 ref（优先使用）"},
  "selector":{"type":"string","description":"CSS 选择器（ref 缺失时的兜底；与 ref 二选一）"}
}
}`)
}

func (browserClick) ReadOnly() bool                 { return false }
func (browserClick) SpaceTag() string               { return spaces.SpaceWork }
func (browserClick) CompactDescription() string     { return compactDesc["browser_click"] }
func (browserClick) CompactSchema() json.RawMessage { return compactSchema["browser_click"] }

func (browserClick) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Ref      int    `json:"ref"`
		Selector string `json:"selector"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.WrapError(tool.CodeValidationError, "invalid args: "+err.Error(), nil), nil
	}
	res, err := browser.Default().Click(ctx, p.Ref, p.Selector)
	if err != nil {
		return browserFail(err, map[string]any{"ref": p.Ref, "selector": p.Selector})
	}
	return tool.WrapResult("ok", map[string]any{
		"ref":      p.Ref,
		"selector": p.Selector,
		"element":  res.Text,
		"message":  "已点击；若页面跳转请重新 browser_snapshot",
	}), nil
}

// ── browser_type ────────────────────────────────────────────────────────

type browserType struct{}

func (browserType) Name() string { return "browser_type" }

func (browserType) Description() string {
	return "向受控浏览器页面的输入框输入文本：传 browser_snapshot 返回的 ref 或 CSS selector 定位（二选一）；submit=true 时提交所在表单。React/Vue 受控组件兼容。页面跳转后 ref 失效需重新 snapshot。"
}

func (browserType) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "ref":{"type":"integer","description":"browser_snapshot 返回的元素 ref（优先使用）"},
  "selector":{"type":"string","description":"CSS 选择器（ref 缺失时的兜底；与 ref 二选一）"},
  "text":{"type":"string","description":"要输入的文本"},
  "submit":{"type":"boolean","description":"输入后是否提交所在表单（默认 false）"}
},
"required":["text"]
}`)
}

func (browserType) ReadOnly() bool                 { return false }
func (browserType) SpaceTag() string               { return spaces.SpaceWork }
func (browserType) CompactDescription() string     { return compactDesc["browser_type"] }
func (browserType) CompactSchema() json.RawMessage { return compactSchema["browser_type"] }

func (browserType) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Ref      int    `json:"ref"`
		Selector string `json:"selector"`
		Text     string `json:"text"`
		Submit   bool   `json:"submit"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.WrapError(tool.CodeValidationError, "invalid args: "+err.Error(), nil), nil
	}
	if p.Text == "" {
		return tool.WrapError(tool.CodeValidationError, "text 必填", nil), nil
	}
	res, err := browser.Default().Type(ctx, p.Ref, p.Selector, p.Text, p.Submit)
	if err != nil {
		return browserFail(err, map[string]any{"ref": p.Ref, "selector": p.Selector})
	}
	return tool.WrapResult("ok", map[string]any{
		"ref":      p.Ref,
		"selector": p.Selector,
		"text":     res.Text,
		"submit":   p.Submit,
	}), nil
}

// ── browser_scroll ──────────────────────────────────────────────────────

type browserScroll struct{}

func (browserScroll) Name() string { return "browser_scroll" }

func (browserScroll) Description() string {
	return "滚动受控浏览器页面：direction=up/down，amount 像素（默认 800）；可选 selector 限定滚动容器。常与 browser_read/browser_snapshot 配合查看长页面。"
}

func (browserScroll) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "direction":{"type":"string","enum":["up","down"],"description":"滚动方向（缺省 down）"},
  "amount":{"type":"integer","description":"滚动像素（默认 800，上限 10000）","minimum":1},
  "selector":{"type":"string","description":"CSS 选择器：在该容器内滚动（缺省滚动整页）"}
}
}`)
}

func (browserScroll) ReadOnly() bool                 { return false }
func (browserScroll) SpaceTag() string               { return spaces.SpaceWork }
func (browserScroll) CompactDescription() string     { return compactDesc["browser_scroll"] }
func (browserScroll) CompactSchema() json.RawMessage { return compactSchema["browser_scroll"] }

func (browserScroll) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Direction string `json:"direction"`
		Amount    int    `json:"amount"`
		Selector  string `json:"selector"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return tool.WrapError(tool.CodeValidationError, "invalid args: "+err.Error(), nil), nil
		}
	}
	res, err := browser.Default().Scroll(ctx, p.Direction, p.Amount, p.Selector)
	if err != nil {
		return browserFail(err, map[string]any{"direction": p.Direction})
	}
	return tool.WrapResult("ok", map[string]any{
		"direction":  p.Direction,
		"scroll_top": res.Text,
	}), nil
}

// ── browser_close ───────────────────────────────────────────────────────

type browserClose struct{}

func (browserClose) Name() string { return "browser_close" }

func (browserClose) Description() string {
	return "关闭受控浏览器：缺省关闭当前页面并结束 Edge 进程、清理临时 profile（任务完成或不再需要浏览器时调用；下次 browser_* 调用会重新拉起）；传 tab_id 只关闭指定标签页（不结束浏览器，返回剩余标签数）。"
}

func (browserClose) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "tab_id":{"type":"string","description":"可选：只关闭该标签页（tab_id 从 browser_tabs 获取）；缺省关闭当前页并结束整个浏览器"}
}
}`)
}

func (browserClose) ReadOnly() bool                 { return false }
func (browserClose) SpaceTag() string               { return spaces.SpaceWork }
func (browserClose) CompactDescription() string     { return compactDesc["browser_close"] }
func (browserClose) CompactSchema() json.RawMessage { return compactSchema["browser_close"] }

func (browserClose) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		TabID string `json:"tab_id"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return tool.WrapError(tool.CodeValidationError, "invalid args: "+err.Error(), nil), nil
		}
	}
	if strings.TrimSpace(p.TabID) != "" {
		// 只关指定标签：关闭前先取标签总数，剩余 = 总数-1（避免关最后一个
		// 标签触发整体 teardown 后再 ListTabs 把浏览器重新拉起）。
		before, _, err := browser.Default().ListTabs(ctx)
		if err != nil {
			return browserFail(err, map[string]any{"tab_id": p.TabID})
		}
		if err := browser.Default().CloseTab(ctx, p.TabID); err != nil {
			return browserFail(err, map[string]any{"tab_id": p.TabID})
		}
		remaining := len(before) - 1
		if remaining < 0 {
			remaining = 0
		}
		return tool.WrapResult("ok", map[string]any{
			"tab_id":    p.TabID,
			"remaining": remaining,
			"message":   "标签页已关闭",
		}), nil
	}
	if err := browser.Default().ClosePage(ctx); err != nil {
		return browserFail(err, nil)
	}
	return tool.WrapResult("ok", map[string]any{"closed": true, "message": "浏览器已关闭，临时 profile 已清理"}), nil
}

// ── browser_tabs ────────────────────────────────────────────────────────

type browserTabs struct{}

func (browserTabs) Name() string { return "browser_tabs" }

func (browserTabs) Description() string {
	return "列出受控浏览器当前打开的全部标签页（[active] 标记当前页，含序号/标题/URL）。多标签时用 browser_new_tab 新建、browser_switch_tab 切换、browser_close(tab_id) 关闭。只读不改变页面。"
}

func (browserTabs) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (browserTabs) ReadOnly() bool                 { return true }
func (browserTabs) SpaceTag() string               { return spaces.SpaceWork }
func (browserTabs) CompactDescription() string     { return compactDesc["browser_tabs"] }
func (browserTabs) CompactSchema() json.RawMessage { return compactSchema["browser_tabs"] }

func (browserTabs) Execute(ctx context.Context, _ json.RawMessage) (string, error) {
	tabs, active, err := browser.Default().ListTabs(ctx)
	if err != nil {
		return browserFail(err, nil)
	}
	return tool.WrapResult("ok", map[string]any{
		"count":  len(tabs),
		"active": active,
		"list":   browserTabsListing(tabs, active),
		"tabs":   tabs, // 结构化列表（id/title/url），供 browser_switch_tab / browser_close(tab_id) 取 tab_id
	}), nil
}

// browserTabsListing 生成标签清单文本：[active] 或对齐空位 + #序号 标题 (URL)。
func browserTabsListing(tabs []browser.TabInfo, active string) string {
	var b strings.Builder
	for i, t := range tabs {
		mark := "        "
		if t.ID == active {
			mark = "[active]"
		}
		fmt.Fprintf(&b, "%s #%d %s (%s)\n", mark, i+1, t.Title, t.URL)
	}
	return strings.TrimRight(b.String(), "\n")
}

// ── browser_new_tab ─────────────────────────────────────────────────────

type browserNewTab struct{}

func (browserNewTab) Name() string { return "browser_new_tab" }

func (browserNewTab) Description() string {
	return "在受控浏览器中新建一个标签页并自动切换为当前页：url 必填（仅 http/https）。新建后原页面的 refs 已失效，需重新 browser_snapshot。"
}

func (browserNewTab) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "url":{"type":"string","description":"要打开的绝对 URL，仅支持 http:// 或 https://（新建后自动切换为当前页）"}
},
"required":["url"]
}`)
}

func (browserNewTab) ReadOnly() bool                 { return false }
func (browserNewTab) SpaceTag() string               { return spaces.SpaceWork }
func (browserNewTab) CompactDescription() string     { return compactDesc["browser_new_tab"] }
func (browserNewTab) CompactSchema() json.RawMessage { return compactSchema["browser_new_tab"] }

func (browserNewTab) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.WrapError(tool.CodeValidationError, "invalid args: "+err.Error(), nil), nil
	}
	if strings.TrimSpace(p.URL) == "" {
		return tool.WrapError(tool.CodeValidationError, "url 必填（仅支持 http/https）", nil), nil
	}
	tab, err := browser.Default().NewTab(ctx, p.URL)
	if err != nil {
		return browserFail(err, map[string]any{"url": p.URL})
	}
	return tool.WrapResult("ok", map[string]any{
		"tab_id":  tab.ID,
		"title":   tab.Title,
		"url":     tab.URL,
		"message": "已新建并切换；原页面的 refs 已失效，需重新 browser_snapshot",
	}), nil
}

// ── browser_switch_tab ──────────────────────────────────────────────────

type browserSwitchTab struct{}

func (browserSwitchTab) Name() string { return "browser_switch_tab" }

func (browserSwitchTab) Description() string {
	return "切换受控浏览器的当前标签页到 tab_id 指定的页（tab_id 从 browser_tabs 获取）。切换后原页面的 refs 已失效，需重新 browser_snapshot。"
}

func (browserSwitchTab) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "tab_id":{"type":"string","description":"要切换到的标签页 id（browser_tabs 返回）"}
},
"required":["tab_id"]
}`)
}

func (browserSwitchTab) ReadOnly() bool                 { return false }
func (browserSwitchTab) SpaceTag() string               { return spaces.SpaceWork }
func (browserSwitchTab) CompactDescription() string     { return compactDesc["browser_switch_tab"] }
func (browserSwitchTab) CompactSchema() json.RawMessage { return compactSchema["browser_switch_tab"] }

func (browserSwitchTab) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		TabID string `json:"tab_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.WrapError(tool.CodeValidationError, "invalid args: "+err.Error(), nil), nil
	}
	if strings.TrimSpace(p.TabID) == "" {
		return tool.WrapError(tool.CodeValidationError, "tab_id 必填（先 browser_tabs 查看标签）", nil), nil
	}
	tab, err := browser.Default().SwitchTab(ctx, p.TabID)
	if err != nil {
		return browserFail(err, map[string]any{"tab_id": p.TabID})
	}
	return tool.WrapResult("ok", map[string]any{
		"tab_id":  tab.ID,
		"title":   tab.Title,
		"message": "已切换；refs 已失效需重新 snapshot",
	}), nil
}
