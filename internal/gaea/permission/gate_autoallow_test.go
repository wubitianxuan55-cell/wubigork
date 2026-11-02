package permission

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

// autoAllowRec 收集 OnAutoAllow 汇报的放行，用于断言账本口径。
type autoAllowRec struct {
	tools   []string
	subs    []string
	sources []string
}

func (r *autoAllowRec) hook(tool, subject, source string) {
	r.tools = append(r.tools, tool)
	r.subs = append(r.subs, subject)
	r.sources = append(r.sources, source)
}

// TestGateHeadlessAutoAllowLedger 锁定第五刀的口径：只有「本该问人、实际没问」
// 的放行才记账；策略本就允许的调用不记（那不是覆盖，是正常路径）；有人可问
// 的交互闸门也不记（那是 ApprovalRequest 的事务，不是授权账本）。
func TestGateHeadlessAutoAllowLedger(t *testing.T) {
	t.Run("ask_without_approver_records_gate", func(t *testing.T) {
		rec := &autoAllowRec{}
		g := NewGate(New("ask", nil, nil, nil), nil)
		g.OnAutoAllow = rec.hook

		allow, _, err := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"git commit -m x"}`), false)
		if err != nil || !allow {
			t.Fatalf("headless ask = (%v,%v), want allow", allow, err)
		}
		if len(rec.tools) != 1 || rec.tools[0] != "bash" || rec.subs[0] != "git commit -m x" {
			t.Fatalf("recorded = %v/%v, want bash/git commit -m x", rec.tools, rec.subs)
		}
		if rec.sources[0] != event.ApprovalSourceGate {
			t.Errorf("source = %q, want %q", rec.sources[0], event.ApprovalSourceGate)
		}
	})

	t.Run("hard_ask_without_approver_records_hard_ask", func(t *testing.T) {
		rec := &autoAllowRec{}
		g := NewGate(New("ask", nil, nil, nil), nil)
		g.AlwaysAsk = map[string]bool{"cost_save": true}
		g.OnAutoAllow = rec.hook

		allow, _, err := g.Check(context.Background(), "cost_save", json.RawMessage(`{"path":"cost.xlsx"}`), false)
		if err != nil || !allow {
			t.Fatalf("headless hard_ask = (%v,%v), want allow", allow, err)
		}
		if len(rec.tools) != 1 || rec.tools[0] != "cost_save" {
			t.Fatalf("recorded tools = %v, want [cost_save]", rec.tools)
		}
		if rec.sources[0] != event.ApprovalSourceHardAsk {
			t.Errorf("source = %q, want %q", rec.sources[0], event.ApprovalSourceHardAsk)
		}
		if rec.subs[0] == "" {
			t.Error("subject is empty — the ledger must say what would have run")
		}
	})

	t.Run("policy_allow_is_not_an_override", func(t *testing.T) {
		rec := &autoAllowRec{}
		g := NewGate(New("allow", nil, nil, nil), nil)
		g.OnAutoAllow = rec.hook

		allow, _, err := g.Check(context.Background(), "read_file", json.RawMessage(`{"path":"a.txt"}`), true)
		if err != nil || !allow {
			t.Fatalf("policy allow = (%v,%v), want allow", allow, err)
		}
		if len(rec.tools) != 0 {
			t.Errorf("recorded %v, want none (nothing was overridden)", rec.tools)
		}
	})

	t.Run("interactive_gate_is_not_ledgered", func(t *testing.T) {
		rec := &autoAllowRec{}
		ap := &stubApprover{allow: true}
		g := NewGate(New("ask", nil, nil, nil), ap)
		g.OnAutoAllow = rec.hook

		if _, _, err := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"go build"}`), false); err != nil {
			t.Fatalf("interactive ask: %v", err)
		}
		if ap.calls != 1 {
			t.Fatalf("approver calls = %d, want 1", ap.calls)
		}
		if len(rec.tools) != 0 {
			t.Errorf("recorded %v, want none (a human answered, that is an attention transaction)", rec.tools)
		}
	})

	t.Run("deny_stays_denied_and_unrecorded", func(t *testing.T) {
		rec := &autoAllowRec{}
		g := NewGate(New("ask", nil, nil, []string{"bash(rm*)"}), nil)
		g.OnAutoAllow = rec.hook

		allow, reason, err := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"rm file"}`), false)
		if err != nil || allow || reason == "" {
			t.Fatalf("deny = (%v,%q,%v), want blocked", allow, reason, err)
		}
		if len(rec.tools) != 0 {
			t.Errorf("recorded %v, want none (deny is not an auto-allow)", rec.tools)
		}
	})
}
