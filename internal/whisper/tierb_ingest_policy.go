// Package whisper — tierb_ingest_policy.go
// 100% 对齐 ackem memory/tierBIngestPolicy.ts
// 分层摄取策略：判断是否应跳过 Tier B ingest

package whisper

import "strings"

// ResolveTierBIngestSkip 判断是否应跳过 Tier B 摄入
func ResolveTierBIngestSkip(skipIngest bool, userMsg string, originFatherRef string) bool {
	if !skipIngest {
		return false
	}
	// 显式 remember 指令不跳过
	if detectMemoryIntent(userMsg) == "remember" {
		return false
	}
	// 用户指称自己的父亲不跳过
	if originFatherRef == "user_family" {
		return false
	}
	return true
}

// detectMemoryIntent 检测记忆意图（简化版）
func detectMemoryIntent(msg string) string {
	memoryKeywords := []string{"记住", "记下", "别忘了", "提醒我", "remember", "记一下"}
	for _, kw := range memoryKeywords {
		if strings.Contains(msg, kw) {
			return "remember"
		}
	}
	return "none"
}
