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
