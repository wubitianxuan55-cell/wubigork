package permission

// W4 按 app 授权验收（电脑操控施工单）：window_id→process:<exe> 解析进
// Subject；同 app 规则放行、他 app 仍 Ask；字符串键优先于钩子；钩子未注入
// 退回裸工具规则。

import (
	"context"
	"encoding/json"
	"testing"
)

func injectWindowResolver(t *testing.T, fn func(windowID int) string) {
	t.Helper()
	prev := WindowSubjectResolver
	WindowSubjectResolver = fn
	t.Cleanup(func() { WindowSubjectResolver = prev })
}

func TestSubjectWindowIDResolver(t *testing.T) {
	injectWindowResolver(t, func(windowID int) string {
		if windowID == 42 {
			return `process:C:\windows\notepad.exe`
		}
		return ""
	})
	if got := Subject(json.RawMessage(`{"window_id":42}`)); got != `process:C:\windows\notepad.exe` {
		t.Fatalf("window_id 必须解析为属主 app: %q", got)
	}
	// 字符串键优先：有 command 时不走钩子。
	if got := Subject(json.RawMessage(`{"window_id":42,"command":"ls"}`)); got != "ls" {
		t.Fatalf("字符串主体键优先: %q", got)
	}
	// 解析失败（窗口亡）回空串。
	if got := Subject(json.RawMessage(`{"window_id":7}`)); got != "" {
		t.Fatalf("解析失败应回空串: %q", got)
	}
	// 钩子未注入：退回裸规则语义。
	WindowSubjectResolver = nil
	if got := Subject(json.RawMessage(`{"window_id":42}`)); got != "" {
		t.Fatalf("无钩子应回空串: %q", got)
	}
}

// recordApprover 记录被询问次数（nil Approver 时 Ask 自动放行=自治语义，
// 无法区分「规则放行」与「无人可问」）。
type recordApprover struct{ asks int }

func (r *recordApprover) Approve(ctx context.Context, tool, subject string, args json.RawMessage) (bool, bool, string, error) {
	r.asks++
	return false, false, "denied-by-test", nil
}

func TestGateAppSubjectAllowAndAsk(t *testing.T) {
	injectWindowResolver(t, func(windowID int) string {
		if windowID == 42 {
			return `process:C:\windows\notepad.exe`
		}
		return ""
	})
	// 同 app 规则放行（不经审批）。
	ap := &recordApprover{}
	g := NewGate(New("ask", []string{`computer_click(process:C:\windows\notepad.exe)`}, nil, nil), ap)
	allow, reason, err := g.Check(context.Background(), "computer_click", json.RawMessage(`{"window_id":42,"element_index":3}`), false)
	if err != nil || !allow {
		t.Fatalf("同 app 规则应放行: allow=%v reason=%q err=%v", allow, reason, err)
	}
	if ap.asks != 0 {
		t.Fatalf("规则放行不得询问审批: asks=%d", ap.asks)
	}
	// 他 app（解析失败/另一窗口）→ 无规则命中 → Ask → 审批被调并拒绝。
	allow, _, _ = g.Check(context.Background(), "computer_click", json.RawMessage(`{"window_id":99}`), false)
	if allow || ap.asks != 1 {
		t.Fatalf("白名单外 app 必须走审批: allow=%v asks=%d", allow, ap.asks)
	}
}
