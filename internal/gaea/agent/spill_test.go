package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spill"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ─── 刀1: spill 泄洪（v4.379，dsh spill-policy 蒸馏）────────────────────────

// bigTool 返回 n 字节定长内容的工具（内容含锚点串供断言）。
type bigTool struct {
	name     string
	readOnly bool
	n        int
}

func (b bigTool) Name() string            { return b.name }
func (b bigTool) Description() string     { return "" }
func (b bigTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (b bigTool) ReadOnly() bool          { return b.readOnly }
func (b bigTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	head := "ANCHOR-HEAD-"
	tail := "-ANCHOR-TAIL"
	return head + strings.Repeat("x", b.n-len(head)-len(tail)) + tail, nil
}

// ≥24KB 的成功结果被泄洪：locator 落在结果尾部、全文可整段取回。
func TestExecuteOneSpillsLargeResult(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(bigTool{name: "big", readOnly: true, n: 30 * 1024})
	a := New(&scriptedProvider{name: "p"}, reg, NewSession(""), Options{Spill: true, DisableVerify: true}, event.Discard)

	out := a.executeOne(context.Background(), provider.ToolCall{ID: "c1", Name: "big", Arguments: "{}"})
	if !strings.Contains(out.output, "[spilled]") || !strings.Contains(out.output, "read_spill") {
		t.Fatalf("locator notice must ride the inline result, got tail: %q", out.output[max(0, len(out.output)-300):])
	}
	// 全文原样可取（锚点头尾俱在——SmartCompress 对未知工具名是原样返回，
	// 但泄洪保的是压缩前原文，这里直接断言原文完整性）。
	full, total, _, err := a.spill.Retrieve("sp-000001", 0, spill.ReadMaxBytes)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if total != 30*1024 || !strings.HasPrefix(full, "ANCHOR-HEAD-") {
		t.Fatalf("spilled text must be the raw full output (total=%d)", total)
	}
}

// 豁免面：read_file 不泄洪（防取回循环）；小结果不泄洪；关开关零行为变化。
func TestExecuteOneSpillExemptionsAndSwitch(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(bigTool{name: "read_file", readOnly: true, n: 30 * 1024})
	reg.Add(bigTool{name: "small", readOnly: true, n: 8 * 1024})
	a := New(&scriptedProvider{name: "p"}, reg, NewSession(""), Options{Spill: true, DisableVerify: true}, event.Discard)

	out := a.executeOne(context.Background(), provider.ToolCall{ID: "c1", Name: "read_file", Arguments: "{}"})
	if strings.Contains(out.output, "[spilled]") {
		t.Fatal("read_file must be exempt from spilling")
	}
	a.executeOne(context.Background(), provider.ToolCall{ID: "c2", Name: "small", Arguments: "{}"})
	if _, _, _, err := a.spill.Retrieve("sp-000001", 0, 0); err == nil {
		t.Fatal("below-threshold result must not spill (no entry created)")
	}

	// 关开关：无库、无 locator、无 panic。
	off := New(&scriptedProvider{name: "p"}, reg, NewSession(""), Options{DisableVerify: true}, event.Discard)
	out = off.executeOne(context.Background(), provider.ToolCall{ID: "c3", Name: "read_file", Arguments: "{}"})
	if strings.Contains(out.output, "[spilled]") || off.spill != nil {
		t.Fatal("spill off must keep legacy behavior")
	}
}

// 极端大结果被全局截断后 locator 仍在（head+tail 保尾纪律）。
func TestSpillNoticeSurvivesTruncation(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(bigTool{name: "big", readOnly: true, n: 120 * 1024})
	a := New(&scriptedProvider{name: "p"}, reg, NewSession(""), Options{Spill: true, DisableVerify: true}, event.Discard)

	out := a.executeOne(context.Background(), provider.ToolCall{ID: "c1", Name: "big", Arguments: "{}"})
	if !out.truncated {
		t.Fatal("precondition: envelope must exceed the global truncation cap")
	}
	if !strings.Contains(out.output, "[spilled]") {
		t.Fatal("locator must survive head+tail truncation (tail-kept)")
	}
}

// read_spill 工具端到端：泄洪后按 locator 分页取回（经 Run 驱动，ctx 盖章
// 走真实 executeOne 注入点）。
func TestReadSpillToolRetrievesSpilledResult(t *testing.T) {
	rs, ok := tool.LookupBuiltin("read_spill")
	if !ok {
		t.Fatal("read_spill builtin not registered")
	}
	reg := tool.NewRegistry()
	reg.Add(bigTool{name: "big", readOnly: false, n: 30 * 1024}) // writer → 批内串行保序
	reg.Add(rs)

	// 跨回合保序：big 与 read_spill 无共享资源键，同批会被 v4.375 分区并行
	// （read_spill 可能先于泄洪执行）——拆回合保证 big 先泄洪。
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{toolCallChunk("c1", "big", `{}`), {Type: provider.ChunkDone}},
		{toolCallChunk("c2", "read_spill", `{"id":"sp-000001","max_bytes":1024}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}}
	a := New(prov, reg, NewSession(""), Options{Spill: true, DisableVerify: true}, event.Discard)
	if _, err := a.Run(context.Background(), "go"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := toolResult(a.session, "read_spill")
	if !strings.Contains(got, "[spill sp-000001") || !strings.Contains(got, "ANCHOR-HEAD-") {
		t.Fatalf("read_spill must return the spilled head with meta, got:\n%s", got[:min(400, len(got))])
	}
	if !strings.Contains(got, "more bytes") {
		t.Fatal("continuation hint must be present")
	}
}

// 未知 locator 如实报错（过期/驱逐不静默给空串）。
func TestReadSpillUnknownIDErrors(t *testing.T) {
	rs, _ := tool.LookupBuiltin("read_spill")
	s := spill.NewStore()
	out, err := rs.Execute(spill.WithStore(context.Background(), s), json.RawMessage(`{"id":"sp-777777"}`))
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unknown locator must error, got %q / %v", out, err)
	}
	// 无库上下文：如实报不可用。
	if _, err := rs.Execute(context.Background(), json.RawMessage(`{"id":"sp-000001"}`)); err == nil {
		t.Fatal("missing store must error")
	}
}

// ─── 刀2: 中断流结构化块保全（v4.379）────────────────────────────────────────

// 终态流错误：assistant 带 calls 落库且每调用有成对结果行（未派发的合成
// 「未执行」占位），历史形态合法（无悬空 tool_calls）。
func TestTerminalStreamErrorPersistsToolCalls(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "writer", readOnly: false})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "writer", `{"path":"a.txt"}`),
			{Type: provider.ChunkError, Err: errors.New("connection reset mid-stream")},
		},
	}}
	a := New(prov, reg, NewSession(""), Options{DisableVerify: true}, event.Discard)
	_, err := a.Run(context.Background(), "go")
	if err == nil {
		t.Fatal("expected terminal stream error")
	}

	var asst *provider.Message
	for i := range a.session.Messages {
		m := &a.session.Messages[i]
		if m.Role == provider.RoleAssistant && len(m.ToolCalls) > 0 {
			asst = m
		}
	}
	if asst == nil {
		t.Fatal("assistant message with tool calls must be persisted")
	}
	if got := toolResult(a.session, "writer"); !strings.Contains(got, "stream interrupted") {
		t.Fatalf("unexecuted call must get a synthetic placeholder result, got %q", got)
	}
	// 成对性：每个 tool_calls id 都有同 id 的 tool 结果行。
	ids := map[string]bool{}
	for _, c := range asst.ToolCalls {
		ids[c.ID] = false
	}
	for _, m := range a.session.Messages {
		if m.Role == provider.RoleTool {
			if _, ok := ids[m.ToolCallID]; ok {
				ids[m.ToolCallID] = true
			}
		}
	}
	for id, paired := range ids {
		if !paired {
			t.Fatalf("call %s has no paired tool result (dangling shape)", id)
		}
	}
}

// 预执行的只读调用在终态路径用真实结果（不是合成占位）。
func TestTerminalStreamErrorKeepsPreexecutedResults(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "reader", readOnly: true})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "reader", `{}`),
			{Type: provider.ChunkError, Err: errors.New("connection reset mid-stream")},
		},
	}}
	a := New(prov, reg, NewSession(""), Options{DisableVerify: true}, event.Discard)
	_, err := a.Run(context.Background(), "go")
	if err == nil {
		t.Fatal("expected terminal stream error")
	}
	// 正常路径同约定：会话里的 tool 结果是信封 JSON，真实预执行结果在 data.result 里。
	if got := toolResult(a.session, "reader"); !strings.Contains(got, "reader done") {
		t.Fatalf("pre-executed read-only result must be persisted verbatim, got %q", got)
	}
}

// 恢复路径（会重试采样）不得落 calls——悬空 assistant(tool_calls) 是非法
// 历史形态；文本保全行为不变。
func TestRecoveryPathDoesNotPersistCalls(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "writer", readOnly: false})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "writer", `{}`),
			{Type: provider.ChunkError, Err: &provider.StreamInterruptedError{Err: errors.New("stream reset")}},
		},
		{{Type: provider.ChunkText, Text: "final answer"}, {Type: provider.ChunkDone}},
	}}
	a := New(prov, reg, NewSession(""), Options{DisableVerify: true}, event.Discard)
	if _, err := a.Run(context.Background(), "go"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, m := range a.session.Messages {
		if m.Role == provider.RoleAssistant && len(m.ToolCalls) > 0 {
			t.Fatal("recovery path must not persist partial assistant tool calls")
		}
	}
	if !strings.Contains(a.session.Messages[len(a.session.Messages)-1].Content, "final answer") {
		t.Fatal("turn must complete after recovery")
	}
}
