package control

// B1（codex 蒸馏第四批 P1）准入与优雅关闭验收。
// 口径=asset-47 §9 A1–A3（A4 连接闸门不在 P1 落地面——决策视图只保留
// Controller/App 生命周期；A8 等价性由「默认 draining=false 走原路径」的
// 既有全量测试覆盖）。

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
)

func countKinds(evs []event.Event, k event.Kind) int {
	n := 0
	for _, e := range evs {
		if e.Kind == k {
			n++
		}
	}
	return n
}

// A1：关闸后 Send 被拒收——结构化通知、不启动回合（无 TurnStarted、
// running 保持 false）。
func TestDrainRejectsSend(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	if err := c.BeginDrain(time.Second); err != nil {
		t.Fatalf("idle BeginDrain: %v", err)
	}
	if !c.Draining() {
		t.Fatal("Draining() = false after BeginDrain")
	}
	c.Send("hello")
	if c.Running() {
		t.Error("running = true, want false（拒收不开跑）")
	}
	evs := rec.snapshot()
	if got := countKinds(evs, event.TurnStarted); got != 0 {
		t.Errorf("TurnStarted = %d, want 0", got)
	}
	if got := countKinds(evs, event.Notice); got != 1 {
		t.Fatalf("Notice = %d, want 1（结构化拒收）", got)
	}
	if !strings.Contains(evs[len(evs)-1].Text, "未被接受") || !strings.Contains(evs[len(evs)-1].Text, "hello") {
		t.Errorf("拒收通知不含消息预览: %q", evs[len(evs)-1].Text)
	}
}

// A2：关闸前已准入的回合跑完才返回——BeginDrain 阻塞到回合收尾，
// TurnDone 恰好一条。
func TestDrainWaitsForAdmittedTurn(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	release := make(chan struct{})
	started := make(chan struct{})
	if !c.tryStartTurn(func(ctx context.Context) error {
		close(started)
		<-release
		return nil
	}) {
		t.Fatal("tryStartTurn = false")
	}
	<-started

	drained := make(chan error, 1)
	go func() { drained <- c.BeginDrain(5 * time.Second) }()
	select {
	case err := <-drained:
		t.Fatalf("BeginDrain returned early while turn in flight: %v", err)
	case <-time.After(80 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-drained:
		if err != nil {
			t.Fatalf("BeginDrain: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("BeginDrain did not return after turn finished")
	}
	if got := countKinds(rec.snapshot(), event.TurnDone); got != 1 {
		t.Errorf("TurnDone = %d, want 1", got)
	}
}

// A3：关闸超时→强制取消在飞回合，BeginDrain 返回超时错误；回合的
// TurnDone{Err} 由既有路径落账。
func TestDrainTimeoutForceCancels(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	started := make(chan struct{})
	var sawCancel atomic.Bool
	if !c.tryStartTurn(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		sawCancel.Store(true)
		return ctx.Err()
	}) {
		t.Fatal("tryStartTurn = false")
	}
	<-started

	err := c.BeginDrain(60 * time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "drain timeout") {
		t.Fatalf("BeginDrain err = %v, want timeout", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sawCancel.Load() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !sawCancel.Load() {
		t.Error("running turn was not cancelled by drain timeout")
	}
	// TurnDone 落账（Err 非空）。
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if countKinds(rec.snapshot(), event.TurnDone) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := countKinds(rec.snapshot(), event.TurnDone); got != 1 {
		t.Errorf("TurnDone = %d, want 1（强退也要落账）", got)
	}
}

// 排队未跑的消息：关闸时逐条通知作废，不静默丢，不计入收敛等待。
func TestDrainDropsQueuedSendsWithNotice(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	block := make(chan struct{})
	started := make(chan struct{})
	if !c.tryStartTurn(func(ctx context.Context) error {
		close(started)
		<-block
		return nil
	}) {
		t.Fatal("tryStartTurn = false")
	}
	<-started
	c.Send("queued-1")
	c.Send("queued-2")

	drained := make(chan error, 1)
	go func() { drained <- c.BeginDrain(5 * time.Second) }()
	// 等两条排队消息的通知先落地（在回合放行之前就该出现）。
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if countKinds(rec.snapshot(), event.Notice) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	notices := countKinds(rec.snapshot(), event.Notice)
	if notices < 2 {
		t.Fatalf("queued-drop notices = %d, want >=2（逐条通知不静默）", notices)
	}
	// 放行在飞回合：队列已作废，drain 只等这一个回合。
	close(block)
	select {
	case err := <-drained:
		if err != nil {
			t.Fatalf("BeginDrain: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("BeginDrain did not converge（作废队列不应拖住收敛）")
	}
	if got := countKinds(rec.snapshot(), event.TurnStarted); got > 1 {
		t.Errorf("TurnStarted = %d, want 1（作废消息不再开跑）", got)
	}
}

// 关闸后系统回合（runGuarded/tryStartTurn 路径）同样拒收。
func TestDrainRejectsSystemTurn(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	if err := c.BeginDrain(time.Second); err != nil {
		t.Fatalf("BeginDrain: %v", err)
	}
	ran := false
	c.runGuarded(func(ctx context.Context) error { ran = true; return nil })
	if ran {
		t.Error("system turn ran during drain, want rejected")
	}
}
