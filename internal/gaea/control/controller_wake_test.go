package control

import (
	"context"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/jobs"
)

// Z6 蒸馏刀：后台任务完成主动唤醒——回调链、空闲判定、自激励预算三面。

// runningLocked 持锁读 running：turnLoop 在 mu 内写该字段，测试轮询裸读
// 即数据竞态（linux -race 实锤，2026-10-30）。
func runningLocked(c *Controller) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

func newWakeTestController(t *testing.T, sink event.Sink) (*Controller, *jobs.Manager, *atomic.Int32, *atomic.Pointer[string]) {
	t.Helper()
	var turns atomic.Int32
	lastP := &atomic.Pointer[string]{}
	if sink == nil {
		sink = event.FuncSink(func(event.Event) {})
	}
	c := New(Options{
		Sink:   sink,
		Runner: wakeRunnerStub{turns: &turns, last: lastP},
	})
	jm := jobs.NewManager(event.FuncSink(func(event.Event) {}))
	c.jobs = jm
	c.WireJobWake()
	return c, jm, &turns, lastP
}

// noticeCounterSink 构造期注入的计数 sink。测试中途热改 c.sink 与回合
// goroutine 的 Emit 读构成数据竞态（linux -race 实锤，2026-10-30）——sink 是
// 构造期注入的契约字段，观察一律在构造期布线。
type noticeCounterSink struct {
	text string
	hits atomic.Int32
}

func (s *noticeCounterSink) Emit(e event.Event) {
	if e.Kind == event.Notice && strings.Contains(e.Text, s.text) {
		s.hits.Add(1)
	}
}

// wakeRunnerStub 立即完成的 Runner 桩：唤醒回合被拉起即计一数，并捕获
// 最近一次模型输入（端到端验证完成注记随唤醒回合进入上下文）。
type wakeRunnerStub struct {
	turns *atomic.Int32
	last  *atomic.Pointer[string] // 共享指针：Runner 以值拷贝进接口，字段须自身为指针
}

func (s wakeRunnerStub) Run(ctx context.Context, input string) (*agent.TurnResult, error) {
	s.turns.Add(1)
	in := input
	s.last.Store(&in)
	return &agent.TurnResult{}, nil
}

func TestWakeJobCompletionStartsIdleTurn(t *testing.T) {
	c, jm, turns, lastP := newWakeTestController(t, nil)
	jm.Start("bash", "long task", func(ctx context.Context, out io.Writer) (string, error) {
		return "ok", nil
	})
	_ = io.Discard // io 保留给 jobs 签名可读性
	// 等完成回调触发唤醒回合（Runner 立即返回 → running 翻转）
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if turns.Load() >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if turns.Load() < 1 {
		t.Fatal("空闲会话的后台完成应拉起唤醒回合")
	}
	// 预算应已消耗 1 且未被真人重置
	c.mu.Lock()
	chain := c.wakeChain
	c.mu.Unlock()
	if chain != 1 {
		t.Fatalf("唤醒链应为 1，实得 %d", chain)
	}
	// 端到端：完成注记由唤醒回合的 Compose 消费进模型输入（含 Z3 防伪造
	// 同意条款与任务名）——DrainCompletedNote 在唤醒回合内已被排空。
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if p := lastP.Load(); p != nil && strings.Contains(*p, "bash") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	p := lastP.Load()
	if p == nil {
		t.Fatal("唤醒回合未捕获到模型输入")
	}
	if !strings.Contains(*p, "bash") || !strings.Contains(*p, "<background-jobs>") {
		t.Fatalf("唤醒回合输入应含 <background-jobs> 段与任务名，实得 %.200q", *p)
	}
	if !strings.Contains(*p, "NOT USER INPUT") {
		t.Fatal("唤醒回合输入缺 Z3 防伪造同意条款")
	}
}

func TestWakeSelfExcitationBudget(t *testing.T) {
	sink := &noticeCounterSink{text: "自激励预算耗尽"}
	c, _, _, _ := newWakeTestController(t, sink)
	// 直接压满预算：连续唤醒 wakeMaxConsecutive 次后再触发应被闸住
	for i := 0; i < wakeMaxConsecutive; i++ {
		c.maybeWakeByJobs()
		// 回合桩立即结束，running 回落（字段由 mu 保护，读必须同样持锁——
		// linux -race 实锤：turnLoop 锁内写 vs 测试裸读）
		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) && runningLocked(c) {
			time.Sleep(2 * time.Millisecond)
		}
	}
	c.mu.Lock()
	chain := c.wakeChain
	c.mu.Unlock()
	if chain != wakeMaxConsecutive {
		t.Fatalf("预算应压满 %d，实得 %d", wakeMaxConsecutive, chain)
	}
	// 再触发：预算耗尽，不再拉起（turn 计数不涨，只发 Notice）
	c.maybeWakeByJobs()
	if sink.hits.Load() != 1 {
		t.Fatal("预算耗尽后应发 Notice 且不再拉起回合")
	}
	// 真人消息复位预算
	c.sendWithRaw("用户回来了", "用户回来了", true)
	c.mu.Lock()
	chain = c.wakeChain
	c.mu.Unlock()
	if chain != 0 {
		t.Fatalf("真人消息应重置唤醒链，实得 %d", chain)
	}
}

func TestWakeRunningSessionSkips(t *testing.T) {
	c, _, turns, _ := newWakeTestController(t, nil)
	// 占住 running：直接拉一个不会立刻结束的回合
	release := make(chan struct{})
	c.tryStartTurn(func(ctx context.Context) error {
		<-release
		return nil
	})
	time.Sleep(20 * time.Millisecond)
	c.maybeWakeByJobs() // 在跑：应静默跳过
	c.mu.Lock()
	chain := c.wakeChain
	c.mu.Unlock()
	if chain != 0 {
		t.Fatalf("在跑会话不应消耗唤醒预算，实得 %d", chain)
	}
	close(release)
	if turns.Load() != 0 {
		t.Fatal("在跑会话不应额外拉起唤醒回合")
	}
}
