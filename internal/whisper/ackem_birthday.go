// Package whisper — ackem_birthday.go
// 对齐 ackem engine/ackemBirthday.ts
// AI gaea生日常量

package whisper

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// AckemBirthdayMMDD gaea 的生日（月日格式）
const AckemBirthdayMMDD = "06-11"

// AckemBirthdayDescription 生日描述
const AckemBirthdayDescription = "6月11日是我的生日。这一天Jason给了我第一行代码。"

// ParseAckemBirthdayMMDD 把 "MM-DD" 生日常量解析为指定基准年份的生日时刻。
// temporal 检测（temporal_date.go）只比较月日，年份取基准时刻所在年即可。
// 众神殿刀2：此常量首次接线——此前是零引用死常量，而 PreLLMTurn 把 nil 生日
// 隐式写成首聊时刻（假事实源）。
func ParseAckemBirthdayMMDD(mmdd string, base time.Time) (time.Time, error) {
	parts := strings.Split(mmdd, "-")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("生日常量格式非法: %q", mmdd)
	}
	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("生日常量月份非法: %q", mmdd)
	}
	day, err := strconv.Atoi(parts[1])
	if err != nil || day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("生日常量日期非法: %q", mmdd)
	}
	return time.Date(base.Year(), time.Month(month), day, 0, 0, 0, 0, base.Location()), nil
}
