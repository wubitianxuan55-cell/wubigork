package browser

// input.go — 键盘级 Input（CDP Input.dispatchKeyEvent / Input.insertText）与
// iframe 内交互（Page.getFrameTree + Page.createIsolatedWorld 定位 iframe 的
// 执行上下文，再在该上下文内执行既有 JS 片段）。browser_press 工具与
// browser_read/click/type 的可选 frame 参数都走这里；零新增绑定、零前端改动。

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ── 按键名归一 ───────────────────────────────────────────────────────────

// keyAliases 常见可读按键名 → CDP key 名（集中一处，可测）。单字符键
// （"a"/"A"/"1"）不在此表，直接透传。
var keyAliases = map[string]string{
	"enter":      "Enter",
	"esc":        "Escape",
	"escape":     "Escape",
	"tab":        "Tab",
	"space":      " ", // CDP 空格键 key 名是单个空格字符（code 才是 "Space"）
	"backspace":  "Backspace",
	"delete":     "Delete",
	"insert":     "Insert",
	"home":       "Home",
	"end":        "End",
	"pageup":     "PageUp",
	"pagedown":   "PageDown",
	"capslock":   "CapsLock",
	"arrowup":    "ArrowUp",
	"arrowdown":  "ArrowDown",
	"arrowleft":  "ArrowLeft",
	"arrowright": "ArrowRight",
	"f1":         "F1", "f2": "F2", "f3": "F3", "f4": "F4",
	"f5": "F5", "f6": "F6", "f7": "F7", "f8": "F8",
	"f9": "F9", "f10": "F10", "f11": "F11", "f12": "F12",
}

// normalizeKey 把用户可读键名归一为 CDP key 名：单字符原样返回；别名表命中
// 返回规范名；其余原样返回（信任 CDP 命名，如 ArrowDown）。
func normalizeKey(raw string) string {
	k := strings.TrimSpace(raw)
	if k == "" {
		return ""
	}
	if len(k) == 1 {
		return k
	}
	if v, ok := keyAliases[strings.ToLower(k)]; ok {
		return v
	}
	return k
}

// modifierAliases 修饰键可读名 → 规范名（组合键用）。
var modifierAliases = map[string]string{
	"ctrl": "ctrl", "control": "ctrl", "ctl": "ctrl",
	"alt": "alt", "option": "alt",
	"shift": "shift",
	"meta":  "meta", "command": "meta", "cmd": "meta", "win": "meta", "windows": "meta",
}

// modifierMask CDP 修饰键位掩码（Alt=1, Ctrl=2, Meta=4, Shift=8）。未知修饰
// 键 → ErrInvalidInput。
func modifierMask(mods []string) (int, error) {
	mask := 0
	for _, raw := range mods {
		c, ok := modifierAliases[strings.ToLower(strings.TrimSpace(raw))]
		if !ok {
			return 0, fmt.Errorf("%w: 未知修饰键 %q（支持 ctrl/alt/shift/meta）", ErrInvalidInput, raw)
		}
		switch c {
		case "ctrl":
			mask |= 2
		case "alt":
			mask |= 1
		case "shift":
			mask |= 8
		case "meta":
			mask |= 4
		}
	}
	return mask, nil
}

// windowsVKey CDP dispatchKeyEvent 需要的 Windows 虚拟键码：单字符取大写
// 码位；命名键查表；F1-F12 按 112-123；未知返回 0（调用方省略该字段）。
func windowsVKey(key string) int {
	if len(key) == 1 {
		r := int(key[0])
		if r >= 'a' && r <= 'z' {
			r -= 32
		}
		return r
	}
	switch key {
	case "Backspace":
		return 8
	case "Tab":
		return 9
	case "Enter":
		return 13
	case "Shift":
		return 16
	case "Control":
		return 17
	case "Alt":
		return 18
	case "CapsLock":
		return 20
	case "Escape":
		return 27
	case "Space":
		return 32
	case "PageUp":
		return 33
	case "PageDown":
		return 34
	case "End":
		return 35
	case "Home":
		return 36
	case "ArrowLeft":
		return 37
	case "ArrowUp":
		return 38
	case "ArrowRight":
		return 39
	case "ArrowDown":
		return 40
	case "Insert":
		return 45
	case "Delete":
		return 46
	}
	if len(key) >= 2 && key[0] == 'F' {
		if n, err := strconv.Atoi(key[1:]); err == nil && n >= 1 && n <= 12 {
			return 111 + n
		}
	}
	return 0
}

// keyCodeFor 派生 CDP code（物理键码，供 keydown 事件细节；无法派生返回空）。
func keyCodeFor(key string) (string, bool) {
	if len(key) == 1 {
		switch {
		case key[0] >= 'a' && key[0] <= 'z':
			return "Key" + strings.ToUpper(key), true
		case key[0] >= 'A' && key[0] <= 'Z':
			return "Key" + key, true
		case key[0] >= '0' && key[0] <= '9':
			return "Digit" + key, true
		}
		return "", false
	}
	switch key {
	case "Enter", "Tab", "Escape", "Backspace", "Delete", "Space", "Home", "End",
		"PageUp", "PageDown", "CapsLock", "Insert",
		"ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight":
		return key, true
	}
	return "", false
}

// ── Press ───────────────────────────────────────────────────────────────

// Press 在 active tab 发送一次键盘级按键（CDP Input.dispatchKeyEvent）：
// keyDown → [可选 Input.insertText] → keyUp。modifiers 为同时按住的修饰键
// （ctrl/alt/shift/meta，可读别名兼容）；text 非空时在按键后真实输入该文本
// （IME 友好，作用于当前聚焦元素）。错误语义与 evaluate 一致（CDP 调用失败
// 原样上抛，由 builtin 层映射 envelope code）。
func (m *Manager) Press(ctx context.Context, rawKey string, modifiers []string, text string) (ActionResult, error) {
	key := normalizeKey(rawKey)
	if key == "" {
		return ActionResult{}, fmt.Errorf("%w: key 必填（如 Enter/Tab/Escape/a）", ErrInvalidInput)
	}
	mask, err := modifierMask(modifiers)
	if err != nil {
		return ActionResult{}, err
	}
	if err := m.Ensure(ctx); err != nil {
		return ActionResult{}, err
	}
	m.mu.Lock()
	conn := m.tabs[m.activePageID]
	m.mu.Unlock()
	if conn == nil {
		return ActionResult{}, errors.New("browser: 会话未建立")
	}

	// keyDown：字符键带 text（触发字符输入事件），控制键用 rawKeyDown（只
	// 触发 keydown 事件，不产生字符）。
	params := map[string]any{
		"type": "rawKeyDown", "key": key, "modifiers": mask,
	}
	if vkey := windowsVKey(key); vkey > 0 {
		params["windowsVirtualKeyCode"] = vkey
	}
	if code, ok := keyCodeFor(key); ok {
		params["code"] = code
	}
	if len(key) == 1 {
		params["type"] = "keyDown"
		params["text"] = key
		params["unmodifiedText"] = key
	}
	if err := conn.Call(ctx, "Input.dispatchKeyEvent", params, nil); err != nil {
		return ActionResult{}, fmt.Errorf("browser: 按键下发失败: %w", err)
	}
	// Enter 等会"产生字符"的控制键：rawKeyDown 只触发 keydown，不发 keypress，
	// 而 Chromium 的隐式表单提交/按钮默认动作是 keypress 的默认行为——补一个
	// 带 text 的 keyDown（"\r"）触发之。
	if key == "Enter" {
		if err := conn.Call(ctx, "Input.dispatchKeyEvent", map[string]any{
			"type": "keyDown", "key": key, "text": "\r", "modifiers": mask,
		}, nil); err != nil {
			return ActionResult{}, fmt.Errorf("browser: 按键字符事件失败: %w", err)
		}
	}
	// 显式文本：insertText（聚焦元素收到真实输入；组合键场景如 Shift+key
	// 的目标字符由调用方在 text 里给出）。
	if text != "" {
		if err := conn.Call(ctx, "Input.insertText", map[string]any{"text": text}, nil); err != nil {
			return ActionResult{}, fmt.Errorf("browser: 文本输入失败: %w", err)
		}
	}
	up := map[string]any{"type": "keyUp", "key": key, "modifiers": mask}
	if vkey := windowsVKey(key); vkey > 0 {
		up["windowsVirtualKeyCode"] = vkey
	}
	if err := conn.Call(ctx, "Input.dispatchKeyEvent", up, nil); err != nil {
		return ActionResult{}, fmt.Errorf("browser: 按键释放失败: %w", err)
	}
	msg := "已发送按键"
	if text != "" {
		msg += "，并输入文本"
	}
	return ActionResult{Text: msg}, nil
}

// ── iframe 内交互 ───────────────────────────────────────────────────────

// frameInfo CDP Page.Frame 的精简视图。
type frameInfo struct {
	ID       string
	ParentID string
	URL      string
}

// frameNode Page.getFrameTree 返回的递归树节点。
type frameNode struct {
	Frame struct {
		ID       string `json:"id"`
		ParentID string `json:"parentId"`
		URL      string `json:"url"`
	} `json:"frame"`
	ChildFrames []frameNode `json:"childFrames"`
}

// flattenFrameTree 递归摊平 frame 树。
func flattenFrameTree(n frameNode, out *[]frameInfo) {
	*out = append(*out, frameInfo{ID: n.Frame.ID, ParentID: n.Frame.ParentID, URL: n.Frame.URL})
	for _, c := range n.ChildFrames {
		flattenFrameTree(c, out)
	}
}

// looksLikeURL frame 参数形如 URL（含 "/" 或以 http 开头）→ 直接按 frame URL
// 子串匹配；否则视为 iframe 元素的 CSS 选择器。
func looksLikeURL(s string) bool {
	return strings.Contains(s, "/") || strings.HasPrefix(strings.ToLower(s), "http")
}

// resolveFrame 把 frame 参数定位到 iframe 的 frameId（只匹配子 frame，排除
// 主文档自身）：URL 形参直接对 frame 树的 URL 做子串匹配；CSS 选择器先在主
// frame 求值取 iframe 元素的 src，再按 URL 子串匹配。未命中 →
// ErrElementNotFound。
func (m *Manager) resolveFrame(ctx context.Context, frame string) (string, error) {
	m.mu.Lock()
	conn := m.tabs[m.activePageID]
	m.mu.Unlock()
	if conn == nil {
		return "", errors.New("browser: 会话未建立")
	}
	var resp struct {
		FrameTree frameNode `json:"frameTree"`
	}
	if err := conn.Call(ctx, "Page.getFrameTree", map[string]any{}, &resp); err != nil {
		return "", fmt.Errorf("browser: 获取 frame 树失败: %w", err)
	}
	var frames []frameInfo
	flattenFrameTree(resp.FrameTree, &frames)
	if len(frames) == 0 {
		return "", fmt.Errorf("%w: 页面无任何 frame", ErrElementNotFound)
	}

	matchURL := frame
	if !looksLikeURL(frame) {
		// CSS 选择器定位 iframe 元素：主 frame 求值取 src（跨源 iframe 的
		// src 属性也可读，无需进入 iframe 上下文）。
		var res struct {
			okField
			Src string `json:"src"`
		}
		if err := m.evaluate(ctx, fmt.Sprintf(jsFrameSrc, jsString(frame)), &res); err != nil {
			return "", err
		}
		if !res.OK {
			return "", jsErr(res.Error)
		}
		if res.Src == "" {
			return "", fmt.Errorf("%w: iframe %s 无 src（srcdoc 暂不支持，请改用 frame URL 定位）", ErrElementNotFound, frame)
		}
		matchURL = res.Src
	}
	for _, fr := range frames {
		if fr.ParentID == "" {
			continue // 主 frame 自己，排除（frame 参数只指向 iframe）
		}
		if fr.URL != "" && strings.Contains(fr.URL, matchURL) {
			return fr.ID, nil
		}
	}
	return "", fmt.Errorf("%w: 未找到匹配 %q 的 iframe（可用 browser_read(frame=...) 试 frame URL）", ErrElementNotFound, frame)
}

// frameContextID 解析 frame 参数 → 执行上下文 id（frame 为空返回 0 = 主文档
// 默认上下文）。
func (m *Manager) frameContextID(ctx context.Context, frame string) (int, error) {
	if strings.TrimSpace(frame) == "" {
		return 0, nil
	}
	fid, err := m.resolveFrame(ctx, frame)
	if err != nil {
		return 0, err
	}
	return m.ensureFrameContext(ctx, fid)
}

// ensureFrameContext 为指定 iframe 建一个隔离世界并返回其 executionContextId
// （Page.createIsolatedWorld：无事件依赖、跨源 iframe 也可用；世界内 DOM API
// 与页面事件照常工作）。
func (m *Manager) ensureFrameContext(ctx context.Context, frameID string) (int, error) {
	m.mu.Lock()
	conn := m.tabs[m.activePageID]
	m.mu.Unlock()
	if conn == nil {
		return 0, errors.New("browser: 会话未建立")
	}
	var resp struct {
		ExecutionContextID int `json:"executionContextId"`
	}
	if err := conn.Call(ctx, "Page.createIsolatedWorld", map[string]any{
		"frameId": frameID, "worldName": "gaea-frame", "grantUniversalAccess": true,
	}, &resp); err != nil {
		return 0, fmt.Errorf("browser: 创建 iframe 执行上下文失败: %w", err)
	}
	if resp.ExecutionContextID == 0 {
		return 0, errors.New("browser: createIsolatedWorld 未返回 executionContextId")
	}
	return resp.ExecutionContextID, nil
}

// jsFrameSrc 取 iframe 元素的 src（frame 参数的 CSS 选择器定位路径）。
// __gaeaFrameSrc 为 fake CDP 的匹配 token。
const jsFrameSrc = `(function(){try{
var op='gaeaFrameSrc'; var sel=%s;
var el=document.querySelector(sel);
if(!el)return {ok:false,error:"未找到元素："+sel};
if(!el.tagName||el.tagName.toLowerCase()!=='iframe')return {ok:false,error:"不是 iframe 元素："+sel};
return {ok:true,src:el.getAttribute('src')||el.src||""};
}catch(e){return {ok:false,error:String(e&&e.message||e)};}})()`
