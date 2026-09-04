package agent

// v4.62 P1 逐 token 流式：持久化子代理运行中的助手文本增量经 SubagentText
// 透传父 sink 的测试。覆盖三段链路：subSinkFor 按 refSrc 把 Text 转标（有
// ref）或丢弃（无 ref）；SubagentText 补打父调用 ID 透传（技能子代理路径）；
// refTextSink 把 Text 原地转标 SubagentText（RunPersistedSubAgent 的 ref 注
// 入点）。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

// TestSubSinkStreamsTextWithRef：有 ref 时 Text 增量转标 SubagentText 透传，
// 打点 ref 与父 task 调用 ID。
func TestSubSinkStreamsTextWithRef(t *testing.T) {
	parent := &collectorSink{}
	s := subSinkFor("call-7", parent, func() string { return "sa_42" })

	s.Emit(event.Event{Kind: event.Text, Text: "第一段"})
	s.Emit(event.Event{Kind: event.Text, Text: "第二段"})

	if len(parent.events) != 2 {
		t.Fatalf("父 sink 收到 %d 条事件，want 2：%+v", len(parent.events), parent.events)
	}
	for i, want := range []string{"第一段", "第二段"} {
		e := parent.events[i]
		if e.Kind != event.SubagentText {
			t.Errorf("事件 %d kind = %v, want SubagentText", i, e.Kind)
		}
		if e.Text != want {
			t.Errorf("事件 %d text = %q, want %q", i, e.Text, want)
		}
		if e.SubagentRef != "sa_42" {
			t.Errorf("事件 %d ref = %q, want sa_42", i, e.SubagentRef)
		}
		if e.ParentToolID != "call-7" {
			t.Errorf("事件 %d parentId = %q, want call-7", i, e.ParentToolID)
		}
	}
}

// TestSubSinkDropsTextWithoutRef：无 ref（临时子代理/Nil refSrc）维持既有
// 行为——Text 一律丢弃，子代理过程噪音不进主聊天。
func TestSubSinkDropsTextWithoutRef(t *testing.T) {
	parent := &collectorSink{}
	s := subSinkFor("call-8", parent, nil)

	s.Emit(event.Event{Kind: event.Text, Text: "不应透传"})
	s.Emit(event.Event{Kind: event.Text, Text: "也不应透传"})

	if len(parent.events) != 0 {
		t.Fatalf("父 sink 收到 %d 条事件，want 0：%+v", len(parent.events), parent.events)
	}
}

// TestSubSinkStampsParentOnSubagentText：技能子代理路径（refTextSink 在外层
// 已转标 ref），subSinkFor 只补 ParentToolID 后透传。
func TestSubSinkStampsParentOnSubagentText(t *testing.T) {
	parent := &collectorSink{}
	s := subSinkFor("call-10", parent, nil) // refSrc=nil：Text 丢弃，但 SubagentText 必须过

	s.Emit(event.Event{Kind: event.SubagentText, Text: "技能子代理增量", SubagentRef: "sa_9"})

	if len(parent.events) != 1 {
		t.Fatalf("父 sink 收到 %d 条事件，want 1：%+v", len(parent.events), parent.events)
	}
	e := parent.events[0]
	if e.Kind != event.SubagentText || e.Text != "技能子代理增量" || e.SubagentRef != "sa_9" {
		t.Fatalf("透传事件字段不符：%+v", e)
	}
	if e.ParentToolID != "call-10" {
		t.Errorf("parentId = %q, want call-10", e.ParentToolID)
	}
}

// TestRefTextSink：Text 原地转标 SubagentText 打 ref；其余 kind 原样透传。
func TestRefTextSink(t *testing.T) {
	parent := &collectorSink{}
	s := refTextSink("sa_77", parent)

	s.Emit(event.Event{Kind: event.Text, Text: "增量"})
	s.Emit(event.Event{Kind: event.ToolDispatch})
	s.Emit(event.Event{Kind: event.Reasoning, Reasoning: "推理不转标"})

	if len(parent.events) != 3 {
		t.Fatalf("inner 收到 %d 条事件，want 3：%+v", len(parent.events), parent.events)
	}
	if parent.events[0].Kind != event.SubagentText || parent.events[0].SubagentRef != "sa_77" {
		t.Errorf("Text 未转标：%+v", parent.events[0])
	}
	if parent.events[1].Kind != event.ToolDispatch {
		t.Errorf("ToolDispatch 应原样透传：%+v", parent.events[1])
	}
	if parent.events[2].Kind != event.Reasoning {
		t.Errorf("Reasoning 应原样透传（由 subSinkFor 决定丢弃）：%+v", parent.events[2])
	}
}
