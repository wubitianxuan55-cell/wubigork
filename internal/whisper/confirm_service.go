// Package whisper — confirm_service.go
// 100% 对齐 ackem desktop-agent/confirm/confirmService.ts
// 确认服务：请求用户确认操作，支持超时

package whisper

import (
	"sync"
	"time"
)

// ConfirmDecision 确认决策
type ConfirmDecision string

const (
	ConfirmAllowed ConfirmDecision = "allowed"
	ConfirmDenied  ConfirmDecision = "denied"
	ConfirmTimeout ConfirmDecision = "timeout"
)

// ConfirmRequest 确认请求
type ConfirmRequest struct {
	RequestID      string `json:"requestId"`
	Action         string `json:"action"`
	Path           string `json:"path"`
	HardBlockReason string `json:"hardBlockReason,omitempty"`
}

// ConfirmService 确认服务
type ConfirmService struct {
	mu      sync.Mutex
	pending map[string]*pendingConfirm
}

type pendingConfirm struct {
	resolve chan ConfirmDecision
	timer   *time.Timer
}

// NewConfirmService 创建确认服务
func NewConfirmService() *ConfirmService {
	return &ConfirmService{pending: make(map[string]*pendingConfirm)}
}

// RequestConfirm 请求用户确认（默认120秒超时）
func (cs *ConfirmService) RequestConfirm(input ConfirmRequest, timeoutMs int) ConfirmDecision {
	if input.HardBlockReason != "" {
		return ConfirmDenied
	}
	if timeoutMs <= 0 {
		timeoutMs = 120000
	}
	if input.RequestID == "" {
		input.RequestID = genHexID()
	}

	result := make(chan ConfirmDecision, 1)
	timer := time.AfterFunc(time.Duration(timeoutMs)*time.Millisecond, func() {
		cs.mu.Lock()
		delete(cs.pending, input.RequestID)
		cs.mu.Unlock()
		result <- ConfirmTimeout
	})

	cs.mu.Lock()
	cs.pending[input.RequestID] = &pendingConfirm{resolve: result, timer: timer}
	cs.mu.Unlock()

	// 广播到前端（由 Wails handler 层桥接）
	// broadcastFn("desktop-agent:confirm:request", input)

	select {
	case decision := <-result:
		return decision
	}
}

// ResolveConfirm 解析确认请求
func (cs *ConfirmService) ResolveConfirm(requestID string, decision ConfirmDecision) bool {
	cs.mu.Lock()
	req, ok := cs.pending[requestID]
	if ok {
		delete(cs.pending, requestID)
	}
	cs.mu.Unlock()

	if ok && req != nil {
		req.timer.Stop()
		req.resolve <- decision
		return true
	}
	return false
}

// CancelAllConfirms 取消所有待确认请求
func (cs *ConfirmService) CancelAllConfirms(reason ConfirmDecision) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for id, req := range cs.pending {
		req.timer.Stop()
		delete(cs.pending, id)
		req.resolve <- reason
	}
}
