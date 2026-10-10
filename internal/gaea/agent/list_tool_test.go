package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── P1-C：结算原因分类与诊断卫生面 ──────────────────────────────────

func TestClassifySubagentStop(t *testing.T) {
	// 被打断：runErr 直接是 ctx.Canceled（子代理观察信号后返回）
	if got := classifySubagentStop(context.Background(), context.Canceled); got != StopInterrupted {
		t.Fatalf("canceled err = %q, want interrupted", got)
	}
	// 被打断：错误链包装（fmt.Errorf %w）
	wrapped := error(context.Canceled)
	for i := 0; i < 3; i++ {
		wrapped = fmt.Errorf("layer: %w", wrapped)
	}
	if got := classifySubagentStop(context.Background(), wrapped); got != StopInterrupted {
		t.Fatalf("wrapped canceled = %q, want interrupted", got)
	}
	// 被打断：runErr 未带 canceled 但运行 ctx 已取消（interrupt 信号在
	// provider 报了别的错之后才被观察到）
	runCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := classifySubagentStop(runCtx, errors.New("model stream reset")); got != StopInterrupted {
		t.Fatalf("canceled ctx = %q, want interrupted", got)
	}
	// 真失败：超时（deadline）不算打断——重派是错误重放，不是口径变化
	dctx, dcancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer dcancel()
	time.Sleep(2 * time.Millisecond)
	if got := classifySubagentStop(dctx, dctx.Err()); got != StopError {
		t.Fatalf("deadline = %q, want error", got)
	}
	// 真失败：普通模型/传输错误
	if got := classifySubagentStop(context.Background(), errors.New("500 bad gateway")); got != StopError {
		t.Fatalf("plain error = %q, want error", got)
	}
}

func TestSanitizeSubagentDiagnostic(t *testing.T) {
	if got := sanitizeSubagentDiagnostic("  500 bad gateway \n"); got != "500 bad gateway" {
		t.Fatalf("短诊断应原样(修整空白), got %q", got)
	}
	long := strings.Repeat("中", subagentDiagnosticCap) // 恰好不超（4096 字节=2048 字…超——每字 3 字节，4096 字节≈1365 字）
	got := sanitizeSubagentDiagnostic(long)
	if len(got) > subagentDiagnosticCap+len("…(diagnostic truncated)") {
		t.Fatalf("超长诊断应被截断到上限附近: %d bytes", len(got))
	}
	if !strings.HasSuffix(got, "…(diagnostic truncated)") {
		t.Fatalf("截断应带标注: %q", got[len(got)-30:])
	}
	// 截断不得劈断 UTF-8 rune：结果仍是合法 UTF-8
	if !utf8ValidString(got[:len(got)-len("…(diagnostic truncated)")]) {
		t.Fatal("截断点应落在 rune 边界")
	}
}

func utf8ValidString(s string) bool {
	for _, r := range s {
		if r == 0xFFFD {
			// 替换字符可能是原文也可能来自坏边界——用 range 遍历本身校验：
			// 坏字节 range 会产生 U+FFFD，但原文不含它
			return false
		}
	}
	return true
}

// ── P1-D：subagent_list 模型面枚举 ─────────────────────────────────

func TestListRuns(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	// 三条不同状态 + 一条损坏 meta
	a, _ := store.PrepareFresh("sys", "work")
	defer a.Release()
	_ = store.MarkRunning(a)
	_ = store.SaveCompleted(a)
	time.Sleep(5 * time.Millisecond)

	b, _ := store.PrepareFresh("sys", "work")
	defer b.Release()
	b.Title = "调研竞品"
	_ = store.MarkQueued(b)

	corrupt := filepath.Join(mustStoreDir(t, store), "sa_broken.meta.json")
	_ = os.WriteFile(corrupt, []byte("{not json"), 0o644)

	runs, err := store.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("ListRuns 条数 = %d（损坏 meta 应跳过）, want 2", len(runs))
	}
	// 新优先：b（后创建）在前
	if runs[0].Ref != b.Ref || runs[0].Status != SubagentQueued || runs[0].Title != "调研竞品" {
		t.Fatalf("首条 = %+v, want %s queued 调研竞品", runs[0], b.Ref)
	}
	if runs[1].Ref != a.Ref || runs[1].Status != SubagentCompleted {
		t.Fatalf("次条 = %+v, want %s completed", runs[1], a.Ref)
	}
}

func mustStoreDir(t *testing.T, s *SubagentStore) string {
	t.Helper()
	d, err := s.dirResolved()
	if err != nil {
		t.Fatalf("dirResolved: %v", err)
	}
	return d
}

func TestListRunsEmptyWhenNoDir(t *testing.T) {
	store := NewSubagentStore(filepath.Join(t.TempDir(), "never-created"))
	runs, err := store.ListRuns()
	if err != nil {
		t.Fatalf("缺目录应返回空而非错误: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("缺目录应空列表, got %d", len(runs))
	}
}

func TestSubagentListToolExecute(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	a, _ := store.PrepareFresh("sys", "work")
	defer a.Release()
	a.Title = "摘要文档"
	_ = store.MarkRunning(a)
	_ = store.SaveCompleted(a)

	tool := NewSubagentListTool(store)
	out, err := tool.Execute(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{a.Ref, "[completed]", "摘要文档", "interrupt_agent", "continue_from"} {
		if !strings.Contains(out, want) {
			t.Fatalf("输出缺 %q:\n%s", want, out)
		}
	}

	// 空会话：诚实空态，不报错
	empty := NewSubagentListTool(NewSubagentStore(filepath.Join(t.TempDir(), "none")))
	out, err = empty.Execute(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatalf("空会话 Execute: %v", err)
	}
	if !strings.Contains(out, "No sub-agent runs") {
		t.Fatalf("空会话应给诚实空态: %s", out)
	}
}

func TestSubagentMetaToolsExcludeControlPlane(t *testing.T) {
	// 控制面三件必须在元工具排除清单里：子代理不见 task/interrupt/list——
	// 中断权与兄弟清单只属于父代理。
	excluded := map[string]bool{}
	for _, n := range SubagentMetaTools() {
		excluded[n] = true
	}
	for _, want := range []string{"task", "run_skill", "install_skill", "exit_plan_mode", "interrupt_agent", "subagent_list"} {
		if !excluded[want] {
			t.Fatalf("元工具排除清单缺 %q: %v", want, SubagentMetaTools())
		}
	}
}
