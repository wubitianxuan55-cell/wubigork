// Package whisper — age_computer.go
// 100% 对齐 ackem memory/ageComputer.ts
// 年龄动态计算：从 ageMeta 反推出生年份 / 动态计算当前年龄

package whisper

import (
	"fmt"
	"strings"
	"time"
)

// InferBirthYear 从 ageMeta 反推出生年份
func InferBirthYear(age int, birthdayMMDD, recordedAt string) int {
	recorded, err := time.Parse(time.RFC3339, recordedAt)
	if err != nil {
		recorded = time.Now()
	}
	recordedYear := recorded.Year()
	if birthdayMMDD == "" {
		return recordedYear - age
	}

	birthday, err := time.Parse("2006-01-02", fmt.Sprintf("%d-%s", recordedYear, birthdayMMDD))
	if err != nil {
		return recordedYear - age
	}

	if recorded.After(birthday) || recorded.Equal(birthday) {
		return recordedYear - age
	}
	return recordedYear - age - 1
}

// ComputeCurrentAge 动态计算当前年龄
func ComputeCurrentAge(age int, birthYear int, birthdayMMDD string, isEstimate bool, recordedAt string) int {
	now := time.Now()

	if birthYear > 0 && birthdayMMDD != "" {
		birthday, err := time.Parse("2006-01-02", fmt.Sprintf("%d-%s", now.Year(), birthdayMMDD))
		if err != nil {
			// 闰年保护
			birthday, _ = time.Parse("2006-01-02", fmt.Sprintf("%d-03-01", now.Year()))
		}
		hasPassed := now.After(birthday) || now.Equal(birthday)
		if hasPassed {
			return now.Year() - birthYear
		}
		return now.Year() - birthYear - 1
	}

	if recordedAt != "" {
		recorded, err := time.Parse(time.RFC3339, recordedAt)
		if err == nil {
			return age + (now.Year() - recorded.Year())
		}
	}

	return age
}

// BuildAgeLine 年龄的自然语言呈现
func BuildAgeLine(fs *FactStore) string {
	var best *Fact
	for _, f := range fs.ListActive() {
		if f.Subcategory == "BASIC_PROFILE" && (f.Subject == "用户" || strings.HasPrefix(f.Subject, "用户")) {
			best = f
			break
		}
	}
	if best == nil || best.AgeMeta == nil || best.AgeMeta.Age <= 0 {
		return ""
	}

	meta := best.AgeMeta
	currentAge := ComputeCurrentAge(meta.Age, meta.BirthYear, meta.BirthdayMMDD, meta.IsEstimate, meta.RecordedAt)

	if meta.BirthYear > 0 && meta.BirthdayMMDD != "" {
		bd := strings.Replace(meta.BirthdayMMDD, "-", "月", 1) + "日"
		return fmt.Sprintf("ta %d 年出生，%s（今年 %d 岁）。", meta.BirthYear, bd, currentAge)
	}
	if meta.IsEstimate {
		return fmt.Sprintf("ta 大约 %d 岁（从对话中推算，可能不太精确）。", currentAge)
	}
	return fmt.Sprintf("ta %d 岁。", currentAge)
}
