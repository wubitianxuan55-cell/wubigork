package agent

// v4.26 对话流式重造：子代理完成回投（SubagentMessage）的发射与透传测试。
// 覆盖两段链路：runSubSession 在子代理成功后向其 sink 发射事件；subSinkFor
// 把事件打点 ParentToolID 后透传父 sink，同时维持对其余 kind 的有意丢弃。

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// collectorSink 收集父 sink 收到的事件（测试断言用）。
type collectorSink struct{ events []event.Event }

func (c *collectorSink) Emit(e event.Event) { c.events = append(c.events, e) }

// TestSubSinkForwardsSubagentMessage 回归：subSinkFor 必须透传 SubagentMessage
// 并打点父 task 调用 ID；其余非工具/用量事件维持丢弃（防子代理过程噪音刷屏）。
func TestSubSinkForwardsSubagentMessage(t *testing.T) {
	parent := &collectorSink{}
	s := subSinkFor("call-9", parent, nil)

	s.Emit(event.Event{Kind: event.SubagentMessage, Text: "子代理最终答复", SubagentRef: "sa_1"})
	s.Emit(event.Event{Kind: event.Text, Text: "中途文本不应透传"})
	s.Emit(event.Event{Kind: event.Reasoning, Reasoning: "中途推理不应透传"})
	s.Emit(event.Event{Kind: event.TurnStarted})
	s.Emit(event.Event{Kind: event.TurnDone})

	if len(parent.events) != 1 {
		t.Fatalf("父 sink 收到 %d 条事件，want 1（仅 SubagentMessage）：%+v", len(parent.events), parent.events)
	}
	got := parent.events[0]
	if got.Kind != event.SubagentMessage {
		t.Fatalf("kind = %v, want SubagentMessage", got.Kind)
	}
	if got.Text != "子代理最终答复" || got.SubagentRef != "sa_1" {
		t.Errorf("text/ref = %q/%q, want 子代理最终答复/sa_1", got.Text, got.SubagentRef)
	}
	if got.ParentToolID != "call-9" {
		t.Errorf("ParentToolID = %q, want call-9（父 task 调用 ID 未打点）", got.ParentToolID)
	}
}

// TestTaskToolReportsSubagentCompletion 回归：task 工具执行成功后，子代理的
// 最终答复（未包 task-result 标签的原文）作为 SubagentMessage 回投父回合。
// 取舍锁定：只回投完成态（err==nil 且文本非空），中途进度不回投。
func TestTaskToolReportsSubagentCompletion(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "found 3 callers of Foo"},
		{Type: provider.ChunkDone},
	}}
	parent := &collectorSink{}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", nil)

	// 模拟 executeOne 的调用上下文：父 task 调用 ID + 父 sink。
	ctx := withCallContext(context.Background(), "call-7", parent, nil)
	out, err := task.Execute(ctx, []byte(`{"prompt":"find callers of Foo"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "found 3 callers of Foo") {
		t.Fatalf("out = %q, want sub-agent final answer", out)
	}

	var msgs []event.Event
	for _, e := range parent.events {
		if e.Kind == event.SubagentMessage {
			msgs = append(msgs, e)
		}
	}
	if len(msgs) != 1 {
		t.Fatalf("收到 %d 条 SubagentMessage, want 1（只回投完成态一次）", len(msgs))
	}
	m := msgs[0]
	// 回投文本是未包 <task-result> 标签的原文——标签只给父模型消费。
	if m.Text != "found 3 callers of Foo" {
		t.Errorf("text = %q, want 未包装的最终答复原文", m.Text)
	}
	if m.ParentToolID != "call-7" {
		t.Errorf("ParentToolID = %q, want call-7", m.ParentToolID)
	}
	if m.SubagentRef != "" {
		t.Errorf("SubagentRef = %q, want 空（无 transcript store = 临时子代理）", m.SubagentRef)
	}
}

// TestTaskToolNoReportOnSubagentError 锁定取舍：子代理失败（无最终答复）时
// 不回投 SubagentMessage——失败信息经 task 工具的 error 结果传达给父模型
// 与前端工具卡，避免重复呈现。
func TestTaskToolNoReportOnSubagentError(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{ // 空文本流 → 无最终答复
		{Type: provider.ChunkDone},
	}}
	parent := &collectorSink{}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", nil)

	ctx := withCallContext(context.Background(), "call-8", parent, nil)
	if _, err := task.Execute(ctx, []byte(`{"prompt":"x"}`)); err == nil {
		t.Fatalf("Execute 应失败（子代理无最终答复）")
	}
	for _, e := range parent.events {
		if e.Kind == event.SubagentMessage {
			t.Fatalf("失败路径不应回投 SubagentMessage，收到：%+v", e)
		}
	}
}
