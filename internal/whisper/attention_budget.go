// Package whisper — attention_budget.go
// 100% 对齐 ackem extensions/policy/attentionBudget.ts
// 注意力预算管理：主动消息频率控制、全局免打扰、冷却管理

package whisper

import (
	"sync"
	"time"
)

const hourMs = int64(time.Hour / time.Millisecond)

// ─── AttentionBudget ─────────────────────────────────────────

// AttentionBudgetState 注意力预算状态
type AttentionBudgetState struct {
	ProactiveMessagesPerHour int              `json:"proactiveMessagesPerHour"`
	LastProactiveAt          []int64          `json:"lastProactiveAt"`            // unix ms
	CategoryCooldown         map[string]int64 `json:"categoryCooldown,omitempty"` // category → cooldown until ms
	GlobalDnd                *GlobalDnd       `json:"globalDnd,omitempty"`
}

// GlobalDnd 全局免打扰
type GlobalDnd struct {
	Until  int64  `json:"until"` // unix ms
	Reason string `json:"reason"`
}

// DefaultAttentionBudget 默认预算：每小时 3 条
func DefaultAttentionBudget() *AttentionBudgetState {
	return &AttentionBudgetState{
		ProactiveMessagesPerHour: 3,
	}
}

// ─── AttentionManager ────────────────────────────────────────

// AttentionManager 注意力管理器
type AttentionManager struct {
	mu    sync.Mutex
	state *AttentionBudgetState
}

// NewAttentionManager 创建注意力管理器
func NewAttentionManager() *AttentionManager {
	return &AttentionManager{
		state: DefaultAttentionBudget(),
	}
}

// State 返回当前状态副本
func (am *AttentionManager) State() AttentionBudgetState {
	am.mu.Lock()
	defer am.mu.Unlock()
	return *am.state
}

// IsBudgetExceeded 检查是否超出预算
func (am *AttentionManager) IsBudgetExceeded(now time.Time) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	nowMs := now.UnixMilli()
	cutoff := nowMs - hourMs

	recent := 0
	for _, t := range am.state.LastProactiveAt {
		if t >= cutoff {
			recent++
		}
	}
	return recent >= am.state.ProactiveMessagesPerHour
}

// IsGlobalDndActive 检查全局免打扰是否生效
func (am *AttentionManager) IsGlobalDndActive(now time.Time) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.state.GlobalDnd == nil {
		return false
	}
	return am.state.GlobalDnd.Until > now.UnixMilli()
}

// RecordProactive 记录一条主动消息
func (am *AttentionManager) RecordProactive(now time.Time) {
	am.mu.Lock()
	defer am.mu.Unlock()

	nowMs := now.UnixMilli()
	cutoff := nowMs - hourMs

	// 裁剪+追加
	var kept []int64
	for _, t := range am.state.LastProactiveAt {
		if t >= cutoff {
			kept = append(kept, t)
		}
	}
	kept = append(kept, nowMs)
	am.state.LastProactiveAt = kept
}

// SetGlobalDnd 设置全局免打扰
func (am *AttentionManager) SetGlobalDnd(until time.Time, reason string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.state.GlobalDnd = &GlobalDnd{Until: until.UnixMilli(), Reason: reason}
}

// ClearGlobalDnd 清除全局免打扰
func (am *AttentionManager) ClearGlobalDnd(reason string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	if am.state.GlobalDnd == nil {
		return
	}
	if reason != "" && am.state.GlobalDnd.Reason != reason {
		return
	}
	am.state.GlobalDnd = nil
}

// SetProactiveLimit 设置每小时主动消息上限
func (am *AttentionManager) SetProactiveLimit(limit int) {
	am.mu.Lock()
	defer am.mu.Unlock()
	if limit > 0 {
		am.state.ProactiveMessagesPerHour = limit
	}
}
