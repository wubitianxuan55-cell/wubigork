package agent

import (
	"context"
	"encoding/json"
	"github.com/gaea/gaea/internal/gaea/event"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// stubGate denies any call whose tool name is in deny; everything else allows.
//
// 竞态治理（挂账 progress.md「stubGate 计数器既有竞态」）：Check 会被并行
// executeOne goroutine 并发调用（executeBatch→runParallel 最多 8 路，以及
// agent_stream 只读预执行 goroutine），checked 计数器必须持锁读写。
type stubGate struct {
	mu      sync.Mutex
	deny    map[string]bool
	checked []string
}

func (g *stubGate) Check(ctx context.Context, toolName string, args json.RawMessage, readOnly bool) (bool, string, error) {
	g.mu.Lock()
	g.checked = append(g.checked, toolName)
	g.mu.Unlock()
	if g.deny[toolName] {
		return false, "denied by test policy", nil
	}
	return true, "", nil
}

// checkedSnapshot 返回已咨询工具名的副本；并发 Check 进行中/结束后读取均安全。
func (g *stubGate) checkedSnapshot() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.checked...)
}

// TestGateBlocksDeniedCall proves executeOne consults the gate after the
// plan-mode check: a denied tool returns a "blocked:" result plus a notice and
// never runs, while an allowed tool runs normally.
func TestGateBlocksDeniedCall(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "bash", readOnly: false})
	reg.Add(fakeTool{name: "read_file", readOnly: true})

	g := &stubGate{deny: map[string]bool{"bash": true}}
	a := New(nil, reg, NewSession(""), Options{Gate: g}, event.Discard)

	blocked := a.executeOne(context.Background(), provider.ToolCall{Name: "bash", Arguments: `{"command":"rm -rf /"}`})
	if !strings.HasPrefix(blocked.output, "blocked:") {
		t.Errorf("denied call result = %q, want a 'blocked:' result", blocked.output)
	}
	if !blocked.blocked || blocked.errMsg == "" {
		t.Errorf("denied call should surface a user-facing block notice, got %+v", blocked)
	}

	ok := a.executeOne(context.Background(), provider.ToolCall{Name: "read_file", Arguments: `{"path":"/a"}`})
	if !strings.Contains(ok.output, "done") {
		t.Errorf("allowed call should run, got %q", ok.output)
	}

	checked := g.checkedSnapshot()
	if len(checked) != 2 {
		t.Errorf("gate consulted %d times, want 2 (%v)", len(checked), checked)
	}
}

// TestNilGateRunsEverything confirms gating is opt-in: with no gate wired, a
// writer call runs unimpeded (backward-compatible default).
func TestNilGateRunsEverything(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "write_file", readOnly: false})

	a := New(nil, reg, NewSession(""), Options{}, event.Discard) // no Gate
	out := a.executeOne(context.Background(), provider.ToolCall{Name: "write_file", Arguments: `{"path":"/a"}`})
	if strings.HasPrefix(out.output, "blocked:") {
		t.Errorf("nil gate should not block: %q", out.output)
	}
}
