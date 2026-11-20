package computer

// W6 状态外发（施工单）：动作工具经 NotifyStatus 发「正在操控」状态，
// app 层装配 StatusHook=core.emit("computer-status", …)（Wails + httpbridge
// 双通道现成）。turn_id 在 gaea 工具层不可得（agent 内部值），恒空串占位
// ——前端按 state/app 关联展示，不做回合级归属。
import "sync"

// StatusHook 状态外发钩子（app.Startup 注入；nil=未装配，静默零开销）。
var (
	statusMu   sync.Mutex
	statusHook func(state, app, window string)
)

// SetStatusHook 装配状态钩子（app.Startup 调用）。
func SetStatusHook(fn func(state, app, window string)) {
	statusMu.Lock()
	defer statusMu.Unlock()
	statusHook = fn
}

// StatusHookPayload 构造事件载荷（验收：四键齐全——单测钉死）。
func StatusHookPayload(state, app, window string) map[string]interface{} {
	return map[string]interface{}{
		"state":   state, // "using" | "idle"
		"app":     app,   // process:<exe 全路径>（空=未知）
		"window":  window,
		"turn_id": "", // 工具层不可得，恒空串占位
	}
}

// NotifyStatus 外发动作状态（工具层调用；钩子未装配=静默）。
func NotifyStatus(state, app, window string) {
	statusMu.Lock()
	hook := statusHook
	statusMu.Unlock()
	if hook != nil {
		hook(state, app, window)
	}
}
