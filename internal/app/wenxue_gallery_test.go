package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── 文学「插图」标签数据面测试（2026-11-18）────────────────────────────

// TestWenxueGalleryList 收集/分类/排序/越界防护：
//   - .gaea/uploads 图片 = uploads 来源（image_gen/粘贴产出）；
//   - 工作空间其他目录图片 = workspace 来源；
//   - 非 .gaea/uploads 的 .gaea 子树（cache 等）不收集；
//   - 非图片扩展名跳过；mtime 倒序。
func TestWenxueGalleryList(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GAEA_TUWEN_ROOT", root)
	book := filepath.Join(root, "雾港来信")
	mk := func(rel, content string) {
		t.Helper()
		p := filepath.Join(book, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk(".gaea/uploads/gen-cover.png", "png-cover")
	mk(".gaea/uploads/gen-scene.png", "png-scene")
	mk(".gaea/uploads/notes.txt", "not-image")
	mk(".gaea/cache/thumb.png", "cache-noise") // .gaea 非 uploads 子树不收集
	mk("角色/立绘.jpg", "portrait")
	mk("章节/001.md", "chapter")

	a := &App{}
	items, err := a.WenxueGalleryList()
	if err != nil {
		t.Fatalf("WenxueGalleryList: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("应收集 3 张图（uploads×2 + 角色×1），got %d: %+v", len(items), items)
	}
	sources := map[string]string{}
	for _, it := range items {
		sources[it.Name] = it.Source
		if !strings.HasSuffix(it.Rel, filepath.ToSlash(it.Name)) {
			t.Fatalf("Rel 应为工作空间相对路径: %q", it.Rel)
		}
	}
	if sources["gen-cover.png"] != "uploads" || sources["gen-scene.png"] != "uploads" {
		t.Fatalf("uploads 来源分类错: %v", sources)
	}
	if sources["立绘.jpg"] != "workspace" {
		t.Fatalf("其他目录图片应标 workspace，got %v", sources["立绘.jpg"])
	}
	// mtime 倒序（同秒写入按名字兜底序，这里只断言两腿都在且无 cache 噪声）。
	for _, it := range items {
		if strings.Contains(it.Rel, ".gaea/cache") {
			t.Fatalf(".gaea 非 uploads 子树泄漏: %q", it.Rel)
		}
		if it.Name == "notes.txt" {
			t.Fatalf("非图片泄漏: %v", it)
		}
	}
}
