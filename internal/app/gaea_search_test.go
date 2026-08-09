package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGaeaFileSearch(t *testing.T) {
	t.Chdir(t.TempDir())
	dirs := []string{"docs", "表格", ".git", "node_modules"}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"docs/成本测算.xlsx":     "x",
		"docs/方案.docx":       "x",
		"表格/报价表.xlsx":        "x",
		".git/secret.docx":   "x",
		"node_modules/a.txt": "x",
	}
	for p, body := range files {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	a := &App{}
	hits := a.GaeaFileSearch("doc", 30)
	found := map[string]bool{}
	for _, h := range hits {
		if strings.Contains(h.Path, ".git") || strings.Contains(h.Path, "node_modules") {
			t.Fatalf("噪音目录泄漏: %s", h.Path)
		}
		found[h.Path] = true
	}
	if !found["docs/方案.docx"] {
		t.Fatalf("未命中 docs/方案.docx: %+v", hits)
	}
	if found["docs/成本测算.xlsx"] {
		t.Fatal("成本测算.xlsx 不应命中 'doc' 查询")
	}

	// 中文查询
	hits = a.GaeaFileSearch("成本", 30)
	found = map[string]bool{}
	for _, h := range hits {
		found[h.Path] = true
	}
	if !found["docs/成本测算.xlsx"] {
		t.Fatalf("中文查询未命中: %+v", hits)
	}

	// 限制数量
	if len(a.GaeaFileSearch("", 2)) > 2 {
		t.Fatal("limit 未生效")
	}
	// 深度上限：深层文件不无限递归（不会 panic 即可）
	deep := filepath.Join("a", "b", "c", "d", "e", "f", "g")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(deep, "deep.txt"), []byte("x"), 0o644)
	_ = a.GaeaFileSearch("deep", 30)
}

func TestGaeaMaterials(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}
	// 修改时间不同的资料文件 + 一个不应收录的 .go + 一个噪音目录里的文件
	if err := os.WriteFile("docs/旧方案.docx", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/新成本测算.xlsx", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/说明.md", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/main.go", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("node_modules", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("node_modules/内藏.xlsx", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 让「新成本测算.xlsx」最新：重新写一次使其 mtime 最大（部分文件系统粒度问题，
	// 这里直接比较集合，不依赖严格排序的唯一性）。
	old := time.Now().Add(-time.Hour)
	_ = os.Chtimes("docs/旧方案.docx", old, old)
	_ = os.Chtimes("docs/说明.md", old, old)

	a := &App{}
	mats := a.GaeaMaterials(100)
	got := map[string]bool{}
	for _, m := range mats {
		got[m.Path] = true
		if strings.HasPrefix(m.Path, "node_modules") {
			t.Fatalf("噪音目录泄漏: %s", m.Path)
		}
	}
	if !got["docs/新成本测算.xlsx"] || !got["docs/旧方案.docx"] || !got["docs/说明.md"] {
		t.Fatalf("资料缺失: %+v", got)
	}
	if got["docs/main.go"] {
		t.Fatal(".go 不应收录")
	}
	if len(mats) > 0 && mats[0].Path != "docs/新成本测算.xlsx" {
		t.Fatalf("应最新在前: %s", mats[0].Path)
	}
}
