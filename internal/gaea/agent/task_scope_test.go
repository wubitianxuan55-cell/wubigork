package agent

// 2026-11-08 众神殿办公板块子代理裁决刀4：DAG 节点作用域（RunNew 面）——
//   - WritePaths 声明 → containment 白名单注册表（自报只读或 WritePathResolver；
//     bash 等无 resolver 写工具出局）+ 执行面 WriteScopePolicy 逐调用包含校验；
//   - ReadOnly → 只读收窄；
//   - 无盖 → 全量注册表+父策略（今日行为）。

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// fakeResolverWriter 是实现 WritePathResolver 的假写工具（write_file 形态，
// 路径从 args.path 解析——真实写工具族同接口）。
type fakeResolverWriter struct{ fakeTool }

func (f fakeResolverWriter) WritePaths(args json.RawMessage) ([]string, error) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Path) == "" {
		return nil, errors.New("no path")
	}
	return []string{p.Path}, nil
}

func TestSubRegForScopeContainableWhitelist(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{{Type: provider.ChunkText, Text: "ok"}, {Type: provider.ChunkDone}}}
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeResolverWriter{fakeTool{name: "write_file", readOnly: false}})
	parentReg.Add(fakeTool{name: "bash", readOnly: false}) // 无 resolver → 出局
	parentReg.Add(fakeTool{name: "read_file", readOnly: true})
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", nil)
	parentReg.Add(task)

	// 写路径声明 → resolver 写工具+只读工具保留，bash 出局。
	ctx := WithSubagentScope(context.Background(), SubagentScope{WritePaths: []string{"/ws/out"}})
	names := task.subRegForScope(ctx).Names()
	if !containsName(names, "write_file") || !containsName(names, "read_file") {
		t.Errorf("containment 白名单应保留 resolver 写工具与只读工具（got %v）", names)
	}
	if containsName(names, "bash") {
		t.Errorf("无 resolver 的 bash 应出局（got %v）", names)
	}
	// 只读作用域 → 只剩自报只读。
	ctx = WithSubagentScope(context.Background(), SubagentScope{ReadOnly: true})
	if names = task.subRegForScope(ctx).Names(); containsName(names, "write_file") || containsName(names, "bash") {
		t.Errorf("只读作用域应只剩只读工具（got %v）", names)
	}
	// 无盖 → 全量（今日行为）。
	if names = task.subRegForScope(context.Background()).Names(); !containsName(names, "bash") {
		t.Errorf("无作用域应保留全量注册表（got %v）", names)
	}
}

// TestRunNewScopedSchemasMatchRegistry 端到端：RunNew 携带写路径声明的 ctx
// 时，发给模型的工具面=containment 白名单（bash 消失）；无盖=父全量。
func TestRunNewScopedSchemasMatchRegistry(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}}}
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeResolverWriter{fakeTool{name: "write_file", readOnly: false}})
	parentReg.Add(fakeTool{name: "bash", readOnly: false})
	parentReg.Add(fakeTool{name: "read_file", readOnly: true})
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", nil)
	parentReg.Add(task)

	ctx := WithSubagentScope(context.Background(), SubagentScope{WritePaths: []string{"/ws/out"}})
	if _, _, err := task.RunNew(ctx, "node work", nil); err != nil {
		t.Fatalf("RunNew: %v", err)
	}
	if containsName(schemaNames(sub.lastReq.Tools), "bash") {
		t.Errorf("scoped 模型面不应含 bash（got %v）", schemaNames(sub.lastReq.Tools))
	}
	if !containsName(schemaNames(sub.lastReq.Tools), "write_file") {
		t.Errorf("scoped 模型面应含 resolver 写工具（got %v）", schemaNames(sub.lastReq.Tools))
	}
}

// TestRunSubSessionScopedPolicyDenial 执行面包含校验端到端（runSubSession 的
// WriteScopePolicy 包裹 + checkToolAccess 既有机制）：声明外写在工具结果里
// 收到拒绝文案（点名列出声明根），声明内放行。
func TestRunSubSessionScopedPolicyDenial(t *testing.T) {
	ws := t.TempDir()
	declared := ws + string(os.PathSeparator) + "out"
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeResolverWriter{fakeTool{name: "write_file", readOnly: false}})
	task := NewTaskTool(&mockProvider{name: "sub", chunks: []provider.Chunk{{Type: provider.ChunkText, Text: "ok"}, {Type: provider.ChunkDone}}},
		nil, parentReg, 20, 0, 0.0, "", "sys", nil).WithAccessPolicy(
		tool.NewPathAccessPolicy(ws, nil, nil, nil))
	parentReg.Add(task)
	ctx := WithSubagentScope(context.Background(), SubagentScope{WritePaths: []string{declared}})
	subReg := task.subRegForScope(ctx)

	toolCallChunks := func(path string) []provider.Chunk {
		argsB, _ := json.Marshal(map[string]string{"path": path, "content": "x"})
		return []provider.Chunk{
			{Type: provider.ChunkToolCallStart, ToolCall: &provider.ToolCall{ID: "c1", Name: "write_file"}},
			{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c1", Name: "write_file", Arguments: string(argsB)}},
			{Type: provider.ChunkDone},
		}
	}
	collect := func(path string) []event.Event {
		prov := &mockProvider{name: "sub", chunks: toolCallChunks(path)}
		var got []event.Event
		sink := event.FuncSink(func(e event.Event) { got = append(got, e) })
		_, _, _ = task.runSubSession(ctx, prov, "go", subReg, sink, nil, 2, nil, nil) // 步数耗尽错误无关紧要，看工具结果
		return got
	}
	toolResults := func(events []event.Event) string {
		var b strings.Builder
		for _, e := range events {
			if e.Kind == event.ToolResult {
				b.WriteString(e.Tool.Output + e.Tool.Err)
			}
		}
		return b.String()
	}

	// 声明外写 → 拒绝文案点名节点声明根。
	if out := toolResults(collect(ws + string(os.PathSeparator) + "elsewhere.txt")); !strings.Contains(out, "declared write paths") {
		t.Errorf("声明外写应收到节点作用域拒绝（got %q）", out)
	}
	// 声明内写 → 正常执行无拒绝。
	if out := toolResults(collect(declared + string(os.PathSeparator) + "a.txt")); strings.Contains(out, "path access denied") {
		t.Errorf("声明内写不应被拒（got %q）", out)
	}
}
