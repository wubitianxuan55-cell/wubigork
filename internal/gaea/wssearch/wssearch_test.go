package wssearch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSearchCJKContent(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "docs", "成本测算.md"),
		"# 成本测算\n\n本项目成本测算总金额为 100 万元，其中材料费占大头。")
	write(t, filepath.Join(dir, "docs", "无关.md"), "这是一份与技术无关的说明文档。")

	hits := Search(dir, "成本", 10)
	if len(hits) == 0 {
		t.Fatal("中文关键词未命中任何文件")
	}
	if hits[0].Path != "docs/成本测算.md" {
		t.Fatalf("应命中 docs/成本测算.md，got %+v", hits)
	}
	if !strings.Contains(hits[0].Snippet, "成本") {
		t.Fatalf("片段应包含关键词: %q", hits[0].Snippet)
	}
}

func TestSearchEnglishAndCSV(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "README.md"), "# Project\n\nThis is the budget report for Q3.")
	write(t, filepath.Join(dir, "data.csv"), "name,amount\n材料费,12000\n人工费,8000\n")

	hits := Search(dir, "budget", 10)
	if len(hits) == 0 || hits[0].Path != "README.md" {
		t.Fatalf("英文关键词未命中 README.md: %+v", hits)
	}

	hits = Search(dir, "材料费", 10)
	if len(hits) == 0 || hits[0].Path != "data.csv" {
		t.Fatalf("CSV 表格内容未命中: %+v", hits)
	}
}

func TestSearchSkipsNoiseDirs(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "docs", "方案.md"), "成本方案正文")
	write(t, filepath.Join(dir, "node_modules", "藏文件.md"), "成本机密内容")
	write(t, filepath.Join(dir, ".git", "hidden.md"), "成本隐藏内容")
	write(t, filepath.Join(dir, ".gaea", "sessions", "20260810.jsonl"), "成本会话记录")
	write(t, filepath.Join(dir, ".gaea", "archive", "old.md"), "成本归档内容")
	// 交付产物（.gaea/exports）应可被搜索
	write(t, filepath.Join(dir, ".gaea", "exports", "周报.md"), "本周成本总结与下周计划")

	hits := Search(dir, "成本", 10)
	foundExport := false
	for _, h := range hits {
		if strings.HasPrefix(h.Path, "node_modules") || strings.HasPrefix(h.Path, ".git") {
			t.Fatalf("噪音目录泄漏: %s", h.Path)
		}
		if strings.HasPrefix(h.Path, ".gaea/sessions") || strings.HasPrefix(h.Path, ".gaea/archive") {
			t.Fatalf(".gaea 噪音子目录泄漏: %s", h.Path)
		}
		if h.Path == ".gaea/exports/周报.md" {
			foundExport = true
		}
	}
	if !foundExport {
		t.Fatal(".gaea/exports 交付产物应可被索引")
	}
}

func TestSearchFilenameFallback(t *testing.T) {
	dir := t.TempDir()
	// 文件名含关键词，但正文不含：应靠文件名保底命中。
	write(t, filepath.Join(dir, "2026-08-10-报价单.md"), "今天没有可索引正文以外的内容。")

	hits := Search(dir, "报价单", 10)
	if len(hits) == 0 {
		t.Fatal("文件名命中未返回")
	}
	if hits[0].Path != "2026-08-10-报价单.md" {
		t.Fatalf("应命中报价单文件: %+v", hits)
	}
}

func TestSearchLimitAndEmpty(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		write(t, filepath.Join(dir, "f"+string(rune('a'+i))+".md"), "成本内容"+string(rune('a'+i)))
	}
	if hits := Search(dir, "成本", 2); len(hits) > 2 {
		t.Fatalf("limit 未生效: %d", len(hits))
	}
	if hits := Search(dir, "", 10); len(hits) != 0 {
		t.Fatalf("空查询应返回空: %+v", hits)
	}
	if hits := Search(dir, "不存在词xyz", 10); len(hits) != 0 {
		t.Fatalf("无命中应返回空: %+v", hits)
	}
}

func TestSearchSnippetEllipsis(t *testing.T) {
	dir := t.TempDir()
	long := "开头内容。" + strings.Repeat("填充", 30) + "预算关键词在这里。" + strings.Repeat("填充", 30) + "结尾内容。"
	write(t, filepath.Join(dir, "long.md"), long)

	hits := Search(dir, "预算", 10)
	if len(hits) == 0 {
		t.Fatal("未命中长文档")
	}
	s := hits[0].Snippet
	if !strings.Contains(s, "预算关键词") {
		t.Fatalf("片段应包含关键词上下文: %q", s)
	}
	if len([]rune(s)) > 200 {
		t.Fatalf("片段过长: %d", len([]rune(s)))
	}
}

func TestExtractTextSkipsBinary(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "data.bin"), string([]byte{0, 1, 2, 3}))
	if _, ok, _ := extractText(filepath.Join(dir, "data.bin")); ok {
		t.Fatal("二进制文件不应被索引")
	}
}

func TestSearchSkippedLargeText(t *testing.T) {
	dir := t.TempDir()
	// >5MB 文本：正文不索引，但文件名命中时返回可见的跳过原因。
	big := filepath.Join(dir, "超大日志.txt")
	if err := os.WriteFile(big, []byte(strings.Repeat("x", 5<<20+1)), 0o644); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "正常.md"), "超大文件处理说明")

	hits := Search(dir, "超大", 10)
	found := false
	for _, h := range hits {
		if h.Path == "超大日志.txt" {
			found = true
			if h.Skipped == "" {
				t.Fatalf("超大文本文件应带跳过原因: %+v", h)
			}
			if !strings.Contains(h.Skipped, "summarize_file") {
				t.Fatalf("跳过原因应指引 summarize_file: %q", h.Skipped)
			}
		}
	}
	if !found {
		t.Fatalf("超大文本文件名命中应返回（带跳过提示）: %+v", hits)
	}
}

func TestSearchTruncatedMark(t *testing.T) {
	dir := t.TempDir()
	// >20 万字符：正文截断，命中带 Truncated 标记。
	body := "预算关键词在开头。" + strings.Repeat("填充", 120000)
	write(t, filepath.Join(dir, "超大报告.md"), body)

	hits := Search(dir, "预算", 10)
	if len(hits) == 0 {
		t.Fatal("截断文件应仍可命中")
	}
	if !hits[0].Truncated {
		t.Fatalf("超大文档应标记截断: %+v", hits[0])
	}
}
