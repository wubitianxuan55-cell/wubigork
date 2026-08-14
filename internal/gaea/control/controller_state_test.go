package control

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// fakeStateRunner 记录输入并按需写入助手消息 / 返回错误，模拟一次回合执行。
type fakeStateRunner struct {
	mu    sync.Mutex
	last  string
	err   error
	onRun func() // 可选：回合中向 executor 会话写入助手消息
}

func (f *fakeStateRunner) Run(ctx context.Context, input string) (*agent.TurnResult, error) {
	f.mu.Lock()
	f.last = input
	f.mu.Unlock()
	if f.onRun != nil {
		f.onRun()
	}
	if f.err != nil {
		return nil, f.err
	}
	return &agent.TurnResult{Success: true}, nil
}

// TestTurnWritesStateNotRunning 验证一次正常回合结束后，会话状态文件
// running=false 且摘要为最后助手消息，供列表识别「未中断」。
func TestTurnWritesStateNotRunning(t *testing.T) {
	dir := t.TempDir()
	exec := agent.New(nil, nil, agent.NewSession("you are gaea"), agent.Options{}, event.Discard)
	run := &fakeStateRunner{onRun: func() {
		exec.Session().Add(provider.Message{Role: provider.RoleAssistant, Content: "已完成第一阶段，准备继续。"})
	}}
	c := New(Options{
		Runner:     run,
		Executor:   exec,
		Sink:       event.Discard,
		SessionDir: dir,
		Label:      "test-model",
	})
	if err := c.runTurnWithRaw(context.Background(), "开始任务", "开始任务"); err != nil {
		t.Fatalf("runTurnWithRaw: %v", err)
	}
	path := c.SessionPath()
	if path == "" {
		t.Fatal("回合后未生成会话路径")
	}
	st, err := session.LoadState(session.StatePath(path))
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if st.Running {
		t.Fatalf("回合正常结束后 state.running = true，want false")
	}
	if st.Summary != "已完成第一阶段，准备继续。" {
		t.Fatalf("state.summary = %q, want 最后助手消息", st.Summary)
	}
}

// TestTurnErrorStillClearsState 验证回合失败时 defer 仍把 running 写回 false，
// 一次失败不会被误判为「未完成」。
func TestTurnErrorStillClearsState(t *testing.T) {
	dir := t.TempDir()
	exec := agent.New(nil, nil, agent.NewSession(""), agent.Options{}, event.Discard)
	c := New(Options{
		Runner:     &fakeStateRunner{err: errors.New("provider 500")},
		Executor:   exec,
		Sink:       event.Discard,
		SessionDir: dir,
		Label:      "test-model",
	})
	if err := c.runTurnWithRaw(context.Background(), "x", "x"); err == nil {
		t.Fatal("期望回合返回错误")
	}
	st, err := session.LoadState(session.StatePath(c.SessionPath()))
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if st.Running {
		t.Fatal("回合失败后 running 仍为 true，want false")
	}
}

// TestHeadlessRunSkipsState 验证 headless Run 路径不写状态文件：
// 只有交互回合（runTurnWithRaw）参与中断标记。
func TestHeadlessRunSkipsState(t *testing.T) {
	dir := t.TempDir()
	exec := agent.New(nil, nil, agent.NewSession(""), agent.Options{}, event.Discard)
	c := New(Options{
		Runner:     &fakeStateRunner{},
		Executor:   exec,
		Sink:       event.Discard,
		SessionDir: dir,
		Label:      "test-model",
	})
	if err := c.Run(context.Background(), "headless 任务"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, err := session.LoadState(session.StatePath(c.SessionPath()))
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if st.Running {
		t.Fatal("headless Run 不应写 running 标记")
	}
	// 无状态文件 → LoadState 零值；若文件被写出来，UpdatedAt 会非零
	if st.UpdatedAt != 0 {
		t.Fatalf("headless Run 不应创建状态文件，got %+v", st)
	}
}

// TestTruncateSummary 验证摘要截断到 240 字符并按 rune 计算（兼容中文）。
func TestTruncateSummary(t *testing.T) {
	long := strings.Repeat("字", 300)
	if got := truncateSummary(long); len([]rune(got)) != 240 {
		t.Fatalf("truncateSummary 长度 = %d rune, want 240", len([]rune(got)))
	}
	if got := truncateSummary("  短摘要  "); got != "短摘要" {
		t.Fatalf("truncateSummary = %q, want 修剪后", got)
	}
	if got := truncateSummary(""); got != "" {
		t.Fatalf("truncateSummary(\"\") = %q, want empty", got)
	}
}
