package permission

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

// TestGateDenyUnapproved 锁定 2026-11-08 众神殿结论第五步（pan_1794098356822）：
// deny_unapproved 开启后，无批准者的 Ask / hard_ask 一律拒绝并记拒绝账，
// 不再走自治放行；默认关闭时既有放行语义逐字节不变；有批准者时开关不生效。
func TestGateDenyUnapproved(t *testing.T) {
	t.Run("ask_without_approver_denied_and_audited", func(t *testing.T) {
		allowRec, denyRec := &autoAllowRec{}, &autoAllowRec{}
		g := NewGate(New("ask", nil, nil, nil), nil)
		g.DenyUnapproved = true
		g.OnAutoAllow = allowRec.hook
		g.OnUnapprovedDeny = denyRec.hook

		allow, reason, err := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"git commit -m x"}`), false)
		if err != nil || allow {
			t.Fatalf("deny-unapproved ask = (%v,%v), want refused", allow, err)
		}
		if !strings.Contains(reason, "no approver") {
			t.Fatalf("refusal reason must say why, got %q", reason)
		}
		if len(allowRec.tools) != 0 {
			t.Fatalf("a refusal must not be booked as an auto-allow: %v", allowRec.tools)
		}
		if len(denyRec.tools) != 1 || denyRec.tools[0] != "bash" || denyRec.subs[0] != "git commit -m x" {
			t.Fatalf("denial ledger = %v/%v, want bash/git commit -m x", denyRec.tools, denyRec.subs)
		}
		if denyRec.sources[0] != event.ApprovalSourceGate {
			t.Errorf("source = %q, want %q", denyRec.sources[0], event.ApprovalSourceGate)
		}
	})

	t.Run("hard_ask_without_approver_denied_and_audited", func(t *testing.T) {
		denyRec := &autoAllowRec{}
		g := NewGate(New("ask", nil, nil, nil), nil)
		g.AlwaysAsk = map[string]bool{"cost_save": true}
		g.DenyUnapproved = true
		g.OnUnapprovedDeny = denyRec.hook

		allow, _, err := g.Check(context.Background(), "cost_save", json.RawMessage(`{"path":"cost.xlsx"}`), false)
		if err != nil || allow {
			t.Fatalf("deny-unapproved hard_ask = (%v,%v), want refused", allow, err)
		}
		if len(denyRec.tools) != 1 || denyRec.sources[0] != event.ApprovalSourceHardAsk {
			t.Fatalf("denial ledger = %v/%v, want cost_save/%s", denyRec.tools, denyRec.sources, event.ApprovalSourceHardAsk)
		}
	})

	t.Run("default_off_keeps_headless_autonomy", func(t *testing.T) {
		allowRec, denyRec := &autoAllowRec{}, &autoAllowRec{}
		g := NewGate(New("ask", nil, nil, nil), nil)
		g.OnAutoAllow = allowRec.hook
		g.OnUnapprovedDeny = denyRec.hook

		allow, _, err := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"git commit -m x"}`), false)
		if err != nil || !allow {
			t.Fatalf("default headless ask = (%v,%v), want allow (unchanged)", allow, err)
		}
		if len(allowRec.tools) != 1 || len(denyRec.tools) != 0 {
			t.Fatalf("default path must book auto-allow only: allow=%v deny=%v", allowRec.tools, denyRec.tools)
		}
	})

	t.Run("policy_allowed_call_unaffected", func(t *testing.T) {
		denyRec := &autoAllowRec{}
		g := NewGate(New("allow", nil, nil, nil), nil)
		g.DenyUnapproved = true
		g.OnUnapprovedDeny = denyRec.hook

		allow, _, err := g.Check(context.Background(), "read_file", json.RawMessage(`{"path":"a.go"}`), true)
		if err != nil || !allow {
			t.Fatalf("policy-allowed call must stay allowed under deny-unapproved: (%v,%v)", allow, err)
		}
		if len(denyRec.tools) != 0 {
			t.Fatalf("nothing was asked, nothing to deny: %v", denyRec.tools)
		}
	})

	t.Run("approver_present_ignores_flag", func(t *testing.T) {
		denyRec := &autoAllowRec{}
		g := NewGate(New("ask", nil, nil, nil), &stubApprover{allow: true})
		g.DenyUnapproved = true
		g.OnUnapprovedDeny = denyRec.hook

		allow, _, err := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"git commit -m x"}`), false)
		if err != nil || !allow {
			t.Fatalf("an approver who allows must win: (%v,%v)", allow, err)
		}
		if len(denyRec.tools) != 0 {
			t.Fatalf("approver path must not book a deny-unapproved refusal: %v", denyRec.tools)
		}
	})
}
