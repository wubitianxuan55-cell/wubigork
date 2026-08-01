// Package whisper — user_presence.go
// 100% 对齐 ackem context/userPresence.ts
// 用户存在检测：参与度级别 + 最近消息片段

package whisper

import (
	"strings"
	"time"
)

const (
	activeNowMin      = 20  // ≤20分钟 = active_now
	recentlyActiveMin = 120 // ≤120分钟 = recently_active
	idleMax           = 480 // ≤480分钟 = idle, >480 = likely_away
)

// ResolveUserEngagement 根据最后活跃时间推断参与度
func ResolveUserEngagement(lastActiveAt time.Time, now time.Time) UserRuntimeContext {
	minutes := int(now.Sub(lastActiveAt).Minutes())
	if minutes < 0 {
		minutes = 9999
	}

	var engagement UserEngagementLevel
	switch {
	case minutes <= activeNowMin:
		engagement = EngagementActiveNow
	case minutes <= recentlyActiveMin:
		engagement = EngagementRecentlyActive
	case minutes <= idleMax:
		engagement = EngagementIdle
	default:
		engagement = EngagementLikelyAway
	}

	return UserRuntimeContext{
		LastActiveAt:         lastActiveAt.Format(time.RFC3339),
		MinutesSinceLastChat: minutes,
		Engagement:           engagement,
	}
}

// LoadRecentUserSnippets 加载最近用户消息片段
func LoadRecentUserSnippets(recentExchanges []string, limit, maxChars int) []string {
	if limit <= 0 {
		limit = 5
	}
	if maxChars <= 0 {
		maxChars = 160
	}

	var result []string
	start := len(recentExchanges) - limit
	if start < 0 {
		start = 0
	}
	for _, s := range recentExchanges[start:] {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		runes := []rune(s)
		if len(runes) > maxChars {
			s = string(runes[:maxChars])
		}
		result = append(result, s)
	}
	return result
}

// ResolveUserRuntimeContext 统一构建用户运行时上下文
func ResolveUserRuntimeContext(lastActiveAt time.Time, recentExchanges []string, now time.Time) UserRuntimeContext {
	ctx := ResolveUserEngagement(lastActiveAt, now)
	ctx.RecentUserSnippets = LoadRecentUserSnippets(recentExchanges, 5, 160)
	return ctx
}
