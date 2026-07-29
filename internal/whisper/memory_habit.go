// Package whisper — memory_habit.go
// 100% 对齐 ackem memory/habitsStore.ts
// 用户习惯槽：短时/长时习惯、时间槽匹配、自动升级降级

package whisper

import (
	"sort"
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
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune(n%10+'0')) + s
		n /= 10
	}
	return s
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
		// 时间段匹配
		if h.HourStart <= h.HourEnd {
			if hour < h.HourStart || hour >= h.HourEnd {
				continue
			}
		} else {
			// 跨天时间段 (如 22-6)
			if hour < h.HourStart && hour >= h.HourEnd {
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

// DecayAndCleanup 衰减过期习惯
func (hs *HabitsStore) DecayAndCleanup(now time.Time) {
	nowMs := now.UnixMilli()
	weekMs := int64(7 * 24 * 3600 * 1000)
	var kept []*UserHabit
	for _, h := range hs.habits {
		// 删除已过期
		if h.ExpiresAt != nil && nowMs > *h.ExpiresAt {
			continue
		}
		// 只衰减短期
		if h.Scope == "short_term" {
			lastAct := h.LastConfirmedAt
			if lastAct == 0 {
				lastAct = h.CreatedAt
			}
			weeksSince := float64(nowMs-lastAct) / float64(weekMs)
			if weeksSince > 0 {
				h.Confidence = clampF(h.Confidence-weeksSince*DecayWeeklyRate, 0, 1)
			}
			if h.Confidence <= DecaySleepThreshold {
				continue
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
