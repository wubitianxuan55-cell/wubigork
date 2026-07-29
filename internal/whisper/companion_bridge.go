// Package whisper — companion_bridge.go
// 100% 对齐 ackem context/companionBridge.ts
// 陪伴在场桥接：解耦 context 与陪伴实例

package whisper

import "time"

// CompanionPresenceSnapshot 陪伴在场快照
type CompanionPresenceSnapshot struct {
	Mode              CompanionPresenceMode `json:"mode"`
	LastInteractionMs int64                 `json:"lastInteractionMs"`
	IdleDurationMs    int64                 `json:"idleDurationMs"`
}

// DefaultCompanionPresence 默认陪伴在场（总是 active）
func DefaultCompanionPresence() CompanionPresenceSnapshot {
	now := time.Now().UnixMilli()
	return CompanionPresenceSnapshot{
		Mode:              CompanionActive,
		LastInteractionMs: now,
		IdleDurationMs:    0,
	}
}

// ReadCompanionPresence 读取陪伴在场上下文
func ReadCompanionPresence(snap *CompanionPresenceSnapshot) CompanionRuntimeContext {
	if snap == nil {
		now := time.Now().UnixMilli()
		return CompanionRuntimeContext{
			Mode:              CompanionActive,
			LastInteractionMs: now,
			IdleDurationMs:    0,
		}
	}
	return CompanionRuntimeContext{
		Mode:              snap.Mode,
		LastInteractionMs: snap.LastInteractionMs,
		IdleDurationMs:    snap.IdleDurationMs,
	}
}
