// Package whisper — memory_habit.go
// 100% 对齐 ackem memory/habitsStore.ts
// 用户习惯槽：短时/长时习惯、时间槽匹配、自动升级降级

package whisper

import (
	"sort"
	"strconv"
	"time"
)

// ─── HabitsStore ──────────────────────────────────────────────

// HabitsStore 用户习惯存储
type HabitsStore struct {
	habits []*UserHabit
}

// NewHabitsStore 创建空习惯库
func NewHabitsStore() *HabitsStore {
	return &HabitsStore{}
}

// upsertKey 生成四元组唯一键
func (h *UserHabit) upsertKey() string {
	wk := "nil"
	if h.Weekday != nil {
		wk = string(rune(*h.Weekday + '0'))
	}
	return h.Type + "|" + wk + "|" + itoa(h.HourStart) + "|" + itoa(h.HourEnd)
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

// Upsert 写入或更新习惯（对齐 upsertHabit）
func (hs *HabitsStore) Upsert(h UserHabit) {
	key := h.upsertKey()
	for i, existing := range hs.habits {
		if existing.upsertKey() == key {
			// 更新现有习惯
			h.ID = existing.ID
			h.CreatedAt = existing.CreatedAt
			if h.UpdatedAt == 0 {
				h.UpdatedAt = time.Now().UnixMilli()
			}
			hs.habits[i] = &h
			return
		}
	}
	// 新习惯
	if h.ID == "" {
		h.ID = genHexID()
	}
	nowMs := time.Now().UnixMilli()
	if h.CreatedAt == 0 {
		h.CreatedAt = nowMs
	}
	if h.UpdatedAt == 0 {
		h.UpdatedAt = nowMs
	}
	hs.habits = append(hs.habits, &h)
}

// MatchHabits 匹配当前时间命中的所有习惯
func (hs *HabitsStore) MatchHabits(now time.Time) []*UserHabit {
	weekday := int(now.Weekday())
	hour := now.Hour()
	nowMs := now.UnixMilli()

	var matched []*UserHabit
	for _, h := range hs.habits {
		// 过期检查
		if h.ExpiresAt != nil && nowMs > *h.ExpiresAt {
			continue
		}
		// weekday 匹配
		if h.Weekday != nil && *h.Weekday != weekday {
			continue
		}
		// 时间段匹配（闭区间，对齐 ackem）
		if h.HourStart <= h.HourEnd {
			if hour < h.HourStart || hour > h.HourEnd {
				continue
			}
		} else {
			// 跨天时间段 (如 22-6)，闭区间：包含6点
			if hour < h.HourStart && hour > h.HourEnd {
				continue
			}
		}
		matched = append(matched, h)
	}

	// 排序：long_term 优先，confidence 降序
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].Scope == "long_term" && matched[j].Scope != "long_term" {
			return true
		}
		if matched[j].Scope == "long_term" && matched[i].Scope != "long_term" {
			return false
		}
		return matched[i].Confidence > matched[j].Confidence
	})

	// 截断短期习惯
	var result []*UserHabit
	shortCount := 0
	for _, h := range matched {
		if h.Scope == "short_term" {
			shortCount++
			if shortCount > MaxShortTermPerDay {
				continue
			}
		}
		result = append(result, h)
	}
	return result
}

// UpgradeToLongTerm 短期→长期升级
func (hs *HabitsStore) UpgradeToLongTerm(id string) {
	for _, h := range hs.habits {
		if h.ID == id {
			h.Scope = "long_term"
			if h.Confidence < LongTermBaseConfidence {
				h.Confidence = LongTermBaseConfidence
			} else {
				h.Confidence = clampF(h.Confidence+LongTermConfidenceIncrement, 0, LongTermConfidenceCap)
			}
			h.UpdatedAt = time.Now().UnixMilli()
			return
		}
	}
}

// DecayAndCleanup 衰减长时习惯 + 清理过期（对齐 ackem habitsStore）
// P0修复: 原实现错误地衰减 short_term 习惯并直接删除；
// ackem 的正确逻辑是衰减 long_term 习惯(久未确认则降级为 short_term)
func (hs *HabitsStore) DecayAndCleanup(now time.Time) {
	nowMs := now.UnixMilli()
	weekMs := int64(7 * 24 * 3600 * 1000)
	var kept []*UserHabit
	for _, h := range hs.habits {
		// 删除已过期的短时习惯
		if h.ExpiresAt != nil && nowMs > *h.ExpiresAt {
			continue
		}
		// 只衰减长时习惯（对齐 ackem: 4周宽限期后开始衰减）
		if h.Scope == "long_term" {
			lastAct := h.LastConfirmedAt
			if lastAct == 0 {
				lastAct = h.CreatedAt
			}
			weeksSince := float64(nowMs-lastAct) / float64(weekMs)
			if weeksSince > DecayWeeksThreshold {
				// 超过宽限期后开始衰减：ackem 公式 (weeksSince - 4 + 1) * 0.1
				h.Confidence = clampF(h.Confidence-(weeksSince-DecayWeeksThreshold+1)*DecayWeeklyRate, 0, 1)
			}
			if h.Confidence <= DecaySleepThreshold {
				// 降级为短期习惯而非删除（对齐 ackem）
				h.Scope = "short_term"
				h.Confidence = 1.0 // 短期习惯初始confidence
			}
		}
		kept = append(kept, h)
	}
	hs.habits = kept
}

// Delete 删除习惯
func (hs *HabitsStore) Delete(id string) {
	for i, h := range hs.habits {
		if h.ID == id {
			hs.habits = append(hs.habits[:i], hs.habits[i+1:]...)
			return
		}
	}
}

// All 返回所有习惯
func (hs *HabitsStore) All() []*UserHabit {
	result := make([]*UserHabit, len(hs.habits))
	copy(result, hs.habits)
	return result
}

// Count 习惯总数
func (hs *HabitsStore) Count() int {
	return len(hs.habits)
}
