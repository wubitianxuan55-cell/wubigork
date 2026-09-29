package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// ── v4.429 小说板块观察池清账（规格 进度计划/gaea-novel-watchpool-20260929.md）──

// TestForEachChapterSkipsGaps N9：中间缺章不再「缺口即停」——缺号后的章必须
// 仍被遍历到（旧实现 ReadChapter 第一个 err 即 break，1、2、4 只扫出 1、2）。
func TestForEachChapterSkipsGaps(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0755); err != nil {
		t.Fatal(err)
	}
	pm := &Manager{Dir: dir, Meta: &types.ProjectMeta{Title: "缺口"}}
	for _, pair := range []struct {
		num  int
		body string
	}{{1, "第一章正文"}, {2, "第二章正文"}, {4, "第四章正文（第 3 章缺失）"}} {
		if err := pm.WriteChapter(pair.num, pair.body); err != nil {
			t.Fatalf("写第 %d 章: %v", pair.num, err)
		}
	}

	var seen []int
	if err := pm.ForEachChapter(func(num int, content string) error {
		seen = append(seen, num)
		if strings.TrimSpace(content) == "" {
			t.Errorf("第 %d 章正文为空", num)
		}
		return nil
	}); err != nil {
		t.Fatalf("ForEachChapter: %v", err)
	}
	if len(seen) != 3 || seen[0] != 1 || seen[1] != 2 || seen[2] != 4 {
		t.Fatalf("缺口后应继续扫到 1、2、4，得到 %v", seen)
	}
}

// TestReadMainlineChapterSummariesExcludesBranches N5：主线口径过滤分支摘要；
// 全量口径（分支浏览器用）保持收全。
func TestReadMainlineChapterSummariesExcludesBranches(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0755); err != nil {
		t.Fatal(err)
	}
	pm := &Manager{Dir: dir, Meta: &types.ProjectMeta{Title: "分支"}}
	if err := pm.WriteChapterSummary(6, &types.ChapterSummary{Title: "第6章", Summary: "主线剧情"}); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapterBranchSummary(6, "a", &types.ChapterSummary{Title: "第6a章", Summary: "分支剧情"}); err != nil {
		t.Fatal(err)
	}

	mainline, err := pm.ReadMainlineChapterSummaries()
	if err != nil {
		t.Fatal(err)
	}
	if len(mainline) != 1 || mainline[0].Summary != "主线剧情" {
		t.Fatalf("主线口径应只含主线摘要，得到 %+v", mainline)
	}

	all, err := pm.ReadAllChapterSummaries()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("全量口径应含分支摘要（分支浏览器特性），得到 %d 条", len(all))
	}
}

// TestMigrateV3ToV4SkippedChaptersFailsClosed N12：有无法迁移的章文件时
// 迁移必须失败且**不落 v4 标记**（标记一落，未迁移章从此从读写视图消失）。
func TestMigrateV3ToV4SkippedChaptersFailsClosed(t *testing.T) {
	dir := t.TempDir()
	chDir := filepath.Join(dir, "chapters")
	if err := os.MkdirAll(chDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chDir, "001.md"), []byte("第一章"), 0644); err != nil {
		t.Fatal(err)
	}
	// 无法解析章号的文件（用户手放/脏数据）：迁移必须如实失败
	if err := os.WriteFile(filepath.Join(chDir, "005x.md"), []byte("形状不对"), 0644); err != nil {
		t.Fatal(err)
	}
	pm := &Manager{Dir: dir, Meta: &types.ProjectMeta{Title: "迁移"}}

	err := pm.MigrateV3ToV4()
	if err == nil {
		t.Fatal("有无法迁移文件时应返回错误")
	}
	if !strings.Contains(err.Error(), "005x.md") {
		t.Fatalf("错误应点名未迁移文件，得到: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".gaea", "v4")); !os.IsNotExist(statErr) {
		t.Fatalf("失败路径不得落 v4 标记，stat err=%v", statErr)
	}

	// 处理掉坏文件后重试应成功
	if err := os.Remove(filepath.Join(chDir, "005x.md")); err != nil {
		t.Fatal(err)
	}
	if err := pm.MigrateV3ToV4(); err != nil {
		t.Fatalf("清理后重试迁移应成功: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".gaea", "v4")); statErr != nil {
		t.Fatalf("成功迁移应落 v4 标记: %v", statErr)
	}
}
