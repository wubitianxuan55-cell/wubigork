package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// ── 写作仪表盘 ───────────────────────────────────────────────

// DashboardData 仪表盘聚合数据
type DashboardData struct {
	// 总览
	TotalWords      int `json:"total_words"`
	ChapterCount    int `json:"chapter_count"`
	AvgWordsPerChap int `json:"avg_words_per_chapter"`
	CharacterCount  int `json:"character_count"`
	LocationCount   int `json:"location_count"`
	CompletedDays   int `json:"completed_days"` // 有写作记录的天数

	// 趋势
	DailyStats        []DailyStat        `json:"daily_stats"`
	ChapterWordCounts []ChapterWordCount `json:"chapter_word_counts"`

	// 目标
	DailyGoal    int     `json:"daily_goal"`    // 日目标字数
	TodayWords   int     `json:"today_words"`   // 今日字数
	GoalProgress float64 `json:"goal_progress"` // 目标进度百分比
	StreakDays   int     `json:"streak_days"`   // 连续写作天数

	// 成就
	Achievements []Achievement `json:"achievements"`
}

// DailyStat 每日统计
type DailyStat struct {
	Date      string `json:"date"` // YYYY-MM-DD
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

// chapterFileRe 匹配主线/分支章节文件名 NNN.md / NNNx.md（与 stats.Collect 一致）。
var chapterFileRe = regexp.MustCompile(`^([0-9]{3})([a-z]?)\.md$`)

// chapterFile 是目录扫描枚举到的章节文件信息。
type chapterFile struct {
	num     int
	branch  string
	content string
	modTime time.Time
}

// listChapterFiles 单次 ReadDir 枚举章节目录下的全部章节文件（章节号、正文、
// 文件 mtime），替代逐个文件探测：章节断档（中间缺章）不会中断统计。
// 读取失败的文件跳过；无 chapters 目录返回空列表。
func listChapterFiles(pm *project.Manager) []chapterFile {
	entries, err := os.ReadDir(filepath.Join(pm.Dir, "chapters"))
	if err != nil {
		return nil
	}
	var out []chapterFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := chapterFileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		num, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join(pm.Dir, "chapters", e.Name()))
		if err != nil {
			continue
		}
		out = append(out, chapterFile{num: num, branch: m[2], content: string(data), modTime: info.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].num != out[j].num {
			return out[i].num < out[j].num
		}
		return out[i].branch < out[j].branch
	})
	return out
}

// chapterTitle 读取章节对应的摘要文件标题（NNN{branch}-summary.json）；
// 无摘要返回空串（与旧逻辑一致）。
func chapterTitle(pm *project.Manager, num int, branch string) string {
	data, err := os.ReadFile(filepath.Join(pm.Dir, "chapters",
		fmt.Sprintf("%03d%s-summary.json", num, branch)))
	if err != nil {
		return ""
	}
	var s types.ChapterSummary
	if json.Unmarshal(data, &s) != nil {
		return ""
	}
	return s.Title
}

// streakDays 计算连续写作天数：从今天（今天未写则昨天）向前数连续有记录的
// 天数。基于真实 mtime 聚合结果，非虚构估计。
func streakDays(dayWords map[string]int) int {
	if len(dayWords) == 0 {
		return 0
	}
	day := time.Now()
	if _, ok := dayWords[day.Format("2006-01-02")]; !ok {
		day = day.AddDate(0, 0, -1)
	}
	streak := 0
	for {
		if _, ok := dayWords[day.Format("2006-01-02")]; !ok {
			break
		}
		streak++
		day = day.AddDate(0, 0, -1)
	}
	return streak
}

// GatherDashboard 收集仪表盘数据（T7-3「名实相符」）：
//   - 章节统计：单次目录扫描枚举章节文件（断档不中断），字数按真实正文统计；
//   - 每日统计：按章节文件 mtime 真实聚合（日期 = 文件修改日，非虚构日期），
//     今日字数 = 今日 mtime 文件的真实字数；
//   - 连续天数/有写作天数：由 mtime 聚合结果计算，全部为真实数据，成就系统
//     不再基于虚构字段。
func GatherDashboard(pm *project.Manager, dailyGoal int) (*DashboardData, error) {
	d := &DashboardData{DailyGoal: dailyGoal}
	if dailyGoal <= 0 {
		d.DailyGoal = 2000
	}

	// 章节统计：单次目录扫描（含断档容错），mtime 用于每日聚合。
	chapters := listChapterFiles(pm)

	dayWords := make(map[string]int)    // 日期 → 字数
	dayChapters := make(map[string]int) // 日期 → 章节数
	chapterWCs := make([]ChapterWordCount, 0, len(chapters))
	for _, ch := range chapters {
		wc := utf8.RuneCountInString(ch.content)
		d.TotalWords += wc
		chapterWCs = append(chapterWCs, ChapterWordCount{
			ChapterNum: ch.num,
			Title:      chapterTitle(pm, ch.num, ch.branch),
			WordCount:  wc,
		})
		date := ch.modTime.Format("2006-01-02")
		dayWords[date] += wc
		dayChapters[date]++
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

	// 每日统计：按文件 mtime 真实聚合（日期升序；无记录则空）。
	dates := make([]string, 0, len(dayWords))
	for date := range dayWords {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	for _, date := range dates {
		d.DailyStats = append(d.DailyStats, DailyStat{
			Date:      date,
			WordCount: dayWords[date],
			Chapters:  dayChapters[date],
		})
	}

	// 今日字数：今日 mtime 章节的真实字数（未写则 0），不再用最近一章冒充。
	todayKey := time.Now().Format("2006-01-02")
	d.TodayWords = dayWords[todayKey]
	d.GoalProgress = float64(d.TodayWords) / float64(d.DailyGoal) * 100
	if d.GoalProgress > 100 {
		d.GoalProgress = 100
	}

	// 有写作记录的天数与连续写作天数：真实聚合（不再用章节数冒充）。
	d.CompletedDays = len(dayWords)
	d.StreakDays = streakDays(dayWords)

	// 成就系统：全部基于真实聚合字段，无需「估算」标注。
	d.Achievements = computeAchievements(d)
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
