// Package whisper — procedural_habits.go
// 100% 对齐 ackem memory/proceduralHabits.ts
// 程序化习惯：持久化记录+计数+已建立判定

package whisper

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// HabitLine 程序化习惯行
type HabitLine struct {
	TS   string `json:"ts"`
	Text string `json:"text"`
}

// ProceduralHabitStore 程序化习惯存储
type ProceduralHabitStore struct {
	mu    sync.RWMutex
	lines []HabitLine
}

// NewProceduralHabitStore 创建程序化习惯存储
func NewProceduralHabitStore() *ProceduralHabitStore {
	return &ProceduralHabitStore{}
}

// AppendHabitLine 追加一条习惯记录
func (s *ProceduralHabitStore) AppendHabitLine(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	s.lines = append(s.lines, HabitLine{
		TS:   time.Now().Format(time.RFC3339),
		Text: trimmed,
	})
}

// ReadHabitLines 读取所有习惯行
func (s *ProceduralHabitStore) ReadHabitLines() []HabitLine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]HabitLine, len(s.lines))
	copy(result, s.lines)
	return result
}

// countHabitOccurrences 统计习惯出现次数
func (s *ProceduralHabitStore) countHabitOccurrences(text string) int {
	key := normalizeHabitKey(text)
	if key == "" {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, l := range s.lines {
		if normalizeHabitKey(l.Text) == key {
			count++
		}
	}
	return count
}

// IsEstablishedHabit 判断是否为已建立的习惯（出现次数≥门槛）
func (s *ProceduralHabitStore) IsEstablishedHabit(text string, minCount int) bool {
	if minCount <= 0 {
		minCount = HabitMinOccurrences
	}
	return s.countHabitOccurrences(text) >= minCount
}

// ListEstablishedHabits 列出所有已建立的习惯
func (s *ProceduralHabitStore) ListEstablishedHabits(minCount int) []string {
	if minCount <= 0 {
		minCount = HabitMinOccurrences
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[string]struct {
		count int
		text  string
	})
	for _, l := range s.lines {
		key := normalizeHabitKey(l.Text)
		if key == "" {
			continue
		}
		prev := counts[key]
		prev.count++
		prev.text = l.Text
		counts[key] = prev
	}

	var out []string
	for _, v := range counts {
		if v.count >= minCount {
			out = append(out, v.text)
		}
	}
	return out
}

// normalizeHabitKey 规范化习惯键
func normalizeHabitKey(text string) string {
	text = strings.TrimSpace(text)
	// 移除标点
	re := regexp.MustCompile(`[。.；;！!？?]`)
	text = re.ReplaceAllString(text, "")
	// 移除空白
	text = strings.ReplaceAll(text, " ", "")
	text = strings.ReplaceAll(text, "\t", "")
	text = strings.ToLower(text)
	runes := []rune(text)
	if len(runes) > HabitKeyMaxLen {
		text = string(runes[:HabitKeyMaxLen])
	}
	return text
}
