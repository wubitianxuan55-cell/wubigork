package stats

// dashboard_test.go — T7-3：DailyStats 按章节文件 mtime 真实聚合（伪造 mtime），
// 章节断档（中间缺章）不中断统计；连续天数/有写作天数基于真实聚合。

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/project"
)

// newProjectWithChapters 建一个临时项目：chapters 目录 + 指定章节文件。
func newProjectWithChapters(t *testing.T, files map[string]string) *project.Manager {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0o755); err != nil {
		t.Fatal(err)
	}
	pm := &project.Manager{Dir: dir}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, "chapters", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pm
}

// forgeMtime 伪造章节文件 mtime（按本地日期）。
func forgeMtime(t *testing.T, pm *project.Manager, name string, daysAgo int) {
	t.Helper()
	when := time.Now().AddDate(0, 0, -daysAgo)
	path := filepath.Join(pm.Dir, "chapters", name)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

// TestGatherDashboardDailyStatsByMtime DailyStats 按文件 mtime 真实聚合：
// 3 章分别伪造为 3 天前/2 天前/今天，日期与字数一一对应；TodayWords 为今天的
// 真实字数；CompletedDays=3；StreakDays=1（昨天无写作，今天不连续）。
func TestGatherDashboardDailyStatsByMtime(t *testing.T) {
	pm := newProjectWithChapters(t, map[string]string{
		"001.md": "第一章正文一二三四五",
		"002.md": "第二章正文一二三四五六",
		"003.md": "第三章正文一二三四五六七",
	})
	forgeMtime(t, pm, "001.md", 3)
	forgeMtime(t, pm, "002.md", 2)
	forgeMtime(t, pm, "003.md", 0)

	d, err := GatherDashboard(pm, 2000)
	if err != nil {
		t.Fatalf("GatherDashboard: %v", err)
	}
	if d.ChapterCount != 3 {
		t.Fatalf("ChapterCount = %d", d.ChapterCount)
	}
	if len(d.DailyStats) != 3 {
		t.Fatalf("DailyStats 应为 3 天（真实 mtime 聚合，非虚构 30 天）: %+v", d.DailyStats)
	}
	// 日期集合应为 3 天前/2 天前/今天
	wantDates := map[string]bool{}
	for _, off := range []int{3, 2, 0} {
		wantDates[time.Now().AddDate(0, 0, -off).Format("2006-01-02")] = true
	}
	gotDates := map[string]bool{}
	for _, ds := range d.DailyStats {
		gotDates[ds.Date] = true
		if ds.WordCount == 0 || ds.Chapters != 1 {
			t.Fatalf("DailyStat 异常: %+v", ds)
		}
	}
	for date := range wantDates {
		if !gotDates[date] {
			t.Fatalf("缺少日期 %s（mtime 聚合）: %+v", date, d.DailyStats)
		}
	}
	// 今日字数 = 第三章（今天的 mtime）的真实字数
	wantToday := len([]rune("第三章正文一二三四五六七"))
	if d.TodayWords != wantToday {
		t.Fatalf("TodayWords = %d, want %d", d.TodayWords, wantToday)
	}
	if d.CompletedDays != 3 {
		t.Fatalf("CompletedDays = %d, want 3", d.CompletedDays)
	}
	if d.StreakDays != 1 {
		t.Fatalf("StreakDays = %d, want 1（仅今天连续）", d.StreakDays)
	}
}

// TestGatherDashboardChapterGap 章节断档（001、003，缺 002）不中断统计：
// 章节数与总字数都正确。
func TestGatherDashboardChapterGap(t *testing.T) {
	pm := newProjectWithChapters(t, map[string]string{
		"001.md": "第一章",
		"003.md": "第三章内容",
	})

	d, err := GatherDashboard(pm, 2000)
	if err != nil {
		t.Fatalf("GatherDashboard: %v", err)
	}
	if d.ChapterCount != 2 {
		t.Fatalf("断档项目应统计到 2 章，实际 %d", d.ChapterCount)
	}
	want := len([]rune("第一章")) + len([]rune("第三章内容"))
	if d.TotalWords != want {
		t.Fatalf("TotalWords = %d, want %d", d.TotalWords, want)
	}
	if len(d.ChapterWordCounts) != 2 {
		t.Fatalf("ChapterWordCounts 应为 2: %+v", d.ChapterWordCounts)
	}
}

// TestGatherDashboardStreakDays 连续写作天数按 mtime 真实计算：
// 今天+昨天 → 2；仅今天 → 1；今天与 2 天前（昨天缺）→ 1。
func TestGatherDashboardStreakDays(t *testing.T) {
	pm := newProjectWithChapters(t, map[string]string{
		"001.md": "昨日正文",
		"002.md": "今日正文",
	})
	forgeMtime(t, pm, "001.md", 1)
	forgeMtime(t, pm, "002.md", 0)

	d, err := GatherDashboard(pm, 2000)
	if err != nil {
		t.Fatalf("GatherDashboard: %v", err)
	}
	if d.StreakDays != 2 {
		t.Fatalf("今天+昨天连续应 StreakDays=2，实际 %d", d.StreakDays)
	}
	if d.CompletedDays != 2 {
		t.Fatalf("CompletedDays = %d, want 2", d.CompletedDays)
	}

	// 断一档：今天 + 前天 → streak=1
	pm2 := newProjectWithChapters(t, map[string]string{
		"001.md": "前日正文",
		"002.md": "今日正文",
	})
	forgeMtime(t, pm2, "001.md", 2)
	forgeMtime(t, pm2, "002.md", 0)
	d2, err := GatherDashboard(pm2, 2000)
	if err != nil {
		t.Fatalf("GatherDashboard: %v", err)
	}
	if d2.StreakDays != 1 {
		t.Fatalf("昨日断档应 StreakDays=1，实际 %d", d2.StreakDays)
	}
}

// TestGatherDashboardEmptyProject 无 chapters 目录：空仪表盘不报错，
// 不产生虚构每日数据。
func TestGatherDashboardEmptyProject(t *testing.T) {
	dir := t.TempDir()
	pm := &project.Manager{Dir: dir}

	d, err := GatherDashboard(pm, 2000)
	if err != nil {
		t.Fatalf("GatherDashboard: %v", err)
	}
	if d.ChapterCount != 0 || len(d.DailyStats) != 0 || d.TodayWords != 0 {
		t.Fatalf("空项目应全零/空: %+v", d)
	}
	if d.StreakDays != 0 || d.CompletedDays != 0 {
		t.Fatalf("空项目连续/完成天数应为 0: %+v", d)
	}
}
