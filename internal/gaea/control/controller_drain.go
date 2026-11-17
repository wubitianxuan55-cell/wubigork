// controller_drain.go 是回合准入与优雅关闭（codex 蒸馏 B1，第四批 P1）。
//
// 机制同构 codex turn_admission 的「关闸与准入同一把锁 + 两计数同零才退出」，
// 折算到 gaea 的单回合 + FIFO 队列形态：
//   - 准入 = 消息被接受（开跑或入队）那一刻，admitted+1；tryStartTurn 的
//     系统回合同理。取消/排队满等拒绝路径不计数。
//   - 收敛 = 「在跑回合 + 已准入消息」两计数同零。gaea 的单 turnLoop 排空
//     队列并落完最后一条 TurnDone 即全部已准入消息完成——该点归零并唤醒
//     等待者（等价 codex 的 permit Drop 广播）。
//   - 关闸后新 Send 拒收并回结构化通知（不静默丢）；已在队列、尚未开跑的
//     消息同样逐条通知后作废（等价 codex 串行化队列的二次准入失败）。
//   - 超时强退：cancel 在飞回合，TurnDone{Err} 由既有路径自然落账——本层
//     不伪造事件。
package control

import (
	"errors"
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
)

// ErrSessionDraining 是关闸后的结构化拒收错误（对齐 codex 的
// serverShuttingDown 可判定拒绝；调用方可 errors.Is 分类，不解析文本）。
var ErrSessionDraining = errors.New("session draining: message not accepted (session is shutting down)")

// DrainTimeoutDefault 是 BeginDrain 的默认超时（上游连接级收尾是 30s；gaea
// 是进程级退出路径，用户点了关闭不应久等——3s 内不收敛就强退）。
const DrainTimeoutDefault = 3 * time.Second

// Draining 报告关闸态（关闸后新回合/新消息不再被接受）。
func (c *Controller) Draining() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.draining
}

// BeginDrain 关闸并等待「在跑回合 + 已准入消息」收敛（B1 两计数同零）。
// 行为：
//   - 首次关闸：置 draining，逐条通知并作废仍在排队、尚未开跑的消息；
//   - timeout 内收敛 → nil；超时 → 强制 cancel 在飞回合并返回超时错误
//     （回合的 TurnDone{Err} 由既有路径落账，不伪造）；
//   - 幂等：重复调用继续等待同一收敛点；
//   - 已经空闲（无在跑、无已准入）→ 立即 nil。
//
// 关闸与准入取同一把 c.mu：sendWithRaw 的「检查 draining—登记 admitted」
// 与本处的「置 draining—快照计数」互斥，不存在关闸漏掉在途准入的窗口。
func (c *Controller) BeginDrain(timeout time.Duration) error {
	if timeout <= 0 {
		timeout = DrainTimeoutDefault
	}
	c.mu.Lock()
	if !c.draining {
		c.draining = true
		// 队列里尚未开跑的消息：已准入（admitted 已计），关闸语义下不再
		// 执行——逐条回结构化通知后作废（不静默丢）。
		dropped := c.pendingSends
		c.pendingSends = nil
		c.admitted -= len(dropped)
		if c.admitted < 0 {
			c.admitted = 0 // 防御：老路径未计入的排队项不产生负计数
		}
		if len(dropped) > 0 {
			c.mu.Unlock()
			for _, req := range dropped {
				c.sink.Emit(noticeDraining(req.input))
			}
			c.mu.Lock()
		}
	}
	if !c.running && c.admitted == 0 {
		c.mu.Unlock()
		return nil
	}
	wait := c.drainWait
	if wait == nil {
		wait = make(chan struct{})
		c.drainWait = wait
	}
	cancel := c.cancel
	c.mu.Unlock()

	select {
	case <-wait:
		return nil
	case <-time.After(timeout):
		// 强退：取消在飞回合。TurnDone{Err} 由 turnLoop 的既有路径发射；
		// admitted 归零由 turnLoop 收尾完成（即使被取消也会走到收尾分支）。
		if cancel != nil {
			cancel()
		}
		return fmt.Errorf("drain timeout after %s: running turn force-cancelled", timeout)
	}
}

// settleDrainLocked 在 turnLoop 收尾（队列空、running 置假）时调用：本批
// 已准入消息全部完成，计数归零并唤醒等待者。调用方必须已持 c.mu。
func (c *Controller) settleDrainLocked() {
	c.admitted = 0
	c.turnStep = stepSnapshot{} // 回合收尾：准入快照随之失效
	if c.drainWait != nil {
		close(c.drainWait)
		c.drainWait = nil
	}
}

// admitLocked 是准入的「检查—登记」原语：关闸后返回 false（调用方回结构
// 化拒收），否则 admitted+1。调用方必须已持 c.mu。
func (c *Controller) admitLocked() bool {
	if c.draining {
		return false
	}
	c.admitted++
	return true
}

// noticeDraining 构造一条关闸拒收通知（含消息预览，让用户知道哪条没发出去）。
func noticeDraining(input string) event.Event {
	preview := input
	if r := []rune(preview); len(r) > 80 {
		preview = string(r[:79]) + "…"
	}
	return event.Event{
		Kind:  event.Notice,
		Level: event.LevelWarn,
		Text:  "会话正在关闭，这条消息未被接受（不会在下次启动时重发）：" + preview,
	}
}
