package tasks

import (
	"context"
	"os/exec"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// spawnIgnorantProcess 启动一个不感知 ctx 的长驻子进程（Windows ping / Unix
// sleep），返回 cmd。协作取消对它无能为力，只有进程树击杀能结束它。
func spawnIgnorantProcess(t *testing.T) *exec.Cmd {
	t.Helper()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "30", "127.0.0.1")
	} else {
		cmd = exec.Command("sleep", "30")
	}
	proc.SetProcessGroupKill(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("启动子进程: %v", err)
	}
	t.Cleanup(func() { proc.KillTree(cmd) })
	return cmd
}

// TestKillRunningKillsRegisteredProcess 强杀运行中任务：OnForceKill 登记的
// 钩子被执行（子进程真死），任务终态 cancelled（用户取消意图胜过 handler
// 返回值）。
func TestKillRunningKillsRegisteredProcess(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	var procDead atomic.Bool
	started := make(chan struct{})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		cmd := spawnIgnorantProcess(t)
		p.OnForceKill(func() { proc.KillTree(cmd) })
		waitCh := make(chan error, 1)
		go func() { waitCh <- cmd.Wait() }()
		close(started)
		select {
		case <-waitCh:
			procDead.Store(true) // 进程已死：钩子生效（或自行退出，用例内 30s 不可能）
		case <-ctx.Done():
			proc.KillTree(cmd) // 兜底：Close 等路径不残留子进程
		}
		return ctx.Err()
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindPriceFetch, "抓取价格", nil)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("任务未开始")
	}
	if err := m.Kill(tk.ID); err != nil {
		t.Fatalf("kill: %v", err)
	}
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusCancelled) {
		t.Fatalf("期望 cancelled，实际 %s", done.Status)
	}
	if !procDead.Load() {
		t.Fatal("强杀钩子未击杀子进程")
	}
}

// TestKillQueuedTaskMarksCancelled 强杀 queued 任务：原子取消、永不运行，
// message 区分强杀措辞。
func TestKillQueuedTaskMarksCancelled(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{MaxConcurrent: 1, BackoffBase: 10 * time.Millisecond})
	block := make(chan struct{})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		<-block
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	defer close(block)
	first, _ := m.Submit(KindFileIndex, "任务一", nil)
	second, _ := m.Submit(KindFileIndex, "任务二", nil)
	deadline := time.Now().Add(2 * time.Second)
	for {
		f, _ := m.Get(first.ID)
		if f.Status == string(StatusRunning) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("第一个任务未开始")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := m.Kill(second.ID); err != nil {
		t.Fatalf("kill: %v", err)
	}
	done := waitTerminal(t, m, second.ID, 3*time.Second)
	if done.Status != string(StatusCancelled) {
		t.Fatalf("期望 cancelled，实际 %s", done.Status)
	}
	if done.Message != "已强制终止" {
		t.Fatalf("期望 message=已强制终止，实际 %q", done.Message)
	}
	if done.StartedAt != 0 {
		t.Fatal("queued 强杀不应进入运行态")
	}
}

// TestKillWithoutHooksEqualsCancel 纯函数任务（未登记钩子）Kill 等价协作
// 取消；重复 Kill / 对已结束任务 Kill 报错。
func TestKillWithoutHooksEqualsCancel(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	started := make(chan struct{})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindPriceFetch, "抓取价格", nil)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("任务未开始")
	}
	if err := m.Kill(tk.ID); err != nil {
		t.Fatalf("kill: %v", err)
	}
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusCancelled) {
		t.Fatalf("期望 cancelled，实际 %s", done.Status)
	}
	if err := m.Kill(tk.ID); err == nil {
		t.Fatal("对已结束任务 Kill 应报错")
	}
	if err := m.Kill("no-such-task"); err == nil {
		t.Fatal("对不存在任务 Kill 应报错")
	}
}

// TestKillHooksClearedAfterAttemptEnd 尝试正常结束后钩子不残留：成功终态的
// 任务再 Kill 报错，且旧尝试登记的钩子不被误触发。
func TestKillHooksClearedAfterAttemptEnd(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	var hookFired atomic.Bool
	started := make(chan struct{})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		p.OnForceKill(func() { hookFired.Store(true) })
		close(started)
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindPriceFetch, "抓取价格", nil)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("任务未开始")
	}
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusSucceeded) {
		t.Fatalf("期望 succeeded，实际 %s", done.Status)
	}
	if err := m.Kill(tk.ID); err == nil {
		t.Fatal("对已成功任务 Kill 应报错")
	}
	time.Sleep(50 * time.Millisecond)
	if hookFired.Load() {
		t.Fatal("终态后旧钩子不应被触发")
	}
}
