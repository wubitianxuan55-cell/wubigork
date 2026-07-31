package stats

import (
	"log/slog"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/project"
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
