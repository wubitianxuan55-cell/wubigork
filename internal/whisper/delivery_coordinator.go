// Package whisper — delivery_coordinator.go
// 100% 对齐 ackem desktop-agent/deliveryCoordinator.ts
// 交付协调器：chat 流式互斥、任务结果排队/推送
package whisper

import (
	"sync"
)

// ─── 交付协调器 ────────────────────────────────────────────────

// DeliveryCoordinator 交付协调器
// 确保 chat 流式输出与 agent 任务结果互斥，避免消息穿插
type DeliveryCoordinator struct {
	mu sync.Mutex

	// 流式输出锁：同一时间仅一个流式输出活跃
	streamingActive bool
	streamingTurnID string

	// 任务结果队列
	pendingResults []AgentTaskResult

	// 回调
	onDeliver func(result AgentTaskResult)
}

// AgentTaskResult agent 任务结果
type AgentTaskResult struct {
	TaskID     string `json:"taskId"`
	TurnID     string `json:"turnId"`
	Success    bool   `json:"success"`
	Summary    string `json:"summary"`
	Content    string `json:"content"`
	MemoryHint string `json:"memoryHint,omitempty"`
}

// NewDeliveryCoordinator 创建交付协调器
func NewDeliveryCoordinator() *DeliveryCoordinator {
	return &DeliveryCoordinator{
		pendingResults: make([]AgentTaskResult, 0),
	}
}

// SetOnDeliver 设置交付回调
func (dc *DeliveryCoordinator) SetOnDeliver(fn func(result AgentTaskResult)) {
	dc.onDeliver = fn
}

// ─── 流式互斥 ──────────────────────────────────────────────────

// AcquireStream 获取流式输出锁
// 返回 true 表示获取成功，false 表示已有其他流式输出活跃
func (dc *DeliveryCoordinator) AcquireStream(turnID string) bool {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if dc.streamingActive {
		return false
	}
	dc.streamingActive = true
	dc.streamingTurnID = turnID
	return true
}

// ReleaseStream 释放流式输出锁
func (dc *DeliveryCoordinator) ReleaseStream(turnID string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if dc.streamingTurnID == turnID {
		dc.streamingActive = false
		dc.streamingTurnID = ""
	}
	// 释放后检查待交付队列
	dc.flushPending()
}

// IsStreaming 当前是否有流式输出
func (dc *DeliveryCoordinator) IsStreaming() bool {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return dc.streamingActive
}

// ─── 任务结果管理 ──────────────────────────────────────────────

// EnqueueResult 入队任务结果
func (dc *DeliveryCoordinator) EnqueueResult(result AgentTaskResult) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.streamingActive {
		// 流式输出中，暂存
		dc.pendingResults = append(dc.pendingResults, result)
	} else {
		// 立即交付
		dc.deliver(result)
	}
}

// PendingCount 待交付结果数
func (dc *DeliveryCoordinator) PendingCount() int {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return len(dc.pendingResults)
}

// FlushAll 立即交付所有待处理结果
func (dc *DeliveryCoordinator) FlushAll() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.flushPending()
}

// ─── 内部方法 ──────────────────────────────────────────────────

func (dc *DeliveryCoordinator) flushPending() {
	for _, r := range dc.pendingResults {
		dc.deliver(r)
	}
	dc.pendingResults = dc.pendingResults[:0]
}

func (dc *DeliveryCoordinator) deliver(result AgentTaskResult) {
	if dc.onDeliver != nil {
		dc.onDeliver(result)
	}
}

// ─── 结果格式化 ────────────────────────────────────────────────

// FormatTaskResultForChat 将任务结果格式化为聊天消息
func FormatTaskResultForChat(result AgentTaskResult) string {
	if !result.Success {
		return "❌ " + result.Summary
	}
	if result.Content != "" {
		return "✅ " + result.Summary + "\n\n" + result.Content
	}
	return "✅ " + result.Summary
}

// FormatTaskResultBrief 简要格式化
func FormatTaskResultBrief(result AgentTaskResult) string {
	if result.Success {
		return "✅ " + result.Summary
	}
	return "❌ " + result.Summary
}
