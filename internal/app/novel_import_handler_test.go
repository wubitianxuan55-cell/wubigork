package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/bookimport"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/project"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestParseTextChapters_SplitsByHeadings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.txt")
	content := "《测试之书》\n作者：佚名\n\n第一章 初遇\n\n雨夜，她推开茶馆的门。\n\n第二章 风波\n\n城外的消息传开了。"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("写文件: %v", err)
	}
	chs, report, err := parseTextChapters(path, bookimport.ParseOptions{ExtractMode: bookimport.ExtractFull})
	if err != nil {
		t.Fatalf("parseTextChapters: %v", err)
	}
	if report.SplitStrategy != "strong" || report.Encoding != "utf-8" {
		t.Fatalf("解析报告异常: %+v", report)
	}
	if len(chs) != 2 {
		t.Fatalf("应解析出 2 章: %d", len(chs))
	}
	if chs[0].Title != "第一章 初遇" {
		t.Fatalf("第一章标题: %q", chs[0].Title)
	}
	if !strings.Contains(chs[0].Content, "《测试之书》") || !strings.Contains(chs[0].Content, "雨夜") {
		t.Fatalf("第一章应包含前置书名与正文: %q", chs[0].Content)
	}
	if chs[1].Title != "第二章 风波" || !strings.Contains(chs[1].Content, "城外的消息") {
		t.Fatalf("第二章解析异常: %+v", chs[1])
	}
}

func TestParseTextChapters_NoHeadings_SingleChapter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.txt")
	if err := os.WriteFile(path, []byte("没有章节标记的一段文字"), 0644); err != nil {
		t.Fatalf("写文件: %v", err)
	}
	chs, report, err := parseTextChapters(path, bookimport.ParseOptions{ExtractMode: bookimport.ExtractFull})
	if err != nil || len(chs) != 1 || chs[0].Title != "全文" {
		t.Fatalf("无章节标记应归为全文一章: %v %v", chs, err)
	}
	if report.SplitStrategy != "single" {
		t.Fatalf("短文本无标题策略应为 single: %+v", report)
	}
}

// ── v4.279：三级分章 + 编码链（规格 docs/distill/02-book-import.md §8.1 验收项）──

func TestParseTextChapters_UTF8BOM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bom.txt")
	content := "第一章 起\n\n风起了。\n\n第二章 落\n\n雨停了。"
	raw := append([]byte{0xEF, 0xBB, 0xBF}, []byte(content)...)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatalf("写文件: %v", err)
	}
	chs, report, err := parseTextChapters(path, bookimport.ParseOptions{ExtractMode: bookimport.ExtractFull})
	if err != nil || len(chs) != 2 {
		t.Fatalf("BOM 文件解析: %v %d", err, len(chs))
	}
	if report.Encoding != "utf-8-sig" {
		t.Fatalf("BOM 编码名 = %q, want utf-8-sig", report.Encoding)
	}
	if strings.Contains(chs[0].Title, "\uFEFF") || !strings.HasPrefix(chs[0].Title, "第一章") {
		t.Fatalf("BOM 污染了标题: %q", chs[0].Title)
	}
}

func TestParseTextChapters_GB18030(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gb.txt")
	content := "第一章 起\n\n风起了。\n\n第二章 落\n\n雨停了。"
	raw, err := simplifiedchinese.GB18030.NewEncoder().Bytes([]byte(content))
	if err != nil {
		t.Fatalf("构造 GB18030 样本: %v", err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatalf("写文件: %v", err)
	}
	chs, report, err := parseTextChapters(path, bookimport.ParseOptions{ExtractMode: bookimport.ExtractFull})
	if err != nil || len(chs) != 2 {
		t.Fatalf("GB18030 文件解析: %v %d", err, len(chs))
	}
	if report.Encoding != "gb18030" || chs[1].Title != "第二章 落" {
		t.Fatalf("GB18030 解码异常: %+v %+v", report, chs[1])
	}
}

func TestParseTextChapters_NoHeadingLongText_WindowSplit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain-long.txt")
	var sb strings.Builder
	for utf8.RuneCountInString(sb.String()) < 8000 {
		sb.WriteString("他推开门，雨还在下，屋檐下的灯影在水面上摇晃了很久很久。")
		sb.WriteString("。")
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		t.Fatalf("写文件: %v", err)
	}
	chs, report, err := parseTextChapters(path, bookimport.ParseOptions{ExtractMode: bookimport.ExtractFull})
	if err != nil {
		t.Fatalf("无标题长文本解析: %v", err)
	}
	if report.SplitStrategy != "window" || len(chs) < 2 {
		t.Fatalf("8000 字无标题应走窗口切分: %+v %d 章", report, len(chs))
	}
	for _, ch := range chs {
		if !strings.HasPrefix(ch.Title, "第") || !strings.HasSuffix(ch.Title, "章") {
			t.Fatalf("窗口章标题应伪造为第N章: %q", ch.Title)
		}
	}
}

func TestParseTextChapters_WeakHeadings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "weak.txt")
	content := "开篇的话\n\n雨夜\n\n她推开门。\n\n晨光\n\n城外的消息传开了。"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("写文件: %v", err)
	}
	chs, report, err := parseTextChapters(path, bookimport.ParseOptions{ExtractMode: bookimport.ExtractFull})
	if err != nil || len(chs) != 3 {
		t.Fatalf("弱标题切分: %v %d 章", err, len(chs))
	}
	if report.SplitStrategy != "weak" || chs[1].Title != "雨夜" {
		t.Fatalf("弱标题策略/标题异常: %+v %+v", report, chs[1])
	}
}

func TestDecodeText_GBK(t *testing.T) {
	gbk, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte("第一章 测试内容"))
	if err != nil {
		t.Fatalf("编码 GBK: %v", err)
	}
	if got := decodeText(gbk); got != "第一章 测试内容" {
		t.Fatalf("GBK 解码失败: %q", got)
	}
	if got := decodeText([]byte("纯 UTF-8 内容")); got != "纯 UTF-8 内容" {
		t.Fatalf("UTF-8 直通失败: %q", got)
	}
}

func TestSanitizeDirName(t *testing.T) {
	if got := sanitizeDirName(`星/落:之*城?`); got != "星_落_之_城_" {
		t.Fatalf("非法字符清洗: %q", got)
	}
	if got := sanitizeDirName("  "); got == "" || got == "." || got == ".." {
		t.Fatalf("空名应回退: %q", got)
	}
}

func TestImportNovelBook_EndToEnd(t *testing.T) {
	a := newCharacterLibTestApp(t)
	a.cfg.NovelsDir = t.TempDir()

	path := filepath.Join(t.TempDir(), "成书.txt")
	content := "第一章 启程\n\n风起于青萍之末。\n\n第二章 归途\n\n灯火渐明。"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("写文件: %v", err)
	}
	res, err := a.ImportNovelBook(path, "成书", "玄幻", "热血")
	if err != nil {
		t.Fatalf("导入: %v", err)
	}
	if res.ChapterCount != 2 || res.Title != "成书" || res.Path == "" {
		t.Fatalf("导入结果异常: %+v", res)
	}
	if res.Encoding != "utf-8" || res.SplitStrategy != "strong" {
		t.Fatalf("导入结果应带解析报告（编码/策略）: %+v", res)
	}
	meta, err := loadProjectMeta(filepath.Join(res.Path, "project.json"))
	if err != nil {
		t.Fatalf("读取项目元数据: %v", err)
	}
	pm := &project.Manager{Dir: res.Path, Meta: meta}
	c1, err := pm.ReadChapter(1)
	if err != nil || !strings.Contains(c1, "风起于青萍之末") {
		t.Fatalf("第一章内容: %q %v", c1, err)
	}
	of, err := pm.ReadOutlines()
	if err != nil || len(of.Nodes) != 2 || of.Nodes[0].Title != "第一章 启程" {
		t.Fatalf("大纲异常: %+v %v", of.Nodes, err)
	}
}

func TestImportNovelBook_Guards(t *testing.T) {
	a := newCharacterLibTestApp(t)
	a.cfg.NovelsDir = t.TempDir()

	if _, err := a.ImportNovelBook("book.pdf", "书", "玄幻", ""); err == nil || !strings.Contains(err.Error(), "TXT") {
		t.Fatalf("不支持格式应报错: %v", err)
	}
	if _, err := a.ImportNovelBook(filepath.Join(t.TempDir(), "nope.txt"), "书", "玄幻", ""); err == nil {
		t.Fatalf("文件不存在应报错")
	}
}

// ── tail 提取范围出口（v4.287，拆书导入线欠账）──

// 构造 N 章强标题 TXT。
func writeNChapters(t *testing.T, dir string, n int) string {
	t.Helper()
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString("第" + strconv.Itoa(i) + "章 标题" + strconv.Itoa(i) + "\n\n本章正文内容，足够长不算过短。\n\n")
	}
	path := filepath.Join(dir, "长书.txt")
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		t.Fatalf("写长书: %v", err)
	}
	return path
}

func TestImportNovelBookEx_TailModeKeepsLastAndRenumbers(t *testing.T) {
	a := newCharacterLibTestApp(t)
	a.cfg.NovelsDir = t.TempDir()

	path := writeNChapters(t, t.TempDir(), 12)
	res, err := a.ImportNovelBookEx(path, "末十章", "玄幻", "", "tail", 10)
	if err != nil {
		t.Fatalf("tail 导入: %v", err)
	}
	if res.ChapterCount != 10 {
		t.Fatalf("应只保留末 10 章: %+v", res)
	}
	// 裁剪告知应透出到导入报告
	hit := false
	for _, w := range res.Warnings {
		if strings.Contains(w.Message, "末尾 10 章") {
			hit = true
		}
	}
	if !hit {
		t.Fatalf("裁剪告警应如实带出: %+v", res.Warnings)
	}
	// 落库章节应从 1 重编号，首章=源第 3 章
	pm, err := project.Open(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer pm.Close()
	of, _ := pm.ReadOutlines()
	if len(of.Nodes) != 10 || of.Nodes[0].Title != "第3章 标题3" {
		t.Fatalf("应从源第 3 章起保留 10 章: %+v", of.Nodes[0])
	}
}

func TestImportNovelBookEx_TailOverMaxFallsBackFull(t *testing.T) {
	a := newCharacterLibTestApp(t)
	a.cfg.NovelsDir = t.TempDir()

	path := writeNChapters(t, t.TempDir(), 12)
	// 60 > 上限 50 → 降级全本（12 章）
	res, err := a.ImportNovelBookEx(path, "全本", "玄幻", "", "tail", 60)
	if err != nil {
		t.Fatalf("降级导入: %v", err)
	}
	if res.ChapterCount != 12 {
		t.Fatalf(">50 应降级全本: %+v", res)
	}

	// 非法模式显式报错
	if _, err := a.ImportNovelBookEx(path, "x", "", "", "middle", 0); err == nil {
		t.Fatal("非法 extractMode 应报错")
	}

	// full 出口与旧绑定行为一致（回归）
	resFull, err := a.ImportNovelBookEx(path, "全本2", "玄幻", "", "full", 0)
	if err != nil || resFull.ChapterCount != 12 {
		t.Fatalf("full 出口回归: %+v %v", resFull, err)
	}
}
