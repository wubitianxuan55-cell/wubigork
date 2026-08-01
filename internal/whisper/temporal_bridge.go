// Package whisper — temporal_bridge.go
// 100% 对齐 ackem engine/temporalAwareness/temporalMemoryBridge.ts
// 时间记忆桥接：基于时间上下文的 Tier B 种子块

package whisper

import (
	"fmt"
	"time"
)

// ─── TemporalSeedBlock ────────────────────────────────────────

// BuildTemporalSeedTierBBlock 构建时间种子的 Tier B 上下文块
// 基于特殊日期检测结果，注入与时间相关的记忆提示
func BuildTemporalSeedTierBBlock(
	signal *TemporalSignal,
	fs *FactStore,
	kg *KnowledgeGraph,
) string {
	if signal == nil || len(signal.SpecialDates) == 0 {
		return ""
	}

	var parts []string
	for _, sd := range signal.SpecialDates {
		if sd.Priority == "high" {
			parts = append(parts, fmt.Sprintf("今天%s。你可以自然地提起这个特别的日子。", sd.Label))
			// 尝试从知识图谱检索相关记忆
			if kg != nil {
				hits := kg.Query(sd.Label, 3)
				for _, t := range hits {
					parts = append(parts, fmt.Sprintf("相关记忆：%s —%s→ %s", t.Subject, t.Predicate, t.Object))
				}
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return "【时间感知 · 今天特别】\n" + joinLines(parts)
}

// ProduceTemporalSignal 生成时间主动触发信号
func ProduceTemporalSignal(
	now time.Time,
	firstMetDate *time.Time,
	ackemBirthday *time.Time,
	gapHours float64,
	stage RelationshipStage,
) *TemporalSignal {
	signal := DetectSpecialDates(now, firstMetDate, ackemBirthday)

	// 重逢信号
	if gapHours >= 1 && gapHours < 72 && stage != StageStranger {
		if signal == nil {
			signal = &TemporalSignal{}
		}
		duration := HumanizeFeltDuration(int(gapHours * 60))
		signal.SpecialDates = append(signal.SpecialDates, SpecialDate{
			Label:    "重逢：" + duration + "没见了",
			Priority: "medium",
		})
		if signal.TemporalHint == nil {
			signal.TemporalHint = &struct {
				DateLabel string
				Priority  string
			}{DateLabel: "重逢", Priority: "medium"}
		}
	}

	return signal
}

// ─── HumanizeFeltDuration ─────────────────────────────────────

func joinLines(lines []string) string {
	result := ""
	for _, l := range lines {
		if l != "" {
			if result != "" {
				result += "\n"
			}
			result += l
		}
	}
	return result
}
