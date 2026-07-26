package stats

import (
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/wubigork/wubigork/internal/project"
)

// Summary 统计数据摘要
type Summary struct {
	TotalWords         int `json:"total_words"`
	ChapterCount       int `json:"chapter_count"`
	AvgWordsPerCh      int `json:"avg_words_per_chapter"`
	ForeshadowTotal    int `json:"foreshadow_total"`
	ForeshadowRevealed int `json:"foreshadow_revealed"`
	CharCount          int `json:"character_count"`
	CharAlive          int `json:"character_alive"`
}

// Collect 收集项目统计数据
func Collect(pm *project.Manager) *Summary {
	s := &Summary{}

	// 章节统计
	var totalWords int
	for i := 1; ; i++ {
		content, err := pm.ReadChapter(i)
		if err != nil {
			break
		}
		s.ChapterCount++
		totalWords += utf8.RuneCountInString(content)
	}
	s.TotalWords = totalWords
	if s.ChapterCount > 0 {
		s.AvgWordsPerCh = totalWords / s.ChapterCount
	}

	// 角色统计
	chars, err := pm.ReadCharacters()
	if err != nil {
		slog.Warn("stats: 读取角色失败", "error", err)
	}
	if chars != nil {
		s.CharCount = len(chars.Characters)
		for _, ch := range chars.Characters {
			if ch.Status == "Alive" || ch.Status == "" {
				s.CharAlive++
			}
		}
	}

	// 伏笔统计
	ff, err := pm.ReadForeshadows()
	if err != nil {
		slog.Warn("stats: 读取伏笔失败", "error", err)
	}
	if ff != nil {
		s.ForeshadowTotal = len(ff.Items)
		for _, f := range ff.Items {
			if f.Status == "revealed" {
				s.ForeshadowRevealed++
			}
		}
	}

	return s
}

// Report 生成可读的统计报告
func (s *Summary) Report() string {
	recoveryRate := 0.0
	if s.ForeshadowTotal > 0 {
		recoveryRate = float64(s.ForeshadowRevealed) / float64(s.ForeshadowTotal) * 100
	}

	return fmt.Sprintf(`📊 作品统计
══════════════════
总字数:      %d
章节数:      %d
平均每章:    %d 字
角色总数:    %d (存活: %d)
伏笔总数:    %d (已回收: %d, 回收率: %.0f%%)
`,
		s.TotalWords,
		s.ChapterCount,
		s.AvgWordsPerCh,
		s.CharCount, s.CharAlive,
		s.ForeshadowTotal, s.ForeshadowRevealed, recoveryRate,
	)
}

// BarChart 简单的 ASCII 柱状图 — 每章字数
func BarChart(pm *project.Manager) string {
	var sb strings.Builder
	sb.WriteString("📊 各章字数\n")
	maxWords := 0
	var words []int
	for i := 1; ; i++ {
		content, err := pm.ReadChapter(i)
		if err != nil {
			break
		}
		w := utf8.RuneCountInString(content)
		words = append(words, w)
		if w > maxWords {
			maxWords = w
		}
	}

	for i, w := range words {
		barLen := int(float64(w) / float64(maxWords) * 30)
		bar := strings.Repeat("█", barLen)
		sb.WriteString(fmt.Sprintf("  第%2d章 %s %d\n", i+1, bar, w))
	}
	return sb.String()
}
