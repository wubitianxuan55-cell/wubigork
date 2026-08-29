package agent

// S1.2 记忆空间隔离器 · A 步：executeOne 把会话空间盖章到工具调用 ctx
//（memory.WithSpace，与 WithQueue 同注入点）——remember/memory_get 等记忆
// 工具经 memory.SpaceFromContext 读到会话空间，写读都限定本空间。

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// memSpaceProbeTool 记录 Execute 时 memory.SpaceFromContext(ctx) 的观察哨：
// 值必须穿过 TaskTool.Execute → runSubSession → 子 runDirect → executeOne
//（memory.WithSpace 盖章点）才能到达这里。
type memSpaceProbeTool struct {
	space atomic.Value
	calls int32
}

func (p *memSpaceProbeTool) Name() string            { return "mem_space_probe" }
func (p *memSpaceProbeTool) Description() string     { return "" }
func (p *memSpaceProbeTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (p *memSpaceProbeTool) ReadOnly() bool          { return true }

func (p *memSpaceProbeTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) {
	p.space.Store(memory.SpaceFromContext(ctx))
	atomic.AddInt32(&p.calls, 1)
	return "probed:" + memory.SpaceFromContext(ctx), nil
}

func (p *memSpaceProbeTool) seenSpace() string {
	if s, ok := p.space.Load().(string); ok {
		return s
	}
	return ""
}

// TestExecuteOneStampsMemorySpaceCtx：play 父的子代理工具调用 ctx 携带 play
//（memory 空间管线）；无标注 ctx（headless 直调）缺省 work。
func TestExecuteOneStampsMemorySpaceCtx(t *testing.T) {
	probe := &memSpaceProbeTool{}
	sub := &scriptedProvider{name: "sub", turns: [][]provider.Chunk{
		{toolCallChunk("c1", "mem_space_probe", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "final"}, {Type: provider.ChunkDone}},
	}}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", nil)
	parentReg.Add(task)
	parentReg.Add(probe)

	// play 父 → 子代理 ctx 带 play → executeOne 盖章 memory 空间 ctx
	ctx := WithSpace(withCallContext(context.Background(), "call_1", &testSink{}, nil), spaces.SpacePlay)
	if _, err := task.Execute(ctx, []byte(`{"prompt":"probe memory space"}`)); err != nil {
		t.Fatalf("task Execute: %v", err)
	}
	if atomic.LoadInt32(&probe.calls) == 0 {
		t.Fatal("子代理没有执行 probe 工具，盖章链未打通")
	}
	if got := probe.seenSpace(); got != spaces.SpacePlay {
		t.Fatalf("记忆工具 ctx 空间 = %q, want %q（play 会话记忆必须落 play）", got, spaces.SpacePlay)
	}
}

// TestExecuteOneMemorySpaceCtxDefaultsToWork：ctx 无空间标注（headless 工具
// 测试、后台 job 重建 ctx）→ memory 空间缺省 work，行为与改造前一致。
func TestExecuteOneMemorySpaceCtxDefaultsToWork(t *testing.T) {
	probe := &memSpaceProbeTool{}
	reg := tool.NewRegistry()
	reg.Add(probe)
	a := New(nil, reg, NewSession(""), Options{}, event.Discard)
	outcome := a.executeOne(withCallContext(context.Background(), "call_1", &testSink{}, nil),
		provider.ToolCall{ID: "c2", Name: "mem_space_probe", Arguments: `{}`})
	if outcome.errMsg != "" {
		t.Fatalf("probe 执行失败: %s", outcome.errMsg)
	}
	if got := probe.seenSpace(); got != spaces.SpaceWork {
		t.Fatalf("缺省记忆工具 ctx 空间 = %q, want %q", got, spaces.SpaceWork)
	}
}
