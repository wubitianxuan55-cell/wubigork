package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/export"
	"github.com/gaea/gaea/internal/stats"
	"github.com/gaea/gaea/internal/style"
)

// ── 导出 2.0 API ────────────────────────────────────────────

// GetCompileTemplates 获取所有编译模板
func (a *App) GetCompileTemplates() []map[string]interface{} {
	templates := export.DefaultTemplates()
	var result []map[string]interface{}
	for _, t := range templates {
		result = append(result, map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"font_family": t.FontFamily,
			"font_size":   t.FontSize,
		})
	}
	return result
}

// ExportHTML 导出 HTML
func (a *App) ExportHTML(templateName string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	templates := export.DefaultTemplates()
	var tmpl export.CompileTemplate
	found := false
	for _, t := range templates {
		if t.Name == templateName {
			tmpl = t
			found = true
			break
		}
	}
	if !found {
		tmpl = templates[0] // 默认网文阅读
	}

	m := export.New(pm)
	html, err := m.ExportHTML("", tmpl)
	if err != nil {
		return nil, err
	}

	// 保存到文件
	outPath := filepath.Join(pm.Dir, pm.Meta.Title+"-"+sanitizeFilename(templateName)+".html")
	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"path": outPath,
		"html": html,
	}, nil
}

func sanitizeFilename(s string) string {
	result := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result += string(r)
		}
	}
	if result == "" {
		result = "export"
	}
	return result
}

// ── 仪表盘 API ──────────────────────────────────────────────

// GetDashboard 获取写作仪表盘数据
func (a *App) GetDashboard(dailyGoal int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	d, err := stats.GatherDashboard(pm, dailyGoal)
	if err != nil {
		return nil, err
	}

	var dailyStats []map[string]interface{}
	for _, ds := range d.DailyStats {
		dailyStats = append(dailyStats, map[string]interface{}{
			"date":       ds.Date,
			"word_count": ds.WordCount,
			"chapters":   ds.Chapters,
		})
	}

	var chapterWCs []map[string]interface{}
	for _, cw := range d.ChapterWordCounts {
		chapterWCs = append(chapterWCs, map[string]interface{}{
			"chapter_num": cw.ChapterNum,
			"title":       cw.Title,
			"word_count":  cw.WordCount,
		})
	}

	var achievements []map[string]interface{}
	for _, a := range d.Achievements {
		achievements = append(achievements, map[string]interface{}{
			"id":          a.ID,
			"name":        a.Name,
			"description": a.Description,
			"icon":        a.Icon,
			"unlocked":    a.Unlocked,
			"progress":    a.Progress,
			"target":      a.Target,
		})
	}

	return map[string]interface{}{
		"total_words":         d.TotalWords,
		"chapter_count":       d.ChapterCount,
		"avg_words":           d.AvgWordsPerChap,
		"character_count":     d.CharacterCount,
		"location_count":      d.LocationCount,
		"daily_stats":         dailyStats,
		"chapter_word_counts": chapterWCs,
		"daily_goal":          d.DailyGoal,
		"today_words":         d.TodayWords,
		"goal_progress":       d.GoalProgress,
		"streak_days":         d.StreakDays,
		"completed_days":      d.CompletedDays,
		"achievements":        achievements,
	}, nil
}

// ── 风格档案 API ────────────────────────────────────────────

// AnalyzeStyle 分析并保存风格档案
func (a *App) AnalyzeStyle() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	analyzer := style.NewAnalyzer(pm, a.client, a.cfg.Model)
	profile, err := analyzer.Analyze()
	if err != nil {
		return nil, err
	}

	if err := style.SaveProfile(pm.Dir, profile); err != nil {
		return nil, err
	}

	var traits []map[string]string
	for k, v := range profile.Traits {
		traits = append(traits, map[string]string{"key": k, "value": v})
	}

	return map[string]interface{}{
		"name":         profile.Name,
		"description":  profile.Description,
		"traits":       traits,
		"raw_markdown": profile.RawMarkdown,
	}, nil
}

// GetStyleProfile 获取当前风格档案
func (a *App) GetStyleProfile() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	profile, err := style.LoadProfile(pm.Dir)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name":         profile.Name,
		"description":  profile.Description,
		"raw_markdown": profile.RawMarkdown,
	}, nil
}

// ImportStyleProfile 导入风格档案
func (a *App) ImportStyleProfile(markdownContent string, profileName string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	profile := style.ImportProfileFromMarkdown(markdownContent)
	profile.Name = profileName

	if err := style.SaveProfile(pm.Dir, profile); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name":   profile.Name,
		"status": "imported",
	}, nil
}
