package jobs

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

// 终态任务超过 terminalJobLimit 时，最旧的被淘汰（Output 转为 unknown），
// 最近的任务仍可读，运行中的任务不受影响。
func TestTerminalJobEviction(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	run := func(_ context.Context, _ io.Writer) (string, error) { return "ok", nil }

	// 先启动并等一个任务完成（它将成为最旧的终态任务）。
	first := m.Start("bash", "first", run)
	m.Wait(context.Background(), []string{first.ID}, 5)

	// 再灌 terminalJobLimit 个新终态任务，把 first 挤出保留窗口。
	for i := 0; i < terminalJobLimit; i++ {
		j := m.Start("bash", "filler", run)
		m.Wait(context.Background(), []string{j.ID}, 5)
	}

	if _, status, ok := m.Output(first.ID); ok {
		t.Fatalf("oldest terminal job should be evicted, got ok=true status=%q", status)
	}

	// 最近一个任务仍在，可读输出。
	last := m.Start("bash", "last", run)
	m.Wait(context.Background(), []string{last.ID}, 5)
	text, status, ok := m.Output(last.ID)
	if !ok || status != Done || !strings.Contains(text, "ok") {
		t.Fatalf("recent job readable, got ok=%v status=%q text=%q", ok, status, text)
	}

	// 运行中任务不被淘汰：启动一个长任务后确认可 Wait 到。
	long := m.Start("bash", "long", func(ctx context.Context, _ io.Writer) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})
	if got := len(m.Running()); got == 0 {
		t.Fatal("running job missing from Running() after pruning")
	}
	m.Kill(long.ID)
	m.Wait(context.Background(), []string{long.ID}, 5)
}
