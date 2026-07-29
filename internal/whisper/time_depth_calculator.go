// Package whisper — time_depth_calculator.go
// 100% 对齐 ackem engine/temporalAwareness/timeDepthCalculator.ts
// 时间深度计算器：输入相识日期和今天，输出"过了多久"的自然语言描述

package whisper

import (
	"fmt"
	"math"
	"regexp"
	"time"
)

// TimeDepthResult 时间深度计算结果
type TimeDepthResult struct {
	Label          string  `json:"label"`
	LabelKey       string  `json:"labelKey"`
	EmotionalWeight float64 `json:"emotionalWeight"`
	IsExactYear    bool    `json:"isExactYear"`
	IsMilestone    bool    `json:"isMilestone"`
	YearsSince     int     `json:"yearsSince"`
	DaysSince      int     `json:"daysSince"`
}

var dateRE = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)

// parseLocalDate 解析 ISO 日期为本地日期
func parseLocalDate(str string) (year, month, day int, ok bool) {
	m := dateRE.FindStringSubmatch(str)
	if m == nil {
		return 0, 0, 0, false
	}
	year, month, day = atoi(m[1]), atoi(m[2]), atoi(m[3])
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return 0, 0, 0, false
	}
	return year, month, day, true
}

func atoi(s string) int {
	var n int
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// ComputeTimeDepth 计算时间深度
func ComputeTimeDepth(firstMetDate string, today time.Time) *TimeDepthResult {
	if firstMetDate == "" {
		return nil
	}

	year, month, day, ok := parseLocalDate(firstMetDate)
	if !ok {
		return nil
	}

	first := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	daysSince := int(todayStart.Sub(first).Hours() / 24)
	if daysSince < 0 {
		return nil
	}

	diffYears := float64(daysSince) / 365.2425
	yearsSince := int(math.Floor(diffYears))
	nearestYear := int(math.Round(diffYears))
	isMilestone := false
	for _, m := range []int{1, 2, 3, 5, 10} {
		if nearestYear == m && nearestYear >= 1 {
			isMilestone = true
			break
		}
	}

	daysSinceLastAnniversary := float64(daysSince) - float64(yearsSince)*365.2425
	distanceToNearestYear := float64(daysSince) - float64(nearestYear)*365.2425

	var label string
	var labelKey string
	var emotionalWeight float64
	isExactYear := false

	switch {
	case nearestYear >= 1 && math.Abs(distanceToNearestYear) <= 15:
		isExactYear = true
		if nearestYear == 1 {
			labelKey = "timeDepth.exactYear"
			label = fmt.Sprintf("整整一年")
		} else {
			labelKey = "timeDepth.exactYears"
			label = fmt.Sprintf("整整%d年", nearestYear)
		}
		emotionalWeight = math.Min(0.95, 0.8+float64(nearestYear)*0.05)

	case daysSince < 30:
		labelKey = "timeDepth.justMet"
		label = "初识不久"
		emotionalWeight = 0.3

	case daysSince < 90:
		labelKey = "timeDepth.overMonth"
		label = "相识月余"
		emotionalWeight = 0.4

	case daysSince < 180:
		labelKey = "timeDepth.halfYear"
		label = "相识半年"
		emotionalWeight = 0.5

	case daysSince < 365:
		labelKey = "timeDepth.overHalfYear"
		label = "相识半年有余"
		emotionalWeight = 0.6

	case daysSinceLastAnniversary <= 90:
		if yearsSince == 1 {
			labelKey = "timeDepth.justOverYear"
			label = "刚过一年"
		} else {
			labelKey = "timeDepth.justOverYears"
			label = fmt.Sprintf("刚过%d年", yearsSince)
		}
		emotionalWeight = math.Min(0.95, 0.75+float64(yearsSince)*0.03)

	case daysSinceLastAnniversary > 275:
		labelKey = "timeDepth.almostNextYear"
		label = fmt.Sprintf("快到%d年", yearsSince+1)
		emotionalWeight = math.Min(0.95, 0.78+float64(yearsSince)*0.04)

	default:
		if yearsSince <= 1 {
			labelKey = "timeDepth.overYear"
			label = "相识一年多"
		} else {
			labelKey = "timeDepth.overYears"
			label = fmt.Sprintf("相识%d年多", yearsSince)
		}
		emotionalWeight = math.Min(0.9, 0.7+float64(yearsSince)*0.04)
	}

	return &TimeDepthResult{
		Label:           label,
		LabelKey:        labelKey,
		EmotionalWeight: emotionalWeight,
		IsExactYear:     isExactYear,
		IsMilestone:     isMilestone,
		YearsSince:      yearsSince,
		DaysSince:       daysSince,
	}
}

// IsAnniversaryWindowActive 整周年 ±15 天窗口内且已满 1 年
func IsAnniversaryWindowActive(firstMetDate string, today time.Time) bool {
	td := ComputeTimeDepth(firstMetDate, today)
	if td == nil || !td.IsExactYear {
		return false
	}
	anniversaryYears := td.YearsSince
	if rounded := int(math.Round(float64(td.DaysSince) / 365.2425)); rounded > anniversaryYears {
		anniversaryYears = rounded
	}
	return anniversaryYears >= 1
}
