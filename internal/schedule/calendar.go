// schedule/calendar.go — 工作日历工具（前端 calendar.ts 的忠实移植，v4.111.0 刀2 口径）。
//
// CPM 与工期全程用工作日整数，本模块负责「工作日序号 ↔ 日历日期」双向换算
// （周末按工作制集合、节假日例外命中即跳过）。扫描上限 3650 天（≈10 年）防呆。
package schedule

import (
	"fmt"
	"regexp"
	"sort"
	"time"
)

// DefaultCalendar 默认基准日历：周一~周五为工作日（getDay 口径 0=周日..6=周六）。
func DefaultCalendar() Calendar {
	return Calendar{Workweek: []int{1, 2, 3, 4, 5}, Holidays: []string{}}
}

const scanLimit = 3650

var isoDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// NormalizeCalendar 归一化日历：空/非法回退默认；去重、裁剪到 0..6、过滤非法日期串。
func NormalizeCalendar(cal *Calendar) Calendar {
	if cal == nil || len(cal.Workweek) == 0 {
		return DefaultCalendar()
	}
	seen := map[int]bool{}
	wk := make([]int, 0, len(cal.Workweek))
	for _, n := range cal.Workweek {
		if n >= 0 && n <= 6 && !seen[n] {
			seen[n] = true
			wk = append(wk, n)
		}
	}
	if len(wk) == 0 {
		return DefaultCalendar()
	}
	sort.Ints(wk)
	holidays := make([]string, 0, len(cal.Holidays))
	for _, d := range cal.Holidays {
		if isoDateRe.MatchString(d) {
			holidays = append(holidays, d)
		}
	}
	return Calendar{Workweek: wk, Holidays: holidays}
}

// parseISO 解析 YYYY-MM-DD 为 UTC 日期；非法输入回退 1970-01-01（与前端 parseISO 同口径）。
func parseISO(iso string) time.Time {
	var y, m, d int
	if _, err := fmt.Sscanf(iso, "%4d-%2d-%2d", &y, &m, &d); err != nil || len(iso) != 10 {
		return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if y <= 0 {
		y = 1970
	}
	if m <= 0 {
		m = 1
	}
	if d <= 0 {
		d = 1
	}
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

// ISOOf 日期 → YYYY-MM-DD（UTC）。
func ISOOf(d time.Time) string {
	return d.Format("2006-01-02")
}

// IsWorkingDate 命中工作制且未落在节假日例外。
func IsWorkingDate(d time.Time, cal Calendar) bool {
	wd := int(d.Weekday()) // time.Weekday 与 JS getDay 同口径：周日=0
	ok := false
	for _, n := range cal.Workweek {
		if n == wd {
			ok = true
			break
		}
	}
	if !ok {
		return false
	}
	for _, h := range cal.Holidays {
		if h == ISOOf(d) {
			return false
		}
	}
	return true
}

// WdToDate 开工日（startISO）起第 idx 个工作日（idx=0 即开工日当日；若开工日恰为
// 非工作日，自动顺延到首个工作日）。idx<0 返回开工日顺延前的原日期。
func WdToDate(startISO string, idx int, calInput *Calendar) time.Time {
	cal := NormalizeCalendar(calInput)
	start := parseISO(startISO)
	if idx < 0 {
		return start
	}
	found := 0
	cur := start
	for !IsWorkingDate(cur, cal) && found < scanLimit {
		cur = cur.AddDate(0, 0, 1)
	}
	found++ // 首个工作日即第 0 个
	for found < idx+1 && found < scanLimit {
		cur = cur.AddDate(0, 0, 1)
		if IsWorkingDate(cur, cal) {
			found++
		}
	}
	return cur
}

// DateToWd 日期 → 工作日序号；非工作日（周末/节假日）或早于开工日返回 (0,false)。
func DateToWd(startISO, dateISO string, calInput *Calendar) (int, bool) {
	cal := NormalizeCalendar(calInput)
	start := parseISO(startISO)
	target := parseISO(dateISO)
	if target.Before(start) {
		return 0, false
	}
	idx := -1
	for cur := start; !cur.After(target) && idx < scanLimit; cur = cur.AddDate(0, 0, 1) {
		if IsWorkingDate(cur, cal) {
			idx++
			if cur.Equal(target) {
				return idx, true
			}
		}
	}
	return 0, false
}

// DeadlineWorkdays 目标竣工日期 → 目标总工期（工作日）：[开工日, 竣工日] 内的
// 工作日计数（竣工日恰为非工作日时自然回落到此前最近工作日，即「第 count 个
// 工作日竣工」）。竣工早于开工 → 0（必然不可达，由倒排校核裁决报告）。
func DeadlineWorkdays(startISO, deadlineISO string, calInput *Calendar) int {
	cal := NormalizeCalendar(calInput)
	start := parseISO(startISO)
	dl := parseISO(deadlineISO)
	if dl.Before(start) {
		return 0
	}
	count := 0
	for cur := start; !cur.After(dl) && count < scanLimit; cur = cur.AddDate(0, 0, 1) {
		if IsWorkingDate(cur, cal) {
			count++
		}
	}
	return count
}

// CdToEf cd（日历天）任务正推换算（v4.150 双工期刀1，镜像前端 calendar.ts
// cdToEf）：完成边界 = 首个 ≥（es 所在工作日 + cd 自然日）的工作日序号
// （ceil 吸附：边界落在周末/节假日即顺延到其后第一个工作日；边界恰为工作日
// 即当日；cd<=0 → es）。扫描上限与 WdToDate 同源（3650 天防呆，越界截断）。
func CdToEf(startISO string, es, cd int, calInput *Calendar) int {
	cal := NormalizeCalendar(calInput)
	if es < 0 {
		es = 0
	}
	if cd < 0 {
		cd = 0
	}
	anchor := WdToDate(startISO, es, &cal)
	boundary := anchor.AddDate(0, 0, cd)
	idx := es
	steps := 0
	for cur := anchor; steps < scanLimit && (cur.Before(boundary) || !IsWorkingDate(cur, cal)); steps++ {
		cur = cur.AddDate(0, 0, 1)
		if IsWorkingDate(cur, cal) {
			idx++
		}
	}
	return idx
}

// CdLatestStart cd（日历天）任务逆推换算（镜像前端 calendar.ts cdLatestStart）：
// 最大的工作日序号 s 使 fwd(s) ≤ lf（fwd=CdToEf，单调不减且有平段）。
// 镜像回退：lf 所在工作日回退 cd 自然日、向下吸附到工作日，再有界双侧校正至
// 满足性质 fwd(s) ≤ lf < fwd(s+1)；下限截到 0（负时差极端场景，两侧镜像一致）。
func CdLatestStart(startISO string, lf, cd int, calInput *Calendar) int {
	cal := NormalizeCalendar(calInput)
	if cd < 0 {
		cd = 0
	}
	if lf < 0 {
		lf = 0
	}
	start := parseISO(startISO)
	boundary := WdToDate(startISO, lf, &cal).AddDate(0, 0, -cd)
	cur := boundary
	for cur.After(start) && !IsWorkingDate(cur, cal) {
		cur = cur.AddDate(0, 0, -1)
	}
	s := 0
	if idx, ok := DateToWd(startISO, ISOOf(cur), &cal); ok && idx > 0 {
		s = idx
	}
	for i := 0; i < scanLimit && s > 0 && CdToEf(startISO, s, cd, &cal) > lf; i++ {
		s--
	}
	for i := 0; i < scanLimit && CdToEf(startISO, s+1, cd, &cal) <= lf; i++ {
		s++
	}
	return s
}
