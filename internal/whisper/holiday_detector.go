// Package whisper — holiday_detector.go
// 100% 对齐 ackem temporalAwareness/holidayDetector.ts
// 节假日检测：公历固定 + 农历预计算(2026-2030) + 公历浮动

package whisper

import "time"

// ─── 公历固定节日 ────────────────────────────────────────────

var staticHolidays = map[string]string{
	"01-01": "元旦", "02-14": "情人节", "03-08": "国际妇女节",
	"04-01": "愚人节", "05-01": "劳动节", "05-20": "520", "05-21": "521",
	"06-01": "儿童节", "10-01": "国庆节", "11-11": "光棍节",
	"12-24": "平安夜", "12-25": "圣诞节", "12-31": "跨年夜",
}

// ─── 农历节日预计算 (2026-2030) ──────────────────────────────

var lunarHolidays = map[string]map[string]string{
	"2026": {"02-17": "春节", "02-22": "元宵节", "05-31": "端午节", "08-19": "七夕", "10-04": "中秋节"},
	"2027": {"02-06": "春节", "02-11": "元宵节", "05-20": "端午节", "08-08": "七夕", "09-23": "中秋节"},
	"2028": {"01-26": "春节", "01-31": "元宵节", "05-08": "端午节", "07-28": "七夕", "09-11": "中秋节"},
	"2029": {"02-13": "春节", "02-18": "元宵节", "05-28": "端午节", "08-17": "七夕", "10-01": "中秋节"},
	"2030": {"02-03": "春节", "02-08": "元宵节", "05-17": "端午节", "08-06": "七夕", "09-19": "中秋节"},
}

// ─── HolidayInfo ─────────────────────────────────────────────

type HolidayCategory string

const (
	HolidayTraditional HolidayCategory = "traditional"
	HolidayWestern     HolidayCategory = "western"
	HolidaySocial      HolidayCategory = "social"
	HolidayFamily      HolidayCategory = "family"
)

type HolidayInfo struct {
	Key      string
	Category HolidayCategory
}

// ─── DetectHoliday ───────────────────────────────────────────

func DetectHoliday(today time.Time) *HolidayInfo {
	mmdd := today.Format("01-02")
	year := today.Format("2006")

	// ① 公历固定
	if name, ok := staticHolidays[mmdd]; ok {
		return &HolidayInfo{Key: name, Category: categorizeHoliday(name)}
	}

	// ② 公历浮动（母亲节/父亲节）
	if name := getFloatingHoliday(today.Year(), int(today.Month()), today.Day()); name != "" {
		return &HolidayInfo{Key: name, Category: HolidayFamily}
	}

	// ③ 农历
	if yearHolidays, ok := lunarHolidays[year]; ok {
		if name, ok := yearHolidays[mmdd]; ok {
			return &HolidayInfo{Key: name, Category: HolidayTraditional}
		}
	}

	return nil
}

func categorizeHoliday(name string) HolidayCategory {
	switch name {
	case "元旦", "国庆节":
		return HolidayTraditional
	case "情人节", "圣诞节", "平安夜", "跨年夜":
		return HolidayWestern
	case "520", "521", "光棍节":
		return HolidaySocial
	case "母亲节", "父亲节", "儿童节", "国际妇女节":
		return HolidayFamily
	}
	return HolidayTraditional
}

// getFloatingHoliday 公历浮动节日：母亲节(5月第2个周日) 父亲节(6月第3个周日)
func getFloatingHoliday(year, month, day int) string {
	if month == 5 && day == nthSundayOfMonth(year, 5, 2) {
		return "母亲节"
	}
	if month == 6 && day == nthSundayOfMonth(year, 6, 3) {
		return "父亲节"
	}
	return ""
}

func nthSundayOfMonth(year, month, n int) int {
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	dayOfWeek := int(first.Weekday())
	return 1 + 7*(n-1) + (7-dayOfWeek)%7
}
