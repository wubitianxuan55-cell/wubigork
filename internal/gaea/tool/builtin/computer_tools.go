package builtin

// computer_tools.go — 电脑操控只读三件（施工单 W3，2026-11-20，默认关）：
// 仅当 [computer_use] enabled=true 时由 boot 调 RegisterComputerTools()
// 注册——默认关=Registry.Schemas() 与现状逐字节一致；输入注入族在 W5。
// 白名单语义（asset-62 §8「未授权 app 连读树都拒」）：非空时 computer_state
// 与 computer_launch 对白名单外 app 拒绝；computer_windows 仅标题清单保留
// 可发现性。错误统一 envelope 承载（照 browser_tools.go 形态）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/computer"
	"github.com/gaea/gaea/internal/gaea/tool"
)

var (
	computerRegisterOnce sync.Once
	computerMu           sync.Mutex
	computerApps         []string // app 白名单（process:<exe 全路径>；空=不限制）
)

// RegisterComputerTools 注册三件只读工具并落白名单/会话 TTL（幂等：
// 工具体只注册一次，白名单与 TTL 每次装配刷新）。boot 在
// [computer_use].enabled=true 时调用。
func RegisterComputerTools(apps []string, idleTTL time.Duration) {
	computerRegisterOnce.Do(func() {
		tool.RegisterBuiltin(computerWindowsTool{})
		tool.RegisterBuiltin(computerStateTool{})
		tool.RegisterBuiltin(computerLaunchTool{})
	})
	computerMu.Lock()
	computerApps = apps
	computerMu.Unlock()
	computer.Configure(computer.Options{IdleTTL: idleTTL})
}

// computerAppsAllowed 白名单判定：空=全放行。
func computerAppsAllowed(app string) bool {
	computerMu.Lock()
	defer computerMu.Unlock()
	if len(computerApps) == 0 {
		return true
	}
	for _, a := range computerApps {
		if a == app {
			return true
		}
	}
	return false
}

// computerFail 统一错误出口（envelope 承载，语义化 code）。
func computerFail(err error) (string, error) {
	code := tool.CodeExecError
	msg := err.Error()
	switch {
	case errors.Is(err, computer.ErrWindowGone):
		code = tool.CodeNotFound
	case errors.Is(err, computer.ErrUnsupported):
		code = "unsupported"
	}
	return tool.WrapError(code, msg, nil), nil
}

// ── computer_windows ────────────────────────────────────────────────────

type computerWindowsTool struct{}

func (computerWindowsTool) Name() string { return "computer_windows" }

func (computerWindowsTool) Description() string {
	return "列出当前可见的顶层窗口（id/app 进程路径/标题/位置）。观察或操控前先取 window_id；随后用 computer_state 读目标窗口的无障碍树。只列清单不含树内容。"
}

func (computerWindowsTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"filter":{"type":"string","description":"按标题或进程路径模糊过滤（大小写不敏感）"}}}`)
}

func (computerWindowsTool) ReadOnly() bool   { return true }
func (computerWindowsTool) SpaceTag() string { return "work" }

func (computerWindowsTool) Execute(_ context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		Filter string `json:"filter"`
	}
	_ = json.Unmarshal(raw, &args)
	wins, err := computer.ListWindows()
	if err != nil {
		return computerFail(err)
	}
	q := strings.ToLower(strings.TrimSpace(args.Filter))
	type winRow struct {
		ID    int    `json:"id"`
		App   string `json:"app"`
		Title string `json:"title"`
		X     int    `json:"x"`
		Y     int    `json:"y"`
		W     int    `json:"w"`
		H     int    `json:"h"`
	}
	out := make([]winRow, 0, len(wins))
	for _, w := range wins {
		if q != "" &&
			!strings.Contains(strings.ToLower(w.Title), q) &&
			!strings.Contains(strings.ToLower(w.App), q) {
			continue
		}
		out = append(out, winRow{ID: w.ID, App: w.App, Title: w.Title, X: w.X, Y: w.Y, W: w.W, H: w.H})
	}
	return tool.WrapResult("ok", map[string]any{
		"count":   len(out),
		"windows": out,
		"notice":  "取 window_id 后用 computer_state 读树；操控（W5 输入族）开启时动作前必须重新观察",
	}), nil
}

// ── computer_state ──────────────────────────────────────────────────────

type computerStateTool struct{}

func (computerStateTool) Name() string { return "computer_state" }

func (computerStateTool) Description() string {
	return "观察一个窗口：返回无障碍树文本（元素序号=随后动作的 element_index，一次观察只对当次有效）与可选截图。include_text 与 include_screenshot 至少要一个。观察版次会因任何新观察失效，动作前必须用最新观察。"
}

func (computerStateTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"window_id":{"type":"integer","description":"computer_windows 返回的窗口 id"},"include_text":{"type":"boolean","description":"返回树文本（默认 true）"},"include_screenshot":{"type":"boolean","description":"附带截图（默认 false；截图随工具结果进对话）"}},"required":["window_id"]}`)
}

func (computerStateTool) ReadOnly() bool   { return true }
func (computerStateTool) SpaceTag() string { return "work" }

func (computerStateTool) Execute(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		WindowID          *int  `json:"window_id"`
		IncludeText       *bool `json:"include_text"`
		IncludeScreenshot bool  `json:"include_screenshot"`
	}
	if err := json.Unmarshal(raw, &args); err != nil || args.WindowID == nil {
		return tool.WrapError(tool.CodeValidationError, "window_id 必填（integer）", map[string]any{"tool": "computer_state"}), nil
	}
	includeText := true
	if args.IncludeText != nil {
		includeText = *args.IncludeText
	}
	sess := computer.Default()

	// 白名单（非空时）：观察前先按窗口进程标识闸（未授权 app 连读树都拒）。
	wins, err := computer.ListWindows()
	if err != nil {
		return computerFail(err)
	}
	var target *computer.Window
	for i := range wins {
		if wins[i].ID == *args.WindowID {
			target = &wins[i]
			break
		}
	}
	if target == nil {
		return tool.WrapError(tool.CodeNotFound, fmt.Sprintf("window id %d 不在当前可见窗口清单里（可能已关闭）", *args.WindowID), map[string]any{"tool": "computer_state"}), nil
	}
	if !computerAppsAllowed(target.App) {
		return tool.WrapError(tool.CodeValidationError,
			"app 不在 [computer_use] apps 白名单，已拒绝观察: "+target.App,
			map[string]any{"tool": "computer_state", "app": target.App}), nil
	}

	snap, err := sess.Observe(*args.WindowID, computer.ObserveOpt{
		IncludeText:       includeText,
		IncludeScreenshot: args.IncludeScreenshot,
	})
	if err != nil {
		return computerFail(err)
	}
	// 截图走 W0 多模态旁路：本地路径随 tool 结果消息进主对话请求
	//（bridge 层统一编码 data URL + 每请求 4 张上限）。
	shotInfo := "（未请求截图）"
	if snap.Shot != nil {
		tool.ImageSinkFromContext(ctx).AddImage(snap.Shot.Path)
		shotInfo = snap.Shot.Path
	}
	return tool.WrapResult("ok", map[string]any{
		"window":       snap.Window,
		"revision":     snap.Revision,
		"text":         snap.Text,
		"shot":         shotInfo,
		"element_hint": "文本里每行行首数字=element_index，只对本次观察（revision）有效；再次观察后全部失效",
	}), nil
}

// ── computer_launch ─────────────────────────────────────────────────────

type computerLaunchTool struct{}

func (computerLaunchTool) Name() string { return "computer_launch" }

func (computerLaunchTool) Description() string {
	return "启动一个 app（exe 全路径或 process:<exe> 标识）并等待其窗口出现在枚举。受 [computer_use] apps 白名单约束；启动后对返回的 window_id 重新 computer_state。"
}

func (computerLaunchTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"app":{"type":"string","description":"exe 全路径或 process:<exe 全路径>"}},"required":["app"]}`)
}

func (computerLaunchTool) ReadOnly() bool   { return false }
func (computerLaunchTool) SpaceTag() string { return "work" }

func (computerLaunchTool) Execute(_ context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		App string `json:"app"`
	}
	if err := json.Unmarshal(raw, &args); err != nil || strings.TrimSpace(args.App) == "" {
		return tool.WrapError(tool.CodeValidationError, "app 必填（exe 全路径或 process:<exe>）", map[string]any{"tool": "computer_launch"}), nil
	}
	if !computerAppsAllowed(args.App) {
		return tool.WrapError(tool.CodeValidationError,
			"app 不在 [computer_use] apps 白名单，已拒绝启动: "+args.App,
			map[string]any{"tool": "computer_launch", "app": args.App}), nil
	}
	win, err := computer.LaunchApp(args.App)
	if err != nil {
		return computerFail(err)
	}
	return tool.WrapResult("ok", map[string]any{
		"window": win,
		"notice": "动作后必须重新 computer_state 获取新观察的元素序号",
	}), nil
}

// ── W5 动作三件：computer_click / computer_type / computer_key ──────────
// 闸门链（缺一不可，施工单 W5）：绑定窗口一致 → 白名单 → deny 表 →
// 双闸门（元素版次/坐标版次）→ 注入。动作成功后强制重观察并回新树
//（省一次往返，同时天然保证「动作后必须刷新」——codex 两段式循环）。

// computerActionGate 动作公共前置：窗口活着 + 白名单（与 state 同语义）。
// ok=false 时 env 已是完整失败信封。
func computerActionGate(windowID int) (win computer.Window, env string, ok bool) {
	wins, err := computer.ListWindows()
	if err != nil {
		env, _ = computerFail(err)
		return win, env, false
	}
	var target *computer.Window
	for i := range wins {
		if wins[i].ID == windowID {
			target = &wins[i]
			break
		}
	}
	if target == nil {
		env, _ = (func() (string, error) {
			return tool.WrapError(tool.CodeNotFound,
				fmt.Sprintf("window id %d 不在当前可见窗口清单里（可能已关闭）", windowID), nil), nil
		})()
		return win, env, false
	}
	if !computerAppsAllowed(target.App) {
		env, _ = (func() (string, error) {
			return tool.WrapError(tool.CodeValidationError,
				"app 不在 [computer_use] apps 白名单，已拒绝操控: "+target.App, nil), nil
		})()
		return win, env, false
	}
	return *target, "", true
}

// computerFailAction 动作族错误 → 语义化信封（stale/forbidden_key/
// coordinates_disabled/screenshot_required 独立 code，模型可分辨该重观察
// 还是该放弃）。
func computerFailAction(err error) (string, error) {
	code := tool.CodeExecError
	switch {
	case errors.Is(err, computer.ErrWindowGone):
		code = tool.CodeNotFound
	case errors.Is(err, computer.ErrStaleObservation):
		code = "stale_observation"
	case errors.Is(err, computer.ErrForbiddenKey):
		code = "forbidden_key"
	case errors.Is(err, computer.ErrCoordinatesDisabled):
		code = "coordinates_disabled"
	case errors.Is(err, computer.ErrScreenshotIDRequired):
		code = "screenshot_required"
	case errors.Is(err, computer.ErrUnsupported):
		code = "unsupported"
	}
	return tool.WrapError(code, err.Error(), nil), nil
}

// computerActionEnvelope 动作成功收口：立即重观察并回新树（新 revision 与
// 新 element_index 一并给出）。重观察失败时如实回报「动作已发生、观察失败」，
// 不谎报成功也不吞异常。
func computerActionEnvelope(sess *computer.Session, win computer.Window, extra map[string]any) (string, error) {
	snap, err := sess.Observe(win.ID, computer.ObserveOpt{IncludeText: true})
	if err != nil {
		extra["reobserve_error"] = err.Error()
		extra["revision"] = nil
		extra["element_hint"] = "重观察失败：下一次任何元素/坐标动作前必须重新 computer_state"
		return tool.WrapResult("ok", extra), nil
	}
	extra["revision"] = snap.Revision
	extra["elements"] = snap.Elements
	extra["text"] = snap.Text
	extra["element_hint"] = "text 每行行首数字=element_index（新一轮观察），旧序号已全部失效"
	return tool.WrapResult("ok", extra), nil
}

type computerClickTool struct{}

func (computerClickTool) Name() string { return "computer_click" }

func (computerClickTool) Description() string {
	return "点击目标窗口：优先 element_index（来自最近一次 computer_state 的树文本行首序号），或窗口相对坐标 x,y（须来自最近一次带截图的观察）。动作后自动返回新树与新一轮序号；换窗或跳过观察直接动作会被拒绝。"
}

func (computerClickTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"window_id":{"type":"integer","description":"目标窗口 id（必须与最近一次 computer_state 同窗）"},"element_index":{"type":"integer","description":"最近一次 computer_state 树文本里的序号（优先通道）"},"x":{"type":"integer","description":"窗口相对 x（坐标通道，须最近一次带截图观察后未再观察）"},"y":{"type":"integer","description":"窗口相对 y"},"button":{"type":"string","enum":["left","right","middle","double"],"description":"默认 left；double=双击"},"count":{"type":"integer","description":"连击次数（默认按 button 语义）"}},"required":["window_id"]}`)
}

func (computerClickTool) ReadOnly() bool   { return false }
func (computerClickTool) SpaceTag() string { return "work" }

func (computerClickTool) Execute(_ context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		WindowID     *int   `json:"window_id"`
		ElementIndex *int   `json:"element_index"`
		X            *int   `json:"x"`
		Y            *int   `json:"y"`
		Button       string `json:"button"`
		Count        int    `json:"count"`
	}
	if err := json.Unmarshal(raw, &args); err != nil || args.WindowID == nil {
		return tool.WrapError(tool.CodeValidationError, "window_id 必填（integer）", map[string]any{"tool": "computer_click"}), nil
	}
	if args.ElementIndex == nil && (args.X == nil || args.Y == nil) {
		return tool.WrapError(tool.CodeValidationError, "element_index 或 x,y 二选一（元素优先）", map[string]any{"tool": "computer_click"}), nil
	}
	sess := computer.Default()
	sess.Touch()
	win, env, ok := computerActionGate(*args.WindowID)
	if !ok {
		return env, nil
	}
	if err := sess.CheckBoundWindow(win.ID); err != nil {
		return computerFailAction(err)
	}
	var actErr error
	if args.ElementIndex != nil {
		actErr = sess.ClickWindowElement(*args.ElementIndex)
	} else {
		actErr = sess.ClickWindow(*args.X, *args.Y, args.Button, args.Count)
	}
	if actErr != nil {
		return computerFailAction(actErr)
	}
	return computerActionEnvelope(sess, win, map[string]any{"action": "click"})
}

type computerTypeTool struct{}

func (computerTypeTool) Name() string { return "computer_type" }

func (computerTypeTool) Description() string {
	return "向目标窗口注入 Unicode 文本（自动聚焦窗口；换行走 Return 键）。目标窗口必须与最近一次 computer_state 同窗；动作后自动返回新树。"
}

func (computerTypeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"window_id":{"type":"integer","description":"目标窗口 id"},"text":{"type":"string","description":"要输入的文本"}},"required":["window_id","text"]}`)
}

func (computerTypeTool) ReadOnly() bool   { return false }
func (computerTypeTool) SpaceTag() string { return "work" }

func (computerTypeTool) Execute(_ context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		WindowID *int   `json:"window_id"`
		Text     string `json:"text"`
	}
	if err := json.Unmarshal(raw, &args); err != nil || args.WindowID == nil {
		return tool.WrapError(tool.CodeValidationError, "window_id 必填（integer）", map[string]any{"tool": "computer_type"}), nil
	}
	sess := computer.Default()
	sess.Touch()
	win, env, ok := computerActionGate(*args.WindowID)
	if !ok {
		return env, nil
	}
	if err := sess.CheckBoundWindow(win.ID); err != nil {
		return computerFailAction(err)
	}
	if err := sess.TypeInto(args.Text); err != nil {
		return computerFailAction(err)
	}
	return computerActionEnvelope(sess, win, map[string]any{"action": "type"})
}

type computerKeyTool struct{}

func (computerKeyTool) Name() string { return "computer_key" }

func (computerKeyTool) Description() string {
	return "向目标窗口注入组合键（keysym 名，+ 连接修饰键，如 Return / Control_L+a / Alt_L+F4 / F5）。Win/Meta/Cmd/Super/OS 族一律本地硬拒；本工具每次调用都要求用户逐条确认。动作后自动返回新树。"
}

func (computerKeyTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"window_id":{"type":"integer","description":"目标窗口 id"},"key":{"type":"string","description":"keysym 组合键：Return / Tab / Escape / Control_L+a / Alt_L+F4 / F5 等"}},"required":["window_id","key"]}`)
}

func (computerKeyTool) ReadOnly() bool   { return false }
func (computerKeyTool) SpaceTag() string { return "work" }

func (computerKeyTool) Execute(_ context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		WindowID *int   `json:"window_id"`
		Key      string `json:"key"`
	}
	if err := json.Unmarshal(raw, &args); err != nil || args.WindowID == nil || args.Key == "" {
		return tool.WrapError(tool.CodeValidationError, "window_id 与 key 必填", map[string]any{"tool": "computer_key"}), nil
	}
	sess := computer.Default()
	sess.Touch()
	win, env, ok := computerActionGate(*args.WindowID)
	if !ok {
		return env, nil
	}
	if err := sess.CheckBoundWindow(win.ID); err != nil {
		return computerFailAction(err)
	}
	if err := sess.PressKeyOn(args.Key); err != nil {
		return computerFailAction(err)
	}
	return computerActionEnvelope(sess, win, map[string]any{"action": "key"})
}
