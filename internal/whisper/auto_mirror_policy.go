// Package whisper — auto_mirror_policy.go
// 100% 对齐 ackem memory/autoMirrorPolicy.ts
// 镜中/矛盾自动检测触发策略

package whisper

// MirrorCheckEarlyMinTurns 早触发最低轮次
const MirrorCheckEarlyMinTurns = 15

// EvaluatePeriodicMemoryAudit 评估是否应触发周期性记忆审计
func EvaluatePeriodicMemoryAudit(turnsSinceLastCheck int, selfFactAddedThisTurn bool) bool {
	if turnsSinceLastCheck >= MirrorCheckIntervalTurns {
		return true
	}
	if selfFactAddedThisTurn && turnsSinceLastCheck >= MirrorCheckEarlyMinTurns {
		return true
	}
	return false
}
