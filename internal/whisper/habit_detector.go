// Package whisper — habit_detector.go
// 100% 对齐 ackem memory/habitDetector.ts
// 时间规律识别：检测用户的时间行为模式

package whisper

import (
	"fmt"
	"sort"
	"time"
)

// TimeHabit 时间规律
type TimeHabit struct {
	Hour         int     `json:"hour"`
	Subcategory  string  `json:"subcategory"`
	Frequency    int     `json:"frequency"`
	AvgIntensity float64 `json:"avgIntensity"`
	Label        string  `json:"label"`
}

// DetectHabits 检测用户时间行为规律（<5ms，每 50 轮调用一次）
func DetectHabits(fs *FactStore) []TimeHabit {
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)

	var habits []TimeHabit
	for hour := 0; hour < 24; hour++ {
		var hourFacts []*Fact
		for _, f := range fs.ListActive() {
			if f.CreatedAt.Hour() == hour && f.CreatedAt.After(thirtyDaysAgo) {
				hourFacts = append(hourFacts, f)
			}
		}
		if len(hourFacts) < 5 {
			continue
		}

		// 按子类分组
		bySub := make(map[string][]*Fact)
		for _, f := range hourFacts {
			bySub[f.Subcategory] = append(bySub[f.Subcategory], f)
		}

		for sub, facts := range bySub {
			if len(facts) < 5 {
				continue
			}
			var totalIntensity float64
			for _, f := range facts {
				if f.EmotionalContext != nil {
					totalIntensity += f.EmotionalContext.Intensity
				}
			}
			avgIntensity := totalIntensity / float64(len(facts))
			habits = append(habits, TimeHabit{
				Hour: hour, Subcategory: sub, Frequency: len(facts),
				AvgIntensity: avgIntensity, Label: buildHabitLabel(hour, sub, len(facts)),
			})
		}
	}

	return habits
}

// FormatHabitHint 生成规律感知提示文本（注入 psycheBlock）
func FormatHabitHint(habits []TimeHabit, currentHour int) string {
	var matched []TimeHabit
	for _, h := range habits {
		if h.Hour == currentHour {
			matched = append(matched, h)
		}
	}
	if len(matched) == 0 {
		return ""
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].AvgIntensity > matched[j].AvgIntensity })
	top := matched[0]
	return fmt.Sprintf("【时间感知】你总是在%s。", top.Label)
}

func buildHabitLabel(hour int, sub string, freq int) string {
	period := "晚上"
	switch {
	case hour < 6:
		period = "凌晨"
	case hour < 12:
		period = "上午"
	case hour < 18:
		period = "下午"
	}
	subLabel := map[string]string{
		"MOOD": "情绪波动", "VULNERABILITIES": "感到脆弱", "OUR_BOND": "想被陪伴",
		"HEALTH": "关注健康", "CAREER": "提及工作", "ROUTINES": "日常习惯",
		"TASTES": "表达喜好", "GOALS": "谈论目标",
	}
	label := subLabel[sub]
	if label == "" {
		label = sub
	}
	return fmt.Sprintf("%s%d点·%s（%d次）", period, hour, label, freq)
}
