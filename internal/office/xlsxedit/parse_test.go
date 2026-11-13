package xlsxedit

// parse_test.go — parse.go 纯函数钉子（波2 扩面）：clean To 转换、delete_rows
// text 条件与 xlsx_profile 概况三处共用同一套判定，这里锁住正/负样本语义。

import (
	"testing"
	"time"
)

func TestParseLooseNumber(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"¥1,299", 1299, true},
		{"￥1，299", 1299, true},
		{"89元", 89, true},
		{"10%", 0.1, true},
		{"10％", 0.1, true}, // 全角百分号
		{"1 234.5", 1234.5, true},
		{" 12 件 ", 12, true},
		{"-3.5", -3.5, true},
		{".5", 0.5, true},
		{"2.5kg", 2.5, true},
		{"100", 100, true},
		// 负样本：不能解析成数字的
		{"", 0, false},
		{"北京", 0, false},
		{"3月1日", 0, false}, // 数字夹在中间，后缀不连续
		{"元", 0, false},    // 只有单位没有数值
		{"a1b2", 0, false}, // 字母数字混杂
		{"-元", 0, false},   // 只有符号
	}
	for _, tc := range cases {
		got, ok := parseLooseNumber(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("parseLooseNumber(%q) = %v, %v; want %v, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestParseFlexibleDate(t *testing.T) {
	want := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	for _, in := range []string{"2025/3/1", "2025.03.01", "2025-3-1", "2025年3月1日", "20250301"} {
		got, ok := parseFlexibleDate(in, 2026)
		if !ok || !got.Equal(want) {
			t.Errorf("parseFlexibleDate(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	// 缺年日期补 defYear
	got, ok := parseFlexibleDate("3月1日", 2026)
	if !ok || got.Year() != 2026 || got.Month() != time.March || got.Day() != 1 {
		t.Errorf("缺年日期应补 defYear: %v, %v", got, ok)
	}
	got, ok = parseFlexibleDate("3.1", 0) // defYear≤0 取当年
	year := time.Now().Year()
	if !ok || got.Year() != year || got.Month() != time.March || got.Day() != 1 {
		t.Errorf("缺年日期应取当年(%d): %v, %v", year, got, ok)
	}
	// 非法日期
	for _, in := range []string{"", "2025/13/1", "2025/2/30", "13月40日", "20250332", "北京"} {
		if _, ok := parseFlexibleDate(in, 2026); ok {
			t.Errorf("parseFlexibleDate(%q) 应解析失败", in)
		}
	}
	// 带时间的 ISO 兜底
	if got, ok := parseFlexibleDate("2025-03-01 08:30:00", 0); !ok || !got.Equal(want) {
		t.Errorf("ISO 兜底 = %v, %v; want %v", got, ok, want)
	}
}

func TestDateShapedHelpers(t *testing.T) {
	// 文本形态：含 8 位纯数字
	for _, in := range []string{"20250301", "2025/3/1", "2026年3月1日", "3月1日"} {
		if !looseDateShaped(in) {
			t.Errorf("looseDateShaped(%q) 应为 true", in)
		}
	}
	if looseDateShaped("100") || looseDateShaped("北京") {
		t.Error("普通数字/文本不是日期形态")
	}
	// 数字格显示形态：必须带分隔符，纯 8 位数字不算（订单号防误判）
	if !displayDateShaped("2025/3/1") || !displayDateShaped("2026年3月1日") {
		t.Error("带分隔符日期显示应为日期形态")
	}
	if displayDateShaped("20250301") {
		t.Error("纯 8 位数字显示不应判为日期形态（存储为数字的编号）")
	}
}
