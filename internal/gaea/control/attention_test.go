package control

// 注意力事务（V5 阶段 1）回归：每张真的弹出的卡必须有成对的解决事件——
// 允许/拒绝/超时/取消/中止 + 触发来源 + 等待时长，落进同一条事件流，
// 日志与轨迹才能「请求 → 解决」重放、计数。此前 Approve()/AnswerQuestion()
// 只投内存 channel，持久层看到的是永远悬空的卡。

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
)

// attentionRecorder 收集事件流，并把卡片 ID 转发给测试线程（模拟前端应答）。
type attentionRecorder struct {
	mu        sync.Mutex
	events    []event.Event
	approvals chan string
	asks      chan string
}

func newAttentionRecorder() *attentionRecorder {
	return &attentionRecorder{
		approvals: make(chan string, 8),
		asks:      make(chan string, 8),
	}
}

func (r *attentionRecorder) Emit(e event.Event) {
	r.mu.Lock()
	r.events = append(r.events, e)
	r.mu.Unlock()
	switch e.Kind {
	case event.ApprovalRequest:
		r.approvals <- e.Approval.ID
	case event.AskRequest:
		r.asks <- e.Ask.ID
	}
}

func (r *attentionRecorder) snapshot() []event.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]event.Event, len(r.events))
	copy(out, r.events)
	return out
}

// approvalPair 返回事件流中的 ApprovalRequest 与紧随的 ApprovalResolved。
func approvalPair(t *testing.T, events []event.Event) (event.Approval, event.ApprovalResolution) {
	t.Helper()
	var req *event.Approval
	var res *event.ApprovalResolution
	for i := range events {
		switch events[i].Kind {
		case event.ApprovalRequest:
			a := events[i].Approval
			req = &a
		case event.ApprovalResolved:
			r := events[i].Resolved
			res = &r
		}
	}
	if req == nil || res == nil {
		t.Fatalf("期望成对事件，实际得到 %d 条: %v", len(events), kindsOf(events))
	}
	if req.ID != res.ID {
		t.Fatalf("请求/解决 ID 不一致: %q vs %q", req.ID, res.ID)
	}
	if req.Tool != res.Tool || req.Subject != res.Subject {
		t.Fatalf("请求/解决目标不一致: %+v vs %+v", *req, *res)
	}
	return *req, *res
}

func kindsOf(events []event.Event) []event.Kind {
	out := make([]event.Kind, 0, len(events))
	for _, e := range events {
		out = append(out, e.Kind)
	}
	return out
}

func TestApprovalResolvedPaired(t *testing.T) {
	t.Run("allow_once", func(t *testing.T) {
		rec := newAttentionRecorder()
		c := New(Options{Sink: rec})
		go func() { c.Approve(<-rec.approvals, DecisionAllowOnce) }()

		allow, _, err := gateApprover{c}.Approve(context.Background(), "bash", "go test", nil)
		if err != nil || !allow {
			t.Fatalf("Approve = (%v,%v)", allow, err)
		}
		req, res := approvalPair(t, rec.snapshot())
		if req.Source != event.ApprovalSourceGate {
			t.Errorf("source = %q, want gate", req.Source)
		}
		if res.Decision != DecisionAllowOnce || res.Outcome != event.OutcomeAllowed {
			t.Errorf("resolved = %+v, want allow_once/allowed", res)
		}
		if res.LatencyMS < 0 {
			t.Errorf("latency = %d, want >= 0", res.LatencyMS)
		}
	})

	t.Run("deny", func(t *testing.T) {
		rec := newAttentionRecorder()
		c := New(Options{Sink: rec})
		go func() { c.Approve(<-rec.approvals, DecisionDeny) }()

		if allow, _, _ := (gateApprover{c}).Approve(context.Background(), "bash", "rm -rf /", nil); allow {
			t.Fatal("deny 应拒绝")
		}
		_, res := approvalPair(t, rec.snapshot())
		if res.Decision != DecisionDeny || res.Outcome != event.OutcomeDenied {
			t.Errorf("resolved = %+v, want deny/denied", res)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		rec := newAttentionRecorder()
		c := New(Options{Sink: rec})
		c.approvalTimeout = 20 * time.Millisecond

		if allow, _, _ := (gateApprover{c}).Approve(context.Background(), "bash", "sleep", nil); allow {
			t.Fatal("超时应按拒绝处理")
		}
		_, res := approvalPair(t, rec.snapshot())
		if res.Decision != DecisionTimeout || res.Outcome != event.OutcomeTimeout {
			t.Errorf("resolved = %+v, want timeout/timeout", res)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		rec := newAttentionRecorder()
		c := New(Options{Sink: rec})
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			_, _, err := gateApprover{c}.Approve(ctx, "bash", "sleep", nil)
			done <- err
		}()
		<-rec.approvals // 卡已弹出
		cancel()
		if err := <-done; err == nil {
			t.Fatal("ctx 取消应返回错误")
		}
		_, res := approvalPair(t, rec.snapshot())
		if res.Outcome != event.OutcomeCancelled || res.Decision != "" {
			t.Errorf("resolved = %+v, want cancelled/empty decision", res)
		}
	})

	t.Run("hard_ask_source", func(t *testing.T) {
		rec := newAttentionRecorder()
		c := New(Options{Sink: rec})
		c.SetPermLevel("yolo") // 逐条确认工具在 yolo 下仍弹卡
		go func() { c.Approve(<-rec.approvals, DecisionAllowOnce) }()

		if allow, _, err := (gateApprover{c}).Approve(context.Background(), "cost_save", "", []byte(`{"title":"钢材"}`)); err != nil || !allow {
			t.Fatalf("cost_save 应放行: %v %v", allow, err)
		}
		req, res := approvalPair(t, rec.snapshot())
		if req.Source != event.ApprovalSourceHardAsk || res.Source != event.ApprovalSourceHardAsk {
			t.Errorf("source = %q/%q, want hard_ask", req.Source, res.Source)
		}
	})

	t.Run("session_grant_short_circuit_emits_no_transaction", func(t *testing.T) {
		rec := newAttentionRecorder()
		c := New(Options{Sink: rec})
		go func() {
			for id := range rec.approvals {
				c.Approve(id, DecisionAllowSession)
			}
		}()
		g := gateApprover{c}
		for i := 0; i < 3; i++ {
			if allow, _, err := g.Approve(context.Background(), "bash", "go build", nil); err != nil || !allow {
				t.Fatalf("call %d = (%v,%v)", i, allow, err)
			}
		}
		var requests, resolutions int
		for _, e := range rec.snapshot() {
			switch e.Kind {
			case event.ApprovalRequest:
				requests++
			case event.ApprovalResolved:
				resolutions++
			}
		}
		if requests != 1 || resolutions != 1 {
			t.Fatalf("requests=%d resolutions=%d, want 1/1（会话放行不弹卡、不产生事务）", requests, resolutions)
		}
		// 没弹卡的两次放行必须留下授权账本：否则事后无法回答「这条规则谁批的」。
		var autoInts []event.AutoApproval
		for _, e := range rec.snapshot() {
			if e.Kind == event.ApprovalAuto {
				autoInts = append(autoInts, e.Auto)
			}
		}
		if len(autoInts) != 2 {
			t.Fatalf("自动放行账本 %d 条, want 2（第 2、3 次由会话规则短路）", len(autoInts))
		}
		for _, a := range autoInts {
			if a.Decided != event.AutoDecidedSessionGrant || a.Source != event.ApprovalSourceGate {
				t.Errorf("账本 = %+v, want session_grant/gate", a)
			}
			if a.Tool != "bash" || a.Subject != "go build" {
				t.Errorf("账本目标 = %q/%q, want bash/go build", a.Tool, a.Subject)
			}
		}
	})
}

// TestAutoApprovalLedger 锁定「机器替人放行」也要留痕：auto/yolo 下没有卡、
// 没有等待，但必须有一条 approval_auto 记录说清「谁放的、放行的是什么」。
func TestAutoApprovalLedger(t *testing.T) {
	t.Run("yolo", func(t *testing.T) {
		rec := newAttentionRecorder()
		c := New(Options{Sink: rec})
		c.SetPermLevel("yolo")

		if allow, _, err := (gateApprover{c}).Approve(context.Background(), "bash", "go test ./...", nil); err != nil || !allow {
			t.Fatalf("yolo 应放行 = (%v,%v)", allow, err)
		}
		events := rec.snapshot()
		var autos []event.AutoApproval
		for _, e := range events {
			switch e.Kind {
			case event.ApprovalAuto:
				autos = append(autos, e.Auto)
			case event.ApprovalRequest, event.ApprovalResolved:
				t.Fatalf("yolo 不该产生注意力事务: %v", kindsOf(events))
			}
		}
		if len(autos) != 1 {
			t.Fatalf("账本 %d 条, want 1: %v", len(autos), kindsOf(events))
		}
		if a := autos[0]; a.Decided != event.AutoDecidedYolo || a.Source != event.ApprovalSourceGate ||
			a.Tool != "bash" || a.Subject != "go test ./..." {
			t.Errorf("yolo 账本 = %+v", a)
		}
	})

	t.Run("auto", func(t *testing.T) {
		rec := newAttentionRecorder()
		c := New(Options{Sink: rec})
		c.SetPermLevel("auto")

		if allow, _, err := (gateApprover{c}).Approve(context.Background(), "write_file", "a.txt", nil); err != nil || !allow {
			t.Fatalf("auto 应放行 = (%v,%v)", allow, err)
		}
		var got *event.AutoApproval
		for _, e := range rec.snapshot() {
			if e.Kind == event.ApprovalAuto {
				a := e.Auto
				got = &a
			}
		}
		if got == nil {
			t.Fatal("auto 放行没有账本")
		}
		if got.Decided != event.AutoDecidedAuto || got.Tool != "write_file" {
			t.Errorf("auto 账本 = %+v", *got)
		}
	})
}

func TestAskResolvedPaired(t *testing.T) {
	t.Run("answered", func(t *testing.T) {
		rec := newAttentionRecorder()
		c := New(Options{Sink: rec})
		go func() {
			c.AnswerQuestion(<-rec.asks, []event.AskAnswer{{QuestionID: "q1", Selected: []string{"A"}}})
		}()

		answers, err := c.Ask(context.Background(), []event.AskQuestion{{ID: "q1", Prompt: "选哪个？"}})
		if err != nil || len(answers) != 1 {
			t.Fatalf("Ask = (%v,%v)", answers, err)
		}
		events := rec.snapshot()
		var req *event.Ask
		var res *event.AskResolution
		for i := range events {
			switch events[i].Kind {
			case event.AskRequest:
				a := events[i].Ask
				req = &a
			case event.AskResolved:
				r := events[i].AskResolved
				res = &r
			}
		}
		if req == nil || res == nil {
			t.Fatalf("期望成对 ask 事件，实际 %v", kindsOf(events))
		}
		if req.ID != res.ID || res.Outcome != event.OutcomeAnswered || res.Answered != 1 {
			t.Errorf("ask 事务 = %+v / %+v", *req, *res)
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		rec := newAttentionRecorder()
		c := New(Options{Sink: rec})
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			_, err := c.Ask(ctx, []event.AskQuestion{{ID: "q1", Prompt: "继续？"}})
			done <- err
		}()
		<-rec.asks
		cancel()
		if err := <-done; err == nil {
			t.Fatal("取消应返回错误")
		}
		events := rec.snapshot()
		var res *event.AskResolution
		for i := range events {
			if events[i].Kind == event.AskResolved {
				r := events[i].AskResolved
				res = &r
			}
		}
		if res == nil || res.Outcome != event.OutcomeCancelled || res.Answered != 0 {
			t.Fatalf("取消后的 ask 事务 = %+v", res)
		}
	})
}
