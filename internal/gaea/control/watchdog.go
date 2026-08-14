// watchdog.go — 运行态看门狗（T6-2.4 质量收敛）。
//
// 对齐声明：v2.13.0 changelog 曾声称实现了「运行态看门狗」，但该声称此前从未
// 落地——internal/ 全库没有 watchdog 实现，超时保护只靠各 gate 的重试上限，
// 前端 store.ts 的 30s 定时器是仅有的「看门狗」（前端侧）。本文件才是真正的
// 实现：进程内运行态看门狗 = 本实现，v2.13.0 声称此前未落地。
//
// 职责：对运行中的单个回合做两层保护——墙钟硬上限（默认 10 分钟）与推进停滞
// 检测（默认 30 秒无新事件）。触发时取消该回合的上下文，与用户 Cancel 走同
// 一条中断链路（controller 的 cancel ctx），回合收尾照常 Emit TurnDone(Err)。
//
// 不误杀长任务：停滞检测只看「推进」信号（新 delta / 新工具调用 / 新事件），
// 不是单纯无输出；工具执行期间（ToolDispatch → ToolResult 之间）与等待用户
// 输入（审批 / 提问）期间豁免停滞计时，因此大文件转换这类长时间无事件输出的
// 工具调用不会被误杀。
package control

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
)

// WatchdogConfig 配置单回合运行态看门狗（T6-2.4）的阈值。
// 语义：字段 == 0 使用 DefaultWatchdog 的对应默认值；字段 < 0 禁用该维度；
// 字段 > 0 使用自定义阈值。两维度均禁用（< 0）时看门狗整体不装配。
type WatchdogConfig struct {
	// WallClock 是单回合墙钟硬上限：回合开始后超过该时长仍未结束即强制终止。
	// 默认 10 分钟；< 0 禁用墙钟限制。
	WallClock time.Duration
	// Stall 是推进停滞阈值：超过该时长没有任何推进事件（新 delta / 新工具
	// 调用 / 新事件）即视为停滞并强制终止。工具执行期间与等待用户输入期间
	// 豁免。默认 30 秒；< 0 禁用停滞检测。
	Stall time.Duration
}

// DefaultWatchdog 是生产默认阈值：墙钟 10 分钟、停滞 30 秒。
var DefaultWatchdog = WatchdogConfig{
	WallClock: 10 * time.Minute,
	Stall:     30 * time.Second,
}

// watchdogCheckInterval 是看门狗轮询间隔；远小于默认停滞阈值，开销可忽略。
const watchdogCheckInterval = 250 * time.Millisecond

// withDefaults 用 DefaultWatchdog 补齐零值字段（< 0 的维度保持禁用）。
func (c WatchdogConfig) withDefaults() WatchdogConfig {
	if c.WallClock == 0 {
		c.WallClock = DefaultWatchdog.WallClock
	}
	if c.Stall == 0 {
		c.Stall = DefaultWatchdog.Stall
	}
	return c
}

// enabled 报告该配置是否启用任何限制。
func (c WatchdogConfig) enabled() bool {
	return c.WallClock > 0 || c.Stall > 0
}

// watchdog 是单个回合的运行态看门狗。控制器在 turnLoop 每个回合开始前调用
// begin、回合结束后调用 end；sink 包装器在每次事件到达时调用 observe 维护
// 推进状态。触发时调用该回合的 cancel（与用户 Cancel 同一条中断链路）取消
// 回合上下文；收尾时由 wrapErr 把返回错误包装为可识别的看门狗超时，
// turnLoop 照常 Emit TurnDone(Err)。
type watchdog struct {
	cfg WatchdogConfig

	// 回合内状态（begin 重置）。stopCh/doneCh/ctx/cancel 是回合级的，begin
	// 时以局部变量传入 goroutine（run），绝不跨回合复用——end/下一个 begin
	// 会重建这些通道，若 goroutine 读字段就会误选到下一回合的通道。
	start    atomic.Int64 // 回合开始 unixnano
	lastProg atomic.Int64 // 最近一次推进事件 unixnano
	toolOpen atomic.Bool  // 有工具调用在途（ToolDispatch 未见对应 ToolResult）
	userWait atomic.Bool  // 等待用户输入（审批/提问）
	fired    atomic.Bool
	reason   atomic.Value // string：触发原因（仅 fired 时有意义）

	// lifeMu 保护 stopCh/doneCh 字段（begin/end 串行调用，仅防御性并发）。
	lifeMu sync.Mutex
	stopCh chan struct{}
	doneCh chan struct{}
}

func newWatchdog(cfg WatchdogConfig) *watchdog {
	return &watchdog{cfg: cfg.withDefaults()}
}

// begin 启动该回合的看门狗：重置推进状态并启动检查 goroutine。
// cancel 是该回合的上下文取消函数（与用户 Cancel 同一链路）。
func (w *watchdog) begin(ctx context.Context, cancel context.CancelFunc) {
	now := time.Now().UnixNano()
	w.start.Store(now)
	w.lastProg.Store(now)
	w.toolOpen.Store(false)
	w.userWait.Store(false)
	w.fired.Store(false)
	w.reason.Store("")
	w.lifeMu.Lock()
	w.stopCh = make(chan struct{})
	w.doneCh = make(chan struct{})
	stopCh := w.stopCh
	doneCh := w.doneCh
	w.lifeMu.Unlock()
	go w.run(ctx, cancel, stopCh, doneCh)
}

// end 停止检查 goroutine 并等待其退出（同步），保证不会跨回合误触发。
func (w *watchdog) end() {
	w.lifeMu.Lock()
	stopCh := w.stopCh
	doneCh := w.doneCh
	w.stopCh = nil
	w.doneCh = nil
	w.lifeMu.Unlock()
	if stopCh == nil {
		return
	}
	close(stopCh)
	<-doneCh
}

func (w *watchdog) run(ctx context.Context, cancel context.CancelFunc, stopCh, doneCh chan struct{}) {
	defer close(doneCh)
	ticker := time.NewTicker(watchdogCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ctx.Done():
			return // 回合以其他方式结束（用户取消/自然收尾）
		case <-ticker.C:
			w.check(ctx, cancel)
		}
	}
}

// check 按配置阈值判断是否触发。
func (w *watchdog) check(ctx context.Context, cancel context.CancelFunc) {
	if w.fired.Load() {
		return
	}
	if ctx.Err() != nil {
		return // 回合已结束（取消），不再触发
	}
	now := time.Now()
	if w.cfg.WallClock > 0 && now.Sub(time.Unix(0, w.start.Load())) > w.cfg.WallClock {
		w.fire(cancel, fmt.Sprintf("wall-clock timeout (%s)", w.cfg.WallClock))
		return
	}
	if w.cfg.Stall > 0 && !w.toolOpen.Load() && !w.userWait.Load() &&
		now.Sub(time.Unix(0, w.lastProg.Load())) > w.cfg.Stall {
		w.fire(cancel, fmt.Sprintf("stalled: no progress for %s", w.cfg.Stall))
	}
}

// fire 触发一次看门狗：取消该回合上下文（与用户 Cancel 同一中断链路）。
func (w *watchdog) fire(cancel context.CancelFunc, reason string) {
	if !w.fired.CompareAndSwap(false, true) {
		return
	}
	w.reason.Store(reason)
	slog.Error("controller: watchdog fired, cancelling turn", "reason", reason)
	if cancel != nil {
		cancel()
	}
}

// firedReason 返回触发原因（未触发时 ok=false）。
func (w *watchdog) firedReason() (reason string, ok bool) {
	if !w.fired.Load() {
		return "", false
	}
	r, _ := w.reason.Load().(string)
	return r, true
}

// wrapErr 在回合结束后把看门狗触发标记附加到返回错误上，使 TurnDone(Err)
// 携带可识别的超时原因；未触发时原样返回。
func (w *watchdog) wrapErr(err error) error {
	reason, fired := w.firedReason()
	if !fired {
		return err
	}
	if err == nil {
		err = context.Canceled
	}
	return fmt.Errorf("turn terminated by watchdog: %s: %w", reason, err)
}

// observe 由 sink 包装器在每次事件到达时调用，维护推进状态。
func (w *watchdog) observe(e event.Event) {
	switch e.Kind {
	case event.TurnDone:
		return // 收尾事件，不算推进
	case event.ToolDispatch:
		w.userWait.Store(false)
		w.toolOpen.Store(true)
		w.touch()
	case event.ToolResult:
		w.toolOpen.Store(false)
		w.touch()
	case event.ApprovalRequest, event.AskRequest:
		w.userWait.Store(true)
		w.touch()
	default:
		// 新 delta / 新事件：推进。也清除用户等待豁免——
		// 审批/提问被答复后回合恢复，恢复后的首个事件即解除豁免。
		w.userWait.Store(false)
		w.touch()
	}
}

func (w *watchdog) touch() {
	w.lastProg.Store(time.Now().UnixNano())
}

// watchdogSink 转发事件到真实 sink，并同步看门狗推进状态（watchdog.observe）。
// 生产装配中控制器把执行器（即 runner）的 sink 一并重接线到该包装器
// （agent.SetSink），因此模型流式输出、工具调用等全部推进事件都经过这里。
type watchdogSink struct {
	inner event.Sink
	wd    *watchdog
}

func (s watchdogSink) Emit(e event.Event) {
	s.wd.observe(e)
	s.inner.Emit(e)
}
