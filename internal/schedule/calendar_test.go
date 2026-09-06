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
