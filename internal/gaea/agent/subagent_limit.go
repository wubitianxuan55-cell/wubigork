package agent

import (
	"context"
	"errors"
	"sync"
)

// 子代理有限并行闸（2026-10，对标 ZCode Agent 编排的 max_concurrency 与
// dsh 子代理控制面的「发射后可预期」纪律）。
//
// Why：v4.63 起同一回合的 N 路 task/run_skill 调用已经真并行（每调用唯一
// spawn 键进同一并行批），但共享的是通用工具批的 runParallel 帽（8）——
// 对本地推理引擎（单卡 MoE）8 路子代理同时打 LLM 是互掐式降速，对远程
// API 则是并发配额/速率限制的直撞。子代理是唯一「每实例都是一整条 LLM
// 循环」的工具，值得一个专属、可配、跨 task/run_skill 共享的并发上限：
// 超出上限的派生调用排队等槽（FIFO 近似），已派发的槽释放后自动起跑。
//
// How：包级信号量（先例：subRunners sync.Map 同为包级）。容量在 boot 装配
// 点由 [agent] subagent_max_parallel 设定（SetSubagentMaxParallel，负/零=
// 默认 3，上限 8 与 runParallel 对齐）；运行期热调不保证（通道容量只在
// Set 时重建，回合空闲期调用是约定）。
//
// 嵌套语义（2026-11-18 嵌套委派开放后）：顶层派发（主会话发起）排队等槽；
// 嵌套派发（子代理发起，caller 级别≥1）走 tryAcquireSubSlot 不排队、槽满
// 快拒——调用方子代理自己持槽等本派生返回，排队会让「等后代的父级」等
// 自己占的名额形成全树自锁（dsh ActivationPool 满拒绝同源洞察）。两路合取
// 保证：任一时刻「持槽者都在推进或可取消」，无 hold-and-wait 死锁面。
//
// 排队诚实化：拿不到槽的顶层调用在等待前先把 transcript 侧车写成 queued
// （MarkQueued），起跑时 MarkRunning 覆写，前端分工面板/会话 tab 如实
// 显示「排队中」而不是假的 running 转轮。

const (
	// DefaultSubagentMaxParallel 是未配置时的并发上限：3 路子代理对单卡
	// 本地引擎与主流远程 API 都是安全档，多路收益靠批内工具并行补足。
	DefaultSubagentMaxParallel = 3
	// MaxSubagentMaxParallel 与 executeBatch 的通用 runParallel 帽对齐：
	// 子代理闸再大也不该超过批次并行度。
	MaxSubagentMaxParallel = 8
)

var (
	subSlotsMu     sync.Mutex
	subSlots       chan struct{}
	subSlotsCap    = DefaultSubagentMaxParallel
	subSlotsActive int // 当前持有槽位的子代理数（观测用）
)

// SetSubagentMaxParallel 设定子代理并发上限。n≤0 取默认值；超过上限取帽。
// 在 boot 装配点（回合空闲期）调用；通道按目标容量重建，运行中热调不保证
// 在跑调用的即时约束。
func SetSubagentMaxParallel(n int) {
	if n <= 0 {
		n = DefaultSubagentMaxParallel
	}
	if n > MaxSubagentMaxParallel {
		n = MaxSubagentMaxParallel
	}
	subSlotsMu.Lock()
	defer subSlotsMu.Unlock()
	if n == subSlotsCap && subSlots != nil {
		return
	}
	subSlotsCap = n
	subSlots = make(chan struct{}, n)
}

// SubagentMaxParallel 返回当前并发上限（含默认值解析，恒 ≥1）。
func SubagentMaxParallel() int {
	subSlotsMu.Lock()
	defer subSlotsMu.Unlock()
	return subSlotsCap
}

// acquireSubSlot 取一个子代理槽位；ctx 取消时如实返回 ctx.Err()（等待中的
// 派生调用随回合取消而放弃，不占槽）。
func acquireSubSlot(ctx context.Context) error {
	subSlotsMu.Lock()
	ch := subSlots
	if ch == nil {
		ch = make(chan struct{}, DefaultSubagentMaxParallel)
		subSlots = ch
	}
	subSlotsMu.Unlock()
	select {
	case ch <- struct{}{}:
		subSlotsMu.Lock()
		subSlotsActive++
		subSlotsMu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// releaseSubSlot 归还槽位。与 acquireSubSlot 严格配对（defer）。
func releaseSubSlot() {
	subSlotsMu.Lock()
	defer subSlotsMu.Unlock()
	if subSlotsActive > 0 {
		subSlotsActive--
	}
	if subSlots != nil && len(subSlots) > 0 {
		<-subSlots
	}
}

// errSubSlotsExhausted 是 tryAcquireSubSlot 的槽满信号（不排队，调用方负责
// 换成带上下文的诚实文案）。
var errSubSlotsExhausted = errors.New("sub-agent slots exhausted")

// tryAcquireSubSlot 是 acquireSubSlot 的不排队形态（2026-11-18 嵌套派发防
// 自锁）：槽满立即返回 errSubSlotsExhausted，不阻塞。Why：嵌套派发的调用方
// 子代理自己正持槽等待本派生返回——若孙代排队等槽，而槽全被「等后代的父级」
// 占着，全树互相等死（dsh ActivationPool「满拒绝不排队」同源洞察，TL;DR#7）。
// gaea 只对嵌套层（caller 级别≥1）启用快拒；顶层派发维持排队语义不变。
func tryAcquireSubSlot() error {
	subSlotsMu.Lock()
	ch := subSlots
	if ch == nil {
		ch = make(chan struct{}, DefaultSubagentMaxParallel)
		subSlots = ch
	}
	subSlotsMu.Unlock()
	select {
	case ch <- struct{}{}:
		subSlotsMu.Lock()
		subSlotsActive++
		subSlotsMu.Unlock()
		return nil
	default:
		return errSubSlotsExhausted
	}
}

// SubagentSlotsBusy 返回当前在跑的子代理数（诊断/测试观测用）。
func SubagentSlotsBusy() int {
	subSlotsMu.Lock()
	defer subSlotsMu.Unlock()
	return subSlotsActive
}
