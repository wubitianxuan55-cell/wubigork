package control

// T3（codex 蒸馏）验收：拒绝理由回喂 / 超时挂起待追认 / 追认重放。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
)

// 拒绝带理由：理由出现在闸门回喂文本（下一轮工具结果）里。
func TestDenialReasonFedBack(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	go func() {
		id := <-rec.approvals
		c.ApproveWithReason(id, DecisionDeny, "这不是我要的写法，用接口重写")
	}()
	_, _, denial, err := gateApprover{c}.Approve(context.Background(), "bash", "rm -rf /", nil)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if !strings.Contains(denial, "已拒绝") || !strings.Contains(denial, "这不是我要的写法，用接口重写") {
		t.Fatalf("拒绝理由未回喂: %q", denial)
	}
	// 拒绝挂账：可追认。
	acts := c.PendingDeniedActions()
	if len(acts) != 1 {
		t.Fatalf("待追认账本 = %d, want 1", len(acts))
	}
	if acts[0].decision != DecisionDeny || acts[0].reason == "" {
		t.Fatalf("账本条目不符: %+v", acts[0])
	}
}

// 拒绝无理由：默认文案兜底（「用户拒绝」）。
func TestDenialDefaultReason(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	go func() {
		c.Approve(<-rec.approvals, DecisionDeny)
	}()
	_, _, denial, err := gateApprover{c}.Approve(context.Background(), "bash", "x", nil)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if !strings.Contains(denial, "用户拒绝") {
		t.Fatalf("默认理由缺失: %q", denial)
	}
}

// 超时：事件为 timeout（与 deny 可区分），挂「待追认」账本（不是 deny 账本）。
func TestTimeoutSuspendsForReinstatement(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	c.approvalTimeout = 20 * time.Millisecond
	_, _, denial, err := gateApprover{c}.Approve(context.Background(), "bash", "sleep", nil)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	_, res := approvalPair(t, rec.snapshot())
	if res.Outcome != event.OutcomeTimeout {
		t.Fatalf("outcome = %q, want timeout", res.Outcome)
	}
	if !strings.Contains(denial, "审批超时") {
		t.Fatalf("超时回喂文案缺失: %q", denial)
	}
	acts := c.PendingDeniedActions()
	if len(acts) != 1 || acts[0].decision != DecisionTimeout {
		t.Fatalf("超时应挂待追认账本: %+v", acts)
	}
}

// 追认：按原参数重放一次审批，放行后写会话授权——模型同参重试直接过闸
// （工具被重新执行一次）。
func TestApproveDeniedReinstates(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	// 第一刀：拒绝挂账。
	denied := make(chan string, 1)
	go func() {
		id := <-rec.approvals
		c.ApproveWithReason(id, DecisionDeny, "先不改")
		denied <- id
	}()
	if _, _, denial, _ := (gateApprover{c}).Approve(context.Background(), "bash", "go test ./...", nil); !strings.Contains(denial, "已拒绝") {
		t.Fatalf("拒绝回喂缺失: %q", denial)
	}
	id := <-denied
	if len(c.PendingDeniedActions()) != 1 {
		t.Fatalf("拒绝应挂账")
	}
	// 追认：新卡弹出，本测试代答 allow_session。
	go func() {
		aid := <-rec.approvals
		c.Approve(aid, DecisionAllowSession)
	}()
	c.ApproveDenied(id)
	// 账本清空；同 tool+subject 再过闸直接短路放行（remember=true，不弹新卡）。
	if len(c.PendingDeniedActions()) != 0 {
		t.Fatalf("追认后账本应清空: %+v", c.PendingDeniedActions())
	}
	allow, remember, denial, err := gateApprover{c}.Approve(context.Background(), "bash", "go test ./...", nil)
	if err != nil || !allow || !remember || denial != "" {
		t.Fatalf("追认后重试 = (%v,%v,%q,%v), want 放行+会话记忆", allow, remember, denial, err)
	}
}

// 追认 hardAsk 卡：放行不写 granted（逐条确认硬纪律），下次调用仍弹卡。
func TestApproveDeniedHardAskNeverGrants(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	denied := make(chan string, 1)
	go func() {
		id := <-rec.approvals
		c.Approve(id, DecisionDeny)
		denied <- id
	}()
	// hardAsk 工具（remember）经 gateApprover 走 alwaysPrompt 通道。
	args := json.RawMessage(`{"name":"x"}`)
	if _, _, denial, _ := (gateApprover{c}).Approve(context.Background(), "remember", "测试记忆条目", args); !strings.Contains(denial, "已拒绝") {
		t.Fatalf("拒绝回喂缺失: %q", denial)
	}
	id := <-denied
	go func() {
		aid := <-rec.approvals
		c.Approve(aid, DecisionAllowSession)
	}()
	c.ApproveDenied(id)
	if len(c.PendingDeniedActions()) != 0 {
		t.Fatalf("追认后账本应清空")
	}
	// 下一次调用仍要弹卡（rec 再收到 ApprovalRequest = 未写 granted 的证据）。
	done := make(chan bool, 1)
	go func() {
		allow, _, _, err := gateApprover{c}.Approve(context.Background(), "remember", "测试记忆条目", args)
		done <- allow && err == nil
	}()
	aid := <-rec.approvals
	c.Approve(aid, DecisionAllowOnce)
	if !<-done {
		t.Fatalf("hardAsk 追认后再次调用应经弹卡正常放行")
	}
}

// 同一账本条目双击/并发追认只能开一张卡；第二次请求不得再弹卡。
func TestApproveDeniedSingleFlight(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	denied := make(chan string, 1)
	go func() {
		id := <-rec.approvals
		c.Approve(id, DecisionDeny)
		denied <- id
	}()
	_, _, _, _ = (gateApprover{c}).Approve(context.Background(), "bash", "go vet ./...", nil)
	id := <-denied

	done := make(chan struct{}, 1)
	go func() {
		c.ApproveDenied(id)
		done <- struct{}{}
	}()
	cardID := <-rec.approvals
	c.ApproveDenied(id) // inFlight：应立即返回，不产生第二张卡
	c.Approve(cardID, DecisionAllowOnce)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("首个追认调用未收尾")
	}
	if len(c.PendingDeniedActions()) != 0 {
		t.Fatalf("单次追认后账本应清空: %+v", c.PendingDeniedActions())
	}
}

// 账本封顶：deniedActionCap+1 条挂账只留最新 50 条。
func TestDeniedActionsCapped(t *testing.T) {
	c := New(Options{})
	for i := 0; i <= deniedActionCap; i++ {
		c.recordDeniedAction("", approvalPrompt{tool: "bash", subject: "x"}, "", "")
	}
	acts := c.PendingDeniedActions()
	if len(acts) != deniedActionCap {
		t.Fatalf("账本 = %d, want %d", len(acts), deniedActionCap)
	}
}

// ctx 取消不挂账：回合终止的拒绝没有重放意义。
func TestCancelDoesNotRecordDenied(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, _, _, err := gateApprover{c}.Approve(ctx, "bash", "sleep", nil)
		done <- err
	}()
	<-rec.approvals
	cancel()
	if err := <-done; err == nil {
		t.Fatal("ctx 取消应返回错误")
	}
	if len(c.PendingDeniedActions()) != 0 {
		t.Fatalf("取消不应挂账: %+v", c.PendingDeniedActions())
	}
}
