package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

func TestChapterBranchSummaryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0755); err != nil {
		t.Fatal(err)
	}
	pm := &Manager{Dir: dir, Meta: &types.ProjectMeta{Title: "测试"}}
	summary := &types.ChapterSummary{Title: "第3a章", Summary: "分支剧情摘要"}

	if err := pm.WriteChapterBranchSummary(3, "a", summary); err != nil {
		t.Fatal(err)
	}
	got, err := pm.ReadChapterBranchSummary(3, "a")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "第3a章" || got.Summary != "分支剧情摘要" {
		t.Fatalf("got %+v", got)
	}
	if want := filepath.Join(dir, "chapters", "003a-summary.json"); pm.ChapterBranchSummaryPath(3, "a") != want {
		t.Fatalf("path = %s, want %s", pm.ChapterBranchSummaryPath(3, "a"), want)
	}
}

// ── T6-7.3 落盘原子化 ──────────────────────────────────────

// assertNoTempLeftovers 断言目录内没有 writeFileAtomic 遗留的临时文件
func assertNoTempLeftovers(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取目录 %s 失败: %v", dir, err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("发现残留临时文件: %s", e.Name())
		}
	}
}

// TestWriteJSONAtomicRoundTrip 成功写入：内容完整且可回读，无临时文件残留
func TestWriteJSONAtomicRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "characters.json")
	cf := &types.CharacterFile{
		Characters: []types.Character{{ID: "c1", Name: "张三", RoleType: "protagonist"}},
	}
	if err := writeJSON(path, cf); err != nil {
		t.Fatal(err)
	}
	got, err := loadJSON[types.CharacterFile](path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Characters) != 1 || got.Characters[0].ID != "c1" || got.Characters[0].Name != "张三" {
		t.Fatalf("内容不符: %+v", got)
	}
	assertNoTempLeftovers(t, dir)
}

// TestWriteJSONAtomicReplacesExisting 覆盖已存在文件：rename 语义，
// 新内容整体替换旧内容（而非追加/截断残留），无临时文件残留
func TestWriteJSONAtomicReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "outline.json")
	if err := os.WriteFile(path, []byte("{\"nodes\":[]}"), 0644); err != nil {
		t.Fatal(err)
	}
	newOf := &types.OutlineFile{Nodes: []types.OutlineNode{{ID: "n1", Title: "第一章", Status: types.OutlinePlanned}}}
	if err := writeJSON(path, newOf); err != nil {
		t.Fatal(err)
	}
	got, err := loadJSON[types.OutlineFile](path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].ID != "n1" || got.Nodes[0].Title != "第一章" {
		t.Fatalf("替换后内容不符: %+v", got)
	}
	// 旧内容不得残留（如旧文件更长时被部分覆盖）
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "{\"nodes\":[]}") {
		t.Fatalf("旧内容残留: %s", raw)
	}
	assertNoTempLeftovers(t, dir)
}

// TestWriteFileAtomicFailureKeepsOriginal 写入失败不破坏原文件：
// 目标路径是已有目录时 rename 必然失败 → 原内容保留、临时文件被清理
func TestWriteFileAtomicFailureKeepsOriginal(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "blocked.json")
	inner := filepath.Join(blocked, "inner")
	if err := os.MkdirAll(inner, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(inner, "keep.txt")
	if err := os.WriteFile(marker, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := writeFileAtomic(blocked, []byte("new")); err == nil {
		t.Fatal("期望写入失败，实际成功")
	}
	// 原目标（目录及其内容）必须原封不动
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "original" {
		t.Fatalf("原文件被破坏: err=%v data=%q", err, data)
	}
	assertNoTempLeftovers(t, dir)
}

// TestWriteFileAtomicBadPathFails 路径非法（父目录不存在）：报错且不产生任何文件
func TestWriteFileAtomicBadPathFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-such-dir", "x.json")
	if err := writeFileAtomic(path, []byte("data")); err == nil {
		t.Fatal("期望写入失败，实际成功")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("不应创建目标文件: %v", err)
	}
	assertNoTempLeftovers(t, dir)
}

// TestWriteChapterAtomicReplaces 章节写入走原子路径：覆盖写生效、可回读、无残留
func TestWriteChapterAtomicReplaces(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0755); err != nil {
		t.Fatal(err)
	}
	pm := &Manager{Dir: dir}
	if err := pm.WriteChapter(2, "旧章节内容"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapter(2, "新章节内容"); err != nil {
		t.Fatal(err)
	}
	got, err := pm.ReadChapter(2)
	if err != nil {
		t.Fatal(err)
	}
	if got != "新章节内容" {
		t.Fatalf("章节内容不符: %q", got)
	}
	assertNoTempLeftovers(t, filepath.Join(dir, "chapters"))
}
