package agent

// D7-B 取消执行闸测试（dsh-ops 超代对账审计落点，2026-11-10）：
// 出口带（agent_stream 干净收尾+ctx 取消=中断，不派发）与入口带
//（executeBatch 拒派发未执行调用、消费已完成预执行结果）。

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// countingTool 计数真执行（验证取消后不再派发）。
type countingTool struct {
	name  string
	calls *atomic.Int32
}

func (c countingTool) Name() string            { return c.name }
func (c countingTool) Description() string     { return "" }
func (c countingTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (c countingTool) ReadOnly() bool          { return false }
func (c countingTool) Execute(context.Context, json.RawMessage) (string, error) {
	c.calls.Add(1)
	return c.name + " executed", nil
}

func TestCancelGateBlocksDispatch(t *testing.T) {
	var execCount atomic.Int32
	reg := tool.NewRegistry()
	reg.Add(countingTool{name: "writer_x", calls: &execCount})
	a := New(nil, reg, NewSession(""), Options{}, event.Discard)

	// 预取消 ctx：整批拒派发。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := []provider.ToolCall{
		{ID: "c1", Name: "writer_x", Arguments: `{}`},
		{ID: "c2", Name: "writer_x", Arguments: `{}`},
	}
	results := a.executeBatch(ctx, calls)
	if n := execCount.Load(); n != 0 {
		t.Fatalf("取消后不得派发（零副作用）: executed=%d", n)
	}
	for i, r := range results {
		env, ok := tool.ParseEnvelope(r)
		if !ok || env.OK {
			t.Fatalf("结果 %d 应为失败信封: %q", i, r)
		}
		if env.Subcode != "NOT_EXECUTED" || env.EffectState != tool.EffectNone || !env.Retryable {
			t.Fatalf("NOT_EXECUTED 语义错: %+v", env)
		}
	}
	// 已完成的预执行结果照常消费（不重跑）。
	a.preMu.Lock()
	a.preOutcomes = map[string]toolOutcome{"c3": {output: tool.WrapResult(tool.CodeOK, map[string]any{"pre": true})}}
	a.preMu.Unlock()
	results2 := a.executeBatch(ctx, []provider.ToolCall{{ID: "c3", Name: "writer_x", Arguments: `{}`}})
	if !strings.Contains(results2[0], `"pre":true`) {
		t.Fatalf("已完成预执行结果应消费: %q", results2[0])
	}
	if n := execCount.Load(); n != 0 {
		t.Fatalf("消费预执行不重跑: executed=%d", n)
	}
}

// cancelledCloseProvider 复现 bridge 取消路径形态：已流出一个工具调用块，
// 然后**干净关流（无 ChunkError）**——旧口径被当正常收尾（err=nil → 派发）。
type cancelledCloseProvider struct{ mockProvider }

func (p *cancelledCloseProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 2)
	tc := provider.ToolCall{ID: "t1", Name: "writer_y", Arguments: "{}"}
	ch <- provider.Chunk{Type: provider.ChunkToolCall, ToolCall: &tc}
	close(ch) // 干净关流（无 ChunkError）——bridge 取消路径形态
	return ch, nil
}

func TestStreamExitGateTreatsCleanCloseAsInterrupted(t *testing.T) {
	var execCount atomic.Int32
	reg := tool.NewRegistry()
	reg.Add(countingTool{name: "writer_y", calls: &execCount})
	a := New(&cancelledCloseProvider{mockProvider{name: "p"}}, reg, NewSession(""), Options{}, event.Discard)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := a.Run(ctx, "hi")
	if err == nil {
		t.Fatal("取消 ctx 的 Run 应返回错误")
	}
	// 出口带生效证据：已流出的工具调用**零执行**（旧口径 err=nil → executeBatch
	// 派发 → writer_y 执行）。
	if n := execCount.Load(); n != 0 {
		t.Fatalf("干净关流+取消不得派发工具: executed=%d", n)
	}
}
