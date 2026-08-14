package control

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/event"
)

// blockRunner 阻塞直到 release 关闭或 ctx 取消；started（可选）在首次 Run
// 进入阻塞前被关闭，供测试同步「回合已开始」。
type blockRunner struct {
	mu        sync.Mutex
	calls     int
	release   chan struct{}
	started   chan struct{}
	startOnce sync.Once
}

func (f *blockRunner) Run(ctx context.Context, input string) (*agent.TurnResult, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.started != nil {
		f.startOnce.Do(func() { close(f.started) })
	}
	select {
	case <-f.release:
		return &agent.TurnResult{Success: true}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// stagedRunner 第一次 Run 阻塞（release/ctx 取消），之后立即成功——
// 用于验证看门狗终止首回合后 Send 队列仍按序继续（T6-2.5 队列行为不破坏）。
type stagedRunner struct {
	mu        sync.Mutex
	calls     int
	inputs    []string
	release   chan struct{}
	started   chan struct{}
	startOnce sync.Once
}

func (f *stagedRunner) Run(ctx context.Context, input string) (*agent.TurnResult, error) {
	f.mu.Lock()
	f.calls++
	n := f.calls
	f.inputs = append(f.inputs, input)
	f.mu.Unlock()
	if f.started != nil {
		f.startOnce.Do(func() { close(f.started) })
	}
	if n == 1 {
		select {
		case <-f.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return &agent.TurnResult{Success: true}, nil
}

func (f *stagedRunner) inputsSnapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.inputs))
	copy(out, f.inputs)
	return out
}

// progressRunner 阻塞期间周期性发射 Text 推进事件（经 New 重接线的包装
// sink——实现了 SetSink，与生产装配一致），用于验证停滞计时被推进重置。
type progressRunner struct {
	mu        sync.Mutex
	sink      event.Sink
	tick      time.Duration
	release   chan struct{}
	started   chan struct{}
	startOnce sync.Once
}

func (f *progressRunner) SetSink(s event.Sink) {
	f.mu.Lock()
	f.sink = s
	f.mu.Unlock()
}

func (f *progressRunner) Run(ctx context.Context, input string) (*agent.TurnResult, error) {
	if f.started != nil {
		f.startOnce.Do(func() { close(f.started) })
	}
	ticker := time.NewTicker(f.tick)
	defer ticker.Stop()
	for {
		select {
		case <-f.release:
			return &agent.TurnResult{Success: true}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			f.mu.Lock()
			s := f.sink
			f.mu.Unlock()
			if s != nil {
				s.Emit(event.Event{Kind: event.Text, Text: "progress"})
			}
		}
	}
}

// waitNTurnDones 轮询直到 sink 出现至少 n 个 TurnDone 并返回它们。
func waitNTurnDones(t *testing.T, sink *recordSink, n int) []event.Event {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		evs := sink.snapshot()
		var out []event.Event
		for _, e := range evs {
			if e.Kind == event.TurnDone {
				out = append(out, e)
			}
		}
		if len(out) >= n {
			return out
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("等待 %d 个 TurnDone 超时（当前 %d 个）", n, sink.count(event.TurnDone))
	return nil
}

// TestWatchdogStallTerminatesTurn 验证停滞回合被看门狗终止并发出
// TurnDone(Err)（原因可识别）与 Notice，控制器回到非运行态。
func TestWatchdogStallTerminatesTurn(t *testing.T) {
	run := &blockRunner{release: make(chan struct{}), started: make(chan struct{})}
	sink := &recordSink{}
	c := New(Options{Runner: run, Sink: sink,
		Watchdog: WatchdogConfig{WallClock: -1, Stall: 150 * time.Millisecond}})

	c.Send("stall-me")
	<-run.started // 回合已进入阻塞

	td := waitNTurnDones(t, sink, 1)[0]
	if td.Err == nil {
		t.Fatal("看门狗应终止停滞回合并带 Err，got nil")
	}
	msg := td.Err.Error()
	if !strings.Contains(msg, "watchdog") || !strings.Contains(msg, "stalled") {
		t.Fatalf("TurnDone.Err = %q, want 看门狗停滞原因", msg)
	}
	if !sink.hasWarnNotice("看门狗") {
		t.Fatal("看门狗触发后应发 LevelWarn Notice（含「看门狗」）")
	}
	if c.Running() {
		t.Fatal("看门狗终止后控制器应回到非运行态")
	}
	close(run.release) // 清理（runner 已因取消返回，此关闭无副作用）
}

// TestWatchdogWallClockTerminatesTurn 验证墙钟超时触发：即使没有停滞
// （本配置禁用停滞维度），回合也会被强制终止。
func TestWatchdogWallClockTerminatesTurn(t *testing.T) {
	run := &blockRunner{release: make(chan struct{}), started: make(chan struct{})}
	sink := &recordSink{}
	c := New(Options{Runner: run, Sink: sink,
		Watchdog: WatchdogConfig{WallClock: 200 * time.Millisecond, Stall: -1}})

	c.Send("slow-turn")
	<-run.started

	td := waitNTurnDones(t, sink, 1)[0]
	if td.Err == nil {
		t.Fatal("墙钟超时后应带 Err，got nil")
	}
	msg := td.Err.Error()
	if !strings.Contains(msg, "watchdog") || !strings.Contains(msg, "wall-clock") {
		t.Fatalf("TurnDone.Err = %q, want 墙钟超时原因", msg)
	}
	if c.Running() {
		t.Fatal("墙钟终止后控制器应回到非运行态")
	}
	close(run.release)
}

// TestWatchdogDoesNotKillFastTurn 验证正常快速回合不被误杀：即使阈值很小，
// 回合正常完成时看门狗不发 Notice、不带 Err。
func TestWatchdogDoesNotKillFastTurn(t *testing.T) {
	run := &blockRunner{release: make(chan struct{})}
	close(run.release) // 立即返回，不阻塞
	sink := &recordSink{}
	c := New(Options{Runner: run, Sink: sink,
		Watchdog: WatchdogConfig{WallClock: 100 * time.Millisecond, Stall: 50 * time.Millisecond}})

	c.Send("fast-turn")
	td := waitNTurnDones(t, sink, 1)[0]
	if td.Err != nil {
		t.Fatalf("快速回合被看门狗误杀：%v", td.Err)
	}
	if sink.hasWarnNotice("看门狗") {
		t.Fatal("快速回合不应触发看门狗 Notice")
	}
	if c.Running() {
		t.Fatal("回合结束后控制器应回到非运行态")
	}
}

// TestWatchdogConfigPerDimension 验证配置阈值逐维度生效：只开停滞时停滞后
// 触发（而非墙钟）；只开墙钟时墙钟触发（停滞维度禁用）；零值配置 = 生产默认。
func TestWatchdogConfigPerDimension(t *testing.T) {
	t.Run("stall-only-config", func(t *testing.T) {
		run := &blockRunner{release: make(chan struct{})}
		sink := &recordSink{}
		c := New(Options{Runner: run, Sink: sink,
			Watchdog: WatchdogConfig{WallClock: -1, Stall: 120 * time.Millisecond}})
		c.Send("x")
		td := waitNTurnDones(t, sink, 1)[0]
		if td.Err == nil || !strings.Contains(td.Err.Error(), "stalled") {
			t.Fatalf("停滞维度配置未生效：%v", td.Err)
		}
		close(run.release)
	})
	t.Run("wall-only-config", func(t *testing.T) {
		run := &blockRunner{release: make(chan struct{})}
		sink := &recordSink{}
		c := New(Options{Runner: run, Sink: sink,
			Watchdog: WatchdogConfig{WallClock: 150 * time.Millisecond, Stall: -1}})
		c.Send("x")
		td := waitNTurnDones(t, sink, 1)[0]
		if td.Err == nil || !strings.Contains(td.Err.Error(), "wall-clock") {
			t.Fatalf("墙钟维度配置未生效：%v", td.Err)
		}
		close(run.release)
	})
	t.Run("zero-config-defaults", func(t *testing.T) {
		// 零值配置 = 生产默认（墙钟 10min / 停滞 30s），不因零值而关闭看门狗。
		c := New(Options{Runner: &blockRunner{release: make(chan struct{})}, Sink: event.Discard})
		if c.watchdog == nil {
			t.Fatal("零值 Watchdog 配置应启用默认看门狗")
		}
		if c.watchdog.cfg.WallClock != DefaultWatchdog.WallClock || c.watchdog.cfg.Stall != DefaultWatchdog.Stall {
			t.Fatalf("默认阈值 = %+v, want %+v", c.watchdog.cfg, DefaultWatchdog)
		}
	})
}

// TestWatchdogProgressResetsStall 验证「推进」信号重置停滞计时：
// 正常持续输出的长回合（周期 Text 事件）不会被停滞检测误杀。
func TestWatchdogProgressResetsStall(t *testing.T) {
	run := &progressRunner{tick: 40 * time.Millisecond, release: make(chan struct{}), started: make(chan struct{})}
	sink := &recordSink{}
	c := New(Options{Runner: run, Sink: sink,
		Watchdog: WatchdogConfig{WallClock: -1, Stall: 200 * time.Millisecond}})

	c.Send("long-progressing-turn")
	<-run.started

	// 持续推进 700ms（远超停滞阈值）：不应触发
	time.Sleep(700 * time.Millisecond)
	if sink.count(event.TurnDone) != 0 {
		t.Fatal("持续推进的回合不应被看门狗终止")
	}
	if sink.hasWarnNotice("看门狗") {
		t.Fatal("持续推进的回合不应触发看门狗 Notice")
	}

	close(run.release)
	td := waitNTurnDones(t, sink, 1)[0]
	if td.Err != nil {
		t.Fatalf("释放后回合应正常完成：%v", td.Err)
	}
}

// TestWatchdogToolExecutionExemptsStall 白盒验证工具执行期间停滞豁免：
// ToolDispatch → ToolResult 之间停滞计时挂起（长工具调用如大文件转换不被误杀），
// 工具结束后再无推进则触发。
func TestWatchdogToolExecutionExemptsStall(t *testing.T) {
	wd := newWatchdog(WatchdogConfig{WallClock: -1, Stall: 100 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	wd.begin(ctx, cancel)
	defer wd.end()
	sink := watchdogSink{inner: event.Discard, wd: wd}

	// 长工具调用在途：停滞豁免
	sink.Emit(event.Event{Kind: event.ToolDispatch, Tool: event.Tool{Name: "format_convert"}})
	time.Sleep(350 * time.Millisecond) // > 停滞阈值，但工具在途
	if wd.fired.Load() {
		t.Fatal("工具执行期间不应触发停滞看门狗")
	}

	// 工具结束且之后无任何推进 → 停滞触发
	sink.Emit(event.Event{Kind: event.ToolResult, Tool: event.Tool{Name: "format_convert"}})
	time.Sleep(350 * time.Millisecond)
	if !wd.fired.Load() {
		t.Fatal("工具结束后无推进应触发停滞看门狗")
	}
	if reason, _ := wd.firedReason(); !strings.Contains(reason, "stalled") {
		t.Fatalf("触发原因 = %q, want stalled", reason)
	}
}

// TestWatchdogUserWaitExemptsStall 白盒验证等待用户输入（审批/提问）期间
// 停滞豁免：用户思考多久都不杀回合，答复后回合恢复再按停滞检测。
func TestWatchdogUserWaitExemptsStall(t *testing.T) {
	wd := newWatchdog(WatchdogConfig{WallClock: -1, Stall: 100 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	wd.begin(ctx, cancel)
	defer wd.end()
	sink := watchdogSink{inner: event.Discard, wd: wd}

	// 等待用户审批：停滞豁免
	sink.Emit(event.Event{Kind: event.ApprovalRequest, Approval: event.Approval{ID: "1", Tool: "bash"}})
	time.Sleep(350 * time.Millisecond)
	if wd.fired.Load() {
		t.Fatal("等待用户输入期间不应触发停滞看门狗")
	}

	// 用户答复后回合恢复（首个推进事件解除豁免）；再无推进 → 停滞触发
	sink.Emit(event.Event{Kind: event.Text, Text: "继续"})
	time.Sleep(350 * time.Millisecond)
	if !wd.fired.Load() {
		t.Fatal("用户等待解除后无推进应触发停滞")
	}
}

// TestWatchdogKilledTurnThenQueueDrains 验证看门狗终止首回合后 Send 队列
// 仍按序排空（T6-2.5 队列行为不破坏）：第二条消息正常执行、正常 TurnDone。
func TestWatchdogKilledTurnThenQueueDrains(t *testing.T) {
	run := &stagedRunner{release: make(chan struct{}), started: make(chan struct{})}
	sink := &recordSink{}
	c := New(Options{Runner: run, Sink: sink,
		Watchdog: WatchdogConfig{WallClock: -1, Stall: 120 * time.Millisecond}})

	c.Send("first")
	<-run.started // 首回合阻塞中
	c.Send("second")

	dones := waitNTurnDones(t, sink, 2)
	if dones[0].Err == nil || !strings.Contains(dones[0].Err.Error(), "watchdog") {
		t.Fatalf("首回合应被看门狗终止：%v", dones[0].Err)
	}
	if dones[1].Err != nil {
		t.Fatalf("队列中第二条消息不应被误杀：%v", dones[1].Err)
	}
	inputs := run.inputsSnapshot()
	if len(inputs) != 2 || inputs[0] != "first" || inputs[1] != "second" {
		t.Fatalf("runner 输入 = %v, want [first second]（队列仍按序排空）", inputs)
	}
	if c.Running() {
		t.Fatal("队列排空后控制器应回到非运行态")
	}
	close(run.release) // 清理（首回合已被取消，此关闭无副作用）
}

// TestWatchdogDisabledWhenConfigNegative 验证两维度均 < 0 时看门狗整体
// 不装配：阻塞回合继续运行，不被强制终止，正常释放后完成。
func TestWatchdogDisabledWhenConfigNegative(t *testing.T) {
	run := &blockRunner{release: make(chan struct{}), started: make(chan struct{})}
	sink := &recordSink{}
	c := New(Options{Runner: run, Sink: sink,
		Watchdog: WatchdogConfig{WallClock: -1, Stall: -1}})
	if c.watchdog != nil {
		t.Fatal("两维度均禁用时不应装配看门狗")
	}

	c.Send("x")
	<-run.started
	time.Sleep(300 * time.Millisecond)
	if !c.Running() {
		t.Fatal("禁用看门狗时回合应继续运行")
	}
	if sink.count(event.TurnDone) != 0 {
		t.Fatal("禁用看门狗时不应有 TurnDone")
	}

	close(run.release)
	td := waitNTurnDones(t, sink, 1)[0]
	if td.Err != nil {
		t.Fatalf("释放后回合应正常完成：%v", td.Err)
	}
}
