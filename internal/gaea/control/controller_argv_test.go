package control

// T10（codex 蒸馏）controller 侧：bash「记住这条命令」走 argv 形态——
// 回写格式与会话规则生效链。

import (
	"context"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

// 回写格式：bash 走 argv 一行式；其他工具保持既有形态。
func TestWritebackArgvFormat(t *testing.T) {
	c := New(Options{})
	var captured []string
	c.persistAllowRule = func(rule string) error {
		captured = append(captured, rule)
		return nil
	}
	c.writebackAllowRule("bash", "git status --porcelain")
	c.writebackAllowRule("cost_save", "写入成本库：测试")
	if len(captured) != 2 {
		t.Fatalf("回写次数不符: %v", captured)
	}
	if captured[0] != `bash prefix ["git","status","--porcelain"]` {
		t.Fatalf("bash 应回写 argv 形态: %q", captured[0])
	}
	if captured[1] != "cost_save(写入成本库：测试)" {
		t.Fatalf("非命令工具保持既有形态: %q", captured[1])
	}
}

// 验收①全链：allow_session 批准 git status 后，同形命令免审批放行，
// 串接提权（git status; echo pwned）仍弹卡。
func TestSessionArgvRuleFlow(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	c.SetPermLevel("ask") // 闸门要真的弹卡（默认级会让 gateApprover 自治放行）
	g := c.newGate()
	// 第一刀：弹卡 → allow_session。
	go func() {
		id := <-rec.approvals
		c.Approve(id, DecisionAllowSession)
	}()
	allow, _, _, err := gateApprover{c}.Approve(context.Background(), "bash", "git status", []byte(`{"command":"git status"}`))
	if err != nil || !allow {
		t.Fatalf("首刀应放行: (%v,%v)", allow, err)
	}
	if c.argvRules == nil || c.argvRules.Len() != 1 {
		t.Fatalf("allow_session 应落会话 argv 规则: %+v", c.argvRules)
	}
	// 同形（前缀扩展）：不再弹卡（rec 无新 ApprovalRequest）。
	requestsBefore := len(rec.snapshot())
	if ok, _, _ := g.Check(context.Background(), "bash", []byte(`{"command":"git status --short"}`), false); !ok {
		t.Fatalf("同前缀变体应免审批放行")
	}
	for _, e := range rec.snapshot()[requestsBefore:] {
		if e.Kind == event.ApprovalRequest {
			t.Fatalf("同形命令不应再弹卡")
		}
	}
	// 串接提权：仍要审批（新卡弹出）。载荷用非删除形态——深删会被 T4 危险
	// 命令硬闸直接拒绝（不弹卡），那是另一条防线，不是本用例的对象。
	done := make(chan bool, 1)
	go func() {
		aid := <-rec.approvals // 弹卡即证据
		c.Approve(aid, DecisionDeny)
		done <- true
	}()
	ok, _, _ := g.Check(context.Background(), "bash", []byte(`{"command":"git status; echo pwned"}`), false)
	<-done
	if ok {
		t.Fatalf("串接提权不得放行")
	}
}
