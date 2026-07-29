// Package whisper — health_policy.go
// 对齐 ackem extensions/policy/evaluate.ts 健康检测部分
// 久坐/喝水/深夜检测 — 供主动消息使用

package whisper

import "strings"

// ─── 健康检测结果 ────────────────────────────────────────────

// HealthCheckResult 健康检测结果
type HealthCheckResult struct {
	ShouldRemind bool
	Reason       string
	Hint         string
}

// EvaluateHealthPolicy 评估是否需要健康提醒
func EvaluateHealthPolicy(hour int, lastUserActivityMinutes int, aff float64) *HealthCheckResult {
	// 深夜提醒
	if hour >= 23 || hour < 5 {
		if lastUserActivityMinutes > 0 && lastUserActivityMinutes < 30 {
			return &HealthCheckResult{
				ShouldRemind: true,
				Reason:       "late_night",
				Hint:         "深夜了，ta还在。温柔地提醒一句注意休息，但不要唠叨。",
			}
		}
	}

	// 久坐提醒（假设每60分钟检查一次）
	if lastUserActivityMinutes > 60 && aff > 30 {
		return &HealthCheckResult{
			ShouldRemind: true,
			Reason:       "sedentary",
			Hint:         "ta可能坐了挺久了。可以自然地提一句'起来活动一下'，但不要像健康App。",
		}
	}

	return nil
}

// IsHealthRelated 检查消息是否涉及健康话题
func IsHealthRelated(msg string) bool {
	healthWords := []string{"久坐", "喝水", "健康", "休息", "睡眠", "熬夜", "累了", "疲劳"}
	for _, w := range healthWords {
		if strings.Contains(msg, w) {
			return true
		}
	}
	return false
}
