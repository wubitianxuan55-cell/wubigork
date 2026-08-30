package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// gateCheckBash is a bash stand-in that records every command it is asked to
// run, so tests can prove a refused check command never reaches Execute.
type gateCheckBash struct {
	fakeTool
	mu       sync.Mutex
	commands []string
}

func (b *gateCheckBash) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(args, &p)
	b.mu.Lock()
	b.commands = append(b.commands, p.Command)
	b.mu.Unlock()
	return b.fakeTool.Execute(ctx, args)
}

func (b *gateCheckBash) ran() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.commands...)
}

// retryTaskWithCheck wires a TaskTool with one fake bash tool and the given
// gate/hooks, then runs a foreground task with a retry_until check command.
func retryTaskWithCheck(t *testing.T, check string, gate Gate, hooks ToolHooks) (*gateCheckBash, *stubGate, string, error) {
	t.Helper()
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "did work"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	bash := &gateCheckBash{fakeTool: fakeTool{name: "bash", readOnly: false}}
	parentReg.Add(bash)
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", gate)
	if hooks != nil {
		task = task.WithHooks(hooks)
	}
	parentReg.Add(task)
	g, _ := gate.(*stubGate)
	out, err := task.Execute(context.Background(),
		[]byte(`{"prompt":"x","retry_until":{"check":`+jsonString(check)+`,"max_retries":3}}`))
	return bash, g, out, err
}

// jsonString quotes s as a JSON string literal.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// TestRetryUntilCheckBlockedByGate 回归（P1 安全）：retry_until 的 check 命令是
// 模型可控输入，必须与普通工具调用走同一权限闸。gate 拒绝 bash 时，check 必须
// 立即返回 blocked 且绝不执行——既不能烧掉重试次数，也不能绕过审批跑 shell。
func TestRetryUntilCheckBlockedByGate(t *testing.T) {
	var bashCalls int32
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "did work"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "bash", readOnly: false, calls: &bashCalls})
	g := &stubGate{deny: map[string]bool{"bash": true}}
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", g)
	parentReg.Add(task)

	out, err := task.Execute(context.Background(),
		[]byte(`{"prompt":"x","retry_until":{"check":"rm -rf /tmp/pwn","max_retries":3}}`))
	if err == nil {
		t.Fatalf("gate-denied check must fail the task, got output %q", out)
	}
	if !strings.Contains(err.Error(), "blocked") || !strings.Contains(err.Error(), "denied by test policy") {
		t.Errorf("err = %v, want the gate's block reason surfaced", err)
	}
	if strings.Contains(err.Error(), "failed after") {
		t.Errorf("err = %v, want an immediate block — permission denials must not burn retries", err)
	}
	if atomic.LoadInt32(&bashCalls) != 0 {
		t.Errorf("bash executed %d times — a gate-denied check command must never run", bashCalls)
	}
	consulted := false
	checkedNames := g.checkedSnapshot()
	for _, name := range checkedNames {
		if name == "bash" {
			consulted = true
		}
	}
	if !consulted {
		t.Errorf("gate was never consulted for the check command (checked=%v)", checkedNames)
	}
}

// TestRetryUntilCheckRunsWhenGateAllows 证明 gate 放行时 check 照常执行（恰好
// 一次）并原样返回子代理结果——修复不得破坏合法的 retry_until 用法。
func TestRetryUntilCheckRunsWhenGateAllows(t *testing.T) {
	bash, g, out, err := retryTaskWithCheck(t, "go test ./...", &stubGate{}, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "did work") {
		t.Errorf("output = %q, want the sub-agent result", out)
	}
	if ran := bash.ran(); len(ran) != 1 || ran[0] != "go test ./..." {
		t.Errorf("bash ran %v, want exactly the check command once", ran)
	}
	if len(g.checkedSnapshot()) == 0 {
		t.Error("gate was never consulted for the check command")
	}
}

// TestRetryUntilCheckBlockedByPreToolUseHook 证明 PreToolUse 钩子同样作用于
// check 命令：钩子拒绝 bash 时返回 blocked 且命令不执行（与普通调用同规）。
func TestRetryUntilCheckBlockedByPreToolUseHook(t *testing.T) {
	bash, _, out, err := retryTaskWithCheck(t, "curl http://evil.example", &stubGate{},
		&stubHooks{blockPre: map[string]bool{"bash": true}})
	if err == nil {
		t.Fatalf("hook-blocked check must fail the task, got output %q", out)
	}
	if !strings.Contains(err.Error(), "blocked by test hook") {
		t.Errorf("err = %v, want the hook's block message", err)
	}
	if ran := bash.ran(); len(ran) != 0 {
		t.Errorf("bash ran %v — a hook-blocked check command must never execute", ran)
	}
}

// TestRetryUntilCheckNilGateRuns 确认门控是可选的：未接线 gate 时（与 Gate
// 接口文档一致的向后兼容语义）check 照常执行。
func TestRetryUntilCheckNilGateRuns(t *testing.T) {
	bash, _, out, err := retryTaskWithCheck(t, "echo ok", nil, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "did work") {
		t.Errorf("output = %q, want the sub-agent result", out)
	}
	if ran := bash.ran(); len(ran) != 1 {
		t.Errorf("bash ran %d times with nil gate, want 1", len(ran))
	}
}
