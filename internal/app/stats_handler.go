package app

import (
	"path/filepath"

	"github.com/gaea/gaea/internal/skill"
	"github.com/gaea/gaea/internal/stats"
	"github.com/gaea/gaea/internal/util"
)

// ── Skill ────────────────────────────────────────────────────

// ListSkills 列出所有 Skill
func (a *App) ListSkills() []map[string]interface{} {
	if a.skillLoader == nil {
		a.skillLoader = skill.NewLoader(filepath.Join(a.cfg.ResourceDir, "skills"))
	}
	skills := a.skillLoader.List()
	result := make([]map[string]interface{}, len(skills))
	for i, s := range skills {
		result[i] = map[string]interface{}{
			"name":        s.Name,
			"description": s.Description,
			"appliesTo":   s.AppliesTo,
			"version":     s.Version,
		}
	}
	return result
}

// ── 统计 ────────────────────────────────────────────────────

// GetStats 获取统计摘要
func (a *App) GetStats() map[string]interface{} {
	pm := a.getPM()
	if pm == nil {
		return nil
	}
	s := stats.Collect(pm)
	return map[string]interface{}{
		"totalWords":         s.TotalWords,
		"chapterCount":       s.ChapterCount,
		"avgWordsPerChapter": s.AvgWordsPerCh,
		"characterCount":     s.CharCount,
		"charAlive":          s.CharAlive,
		"foreshadowTotal":    s.ForeshadowTotal,
		"foreshadowRevealed": s.ForeshadowRevealed,
		"foreshadowRate":     float64(s.ForeshadowRevealed) / float64(util.Max(s.ForeshadowTotal, 1)) * 100,
	}
}

// ── 配置 ──────────────────────────────────────────────────────

// GetConfig 返回当前配置
func (a *App) GetConfig() map[string]string {
	return map[string]string{
		"model":     a.cfg.Model,
		"baseURL":   a.cfg.XaiAPIBaseURL,
		"tokenPath": a.cfg.TokenStorePath,
	}
}
