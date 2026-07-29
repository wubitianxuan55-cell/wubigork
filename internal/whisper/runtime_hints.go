// Package whisper — runtime_hints.go
// 100% 对齐 ackem context/runtimeHints.ts
// 运行时提示格式化：将 RuntimeContext 转为可注入 LLM 的说明块

package whisper

import (
	"fmt"
	"strings"
)

// FormatActivityHint 生活场景 hint（CTX-A）；confidence 过低时不输出
func FormatActivityHint(runtime RuntimeContext) string {
	a := runtime.Activity
	if a.Confidence < 0.4 || a.Category == ActivityUnknown {
		return ""
	}
	return fmt.Sprintf("用户当前场景：%s（置信 %d%%）", a.Label, int(a.Confidence*100))
}

// FormatRuntimeContextHint 将 RuntimeContext 格式化为可注入 LLM 的说明块
func FormatRuntimeContextHint(runtime RuntimeContext) string {
	user := runtime.User
	companion := runtime.Companion
	timeCtx := runtime.Time

	var lines []string
	lines = append(lines,
		fmt.Sprintf("【运行时上下文】本地 %s %s（%s）", timeCtx.LocalDate, timeCtx.LocalTime, timeCtx.TimeOfDay),
		fmt.Sprintf("用户最后活跃：%d 分钟前，参与度 %s", user.MinutesSinceLastChat, user.Engagement),
		fmt.Sprintf("陪伴在场：%s，空闲 %d 分钟", companion.Mode, companion.IdleDurationMs/60000),
	)

	if hint := FormatActivityHint(runtime); hint != "" {
		lines = append(lines, hint)
	}

	switch user.Engagement {
	case EngagementActiveNow, EngagementRecentlyActive:
		lines = append(lines,
			"用户此刻很可能醒着且在线；不要假设 ta 在睡觉或 offline。",
			"若记忆里有熬夜/补觉，以当前仍在互动为准。",
		)
		if len(user.RecentUserSnippets) > 0 {
			snipLines := []string{"用户最近说的话："}
			for i, s := range user.RecentUserSnippets {
				snipLines = append(snipLines, fmt.Sprintf("%d. %s", i+1, s))
			}
			lines = append(lines, snipLines...)
		}
	case EngagementIdle:
		lines = append(lines, "用户可能暂时离开；不要笃定 ta 一定在睡觉。")
	default:
		lines = append(lines, "用户已较长时间未对话；可温和推测，但不要写死「一定在睡觉」。")
	}

	if companion.Mode == CompanionSleeping {
		lines = append(lines, "系统推断用户可能已休息（长时间无交互且处于深夜窗口）。")
	}

	return strings.Join(lines, "\n")
}

// FormatUserPresenceHint 日记等场景专用的用户状态 hint
func FormatUserPresenceHint(runtime RuntimeContext) string {
	return FormatRuntimeContextHint(runtime)
}
