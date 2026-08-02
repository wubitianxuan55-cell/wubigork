// Package whisper — special_date_detector.go
// 100% 对齐 ackem engine/temporalAwareness/specialDateDetector.ts
// 增强版特殊日期检测：聚合生日/锚点/节假日4数据源
// 基础版 DetectSpecialDates 见 temporal_date.go

package whisper

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// SpecialDateV2 增强版特殊日期（对齐ackem SpecialDate）
type SpecialDateV2 struct {
	Type               string           `json:"type"`
	Title              string           `json:"title"`
	Subject            string           `json:"subject,omitempty"`
	DaysSince          int              `json:"daysSince,omitempty"`
	YearsSince         int              `json:"yearsSince,omitempty"`
	TimeDepth          *TimeDepthResult `json:"timeDepth,omitempty"`
	LinkedFactIDs      []string         `json:"linkedFactIds,omitempty"`
	EmotionalIntensity float64          `json:"emotionalIntensity,omitempty"`
}

// BirthdayEntryV2 生日条目
type BirthdayEntryV2 struct {
	Subject      string `json:"subject"`
	BirthdayMMDD string `json:"birthdayMMDD"`
}

// AnchorEntryV2 锚点条目
type AnchorEntryV2 struct {
	AnchorDate         string  `json:"anchor_date"`
	AnchorType         string  `json:"anchor_type"`
	LinkedFactIDs      string  `json:"linked_fact_ids"`
	EmotionalIntensity float64 `json:"emotional_intensity"`
}

// DetectSpecialDatesV2 增强版特殊日期检测（聚合4数据源）
func DetectSpecialDatesV2(today time.Time, firstMetDate, ackemBirthday string,
	birthdays []BirthdayEntryV2, anchors []AnchorEntryV2) []SpecialDateV2 {

	todayMMDD := fmt.Sprintf("%02d-%02d", int(today.Month()), today.Day())
	var results []SpecialDateV2

	// 源0: Ackem生日
	if ackemBirthday != "" && len(ackemBirthday) >= 10 && ackemBirthday[5:10] == todayMMDD {
		td := ComputeTimeDepth(ackemBirthday, today)
		ys := 0
		if td != nil {
			ys = td.YearsSince
		}
		title := fmt.Sprintf("Hermes%d岁生日", ys)
		if ys == 1 {
			title = "Hermes1岁生日"
		}
		results = append(results, SpecialDateV2{
			Type: "ackem_birthday", Title: title, YearsSince: ys,
			EmotionalIntensity: math.Min(1.0, 0.7+float64(ys)*0.05),
		})
	}

	// 源1: 相识周年
	if IsAnniversaryWindowActive(firstMetDate, today) {
		if td := ComputeTimeDepth(firstMetDate, today); td != nil {
			ay := td.YearsSince
			if r := int(float64(td.DaysSince)/365.2425 + 0.5); r > ay {
				ay = r
			}
			if ay >= 1 {
				title := fmt.Sprintf("相识%d周年", ay)
				if ay == 1 {
					title = "相识1周年"
				}
				results = append(results, SpecialDateV2{
					Type: "first_met_anniversary", Title: title, DaysSince: td.DaysSince,
					YearsSince: ay, TimeDepth: td,
					EmotionalIntensity: math.Min(0.95, 0.6+float64(ay)*0.1),
				})
			}
		}
	}

	// 源2: 生日
	seen := map[string]bool{}
	for _, b := range birthdays {
		if b.BirthdayMMDD != todayMMDD {
			continue
		}
		k := b.Subject + "_" + b.BirthdayMMDD
		if seen[k] {
			continue
		}
		seen[k] = true
		results = append(results, SpecialDateV2{
			Type: "birthday", Title: b.Subject + "生日", Subject: b.Subject,
			EmotionalIntensity: 1.0,
		})
	}

	// 源3: 时间锚点
	for _, a := range anchors {
		if a.AnchorType == "fuzzy" || len(a.AnchorDate) < 10 || a.AnchorDate[5:10] != todayMMDD {
			continue
		}
		stype := "recurring_memory"
		title := "周期记忆"
		switch a.AnchorType {
		case "relationship":
			stype = "relationship"
			title = "关系纪念"
		case "milestone":
			stype = "milestone"
			title = "里程碑"
		}
		results = append(results, SpecialDateV2{
			Type: stype, Title: title, EmotionalIntensity: a.EmotionalIntensity,
		})
	}

	// 排序
	order := map[string]int{"ackem_birthday": 0, "first_met_anniversary": 0, "relationship": 0, "birthday": 1, "milestone": 2, "holiday": 3, "recurring_memory": 4}
	sort.Slice(results, func(i, j int) bool {
		if oi, oj := order[results[i].Type], order[results[j].Type]; oi != oj {
			return oi < oj
		}
		return results[i].EmotionalIntensity > results[j].EmotionalIntensity
	})
	return results
}
