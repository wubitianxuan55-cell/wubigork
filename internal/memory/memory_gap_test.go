package memory

// memory_gap_test.go — T7-3：「章节断档即停」修复：BuildFromProject 改用
// project.ReadAllChapterSummaries 目录扫描，中间缺章不再提前终止索引。

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/project"
)

// TestBuildFromProjectGapTolerant 章节断档（001、003，缺 002）时仍能索引到
// 断档后的章节摘要，且章节号按文件名解析正确。
func TestBuildFromProjectGapTolerant(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeSummary(t, dir, "001-summary.json", "第一章", "主角进入青云宗")
	writeSummary(t, dir, "003-summary.json", "第三章", "主角突破瓶颈")
	pm := &project.Manager{Dir: dir}

	idx, err := BuildFromProject(pm)
	if err != nil {
		t.Fatalf("BuildFromProject: %v", err)
	}
	if len(idx.memories) != 2 {
		t.Fatalf("断档项目应索引 2 条记忆，实际 %d: %+v", len(idx.memories), idx.memories)
	}
	nums := map[int]bool{}
	for _, m := range idx.memories {
		nums[m.ChapterNum] = true
	}
	if !nums[1] || !nums[3] {
		t.Fatalf("章节号应为 1 和 3（断档后的 3 不能被跳过）: %v", nums)
	}
}

// TestBuildFromProjectBranchSummary 分支摘要（001a）也纳入索引，章节号取主章节号。
func TestBuildFromProjectBranchSummary(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeSummary(t, dir, "002-summary.json", "第二章", "主线：商会谈判")
	writeSummary(t, dir, "002a-summary.json", "第二章a", "分支：谈判破裂")
	pm := &project.Manager{Dir: dir}

	idx, err := BuildFromProject(pm)
	if err != nil {
		t.Fatalf("BuildFromProject: %v", err)
	}
	if len(idx.memories) != 2 {
		t.Fatalf("应索引 2 条（主线+分支），实际 %d", len(idx.memories))
	}
	for _, m := range idx.memories {
		if m.ChapterNum != 2 {
			t.Fatalf("分支摘要章节号应为 2，实际 %d（%+v）", m.ChapterNum, m)
		}
	}
}

// TestBuildFromProjectNoChaptersDir 无 chapters 目录：空索引且不报错。
func TestBuildFromProjectNoChaptersDir(t *testing.T) {
	dir := t.TempDir()
	pm := &project.Manager{Dir: dir}

	idx, err := BuildFromProject(pm)
	if err != nil {
		t.Fatalf("BuildFromProject 不应报错: %v", err)
	}
	if len(idx.memories) != 0 {
		t.Fatalf("空项目应返回空索引，实际 %d", len(idx.memories))
	}
}

func writeSummary(t *testing.T, dir, name, title, summary string) {
	t.Helper()
	data := []byte(`{"title":"` + title + `","summary":"` + summary + `"}`)
	if err := os.WriteFile(filepath.Join(dir, "chapters", name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}
