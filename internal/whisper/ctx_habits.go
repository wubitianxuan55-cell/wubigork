// Package whisper — ctx_habits.go
// 100% 对齐 ackem context/ctxB2Habits.ts
// CTX-B2：从程序化习惯推断用户活动（避免瞎猜）

package whisper

import (
	"regexp"
	"strings"
)

var workHabitRE = regexp.MustCompile(`(?i)开会|办公|写代码|编程|上班|工作|加班|ddl`)
var restHabitRE = regexp.MustCompile(`(?i)睡觉|休息|熬夜|补觉`)

// ResolveActivityFromEstablishedHabits 从已建立的习惯推断用户活动
// habits 为已建立的程序化习惯文本列表
func ResolveActivityFromEstablishedHabits(habits []string) *UserActivityContext {
	if len(habits) == 0 {
		return nil
	}

	joined := strings.Join(habits, " ")

	if workHabitRE.MatchString(joined) {
		return &UserActivityContext{
			Category:   ActivityWork,
			Tense:      TensePresent,
			Label:      "工作·进行中",
			Confidence: 0.78,
			Source:     []string{"ctx-b2:habit_established", "procedural-memory"},
		}
	}
	if restHabitRE.MatchString(joined) {
		return &UserActivityContext{
			Category:   ActivityRest,
			Tense:      TensePresent,
			Label:      "休息·进行中",
			Confidence: 0.75,
			Source:     []string{"ctx-b2:habit_established", "procedural-memory"},
		}
	}

	return &UserActivityContext{
		Category:   ActivityDaily,
		Tense:      TensePresent,
		Label:      "日常·进行中",
		Confidence: 0.72,
		Source:     []string{"ctx-b2:habit_established", "procedural-memory"},
	}
}
