package control

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
)

type typedNilControllerSink struct{}

func (*typedNilControllerSink) Emit(event.Event) {}

func TestNewTreatsTypedNilSinkAsDiscard(t *testing.T) {
	var sink *typedNilControllerSink
	c := New(Options{Sink: sink})

	c.notice("typed nil sink should not panic")
}

// approvalIDs returns a Controller whose Sink forwards each ApprovalRequest's ID
// onto the channel, plus a counter of how many requests it emitted.
func approvalIDs() (*Controller, chan string, *int) {
	ids := make(chan string, 8)
	prompts := 0
	c := New(Options{Sink: event.FuncSink(func(e event.Event) {
		if e.Kind == event.ApprovalRequest {
			prompts++
			ids <- e.Approval.ID
		}
	})})
	return c, ids, &prompts
}

// TestApprovalAllowOnce drives the happy path: the gate emits an ApprovalRequest,
// the (fake) frontend answers allow, and the gate returns allow with no grant.
func TestApprovalAllowOnce(t *testing.T) {
	c, ids, _ := approvalIDs()
	go func() { c.Approve(<-ids, true, false, false) }()

	allow, remember, err := gateApprover{c}.Approve(context.Background(), "bash", "go test", nil)
	if err != nil || !allow || remember {
		t.Fatalf("Approve = (%v,%v,%v), want allow once", allow, remember, err)
	}
}

// TestApprovalDeny confirms a declined call returns allow=false.
func TestApprovalDeny(t *testing.T) {
	c, ids, _ := approvalIDs()
	go func() { c.Approve(<-ids, false, false, false) }()

	allow, _, err := gateApprover{c}.Approve(context.Background(), "bash", "rm -rf /", nil)
	if err != nil || allow {
		t.Fatalf("Approve = (%v,%v), want deny", allow, err)
	}
}

// TestApprovalSessionGrant proves an "allow this session" answer short-circuits
// later prompts for the same tool+subject: only the first reaches the frontend.
func TestApprovalSessionGrant(t *testing.T) {
	c, ids, prompts := approvalIDs()
	go func() {
		for id := range ids {
			c.Approve(id, true, true, false)
		}
	}()

	for i := 0; i < 3; i++ {
		allow, _, err := gateApprover{c}.Approve(context.Background(), "bash", "go build", nil)
		if err != nil || !allow {
			t.Fatalf("call %d = (%v,%v), want allow", i, allow, err)
		}
	}
	if *prompts != 1 {
		t.Errorf("prompted %d times, want 1 (session grant should short-circuit)", *prompts)
	}
}

// TestCostSaveAlwaysRequiresApproval 回归：成本库写入必须逐条确认——
// yolo 权限级别不自动放行，且「本会话允许」不会被记忆（每次仍弹审批）。
func TestCostSaveAlwaysRequiresApproval(t *testing.T) {
	c, ids, prompts := approvalIDs()
	c.SetPermLevel("yolo")
	g := gateApprover{c}

	// 第一次：yolo 下也触发审批，批准后放行
	done := make(chan bool, 1)
	go func() {
		allow, _, err := g.Approve(context.Background(), "cost_save", "", json.RawMessage(`{"title":"测试条目","price":123.45,"unit":"台班","source":"本次测算"}`))
		done <- allow && err == nil
	}()
	select {
	case id := <-ids:
		c.Approve(id, true, true, false) // 用户选「本会话允许」
	case <-time.After(2 * time.Second):
		t.Fatal("yolo 下 cost_save 未触发审批，被自动放行")
	}
	if !<-done {
		t.Fatal("批准后 cost_save 应放行")
	}

	// 第二次：会话放行不应被记忆，必须再次审批
	go func() {
		_, _, _ = g.Approve(context.Background(), "cost_save", "", json.RawMessage(`{"title":"测试条目","price":123.45,"unit":"台班","source":"本次测算"}`))
	}()
	select {
	case id := <-ids:
		c.Approve(id, true, false, false)
	case <-time.After(2 * time.Second):
		t.Fatal("cost_save 被会话放行记忆跳过，未再次审批")
	}
	if *prompts != 2 {
		t.Fatalf("cost_save 触发审批 %d 次，want 2（每次都必须确认）", *prompts)
	}
}

// TestCostSaveApprovalSubject 验证审批摘要包含条目名称/单价/单位/来源。
func TestCostSaveApprovalSubject(t *testing.T) {
	s := costSaveApprovalSubject(json.RawMessage(`{"title":"HP300 高频液压振动锤","price":4500,"unit":"台班","spec":"300kW","source":"本次测算","category":"机械"}`))
	for _, want := range []string{"写入成本库：HP300 高频液压振动锤", "单价 ¥4500.00", "单位 台班", "规格 300kW", "来源 本次测算"} {
		if !strings.Contains(s, want) {
			t.Fatalf("subject %q 缺少 %q", s, want)
		}
	}
}

// TestMemoryWriteAlwaysRequiresApproval 回归：记忆/知识库写入也必须逐条确认——
// yolo 权限级别不自动放行，会话放行不被记忆。
func TestMemoryWriteAlwaysRequiresApproval(t *testing.T) {
	c, ids, prompts := approvalIDs()
	c.SetPermLevel("yolo")
	g := gateApprover{c}

	for _, tool := range []string{"remember", "knowledge_add"} {
		args := json.RawMessage(`{"title":"测试条目","description":"示例描述","category":"经验总结","body":"正文"}`)
		done := make(chan bool, 1)
		go func() {
			allow, _, err := g.Approve(context.Background(), tool, "", args)
			done <- allow && err == nil
		}()
		select {
		case id := <-ids:
			c.Approve(id, true, true, false) // 选「本会话允许」
		case <-time.After(2 * time.Second):
			t.Fatalf("%s 在 yolo 下未触发审批，被自动放行", tool)
		}
		if !<-done {
			t.Fatalf("%s 批准后应放行", tool)
		}
	}
	if *prompts != 2 {
		t.Fatalf("持久化写入触发审批 %d 次，want 2", *prompts)
	}
}

// TestMemoryApprovalSubjects 验证记忆/知识库审批摘要包含关键字段。
func TestMemoryApprovalSubjects(t *testing.T) {
	rm := rememberApprovalSubject(json.RawMessage(`{"name":"prefers-tabs","title":"用户偏好","description":"喜欢先给大纲再展开","type":"user"}`))
	for _, want := range []string{"写入永久记忆：用户偏好", "喜欢先给大纲再展开", "类型 user"} {
		if !strings.Contains(rm, want) {
			t.Fatalf("remember subject %q 缺少 %q", rm, want)
		}
	}
	ka := knowledgeAddApprovalSubject(json.RawMessage(`{"title":"土壤修复验收标准","category":"规范标准","body":"验收应包含…","source":"生态环境部"}`))
	for _, want := range []string{"写入知识库：土壤修复验收标准", "分类 规范标准", "来源 生态环境部"} {
		if !strings.Contains(ka, want) {
			t.Fatalf("knowledge_add subject %q 缺少 %q", ka, want)
		}
	}
}

// TestApprovalCtxCancel ensures a cancelled turn unblocks the gate with an error
// (rather than hanging) when no one answers.
func TestApprovalCtxCancel(t *testing.T) {
	c := New(Options{Sink: event.Discard})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	allow, _, err := gateApprover{c}.Approve(ctx, "bash", "x", nil)
	if err == nil || allow {
		t.Fatalf("Approve on cancelled ctx = (%v,%v), want (false, error)", allow, err)
	}
}

// TestApprovalAbort 验证「拒绝并终止本轮」（codex ReviewDecision::Abort）：
// 闸门按拒绝返回（allow=false、不记会话放行、无错误），同时触发回合取消。
func TestApprovalAbort(t *testing.T) {
	c, ids, _ := approvalIDs()
	cancelled := make(chan struct{}, 1)
	c.mu.Lock()
	c.cancel = func() { cancelled <- struct{}{} }
	c.mu.Unlock()

	go func() { c.Approve(<-ids, false, false, true) }()

	allow, remember, err := gateApprover{c}.Approve(context.Background(), "bash", "rm -rf /", nil)
	if err != nil || allow || remember {
		t.Fatalf("Approve = (%v,%v,%v), want deny-with-abort", allow, remember, err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("abort 未触发回合取消")
	}
}
