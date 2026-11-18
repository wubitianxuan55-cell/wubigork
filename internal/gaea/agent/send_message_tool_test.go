package agent

// send_message 三态路由测试（2026-11-18 DSH 式通用智能体工具集刀2）：
// running=直穿 steer；queued=排队落盘（起跑 flush）；终态=followup 续聊。
// 另钉 task 模型 schema 瘦身断言（专业收窄参数退役）。

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func newSendMessageHarness(t *testing.T, followUp SubagentFollowUpRunner) (*SendMessageTool, *SubagentStore) {
	t.Helper()
	store := NewSubagentStore(t.TempDir())
	return NewSendMessageTool(store, followUp), store
}

func TestSendMessageRoutingRunning(t *testing.T) {
	tl, store := newSendMessageHarness(t, nil)
	a, _ := store.PrepareFresh("sys", "work")
	defer a.Release()
	_ = store.MarkRunning(a)

	// 在跑登记（真 AgentRunner，同 steer_subagent_test 桩形）
	sub := New(nil, nil, nil, Options{}, nil)
	subRuns.Store(a.Ref, &subRunHandle{runner: sub, cancel: func() {}})
	defer subRuns.Delete(a.Ref)

	out, err := tl.Execute(context.Background(), mustJSON(map[string]string{"agent_id": a.Ref, "message": "补充口径"}))
	if err != nil {
		t.Fatalf("running 直穿: %v", err)
	}
	if !strings.Contains(out, "delivered to running") {
		t.Fatalf("running 分支文案不符: %s", out)
	}
	it, ok := sub.consumeSteer()
	if !ok || it.text != "补充口径" {
		t.Fatalf("steer 队列未收到: %+v ok=%v", it, ok)
	}
}

func TestSendMessageRoutingQueuedAndFlush(t *testing.T) {
	tl, store := newSendMessageHarness(t, nil)
	a, _ := store.PrepareFresh("sys", "work")
	defer a.Release()
	_ = store.MarkQueued(a)

	// queued：入队落盘
	out, err := tl.Execute(context.Background(), mustJSON(map[string]string{"agent_id": a.Ref, "message": "排队补充"}))
	if err != nil {
		t.Fatalf("queued 入队: %v", err)
	}
	if !strings.Contains(out, "queued") {
		t.Fatalf("queued 分支文案不符: %s", out)
	}
	if got := store.TakePending(a.Ref); len(got) != 1 || got[0] != "排队补充" {
		t.Fatalf("pending 未落盘: %v", got)
	}
	// 取空后重取=空（重放不重复投递）
	if got := store.TakePending(a.Ref); len(got) != 0 {
		t.Fatalf("TakePending 应取空: %v", got)
	}

	// 满员拒（帽 8，第 9 条给可执行文案）
	for i := 0; i < maxPendingMessages; i++ {
		_ = store.AppendPending(a.Ref, "m")
	}
	_, err = tl.Execute(context.Background(), mustJSON(map[string]string{"agent_id": a.Ref, "message": "overflow"}))
	if err == nil || !strings.Contains(err.Error(), "queue full") {
		t.Fatalf("满员应拒并给可执行文案: %v", err)
	}
}

func TestSendMessageRoutingFollowUp(t *testing.T) {
	var gotRef, gotMsg string
	tl, store := newSendMessageHarness(t, func(ctx context.Context, ref, prompt string) error {
		gotRef, gotMsg = ref, prompt
		return nil
	})
	a, _ := store.PrepareFresh("sys", "work")
	defer a.Release()
	_ = store.MarkRunning(a)
	_ = store.SaveCompleted(a)

	out, err := tl.Execute(context.Background(), mustJSON(map[string]string{"agent_id": a.Ref, "message": "续聊一句"}))
	if err != nil {
		t.Fatalf("终态 followup: %v", err)
	}
	if !strings.Contains(out, "follow-up started") || gotRef != a.Ref || gotMsg != "续聊一句" {
		t.Fatalf("followup 分支不符: out=%s ref=%s msg=%s", out, gotRef, gotMsg)
	}
}

func TestSendMessageSubtreeGuard(t *testing.T) {
	tl, store := newSendMessageHarness(t, nil)
	// 子代理调用面：兄弟运行不可传话（子树守卫，与 interrupt_agent 同款）
	l1, _ := store.PrepareFresh("sys", "work")
	defer l1.Release()
	l1.ParentRef = ""
	_ = store.MarkQueued(l1)
	sibling, _ := store.PrepareFresh("sys", "work")
	defer sibling.Release()
	_ = store.MarkQueued(sibling)
	_, err := tl.Execute(WithSubagentRef(context.Background(), l1.Ref), mustJSON(map[string]string{"agent_id": sibling.Ref, "message": "x"}))
	if err == nil || !strings.Contains(err.Error(), "outside your own sub-agent tree") {
		t.Fatalf("兄弟应被子树守卫拒绝: %v", err)
	}
}

// task 模型 schema 瘦身断言（刀1）：专业收窄参数从 Schema/CompactSchema/两段
// 描述退役——模型面不再暴露 readonly/write_paths/tools 白名单等专业调度面。
func TestTaskSchemaRetiresSpecialistParams(t *testing.T) {
	task := NewTaskTool(&mockProvider{name: "p"}, nil, tool.NewRegistry(), 10, 0, 0.0, "", "sys", nil)
	for _, surface := range []string{string(task.Schema()), string(task.CompactSchema())} {
		for _, banned := range []string{"readonly", "write_paths", "\"tools\"", "continue_from", "output_schema", "retry_until", "max_steps"} {
			if strings.Contains(surface, banned) {
				t.Fatalf("模型面仍含退役参数 %q: %s", banned, surface)
			}
		}
	}
	for _, banned := range []string{"readonly", "write_paths", "continue_from", "output_schema"} {
		if strings.Contains(task.CompactDescription(), banned) {
			t.Fatalf("CompactDescription 仍含退役参数 %q: %s", banned, task.CompactDescription())
		}
	}
}
