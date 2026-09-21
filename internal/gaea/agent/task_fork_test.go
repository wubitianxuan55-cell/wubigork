package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ─── 刀1: Fork 型子代理（v4.380，dsh subagent-fork-in-process 蒸馏）────────────
// ─── 刀2: 子代理结构化输出补完（schema 注入 + 同会话 terminal guard）──────────

// recordingScripted = scriptedProvider（多轮脚本）+ 请求记录（fork/guard 断言
// 要看子代理首个请求的形状，scriptedProvider 不记录请求）。
type recordingScripted struct {
	scriptedProvider
	reqs []provider.Request
}

func (m *recordingScripted) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	m.reqs = append(m.reqs, req)
	return m.scriptedProvider.Stream(ctx, req)
}

func newRecordingScripted(turns [][]provider.Chunk) *recordingScripted {
	return &recordingScripted{scriptedProvider: scriptedProvider{name: "p", turns: turns}}
}

func taskToolForTest(prov provider.LLMProvider, parentReg *tool.Registry) *TaskTool {
	return NewTaskTool(prov, nil, parentReg, 20, 100000, 0.5, "", "", nil)
}

// fork=父会话平衡前缀做种子：子代理首个请求 = 父 system + 父历史 + 任务
// prompt（字节同源），且工具 schema 与父注册表对齐（KV 缓存继承的形状前提）。
func TestForkSeedsParentHistoryAndCacheShape(t *testing.T) {
	turns := [][]provider.Chunk{
		{toolCallChunk("c1", "task", `{"prompt":"sub do X","fork":true}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "sub answer"}, {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}
	rp := newRecordingScripted(turns)
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "grep", readOnly: true})
	tt := taskToolForTest(rp, reg)
	reg.Add(tt)

	parent := NewSession("PARENT-SYSTEM")
	parent.Add(provider.Message{Role: provider.RoleUser, Content: "earlier context ZZZ"})
	parent.Add(provider.Message{Role: provider.RoleAssistant, Content: "working on it"})
	a := New(rp, reg, parent, Options{DisableVerify: true}, event.Discard)
	if _, err := a.Run(context.Background(), "delegate please"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 子代理首个请求：记录里找 lastUser == 任务 prompt 的那一条。
	var sub *provider.Request
	for i := range rp.reqs {
		if lastUser(rp.reqs[i]) == "sub do X" {
			sub = &rp.reqs[i]
			break
		}
	}
	if sub == nil {
		t.Fatalf("sub-agent request not recorded; got %d requests", len(rp.reqs))
	}
	if len(sub.Messages) == 0 || sub.Messages[0].Role != provider.RoleSystem || sub.Messages[0].Content != "PARENT-SYSTEM" {
		t.Fatalf("fork seed must carry the parent system bytes verbatim, got first=%+v", sub.Messages[0])
	}
	joined := ""
	for _, m := range sub.Messages {
		joined += m.Content + "\n"
	}
	if !strings.Contains(joined, "earlier context ZZZ") || !strings.Contains(joined, "working on it") {
		t.Fatalf("fork seed must carry the full parent history, got:\n%s", joined)
	}
	if strings.Contains(sub.Messages[0].Content, DefaultTaskSystemPrompt) {
		t.Fatal("fork must not prepend the template system prompt (breaks byte-identical prefix)")
	}
	// 缓存形状：子请求的 schema 集合与父请求逐一同形（ActiveSchemas 对齐）。
	var parentReq *provider.Request
	for i := range rp.reqs {
		if lastUser(rp.reqs[i]) == "delegate please" {
			parentReq = &rp.reqs[i]
			break
		}
	}
	if parentReq == nil {
		t.Fatal("parent request not recorded")
	}
	if len(sub.Tools) != len(parentReq.Tools) {
		t.Fatalf("fork sub request must align tool schemas with the parent request, got %d vs parent %d", len(sub.Tools), len(parentReq.Tools))
	}
	// 子代理最终答复抵达父级。
	if got := toolResult(a.session, "task"); !strings.Contains(got, "sub answer") {
		t.Fatalf("parent must receive the sub answer, got %q", got)
	}
}

// fork 组合禁忌与无父上下文形态。
func TestForkValidationErrors(t *testing.T) {
	tt := taskToolForTest(&mockProvider{name: "p"}, tool.NewRegistry())
	ctx := context.Background()
	cases := []struct {
		name string
		args string
		want string
	}{
		{"background", `{"prompt":"x","fork":true,"run_in_background":true}`, "fork cannot be used with run_in_background"},
		{"continue_from", `{"prompt":"x","fork":true,"continue_from":"sa_1"}`, "fork cannot be used with continue_from"},
		{"retry_until", `{"prompt":"x","fork":true,"retry_until":{"check":"true"}}`, "fork cannot be used with retry_until"},
	}
	for _, c := range cases {
		if _, err := tt.Execute(ctx, json.RawMessage(c.args)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: want error containing %q, got %v", c.name, c.want, err)
		}
	}
	// 直接 Execute（无 ToolContext）：无父会话可种子，如实报错。
	if _, err := tt.Execute(ctx, json.RawMessage(`{"prompt":"x","fork":true}`)); err == nil || !strings.Contains(err.Error(), "parent conversation context") {
		t.Fatalf("fork without parent context must error, got %v", err)
	}
}

// schema 注入：子代理能在 prompt 里看见格式指令。
func TestOutputSchemaInjectedIntoSubPrompt(t *testing.T) {
	turns := [][]provider.Chunk{
		{toolCallChunk("c1", "task", `{"prompt":"report","output_schema":{"type":"object","properties":{"answer":{"type":"string"}}}}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: `{"answer":"foo"}`}, {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}
	rp := newRecordingScripted(turns)
	reg := tool.NewRegistry()
	tt := taskToolForTest(rp, reg)
	reg.Add(tt)
	a := New(rp, reg, NewSession(""), Options{DisableVerify: true}, event.Discard)
	if _, err := a.Run(context.Background(), "go"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	var sub *provider.Request
	for i := range rp.reqs {
		if strings.HasSuffix(strings.TrimSpace(lastUser(rp.reqs[i])), `"answer"}`) ||
			strings.Contains(lastUser(rp.reqs[i]), "[OUTPUT FORMAT]") {
			sub = &rp.reqs[i]
			break
		}
	}
	if sub == nil {
		t.Fatal("sub request with injected schema not recorded")
	}
	if lu := lastUser(*sub); !strings.Contains(lu, "[OUTPUT FORMAT]") || !strings.Contains(lu, `"answer"`) {
		t.Fatalf("schema must be injected into the sub prompt, got last user:\n%s", lu)
	}
}

// terminal guard：终答非 JSON → 同会话纠偏重入（有界）→ 剥围栏回收合法 JSON。
func TestOutputSchemaGuardRecoversInSession(t *testing.T) {
	turns := [][]provider.Chunk{
		{toolCallChunk("c1", "task", `{"prompt":"report","output_schema":{"type":"object"}}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "I think the answer is foo."}, {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "```json\n{\"answer\":\"foo\"}\n```"}, {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}
	rp := newRecordingScripted(turns)
	reg := tool.NewRegistry()
	tt := taskToolForTest(rp, reg)
	reg.Add(tt)
	a := New(rp, reg, NewSession(""), Options{DisableVerify: true}, event.Discard)
	if _, err := a.Run(context.Background(), "go"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// executeOne 会把 task 结果包进信封 JSON——透过信封断言内容。
	got := toolResult(a.session, "task")
	if strings.Contains(got, "[output_schema:") {
		t.Fatalf("recovered result must not carry the diagnostic prefix, got:\n%s", got)
	}
	if !strings.Contains(got, "answer") || !strings.Contains(got, "foo") {
		t.Fatalf("recovered structured answer must reach the parent, got:\n%s", got)
	}
	// guard 重入的 nudge 确实发出（在某个请求的末条 user 里）。
	nudged := false
	for _, req := range rp.reqs {
		if lastUser(req) == schemaRetryNudge {
			nudged = true
		}
	}
	if !nudged {
		t.Fatal("guard re-entry nudge must be sent in-session")
	}
}

// guard 耗尽：如实降级为诊断前缀+原样文本（宁带原样不造假）。
func TestOutputSchemaGuardGivesUpHonestly(t *testing.T) {
	nonJSON := []provider.Chunk{{Type: provider.ChunkText, Text: "nope, not json"}, {Type: provider.ChunkDone}}
	turns := [][]provider.Chunk{
		{toolCallChunk("c1", "task", `{"prompt":"report","output_schema":{"type":"object"}}`), {Type: provider.ChunkDone}},
		nonJSON, // 子代理首轮
		nonJSON, // guard 重试 1
		nonJSON, // guard 重试 2
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}
	rp := newRecordingScripted(turns)
	reg := tool.NewRegistry()
	tt := taskToolForTest(rp, reg)
	reg.Add(tt)
	a := New(rp, reg, NewSession(""), Options{DisableVerify: true}, event.Discard)
	if _, err := a.Run(context.Background(), "go"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := toolResult(a.session, "task")
	if !strings.Contains(got, "[output_schema:") || !strings.Contains(got, "nope, not json") {
		t.Fatalf("exhausted guard must degrade honestly with the raw text, got:\n%s", got)
	}
}

// stripJSONFences 单元：围栏/裸 JSON/非 JSON 三态。
func TestStripJSONFences(t *testing.T) {
	if got := stripJSONFences("```json\n{\"a\":1}\n```"); got != `{"a":1}` {
		t.Fatalf("fenced json must be stripped, got %q", got)
	}
	if got := stripJSONFences(`{"a":1}`); got != `{"a":1}` {
		t.Fatalf("plain json must pass through, got %q", got)
	}
	if got := stripJSONFences("plain text"); got != "plain text" {
		t.Fatalf("non-fenced text must pass through, got %q", got)
	}
}
