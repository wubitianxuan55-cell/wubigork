// Package whisper — plan_date_window.go
// 100% 对齐 ackem context/planDateWindow.ts
// 日期窗口解析：从 PLANS/COMMITMENTS 事实中提取日期范围+活动推断

package whisper

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TemporalFactRef 时间事实引用
type TemporalFactRef struct {
	Subcategory string `json:"subcategory"`
	Summary     string `json:"summary"`
}

// ParsedPlanWindow 解析后的计划窗口
type ParsedPlanWindow struct {
	Category    UserActivityCategory `json:"category"`
	StartDay    string               `json:"startDay"`
	EndDay      string               `json:"endDay"`
	Subcategory string               `json:"subcategory"`
}

// ToLocalDayKey 本地日期键 YYYY-MM-DD
func ToLocalDayKey(d time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year(), int(d.Month()), d.Day())
}

func dayKeyToTime(day string) time.Time {
	parts := strings.Split(day, "-")
	if len(parts) != 3 {
		return time.Now()
	}
	y, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	d, _ := strconv.Atoi(parts[2])
	return time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.Local)
}

func addDaysToKey(day string, delta int) string {
	t := dayKeyToTime(day)
	return ToLocalDayKey(t.AddDate(0, 0, delta))
}

var (
	travelKW = regexp.MustCompile(`(?i)旅游|出游|旅行|出差|航班|景点|酒店|度假`)
	workKW   = regexp.MustCompile(`(?i)开会|项目|加班|上班|办公|ddl|deadline|赶工`)
	studyKW  = regexp.MustCompile(`(?i)考试|复习|备考|论文|上课`)
)

func inferCategory(summary string) UserActivityCategory {
	switch {
	case travelKW.MatchString(summary):
		return ActivityTravel
	case studyKW.MatchString(summary):
		return ActivityStudy
	case workKW.MatchString(summary):
		return ActivityWork
	default:
		return ActivityTravel
	}
}

// ParseDateRangeFromText 从文本解析日期范围
func ParseDateRangeFromText(text string, refYear int, now time.Time) (start, end string, ok bool) {
	t := strings.TrimSpace(text)
	if t == "" {
		return "", "", false
	}

	// ISO范围: 2024-03-15 至 2024-03-20
	isoRE := regexp.MustCompile(`(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})\s*(?:日)?\s*(?:至|到|-|~|—)\s*(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})`)
	if m := isoRE.FindStringSubmatch(t); m != nil {
		y1, _ := strconv.Atoi(m[1])
		m1, _ := strconv.Atoi(m[2])
		d1, _ := strconv.Atoi(m[3])
		y2, _ := strconv.Atoi(m[4])
		m2, _ := strconv.Atoi(m[5])
		d2, _ := strconv.Atoi(m[6])
		return fmt.Sprintf("%04d-%02d-%02d", y1, m1, d1), fmt.Sprintf("%04d-%02d-%02d", y2, m2, d2), true
	}

	// 中文范围: 3月15日 至 4月20日
	cnRE := regexp.MustCompile(`(\d{1,2})\s*月\s*(\d{1,2})\s*日?\s*(?:至|到|-|~|—)\s*(\d{1,2})\s*月\s*(\d{1,2})\s*日?`)
	if m := cnRE.FindStringSubmatch(t); m != nil {
		m1, _ := strconv.Atoi(m[1])
		d1, _ := strconv.Atoi(m[2])
		m2, _ := strconv.Atoi(m[3])
		d2, _ := strconv.Atoi(m[4])
		return fmt.Sprintf("%04d-%02d-%02d", refYear, m1, d1), fmt.Sprintf("%04d-%02d-%02d", refYear, m2, d2), true
	}

	// 中文单日: 3月15日
	cnSingle := regexp.MustCompile(`(\d{1,2})\s*月\s*(\d{1,2})\s*日?`)
	if m := cnSingle.FindStringSubmatch(t); m != nil {
		m1, _ := strconv.Atoi(m[1])
		d1, _ := strconv.Atoi(m[2])
		day := fmt.Sprintf("%04d-%02d-%02d", refYear, m1, d1)
		return day, day, true
	}

	base := ToLocalDayKey(now)

	// 明天/后天/下周
	if strings.Contains(t, "明天") {
		d := addDaysToKey(base, 1)
		return d, d, true
	}
	if strings.Contains(t, "后天") {
		d := addDaysToKey(base, 2)
		return d, d, true
	}
	if strings.Contains(t, "下周") {
		return addDaysToKey(base, 1), addDaysToKey(base, 7), true
	}

	return "", "", false
}

// ParsePlanWindowsFromFacts 从事实列表解析计划窗口
func ParsePlanWindowsFromFacts(facts []TemporalFactRef, now time.Time) []ParsedPlanWindow {
	refYear := now.Year()
	var out []ParsedPlanWindow
	for _, f := range facts {
		if f.Subcategory != "PLANS" && f.Subcategory != "COMMITMENTS" {
			continue
		}
		start, end, ok := ParseDateRangeFromText(f.Summary, refYear, now)
		if !ok {
			continue
		}
		if start > end {
			start, end = end, start
		}
		out = append(out, ParsedPlanWindow{
			Category:    inferCategory(f.Summary),
			StartDay:    start,
			EndDay:      end,
			Subcategory: f.Subcategory,
		})
	}
	return out
}

// TenseForPlanWindow 计算计划窗口时态
func TenseForPlanWindow(startDay, endDay string, now time.Time) ActivityTense {
	today := ToLocalDayKey(now)
	if today < startDay {
		return TenseFuture
	}
	if today > endDay {
		return TensePast
	}
	return TensePresent
}

func scorePlanWindow(w ParsedPlanWindow, tense ActivityTense, today string) float64 {
	switch tense {
	case TensePresent:
		return 100
	case TenseFuture:
		daysUntil := math.Max(0, dayKeyToTime(w.StartDay).Sub(dayKeyToTime(today)).Hours()/24)
		if daysUntil <= 7 {
			return 85 - daysUntil
		}
		return 50
	case TensePast:
		daysSince := math.Max(0, dayKeyToTime(today).Sub(dayKeyToTime(w.EndDay)).Hours()/24)
		if daysSince <= 3 {
			return 40 - daysSince
		}
		return 0
	}
	return 0
}

type scoredWindow struct {
	ParsedPlanWindow
	Tense ActivityTense
	Score float64
}

// ResolveActivityFromTemporalFacts 从时间事实推断活动
func ResolveActivityFromTemporalFacts(facts []TemporalFactRef, now time.Time) *UserActivityContext {
	windows := ParsePlanWindowsFromFacts(facts, now)
	if len(windows) == 0 {
		return nil
	}

	today := ToLocalDayKey(now)
	var scored []scoredWindow
	for _, w := range windows {
		tense := TenseForPlanWindow(w.StartDay, w.EndDay, now)
		s := scorePlanWindow(w, tense, today)
		if s > 0 {
			scored = append(scored, scoredWindow{w, tense, s})
		}
	}
	if len(scored) == 0 {
		return nil
	}

	// 简单选最高分
	best := scored[0]
	for _, s := range scored[1:] {
		if s.Score > best.Score {
			best = s
		}
	}

	corpus := ""
	for _, f := range facts {
		corpus += f.Summary + " "
	}
	explicitDates := regexp.MustCompile(`\d{1,2}\s*月|\d{4}[-/]`).MatchString(corpus)

	confidence := 0.72
	switch best.Tense {
	case TensePresent:
		confidence = 0.88
	case TenseFuture:
		confidence = 0.82
	}
	if !explicitDates {
		confidence -= 0.08
	}

	label := buildActivityLabel(best.Category, best.Tense)
	return &UserActivityContext{
		Category:   best.Category,
		Tense:      best.Tense,
		Label:      label,
		Confidence: math.Round(confidence*100) / 100,
		Source:     []string{"memory:" + best.Subcategory, "ctx-b:date_window"},
	}
}
