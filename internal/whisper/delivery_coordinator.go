// Package whisper — delivery_coordinator.go
// 100% 对齐 ackem desktop-agent/deliveryCoordinator.ts
// 消息分发协调：流式状态追踪 + 排队投递

package whisper

import "sync"

// DeliveryCoordinator 消息分发协调器
type DeliveryCoordinator struct {
	mu        sync.RWMutex
	streaming map[string]bool // sessionID → 是否正在流式输出
}

// NewDeliveryCoordinator 创建分发协调器
func NewDeliveryCoordinator() *DeliveryCoordinator {
	return &DeliveryCoordinator{streaming: make(map[string]bool)}
}

// MarkChatStreamStart 标记聊天流开始
func (dc *DeliveryCoordinator) MarkChatStreamStart(sessionID string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.streaming[sessionID] = true
}

// MarkChatStreamEnd 标记聊天流结束
func (dc *DeliveryCoordinator) MarkChatStreamEnd(sessionID string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	delete(dc.streaming, sessionID)
}

// IsChatStreaming 检查是否正在流式输出
func (dc *DeliveryCoordinator) IsChatStreaming(sessionID string) bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.streaming[sessionID]
}

// TaskDeliveryPayload 任务投递负载
type TaskDeliveryPayload struct {
	SessionID   string `json:"sessionId"`
	TaskPlanID  string `json:"taskPlanId,omitempty"`
	GoalSummary string `json:"goalSummary"`
	AllPassed   bool   `json:"allPassed"`
	Text        string `json:"text"`
	Queued      bool   `json:"queued"`
}

// DeliverTaskResult 投递任务结果（排队感知）
func (dc *DeliveryCoordinator) DeliverTaskResult(payload TaskDeliveryPayload) TaskDeliveryPayload {
	dc.mu.RLock()
	queued := dc.streaming[payload.SessionID]
	dc.mu.RUnlock()

	payload.Queued = queued
	return payload
}
