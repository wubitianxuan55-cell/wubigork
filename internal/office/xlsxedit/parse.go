package xlsxedit

// parse.go — 人手打值解析纯函数（波2 扩面：移植上游 dsh-office OfficeHelper.cs
// 的 ParseNumber/ParseDate/Tidy 语义，只读不改文件）。clean To 转换、
// delete_rows text 条件与 xlsx_profile 列概况共用同一套判定，避免三处漂移。

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// tidyText 归一文本：全角空格/不换行空格/制表符折成半角空格，去首尾空白并
// 折叠连续空格（上游 Tidy 同语义）。取值改名/判重的键都过这里。
func tidyText(s string) string {
	s = strings.ReplaceAll(s, "\u00A0", " ")
	s = strings.ReplaceAll(s, "\u3000", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.TrimSpace(s)
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}

// looseNumberRe 宽松数字形态：可选货币符 + 数值 + 可选百分号 + 可选中文单位
// 后缀（1-3 个非数字字符，如 "元"/"件"）或常见英文单位（上游同款正则）。
var looseNumberRe = regexp.MustCompile(
	`^[¥￥$€£]?([-+]?\d*\.?\d+(?:[eE][-+]?\d+)?)?(%|％)?(?:[^\d\s\-.,A-Za-z%％]{1,3}|kg|g|km|cm|mm|m|pcs|k|w)?$`)

// parseLooseNumber 把人手打的数字文本解析成数值："¥1,299" → 1299、"89元" → 89、
// "10%" → 0.1、"1 234.5" → 1234.5。返回（值, 是否解析成功）；纯数字文本天然
// 命中。百分比按 /100 换算（与 Excel 把 % 读成小数一致）。
func parseLooseNumber(text string) (float64, bool) {
	s := tidyText(text)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "，", "")
	s = strings.ReplaceAll(s, " ", "")
	m := looseNumberRe.FindStringSubmatch(s)
	if m == nil || m[1] == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	if m[2] != "" {
		n /= 100
	}
	return n, true
}

var (
	// fullDateRe 8 位紧凑日期：20250301。
	fullDateRe = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`)
	// ymdDateRe 带年月日的完整日期：2025/3/1、2025.03.01、2026年3月1日
	// （分隔符 -/.年 与 -/.月/日；尾部允许跟时间等杂注）。
	ymdDateRe = regexp.MustCompile(`^(\d{4}) *[-/.年] *(\d{1,2}) *[-/.月] *(\d{1,2}) *日?(?: .*)?$`)
	// mdDateRe 缺年日期：3月1日、3.1、3-1（年取 defYear）。
	mdDateRe = regexp.MustCompile(`^(\d{1,2}) *[-/.月] *(\d{1,2}) *日?$`)
	// textDateShapedRe 文本格的日期形态：额外接受 8 位纯数字（20250301 的
	// 手打写法）——用于「写成文本的日期」识别。
	textDateShapedRe = regexp.MustCompile(
		`^\d{8}$|^\d{4} *[-/.年] *\d{1,2} *[-/.月] *\d{1,2}.*$|^\d{1,2} *[-/.月] *\d{1,2} *日?$`)
	// displayDateShapedRe 数字格显示值的日期形态：必须带分隔符或年月日字样；
	// 纯 8 位数字不算——存储为数字的普通编号（订单号 20250301）不该被当成
	// 已格式化的日期。
	displayDateShapedRe = regexp.MustCompile(
		`^\d{4} *[-/.年] *\d{1,2} *[-/.月] *\d{1,2}( *日?)?( .*)?$|^\d{1,2} *[-/.月] *\d{1,2} *日?$`)
)

// parseFlexibleDate 把人手打的日期文本解析成日期："2025/3/1" "2025.03.01"
// "20250301" "2026年3月1日"，缺年的 "3月1日" 补 defYear（defYear≤0 取当年）。
// 返回（日期, 是否解析成功）；解析失败返回 false，调用方原样保留该格。
func parseFlexibleDate(text string, defYear int) (time.Time, bool) {
	s := tidyText(text)
	if s == "" {
		return time.Time{}, false
	}
	year := defYear
	if year <= 0 {
		year = time.Now().Year()
	}
	var y, mo, d int
	if m := fullDateRe.FindStringSubmatch(s); m != nil {
		y, mo, d = ato3(m[1], m[2], m[3])
	} else if m := ymdDateRe.FindStringSubmatch(s); m != nil {
		y, mo, d = ato3(m[1], m[2], m[3])
	} else if m := mdDateRe.FindStringSubmatch(s); m != nil {
		y, mo, d = ato3(strconv.Itoa(year), m[1], m[2])
	} else {
		return fallbackDateLayouts(s)
	}
	return buildDate(y, mo, d)
}

// ato3 三个数字组一次性转 int（正则保证了形态，出错按 0 处理交给 buildDate 拒绝）。
func ato3(a, b, c string) (int, int, int) {
	y, _ := strconv.Atoi(a)
	mo, _ := strconv.Atoi(b)
	d, _ := strconv.Atoi(c)
	return y, mo, d
}

// buildDate 组装日期并做范围校验（1900-2200、月 1-12、日不超当月天数）。
func buildDate(y, mo, d int) (time.Time, bool) {
	if y < 1900 || y > 2200 || mo < 1 || mo > 12 || d < 1 {
		return time.Time{}, false
	}
	lastDay := time.Date(y, time.Month(mo)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if d > lastDay {
		return time.Time{}, false
	}
	return time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC), true
}

// fallbackDateLayouts 兜底布局：ISO 带时间的写法（上游 DateTime.TryParse 的
// 缩量版；纯日期形态已被上面三个正则覆盖）。
func fallbackDateLayouts(s string) (time.Time, bool) {
	layouts := []string{
		"2006-01-02T15:04:05", "2006-01-02 15:04:05",
		"2006/01/02 15:04", "2006-01-02 15:04",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil && t.Year() >= 1900 {
			return t, true
		}
	}
	return time.Time{}, false
}

// looseDateShaped 判断文本是否「日期形态」（含 8 位纯数字），与
// parseFlexibleDate 联用识别写成文本的日期。
func looseDateShaped(s string) bool {
	return textDateShapedRe.MatchString(tidyText(s))
}

// displayDateShaped 判断数字格的显示值是否日期形态（须带分隔符；见
// displayDateShapedRe 注）。
func displayDateShaped(s string) bool {
	return displayDateShapedRe.MatchString(tidyText(s))
}
