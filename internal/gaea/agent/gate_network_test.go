package agent

// P8（codex 蒸馏 B6 折算，第四批）验收：
// ① 断网闸拒绝网络能力工具、放行本地工具、allow=true 零介入；
// ② panic 隔离（已有机制验收落袋）：注入 panic 工具后宿主存活、
//    回合失败可解释（errMsg 带 panic）、同批其余工具照常完成。

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

type allowAllGate struct{}

func (allowAllGate) Check(context.Context, string, json.RawMessage, bool) (bool, string, error) {
	return true, "", nil
}

func TestNetworkGateBlocksNetworkToolsOnly(t *testing.T) {
	inner := allowAllGate{}
	g := WrapNetworkGate(inner, false)

	for _, name := range []string{"web_search", "web_fetch", "browser_navigate", "browser_run_js", "mcp__github__create_issue"} {
		allow, reason, err := g.Check(context.Background(), name, nil, false)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if allow {
			t.Errorf("%s: 断网闸放行了网络工具", name)
		}
		if !strings.Contains(reason, "network") {
			t.Errorf("%s: 拒绝理由不可操作: %q", name, reason)
		}
	}
	// 本地工具（含 bash——它有自己的子进程网络开关）照常放行。
	for _, name := range []string{"bash", "read_file", "edit_file", "todo_write"} {
		allow, _, err := g.Check(context.Background(), name, nil, false)
		if err != nil || !allow {
			t.Errorf("%s: 本地工具被误拒 allow=%v err=%v", name, allow, err)
		}
	}
	// 开关闭合时零介入：原闸门原样返回（现状红线）。
	if got := WrapNetworkGate(inner, true); got != any(inner) {
		t.Errorf("allow=true 必须原样返回内层闸门（%T）", got)
	}
}

// panic 工具：执行即 panic（P8 隔离面的注入源）。
type panicTool struct{ name string }

func (p panicTool) Name() string            { return p.name }
func (p panicTool) Description() string     { return "" }
func (p panicTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (p panicTool) ReadOnly() bool          { return false }
func (p panicTool) Execute(context.Context, json.RawMessage) (string, error) {
	panic("boom: injected tool panic")
}

func TestPanicToolIsolatedBatchSurvives(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(panicTool{name: "crash_tool"})
	var healthyCalls int32
	reg.Add(fakeTool{name: "healthy", readOnly: true, calls: &healthyCalls})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "crash_tool", `{}`),
			toolCallChunk("c2", "healthy", `{}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "recovered"}, {Type: provider.ChunkDone}},
	}}
	sink := &collectingSink{}
	a := New(prov, reg, NewSession(""), Options{DisableVerify: true}, sink)
	res, err := a.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("Run: %v（panic 工具不得拖死回合）", err)
	}
	if res == nil || res.Summary != "recovered" {
		t.Fatalf("回合未恢复完成: %+v", res)
	}
	_, results := stampedToolEvents(t, sink.all())
	var crashErrMsg string
	for _, e := range results {
		if e.Tool.Name == "crash_tool" {
			crashErrMsg = e.Tool.Err
		}
	}
	if !strings.Contains(crashErrMsg, "panic") {
		t.Errorf("panic 工具的错误不可解释: %q", crashErrMsg)
	}
	if atomic.LoadInt32(&healthyCalls) == 0 {
		t.Error("同批健康工具未执行（隔离应只吞 panic，不吞批次）")
	}
}
