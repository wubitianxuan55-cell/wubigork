package stats

import (
	"sort"
	"time"

	"github.com/wubigork/wubigork/internal/project"
)

// ── 写作仪表盘 ───────────────────────────────────────────────

// DashboardData 仪表盘聚合数据
type DashboardData struct {
	// 总览
	TotalWords      int     `json:"total_words"`
	ChapterCount    int     `json:"chapter_count"`
	AvgWordsPerChap int     `json:"avg_words_per_chapter"`
	CharacterCount  int     `json:"character_count"`
	LocationCount   int     `json:"location_count"`
	CompletedDays   int     `json:"completed_days"` // 有写作记录的天数

	// 趋势
	DailyStats      []DailyStat  `json:"daily_stats"`
	ChapterWordCounts []ChapterWordCount `json:"chapter_word_counts"`

	// 目标
	DailyGoal       int     `json:"daily_goal"`       // 日目标字数
	TodayWords      int     `json:"today_words"`      // 今日字数
	GoalProgress    float64 `json:"goal_progress"`    // 目标进度百分比
	StreakDays      int     `json:"streak_days"`      // 连续写作天数

	// 成就
	Achievements    []Achievement `json:"achievements"`
}

// DailyStat 每日统计
type DailyStat struct {
	Date      string `json:"date"`      // YYYY-MM-DD
	WordCount int    `json:"word_count"`
	Chapters  int    `json:"chapters"`
}

// ChapterWordCount 章节字数
type ChapterWordCount struct {
	ChapterNum int    `json:"chapter_num"`
	Title      string `json:"title"`
	WordCount  int    `json:"word_count"`
}

// Achievement 写作成就
type Achievement struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Unlocked    bool   `json:"unlocked"`
	Progress    int    `json:"progress"`
	Target      int    `json:"target"`
}

// GatherDashboard 收集仪表盘数据
func GatherDashboard(pm *project.Manager, dailyGoal int) (*DashboardData, error) {
	d := &DashboardData{DailyGoal: dailyGoal}
	if dailyGoal <= 0 {
		d.DailyGoal = 2000
	}

	// 章节统计
	var chapterWCs []ChapterWordCount
	for chapterNum := 1; ; chapterNum++ {
		content, err := pm.ReadChapter(chapterNum)
		if err != nil {
			break
		}
		summary, _ := pm.ReadChapterSummary(chapterNum)

		wc := len([]rune(content))
		d.TotalWords += wc

		title := ""
		if summary != nil {
			title = summary.Title
		}

		chapterWCs = append(chapterWCs, ChapterWordCount{
			ChapterNum: chapterNum,
			Title:      title,
			WordCount:  wc,
		})
	}

	d.ChapterCount = len(chapterWCs)
	d.ChapterWordCounts = chapterWCs

	if d.ChapterCount > 0 {
		d.AvgWordsPerChap = d.TotalWords / d.ChapterCount
	}

	// 角色统计
	chars, err := pm.ReadCharacters()
	if err == nil && chars != nil {
		d.CharacterCount = len(chars.Characters)
	}

	// Lorebook 地点统计
	lorebook, err := pm.ReadLorebook()
	if err == nil && lorebook != nil {
		for _, e := range lorebook.Entries {
			if e.Category == "location" {
				d.LocationCount++
			}
		}
	}

	// 每日统计（简化：基于章节摘要时间 — 实际需文件修改时间）
	// 当前无文件修改时间追踪，基于章节数估算
	today := time.Now().Format("2006-01-02")
	for i := 0; i < d.ChapterCount && i < 30; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		wc := 0
		if i < len(chapterWCs) {
			wc = chapterWCs[len(chapterWCs)-1-i].WordCount
		}
		d.DailyStats = append(d.DailyStats, DailyStat{
			Date:      date,
			WordCount: wc,
			Chapters:  min(1, len(chapterWCs)-i),
		})
	}
	sort.Slice(d.DailyStats, func(i, j int) bool {
		return d.DailyStats[i].Date < d.DailyStats[j].Date
	})

	// 今日字数（最近一章的字数作为估计）
	if d.ChapterCount > 0 {
		d.TodayWords = chapterWCs[d.ChapterCount-1].WordCount
	}
	d.GoalProgress = float64(d.TodayWords) / float64(d.DailyGoal) * 100
	if d.GoalProgress > 100 {
		d.GoalProgress = 100
	}

	// 连续天数（简化）
	d.StreakDays = min(d.ChapterCount, 30)
	if d.ChapterCount > 0 {
		d.CompletedDays = d.ChapterCount
	}

	// 成就系统
	d.Achievements = computeAchievements(d)

	_ = today
	return d, nil
}

func computeAchievements(d *DashboardData) []Achievement {
	return []Achievement{
		{
			ID: "first_chapter", Name: "第一章", Description: "完成第一章创作",
			Icon: "✍️", Unlocked: d.ChapterCount >= 1,
			Progress: min(d.ChapterCount, 1), Target: 1,
		},
		{
			ID: "five_chapters", Name: "五章达成", Description: "完成五章创作",
			Icon: "📖", Unlocked: d.ChapterCount >= 5,
			Progress: min(d.ChapterCount, 5), Target: 5,
		},
		{
			ID: "ten_k_words", Name: "万言书", Description: "累计创作 10,000 字",
			Icon: "🏆", Unlocked: d.TotalWords >= 10000,
			Progress: min(d.TotalWords, 10000), Target: 10000,
		},
		{
			ID: "fifty_k_words", Name: "中篇达成", Description: "累计创作 50,000 字",
			Icon: "🌟", Unlocked: d.TotalWords >= 50000,
			Progress: min(d.TotalWords, 50000), Target: 50000,
		},
		{
			ID: "hundred_k_words", Name: "长篇巨著", Description: "累计创作 100,000 字",
			Icon: "👑", Unlocked: d.TotalWords >= 100000,
			Progress: min(d.TotalWords, 100000), Target: 100000,
		},
		{
			ID: "three_chars", Name: "群像剧", Description: "创建 3 个以上角色",
			Icon: "👥", Unlocked: d.CharacterCount >= 3,
			Progress: min(d.CharacterCount, 3), Target: 3,
		},
		{
			ID: "daily_goal", Name: "日拱一卒", Description: "单日达到字数目标",
			Icon: "🎯", Unlocked: d.TodayWords >= d.DailyGoal,
			Progress: min(d.TodayWords, d.DailyGoal), Target: d.DailyGoal,
		},
		{
			ID: "seven_day_streak", Name: "周更达人", Description: "连续 7 天创作",
			Icon: "🔥", Unlocked: d.StreakDays >= 7,
			Progress: min(d.StreakDays, 7), Target: 7,
		},
	}
}
