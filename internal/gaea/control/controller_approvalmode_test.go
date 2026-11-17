package control

// T4 接线验收：[permissions].approval_mode="never" 经 Options 注入，
// gate 的 Ask 一律硬拒绝给理由且不弹卡；""=现状（弹卡、既有拒绝文案）。

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/permission"
)

func TestApprovalModeOptionWiring(t *testing.T) {
	// boot 装配恒走 permission.New（解析 "ask" 缺省）；测试同构。
	policy := permission.New("ask", nil, nil, nil)
	commitArgs := []byte(`{"command":"git commit -m x"}`)

	// never：Ask 硬拒（普通闸 + hardAsk 皆然），理由含 never，卡不弹。
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec, Policy: policy, ApprovalMode: "never"})
	g := c.newGate()
	allow, reason, err := g.Check(context.Background(), "bash", commitArgs, false)
	if err != nil || allow || !strings.Contains(reason, "never") {
		t.Fatalf("never 模式 Ask = (%v,%q,%v), want 拒绝+never 理由", allow, reason, err)
	}
	allow, reason, err = g.Check(context.Background(), "remember", []byte(`{"name":"x"}`), false)
	if err != nil || allow || !strings.Contains(reason, "never") {
		t.Fatalf("never 模式 hardAsk = (%v,%q,%v), want 拒绝+never 理由", allow, reason, err)
	}
	if n := len(rec.snapshot()); n != 0 {
		t.Fatalf("never 模式不得弹卡, got %d 事件", n)
	}

	// 现状（""）：弹卡走既有语义——拒绝文案是默认 declined 文本，
	// 不带 never 字样（与 never 模式的硬拒在文本上可区分）。
	rec2 := newAttentionRecorder()
	c2 := New(Options{Sink: rec2, Policy: policy})
	g2 := c2.newGate()
	go func() {
		c2.Approve(<-rec2.approvals, DecisionDeny)
	}()
	allow, reason, err = g2.Check(context.Background(), "bash", commitArgs, false)
	if err != nil || allow {
		t.Fatalf("现状拒绝 = (%v,%q,%v)", allow, reason, err)
	}
	if strings.Contains(reason, "never") {
		t.Fatalf("现状拒绝不应带 never 理由: %q", reason)
	}
	if !strings.Contains(reason, "已拒绝") {
		t.Fatalf("现状应走既有拒绝文案（T3 中文回喂）: %q", reason)
	}
}
