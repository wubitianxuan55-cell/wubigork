package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// newTestProject 构造一个含元数据（标题/作者）与章节文件、可选世界观的测试项目。
func newTestProject(t *testing.T, meta types.ProjectMeta, chapters map[string]string, worldview string) *project.Manager {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0755); err != nil {
		t.Fatal(err)
	}
	meta.SchemaVersion = 1
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	if worldview != "" {
		if err := os.WriteFile(filepath.Join(dir, "worldview.md"), []byte(worldview), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range chapters {
		if err := os.WriteFile(filepath.Join(dir, "chapters", name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return &project.Manager{Dir: dir, Meta: &meta}
}

// readZipEntry 读取 zip（EPUB）内指定条目内容。
func readZipEntry(t *testing.T, zipPath, name string) string {
	t.Helper()
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("打开 EPUB: %v", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("打开 zip 条目 %s: %v", name, err)
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("读取 zip 条目 %s: %v", name, err)
			}
			return string(data)
		}
	}
	t.Fatalf("zip 中找不到条目 %s", name)
	return ""
}

// ── T6-7.1 导出整改 ────────────────────────────────────────

// TestExportTXT_ContainsTitleWorldviewChapters TXT 导出包含标题/题材/文风/世界观与章节正文
func TestExportTXT_ContainsTitleWorldviewChapters(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "星海彼岸", Genre: "科幻", Style: "硬核"},
		map[string]string{
			"001.md": "第一段正文。\n\n第二段正文。",
			"002.md": "第二章正文。",
		},
		"这是一个奇幻世界设定。")
	out := filepath.Join(t.TempDir(), "novel.txt")
	m := New(pm)
	got, err := m.ExportTXT(out)
	if err != nil {
		t.Fatalf("ExportTXT: %v", err)
	}
	if got != out {
		t.Errorf("返回值 = %q, want %q", got, out)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{"星海彼岸", "科幻", "硬核", "【世界观】", "这是一个奇幻世界设定。", "第 1 章", "第一段正文。", "第 2 章", "第二章正文。"} {
		if !strings.Contains(s, want) {
			t.Errorf("TXT 导出缺少 %q", want)
		}
	}
	if m.FailedChapters != 0 {
		t.Errorf("FailedChapters = %d, want 0", m.FailedChapters)
	}
}

// TestExportMarkdown_ContainsWorldview MD 导出补世界观（与 TXT 对齐）
func TestExportMarkdown_ContainsWorldview(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "测试", Genre: "玄幻", Style: "轻松"},
		map[string]string{"001.md": "正文。"}, "世界观内容-AB12")
	out := filepath.Join(t.TempDir(), "novel.md")
	m := New(pm)
	if _, err := m.ExportMarkdown(out); err != nil {
		t.Fatalf("ExportMarkdown: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "## 世界观") || !strings.Contains(s, "世界观内容-AB12") {
		t.Errorf("Markdown 导出应包含世界观，实际内容: %s", s)
	}
	if !strings.Contains(s, "## 第 1 章") {
		t.Errorf("Markdown 导出应包含章节标题")
	}
}

// TestExportTXT_Markdown_WorldviewConsistency TXT 与 MD 均含世界观内容
func TestExportTXT_Markdown_WorldviewConsistency(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "一致性", Genre: "都市", Style: "写实"},
		map[string]string{"001.md": "正文。"}, "双格式共同世界观。")
	m := New(pm)
	dir := t.TempDir()
	if _, err := m.ExportTXT(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ExportMarkdown(filepath.Join(dir, "a.md")); err != nil {
		t.Fatal(err)
	}
	txt, _ := os.ReadFile(filepath.Join(dir, "a.txt"))
	md, _ := os.ReadFile(filepath.Join(dir, "a.md"))
	if !strings.Contains(string(txt), "双格式共同世界观。") {
		t.Errorf("TXT 缺少世界观")
	}
	if !strings.Contains(string(md), "双格式共同世界观。") {
		t.Errorf("MD 缺少世界观（与 TXT 不一致）")
	}
}

// TestExport_ChapterReadFailure_WarnsAndCounts 章节读取失败：slog.Warn + 失败计数（部分失败可见）
func TestExport_ChapterReadFailure_WarnsAndCounts(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "失败", Genre: "悬疑", Style: "冷峻"},
		map[string]string{"001.md": "第一章正文。"}, "")
	// 断链符号链接：目录中存在 002.md 条目但读取必然失败（os.ReadDir 可列出，ReadFile 报错）
	link := filepath.Join(pm.Dir, "chapters", "002.md")
	if err := os.Symlink(filepath.Join(pm.Dir, "chapters", "002-missing.md"), link); err != nil {
		t.Skipf("当前环境无法创建符号链接，跳过失败分支用例: %v", err)
	}

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(prev)

	m := New(pm)
	out := filepath.Join(t.TempDir(), "fail.txt")
	if _, err := m.ExportTXT(out); err != nil {
		t.Fatalf("ExportTXT: %v", err)
	}
	if m.FailedChapters != 1 {
		t.Errorf("FailedChapters = %d, want 1", m.FailedChapters)
	}
	logText := buf.String()
	if !strings.Contains(logText, "exportTXT") || !strings.Contains(logText, "读取章节失败") {
		t.Errorf("应记录 slog.Warn，实际日志: %s", logText)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "第一章正文。") {
		t.Errorf("成功章节仍应导出")
	}
}

// TestSanitizeFilename_WindowsReservedNames Windows 保留名（含带扩展名形式）加前缀规避
func TestSanitizeFilename_WindowsReservedNames(t *testing.T) {
	cases := map[string]string{
		"CON": "_CON", "con": "_con", "CON.txt": "_CON.txt",
		"PRN": "_PRN", "AUX": "_AUX", "NUL": "_NUL",
		"COM1": "_COM1", "COM9": "_COM9", "com5": "_com5",
		"LPT1": "_LPT1", "LPT9": "_LPT9", "lpt3": "_lpt3",
		"我的小说": "我的小说",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestSanitizeFilename_TrailingDotsAndSpaces 尾部点/空格剔除与非法字符替换
func TestSanitizeFilename_TrailingDotsAndSpaces(t *testing.T) {
	cases := map[string]string{
		"书名.":      "书名",
		"书名...":    "书名",
		"书名 ":      "书名",
		"书名 .":     "书名",
		"a/b:c*d?": "a_b_c_d_",
		"正常书名":     "正常书名",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestExportTXT_CreatesParentDir 导出目标父目录不存在时自动创建
func TestExportTXT_CreatesParentDir(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "父目录", Genre: "奇幻", Style: "史诗"},
		map[string]string{"001.md": "正文。"}, "")
	root := t.TempDir()
	out := filepath.Join(root, "deep", "nested", "novel.txt")
	if _, err := New(pm).ExportTXT(out); err != nil {
		t.Fatalf("ExportTXT 应自动创建父目录: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("导出文件不存在: %v", err)
	}
}

// TestExportMarkdown_CreatesParentDir 导出目标父目录不存在时自动创建
func TestExportMarkdown_CreatesParentDir(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "父目录", Genre: "奇幻", Style: "史诗"},
		map[string]string{"001.md": "正文。"}, "")
	root := t.TempDir()
	out := filepath.Join(root, "deep", "nested", "novel.md")
	if _, err := New(pm).ExportMarkdown(out); err != nil {
		t.Fatalf("ExportMarkdown 应自动创建父目录: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("导出文件不存在: %v", err)
	}
}

// TestExportEPUB_CreatesParentDir EPUB 写入前自动创建父目录
func TestExportEPUB_CreatesParentDir(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "父目录EPUB", Genre: "奇幻", Style: "史诗"},
		map[string]string{"001.md": "正文。"}, "")
	root := t.TempDir()
	out := filepath.Join(root, "deep", "nested", "novel.epub")
	if _, err := New(pm).ExportEPUB(out); err != nil {
		t.Fatalf("ExportEPUB 应自动创建父目录: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("导出文件不存在: %v", err)
	}
}

// TestExportEPUB_ChapterUsesUnifiedSegmenter EPUB 章节与 HTML 导出共用 markdownToHTML 分段器
func TestExportEPUB_ChapterUsesUnifiedSegmenter(t *testing.T) {
	content := "### 楔子\n\n第一段。\n\n- 要点一\n- 要点二\n\n第二段。"
	pm := newTestProject(t, types.ProjectMeta{Title: "分段器", Genre: "玄幻", Style: "热血"},
		map[string]string{"001.md": content}, "")
	out := filepath.Join(t.TempDir(), "novel.epub")
	if _, err := New(pm).ExportEPUB(out); err != nil {
		t.Fatalf("ExportEPUB: %v", err)
	}
	ch := readZipEntry(t, out, "EPUB/xhtml/chapter_001.xhtml")
	if !strings.Contains(ch, markdownToHTML(content)) {
		t.Errorf("EPUB 章节未使用统一分段器 markdownToHTML\n章节内容: %s", ch)
	}
	// markdownToHTML 段落输出为 "<p>\n正文 \n</p>"（逐元素换行拼接），按实际产物断言
	for _, frag := range []string{"<h3>楔子</h3>", "<li>要点一</li>", "<p>\n第二段。 \n</p>"} {
		if !strings.Contains(ch, frag) {
			t.Errorf("EPUB 章节缺少 %q", frag)
		}
	}
}

// TestExportEPUB_AuthorFromProjectMeta EPUB 作者取自项目元数据
func TestExportEPUB_AuthorFromProjectMeta(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "作者", Genre: "现实", Style: "细腻", Author: "星河漫游者"},
		map[string]string{"001.md": "正文。"}, "")
	out := filepath.Join(t.TempDir(), "novel.epub")
	if _, err := New(pm).ExportEPUB(out); err != nil {
		t.Fatalf("ExportEPUB: %v", err)
	}
	opf := readZipEntry(t, out, "EPUB/package.opf")
	if !strings.Contains(opf, "<dc:creator") || !strings.Contains(opf, "星河漫游者") {
		t.Errorf("EPUB 作者应取自项目元数据，package.opf: %s", opf)
	}
}

// TestExportEPUB_AuthorFallback 项目未配置作者时回退品牌名 gaea
func TestExportEPUB_AuthorFallback(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "无作者", Genre: "现实", Style: "细腻"},
		map[string]string{"001.md": "正文。"}, "")
	out := filepath.Join(t.TempDir(), "novel.epub")
	if _, err := New(pm).ExportEPUB(out); err != nil {
		t.Fatalf("ExportEPUB: %v", err)
	}
	opf := readZipEntry(t, out, "EPUB/package.opf")
	if !strings.Contains(opf, "<dc:creator") || !strings.Contains(opf, "gaea") {
		t.Errorf("无元数据作者时应回退 gaea，package.opf: %s", opf)
	}
}

// TestExportAll_CreatesExportDir 一键导出在项目 export/ 目录产出 txt/md/epub 三格式
func TestExportAll_CreatesExportDir(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "合集", Genre: "奇幻", Style: "轻快"},
		map[string]string{"001.md": "正文。"}, "世界观。")
	res, err := New(pm).ExportAll()
	if err != nil {
		t.Fatal(err)
	}
	for ext, p := range res {
		if strings.HasPrefix(p, "失败") {
			t.Errorf("ExportAll %s 失败: %s", ext, p)
			continue
		}
		if _, err := os.Stat(p); err != nil {
			t.Errorf("ExportAll %s 产物不存在: %v", ext, err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(pm.Dir, "export"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("export 目录应有 3 个文件，实际 %d", len(entries))
	}
}
