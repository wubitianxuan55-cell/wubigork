package computer

// W6 验收 1（施工单）：事件载荷四键齐全（state/app/window/turn_id）。
import "testing"

func TestStatusHookPayload(t *testing.T) {
	p := StatusHookPayload("using", `process:C:\x\notepad.exe`, "记事本")
	for _, k := range []string{"state", "app", "window", "turn_id"} {
		if _, ok := p[k]; !ok {
			t.Fatalf("载荷缺键 %s: %v", k, p)
		}
	}
	if p["state"] != "using" || p["app"] != `process:C:\x\notepad.exe` || p["window"] != "记事本" {
		t.Fatalf("载荷值错: %v", p)
	}
}

func TestNotifyStatusHook(t *testing.T) {
	var got struct{ state, app, window string }
	SetStatusHook(func(state, app, window string) { got = struct{ state, app, window string }{state, app, window} })
	t.Cleanup(func() { SetStatusHook(nil) })
	NotifyStatus("idle", "app", "win")
	if got.state != "idle" || got.app != "app" || got.window != "win" {
		t.Fatalf("钩子未收到状态: %+v", got)
	}
	// 未装配钩子=静默（nil 安全）。
	SetStatusHook(nil)
	NotifyStatus("using", "app", "win")
}
