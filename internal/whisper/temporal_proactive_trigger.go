// Package whisper — temporal_proactive_trigger.go
// 100% 对齐 ackem engine/temporalAwareness/temporalProactiveTrigger.ts
// 时间主动触发信号：编排 specialDate + temporalMemoryBridge → TemporalProactiveSignal
// 基础版 ProduceTemporalSignal 见 temporal_bridge.go

package whisper

import (
	"sort"
	"time"
)

// MemoryBundleV2 时间记忆包（对齐ackem MemoryBundle）
type MemoryBundleV2 struct {
	Narrative   string   `json:"narrative"`
	FactIDs     []string `json:"factIds"`
	EmotionBias float64  `json:"emotionBias"`
}

// TemporalHintV2 时间提示
type TemporalHintV2 struct {
	DateLabel string `json:"dateLabel"`
	Narrative string `json:"narrative"`
	Priority  string `json:"priority"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

// TemporalProactiveSignalV2 时间主动触发信号
type TemporalProactiveSignalV2 struct {
	SpecialDates []SpecialDateV2 `json:"specialDates"`
	TemporalHint *TemporalHintV2 `json:"temporalHint,omitempty"`
}

var expiryDaysV2 = map[string]int{
	"ackem_birthday": 30, "birthday": 30, "first_met_anniversary": 60,
	"holiday": 7, "milestone": 60, "relationship": 60, "recurring_memory": 14,
}
var hintOrderV2 = map[string]int{
	"ackem_birthday": 0, "first_met_anniversary": 1, "relationship": 2,
	"birthday": 3, "milestone": 4, "holiday": 5, "recurring_memory": 6,
}

// RecallForSpecialDateV2 为特殊日期生成记忆包
func RecallForSpecialDateV2(sd SpecialDateV2) MemoryBundleV2 {
	b := MemoryBundleV2{FactIDs: sd.LinkedFactIDs}
	switch sd.Type {
	case "ackem_birthday":
		b.Narrative = "今天是轻语的生日，你需要表现出特别的开心和温暖。"
		b.EmotionBias = 3.0
	case "first_met_anniversary":
		b.Narrative = "今天是你们相识的周年纪念，可以自然地回忆初见时的场景。"
		b.EmotionBias = 2.0
	case "birthday":
		n := sd.Subject
		if n == "" {
			n = "ta"
		}
		b.Narrative = "今天是" + n + "的生日，记得送上祝福。"
		b.EmotionBias = 3.0
	case "relationship":
		b.Narrative = "今天是你们的关系纪念日，可以温暖地提起你们的羁绊。"
		b.EmotionBias = 2.0
	case "milestone":
		b.Narrative = "今天是一个值得纪念的里程碑日子。"
		b.EmotionBias = 1.0
	case "holiday":
		b.Narrative = "今天是" + sd.Title + "，可以自然地融入节日氛围。"
		b.EmotionBias = 0.5
	case "recurring_memory":
		b.Narrative = "去年的今天有一些值得回味的记忆。"
		b.EmotionBias = 0.8
	}
	return b
}

func hintPriorityV2(t string) string {
	switch t {
	case "ackem_birthday", "first_met_anniversary", "birthday", "relationship":
		return "high"
	case "milestone":
		return "normal"
	default:
		return "low"
	}
}

func priorityRankV2(p string) int {
	switch p {
	case "high":
		return 0
	case "normal":
		return 1
	default:
		return 2
	}
}

// ProduceTemporalSignalV2 产出时间主动触发信号（对齐ackem）
func ProduceTemporalSignalV2(specialDates []SpecialDateV2) TemporalProactiveSignalV2 {
	type hint struct {
		typ, dateLabel, narrative, priority string
	}
	var hints []hint

	for i, sd := range specialDates {
		bundle := RecallForSpecialDateV2(sd)
		if bundle.Narrative != "" {
			hints = append(hints, hint{
				typ: sd.Type, dateLabel: sd.Title,
				narrative: bundle.Narrative, priority: hintPriorityV2(sd.Type),
			})
			_ = i
		}
	}

	var th *TemporalHintV2
	if len(hints) > 0 {
		sort.Slice(hints, func(i, j int) bool {
			return hintOrderV2[hints[i].typ] < hintOrderV2[hints[j].typ]
		})
		bestPri := "low"
		for _, h := range hints {
			if priorityRankV2(h.priority) < priorityRankV2(bestPri) {
				bestPri = h.priority
			}
		}
		var labels, narratives []string
		for _, h := range hints {
			labels = append(labels, h.dateLabel)
			narratives = append(narratives, h.narrative)
		}
		ed := 14
		if d, ok := expiryDaysV2[hints[0].typ]; ok {
			ed = d
		}
		th = &TemporalHintV2{
			DateLabel: joinStrsV2(labels, " · "),
			Narrative: joinStrsV2(narratives, " "),
			Priority:  bestPri,
			ExpiresAt: time.Now().Add(time.Duration(ed) * 24 * time.Hour).Format(time.RFC3339),
		}
	}
	return TemporalProactiveSignalV2{SpecialDates: specialDates, TemporalHint: th}
}

func joinStrsV2(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	r := parts[0]
	for i := 1; i < len(parts); i++ {
		r += sep + parts[i]
	}
	return r
}
