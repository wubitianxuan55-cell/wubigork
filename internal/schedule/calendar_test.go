package schedule

// calendar_test.go — 工作日历用例与前端 calendar.test.ts 口径镜像。

import (
	"testing"
)

func TestNormalizeCalendarDefaults(t *testing.T) {
	if got := NormalizeCalendar(nil); len(got.Workweek) != 5 || got.Workweek[0] != 1 {
		t.Fatalf("%+v", got)
	}
	if got := NormalizeCalendar(&Calendar{Workweek: []int{}}); len(got.Workweek) != 5 {
		t.Fatalf("空工作制应回退默认：%+v", got)
	}
	got := NormalizeCalendar(&Calendar{Workweek: []int{6, 6, 0, 9, -1}, Holidays: []string{"2026-10-01", "bad"}})
	if len(got.Workweek) != 2 || got.Workweek[0] != 0 || got.Workweek[1] != 6 {
		t.Fatalf("去重/裁剪失败：%+v", got)
	}
	if len(got.Holidays) != 1 {
		t.Fatalf("非法日期应滤除：%+v", got.Holidays)
	}
}

func TestWdToDateRoundTrip(t *testing.T) {
	// 2026-09-01 是周二；第 0 个工作日=当天，第 4 个=下周一 09-07（周末跳过）
	start := "2026-09-01"
	if d := WdToDate(start, 0, nil); ISOOf(d) != "2026-09-01" {
		t.Fatalf("wd0 = %s", ISOOf(d))
	}
	if d := WdToDate(start, 4, nil); ISOOf(d) != "2026-09-07" {
		t.Fatalf("wd4 = %s", ISOOf(d))
	}
	// 互逆：dateToWd(wdToDate(idx)) == idx
	for _, idx := range []int{0, 1, 4, 5, 9} {
		wd, ok := DateToWd(start, ISOOf(WdToDate(start, idx, nil)), nil)
		if !ok || wd != idx {
			t.Fatalf("互逆失败 idx=%d → %d,%v", idx, wd, ok)
		}
	}
	// 开工日为周六：顺延到周一为第 0 个工作日
	if d := WdToDate("2026-09-05", 0, nil); ISOOf(d) != "2026-09-07" {
		t.Fatalf("周六开工应顺延：%s", ISOOf(d))
	}
	// 节假日例外命中即跳过
	cal := Calendar{Workweek: []int{1, 2, 3, 4, 5}, Holidays: []string{"2026-10-01"}}
	if d := WdToDate("2026-09-30", 1, &cal); ISOOf(d) != "2026-10-02" {
		t.Fatalf("节假日应跳过：%s", ISOOf(d))
	}
	// 非工作日与早于开工日 → false
	if _, ok := DateToWd(start, "2026-09-05", nil); ok {
		t.Fatal("周六应非工作日")
	}
	if _, ok := DateToWd(start, "2026-08-31", nil); ok {
		t.Fatal("早于开工日应 false")
	}
}

// ── 双工期口径（v4.150 刀1）：cd=日历天换算（镜像 TS calendar.test.ts cdToEf/cdLatestStart）──

func TestCdToEf(t *testing.T) {
	mon := "2026-09-07"
	if got := CdToEf(mon, 0, 28, nil); got != 20 {
		t.Fatalf("28cd = %d, want 20（等效跨度）", got)
	}
	if got := CdToEf(mon, 0, 1, nil); got != 1 {
		t.Fatalf("1cd = %d", got)
	}
	if got := CdToEf(mon, 0, 0, nil); got != 0 {
		t.Fatalf("0cd 应回 es")
	}
	// ceil 吸附折叠：边界落周末（26/27）与恰为工作日（28）同 ef
	for _, cd := range []int{26, 27, 28} {
		if got := CdToEf(mon, 0, cd, nil); got != 20 {
			t.Fatalf("%dcd = %d, want 20", cd, got)
		}
	}
	// es>0 锚点随行：周四起 7cd → ef=8（跨度 5）
	if got := CdToEf(mon, 3, 7, nil); got != 8 {
		t.Fatalf("es3+7cd = %d, want 8", got)
	}
	// 节假日窗口：边界前一节假日不计入索引
	cal := Calendar{Workweek: []int{1, 2, 3, 4, 5}, Holidays: []string{"2026-09-16"}}
	if got := CdToEf(mon, 0, 10, &cal); got != 7 {
		t.Fatalf("holiday 10cd = %d, want 7", got)
	}
	if got := CdToEf(mon, 0, 10, nil); got != 8 {
		t.Fatalf("plain 10cd = %d, want 8", got)
	}
	// 开工日为非工作日：锚点顺延后起算
	if got := CdToEf("2026-09-12", 0, 3, nil); got != 3 {
		t.Fatalf("sat start 3cd = %d, want 3", got)
	}
}

func TestCdLatestStart(t *testing.T) {
	mon := "2026-09-07"
	// 性质 fwd(s) ≤ lf < fwd(s+1)：fwd(0)=20、fwd(5)=25
	if got := CdLatestStart(mon, 20, 28, nil); got != 0 {
		t.Fatalf("lf20 = %d, want 0", got)
	}
	if got := CdLatestStart(mon, 25, 28, nil); got != 5 {
		t.Fatalf("lf25 = %d, want 5", got)
	}
	// 平段吸附：26cd 时 fwd(0)=fwd(1)=fwd(2)=20，取最大 s=2
	if got := CdLatestStart(mon, 20, 26, nil); got != 2 {
		t.Fatalf("flat = %d, want 2", got)
	}
	// 负时差极端：下限截 0
	if got := CdLatestStart(mon, 0, 28, nil); got != 0 {
		t.Fatalf("lf0 = %d", got)
	}
	if got := CdLatestStart(mon, 19, 28, nil); got != 0 {
		t.Fatalf("lf19 = %d", got)
	}
}

// ── v4.155 双工期欠账放开：CdEarliestStart（正推单调逆查，镜像 TS cdEarliestStart）──

func TestCdEarliestStart(t *testing.T) {
	mon := "2026-01-05" // 周一开工锚点（周一~五无节假日，与 v4.155 镜像表同源）
	// 锚点表 cd=7（fwd(s,7)=5..15 单调）：target→最小 s
	for _, c := range []struct{ target, want int }{
		{5, 0}, {6, 1}, {9, 4}, {10, 5}, {11, 6}, {12, 7},
	} {
		if got := CdEarliestStart(mon, c.target, 7, nil); got != c.want {
			t.Fatalf("cd7 target%d = %d, want %d", c.target, got, c.want)
		}
	}
	// 锚点表 cd=5（fwd(s,5) 平段 0-2→5、5-7→10）：最小性取平台头
	for _, c := range []struct{ target, want int }{
		{5, 0}, {6, 3}, {10, 5}, {11, 8},
	} {
		if got := CdEarliestStart(mon, c.target, 5, nil); got != c.want {
			t.Fatalf("cd5 target%d = %d, want %d", c.target, got, c.want)
		}
	}
	// target ≤ 0 → 0
	if got := CdEarliestStart(mon, 0, 7, nil); got != 0 {
		t.Fatalf("target0 = %d", got)
	}
	if got := CdEarliestStart(mon, -3, 5, nil); got != 0 {
		t.Fatalf("target-3 = %d", got)
	}
	// 最小性性质：CdToEf(s) ≥ target 且（s>0 时）CdToEf(s−1) < target
	for target := 1; target <= 40; target++ {
		s := CdEarliestStart(mon, target, 7, nil)
		if s < 0 || CdToEf(mon, s, 7, nil) < target {
			t.Fatalf("target%d: fwd(s=%d)=%d 未达下界", target, s, CdToEf(mon, s, 7, nil))
		}
		if s > 0 && CdToEf(mon, s-1, 7, nil) >= target {
			t.Fatalf("target%d: s=%d 非最小（fwd(s−1)=%d）", target, s, CdToEf(mon, s-1, 7, nil))
		}
	}
}
