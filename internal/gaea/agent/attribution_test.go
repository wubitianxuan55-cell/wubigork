package agent

// B5 工具归因（第四批 P3）：工具/用量事件的 Model+Turn 盖戳。
//
// 验收口径（docs/codex-distill/asset-50 §9）：
//   A5 工具事件携带的回合号与回合边界一致（第 N 次 Run 的调用盖 N）；
//   A6 并行同名调用 ID 不串、盖戳不串；
//   未配置 ModelName 时事件不带模型（旧日志形态，读端按未知处理）。

import (
	"context"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// collectingSink 收集全部事件供事后断言（并发安全：批量执行器工作协程同发）。
type collectingSink struct {
	mu     sync.Mutex
	events []event.Event
}

func (s *collectingSink) Emit(e event.Event) {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
}

func (s *collectingSink) all() []event.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]event.Event, len(s.events))
	copy(out, s.events)
	return out
}

func stampedToolEvents(t *testing.T, evs []event.Event) (dispatches, results []event.Event) {
	t.Helper()
	for _, e := range evs {
		switch e.Kind {
		case event.ToolDispatch:
			if !e.Tool.Partial { // 只断言完整派发（pre-exec 起点卡同样盖戳，单独核）
				dispatches = append(dispatches, e)
			}
		case event.ToolResult:
			results = append(results, e)
		}
	}
	return dispatches, results
}

// TestToolEventsCarryTurnAndModel（A5）：两次 Run 各带一次工具调用——
// 第 1/2 次调用的 dispatch/result 盖 turn=1/2 与 modelName；Usage 同轮同号。
func TestToolEventsCarryTurnAndModel(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "echo", readOnly: true})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{toolCallChunk("c1", "echo", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "r1"}, {Type: provider.ChunkDone}},
		{toolCallChunk("c2", "echo", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "r2"}, {Type: provider.ChunkDone}},
	}}
	sink := &collectingSink{}
	a := New(prov, reg, NewSession(""), Options{DisableVerify: true, ModelName: "test-model"}, sink)
	for _, in := range []string{"q1", "q2"} {
		if _, err := a.Run(context.Background(), in); err != nil {
			t.Fatalf("Run(%q): %v", in, err)
		}
	}
	dispatches, results := stampedToolEvents(t, sink.all())
	if len(dispatches) != 2 || len(results) != 2 {
		t.Fatalf("dispatches=%d results=%d, want 2/2", len(dispatches), len(results))
	}
	for _, e := range append(append([]event.Event{}, dispatches...), results...) {
		if e.Tool.Model != "test-model" {
			t.Errorf("%v %s: Model = %q, want test-model", e.Kind, e.Tool.ID, e.Tool.Model)
		}
	}
	if dispatches[0].Turn != 1 || dispatches[1].Turn != 2 {
		t.Errorf("dispatch turns = %d,%d, want 1,2", dispatches[0].Turn, dispatches[1].Turn)
	}
	if results[0].Turn != 1 || results[1].Turn != 2 {
		t.Errorf("result turns = %d,%d, want 1,2", results[0].Turn, results[1].Turn)
	}
	if dispatches[0].Tool.ID != "c1" || dispatches[1].Tool.ID != "c2" {
		t.Errorf("dispatch ids = %s,%s, want c1,c2（调用与回合不串）", dispatches[0].Tool.ID, dispatches[1].Tool.ID)
	}
	// 请求头也带模型 id（比 Provider 名细一档）。
	var header *event.RequestHeaderInfo
	for _, e := range sink.all() {
		if e.Kind == event.RequestHeader {
			h := e.Header
			header = &h
		}
	}
	if header == nil || header.Model != "test-model" {
		t.Errorf("RequestHeader.Model = %+v, want test-model", header)
	}
}

// TestConcurrentSameNameCallsNotCrossStamped（A6）：同一轮两条同名调用
// （批量执行器并行面）——ID 互异、盖戳一致、不串号。
func TestConcurrentSameNameCallsNotCrossStamped(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "echo", readOnly: false})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			toolCallChunk("a1", "echo", `{"i":1}`),
			toolCallChunk("a2", "echo", `{"i":2}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}}
	sink := &collectingSink{}
	a := New(prov, reg, NewSession(""), Options{DisableVerify: true, ModelName: "m-conc"}, sink)
	if _, err := a.Run(context.Background(), "go"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	dispatches, results := stampedToolEvents(t, sink.all())
	if len(dispatches) != 2 || len(results) != 2 {
		t.Fatalf("dispatches=%d results=%d, want 2/2（同名并行各一条）", len(dispatches), len(results))
	}
	ids := map[string]bool{}
	for _, e := range append(dispatches, results...) {
		ids[e.Tool.ID] = true
		if e.Turn != 1 {
			t.Errorf("%v %s: Turn = %d, want 1", e.Kind, e.Tool.ID, e.Turn)
		}
		if e.Tool.Model != "m-conc" {
			t.Errorf("%v %s: Model = %q, want m-conc", e.Kind, e.Tool.ID, e.Tool.Model)
		}
	}
	if len(ids) != 2 {
		t.Errorf("distinct ids = %d (%v), want 2（并行同名不串号）", len(ids), ids)
	}
}

// TestModelNameEmptyLeavesStampsEmpty：未配置 ModelName（旧装配路径）→
// 事件 Model 为空、Turn 照盖——旧行为兼容面。
func TestModelNameEmptyLeavesStampsEmpty(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "echo", readOnly: true})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{toolCallChunk("c1", "echo", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "ok"}, {Type: provider.ChunkDone}},
	}}
	sink := &collectingSink{}
	a := New(prov, reg, NewSession(""), Options{DisableVerify: true}, sink)
	if _, err := a.Run(context.Background(), "q"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	dispatches, results := stampedToolEvents(t, sink.all())
	if len(dispatches) != 1 || len(results) != 1 {
		t.Fatalf("dispatches=%d results=%d, want 1/1", len(dispatches), len(results))
	}
	if dispatches[0].Tool.Model != "" || results[0].Tool.Model != "" {
		t.Errorf("Model = %q/%q, want 空（未配置不盖）", dispatches[0].Tool.Model, results[0].Tool.Model)
	}
	if dispatches[0].Turn != 1 {
		t.Errorf("Turn = %d, want 1（回合号与模型名相互独立）", dispatches[0].Turn)
	}
}
