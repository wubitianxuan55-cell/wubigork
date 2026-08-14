package control

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/event"
)

// queueRunner 记录每次 Run 的输入；release 关闭前 Run 会阻塞，用于制造
// 「回合进行中」窗口以测试 Send 排队（T6-2.5）。
type queueRunner struct {
	mu      sync.Mutex
	inputs  []string
	calls   int
	release chan struct{}
}

func (f *queueRunner) Run(ctx context.Context, input string) (*agent.TurnResult, error) {
	f.mu.Lock()
	f.inputs = append(f.inputs, input)
	f.calls++
	f.mu.Unlock()
	select {
	case <-f.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &agent.TurnResult{Success: true}, nil
}

func (f *queueRunner) snap() ([]string, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.inputs))
	copy(out, f.inputs)
	return out, f.calls
}

// recordSink 记录事件，供测试断言入队/队满 notice 与 TurnDone。
type recordSink struct {
	mu     sync.Mutex
	events []event.Event
}

func (s *recordSink) Emit(e event.Event) {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
}

func (s *recordSink) snapshot() []event.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]event.Event, len(s.events))
	copy(out, s.events)
	return out
}

func (s *recordSink) count(kind event.Kind) int {
	n := 0
	for _, e := range s.snapshot() {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

func (s *recordSink) notices(substr string) int {
	n := 0
	for _, e := range s.snapshot() {
		if e.Kind == event.Notice && strings.Contains(e.Text, substr) {
			n++
		}
	}
	return n
}

func (s *recordSink) hasWarnNotice(substr string) bool {
	for _, e := range s.snapshot() {
		if e.Kind == event.Notice && e.Level == event.LevelWarn && strings.Contains(e.Text, substr) {
			return true
		}
	}
	return false
}

func eventuallySend(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within 3s")
}

// TestSendQueuesWhileRunningAndDrainsInOrder 验证 T6-2.5：回合进行中 Send
// 不再被静默丢弃，而是入队并在回合结束后按序执行。
func TestSendQueuesWhileRunningAndDrainsInOrder(t *testing.T) {
	run := &queueRunner{release: make(chan struct{})}
	sink := &recordSink{}
	c := New(Options{Runner: run, Sink: sink})

	c.Send("first") // 无在途回合：立即启动
	eventuallySend(t, func() bool { _, n := run.snap(); return n == 1 })

	// 回合进行中：后续消息应入队（notice 提示），而不是被忽略
	c.Send("second")
	c.Send("third")
	if got := sink.notices("发送队列"); got != 2 {
		t.Fatalf("入队 notice 数量 = %d, want 2", got)
	}

	close(run.release)
	eventuallySend(t, func() bool { _, n := run.snap(); return n == 3 })

	inputs, _ := run.snap()
	if want := []string{"first", "second", "third"}; !reflect.DeepEqual(inputs, want) {
		t.Fatalf("runner 输入 = %v, want %v（按序执行）", inputs, want)
	}
	if got := sink.count(event.TurnDone); got != 3 {
		t.Fatalf("TurnDone 数量 = %d, want 3（每个回合一次）", got)
	}
	if c.Running() {
		t.Fatal("队列排空后控制器应回到非运行态")
	}
}

// TestSendQueueFullRejectsWithNotice 验证队满拒绝：队列上限 sendQueueLimit 条，
// 超出时新消息被拒绝并发出明确错误 notice，且被拒绝的消息不会被执行。
func TestSendQueueFullRejectsWithNotice(t *testing.T) {
	run := &queueRunner{release: make(chan struct{})}
	sink := &recordSink{}
	c := New(Options{Runner: run, Sink: sink})

	c.Send("first")
	eventuallySend(t, func() bool { _, n := run.snap(); return n == 1 })

	// 填满队列（上限 sendQueueLimit 条）
	for i := 0; i < sendQueueLimit; i++ {
		c.Send(fmt.Sprintf("queued-%d", i))
	}
	// 第 9 条：队满拒绝
	c.Send("overflow")

	close(run.release)
	eventuallySend(t, func() bool { _, n := run.snap(); return n == 1+sendQueueLimit })

	inputs, _ := run.snap()
	if len(inputs) != 1+sendQueueLimit {
		t.Fatalf("runner 收到 %d 条, want %d", len(inputs), 1+sendQueueLimit)
	}
	for i := 0; i < sendQueueLimit; i++ {
		if inputs[1+i] != fmt.Sprintf("queued-%d", i) {
			t.Fatalf("input[%d] = %q, want queued-%d（按序执行）", 1+i, inputs[1+i], i)
		}
	}
	if inputs[len(inputs)-1] == "overflow" {
		t.Fatal("队满被拒绝的消息不应被执行")
	}
	if !sink.hasWarnNotice("发送队列已满") {
		t.Fatal("队满时未发出明确错误 notice（含「发送队列已满」）")
	}
}

// TestSendWithoutRunningTurnStartsImmediately 验证非 running 期行为不变：
// 无在途回合时 Send 立即各自启动回合，不入队、无队列 notice。
func TestSendWithoutRunningTurnStartsImmediately(t *testing.T) {
	run := &queueRunner{release: make(chan struct{})}
	close(run.release) // 不阻塞任何回合
	sink := &recordSink{}
	c := New(Options{Runner: run, Sink: sink})

	c.Send("a")
	eventuallySend(t, func() bool { _, n := run.snap(); return n == 1 })
	c.Send("b")
	eventuallySend(t, func() bool { _, n := run.snap(); return n == 2 })

	if got := sink.notices("发送队列"); got != 0 {
		t.Fatalf("非运行期 Send 不应产生入队 notice，got %d", got)
	}
}

// TestSendQueuePreservesInputThroughQueue 验证排队消息的 input 原样保留：
// SendWithRaw 入队后，回合结束时用同一 input 执行（不串台、不丢失）。
func TestSendQueuePreservesInputThroughQueue(t *testing.T) {
	run := &queueRunner{release: make(chan struct{})}
	c := New(Options{Runner: run})

	c.SendWithRaw("first-input", "first-raw")
	eventuallySend(t, func() bool { _, n := run.snap(); return n == 1 })

	c.SendWithRaw("second-input", "second-raw")
	close(run.release)
	eventuallySend(t, func() bool { _, n := run.snap(); return n == 2 })

	inputs, _ := run.snap()
	if inputs[1] != "second-input" {
		t.Fatalf("排队消息 input = %q, want second-input（原样保留）", inputs[1])
	}
}
