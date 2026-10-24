package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/gaea/hook"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

type turnCaptureProvider struct {
	mu       sync.Mutex
	turns    [][]provider.Chunk
	requests []provider.Request
}

func (p *turnCaptureProvider) Name() string { return "turn-capture" }

func (p *turnCaptureProvider) Stream(_ context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.mu.Lock()
	p.requests = append(p.requests, req)
	idx := len(p.requests) - 1
	p.mu.Unlock()
	if idx >= len(p.turns) {
		idx = len(p.turns) - 1
	}
	ch := make(chan provider.Chunk, len(p.turns[idx]))
	for _, c := range p.turns[idx] {
		ch <- c
	}
	close(ch)
	return ch, nil
}

func (p *turnCaptureProvider) Chat(ctx context.Context, req provider.Request) (*provider.Completion, error) {
	return provider.ChatFromStream(ctx, p, req)
}

func (p *turnCaptureProvider) lastRequest() provider.Request {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.requests) == 0 {
		return provider.Request{}
	}
	return p.requests[len(p.requests)-1]
}

type accessProbeTool struct {
	path  string
	calls *int32
}

func (t accessProbeTool) Name() string            { return "probe" }
func (t accessProbeTool) Description() string     { return "probe" }
func (t accessProbeTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (t accessProbeTool) ReadOnly() bool          { return true }
func (t accessProbeTool) ReadTarget(json.RawMessage) (string, bool) {
	return t.path, true
}
func (t accessProbeTool) Execute(context.Context, json.RawMessage) (string, error) {
	if t.calls != nil {
		atomic.AddInt32(t.calls, 1)
	}
	return "executed", nil
}

func TestTaskSubagentInheritsAccessPolicy(t *testing.T) {
	workspace := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	var calls int32
	parentReg := tool.NewRegistry()
	parentReg.Add(accessProbeTool{path: outside, calls: &calls})
	child := &turnCaptureProvider{turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "probe", `{}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}}
	task := NewTaskTool(child, nil, parentReg, 20, 0, 0, "", "sys", nil).
		WithAccessPolicy(tool.NewPathAccessPolicy(workspace, nil, nil, nil))

	if _, err := task.Execute(context.Background(), []byte(`{"prompt":"probe access"}`)); err != nil {
		t.Fatalf("task Execute: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("subagent executed outside tool %d time(s); access policy was not inherited", got)
	}
	req := child.lastRequest()
	if len(req.Messages) == 0 || !strings.Contains(req.Messages[len(req.Messages)-1].Content, "path access denied") {
		t.Fatalf("subagent did not receive the access denial: %+v", req.Messages)
	}
}

func TestTaskSubagentHooksUseChildSessionID(t *testing.T) {
	workspace := t.TempDir()
	var payloads []hook.Payload
	spawner := func(_ context.Context, in hook.SpawnInput) hook.SpawnResult {
		var p hook.Payload
		if err := json.Unmarshal([]byte(in.Stdin), &p); err != nil {
			t.Fatalf("decode hook payload: %v", err)
		}
		payloads = append(payloads, p)
		return hook.SpawnResult{ExitCode: 0}
	}
	runner := hook.NewRunner([]hook.ResolvedHook{{
		HookConfig: hook.HookConfig{Command: "probe"},
		Event:      hook.PreToolUse,
	}}, workspace, spawner, nil)

	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "probe", readOnly: true})
	child := &turnCaptureProvider{turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "probe", `{}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}}
	var childID string
	task := NewTaskTool(child, nil, parentReg, 20, 0, 0, "", "sys", nil).
		WithHooksForSession(func(id string) ToolHooks {
			childID = id
			return runner.WithSessionID(id)
		})

	if _, err := task.Execute(context.Background(), []byte(`{"prompt":"hook session"}`)); err != nil {
		t.Fatalf("task Execute: %v", err)
	}
	if !strings.HasPrefix(childID, "sa_") {
		t.Fatalf("child hook session id = %q, want sa_ prefix", childID)
	}
	if len(payloads) == 0 {
		t.Fatal("child hook did not fire")
	}
	for _, p := range payloads {
		if p.SessionID != childID {
			t.Errorf("hook sessionId = %q, want child %q", p.SessionID, childID)
		}
	}
}

func TestParentPreToolHookBlocksSubagentBash(t *testing.T) {
	workspace := t.TempDir()
	var calls int32
	spawner := func(_ context.Context, in hook.SpawnInput) hook.SpawnResult {
		var p hook.Payload
		_ = json.Unmarshal([]byte(in.Stdin), &p)
		if p.ToolName == "bash" {
			return hook.SpawnResult{ExitCode: 2, Stderr: "bash denied by parent hook"}
		}
		return hook.SpawnResult{ExitCode: 0}
	}
	runner := hook.NewRunner([]hook.ResolvedHook{{
		HookConfig: hook.HookConfig{Command: "guard", Match: "bash"},
		Event:      hook.PreToolUse,
	}}, workspace, spawner, nil)
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "bash", readOnly: false, calls: &calls})
	child := &turnCaptureProvider{turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "bash", `{"command":"echo escaped"}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}}
	task := NewTaskTool(child, nil, parentReg, 20, 0, 0, "", "sys", nil).
		WithHooksForSession(func(id string) ToolHooks { return runner.WithSessionID(id) })

	if _, err := task.Execute(context.Background(), []byte(`{"prompt":"do not run bash"}`)); err != nil {
		t.Fatalf("task Execute: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("blocked subagent bash executed %d time(s)", got)
	}
}

func TestRetryUntilUsesChildHooks(t *testing.T) {
	workspace := t.TempDir()
	var calls int32
	spawner := func(_ context.Context, in hook.SpawnInput) hook.SpawnResult {
		var p hook.Payload
		_ = json.Unmarshal([]byte(in.Stdin), &p)
		if p.ToolName == "bash" {
			return hook.SpawnResult{ExitCode: 2, Stderr: "check denied"}
		}
		return hook.SpawnResult{ExitCode: 0}
	}
	runner := hook.NewRunner([]hook.ResolvedHook{{
		HookConfig: hook.HookConfig{Command: "check-guard", Match: "bash"},
		Event:      hook.PreToolUse,
	}}, workspace, spawner, nil)
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "bash", readOnly: false, calls: &calls})
	child := &turnCaptureProvider{turns: [][]provider.Chunk{{
		{Type: provider.ChunkText, Text: "candidate"},
		{Type: provider.ChunkDone},
	}}}
	task := NewTaskTool(child, nil, parentReg, 20, 0, 0, "", "sys", nil).
		WithHooksForSession(func(id string) ToolHooks { return runner.WithSessionID(id) })

	_, err := task.Execute(context.Background(), []byte(`{"prompt":"finish","retry_until":{"check":"echo ok","max_retries":1}}`))
	if err == nil || !strings.Contains(err.Error(), "check command blocked") {
		t.Fatalf("retry check should be blocked by child hooks, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("blocked retry check executed %d time(s)", got)
	}
}

func TestSubagentSessionIDIsUniqueForEphemeralRuns(t *testing.T) {
	child := &turnCaptureProvider{turns: [][]provider.Chunk{{
		{Type: provider.ChunkText, Text: "ok"},
		{Type: provider.ChunkDone},
	}}}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(child, nil, parentReg, 20, 0, 0, "", "sys", nil)
	var ids []string
	task.WithHooksForSession(func(id string) ToolHooks {
		ids = append(ids, id)
		return nil
	})
	for i := 0; i < 2; i++ {
		if _, err := task.Execute(context.Background(), []byte(fmt.Sprintf(`{"prompt":"run %d"}`, i))); err != nil {
			t.Fatalf("task %d: %v", i, err)
		}
	}
	for _, id := range ids {
		if !strings.HasPrefix(id, "sa_") {
			t.Errorf("ephemeral child session id = %q, want sa_ prefix", id)
		}
	}
	if len(ids) != 2 || ids[0] == ids[1] {
		t.Fatalf("ephemeral child session ids = %v, want two unique ids", ids)
	}
}
