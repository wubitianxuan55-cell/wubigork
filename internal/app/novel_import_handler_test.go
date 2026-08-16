package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	chs, err := parseTextChapters(path)
	if err != nil {
		t.Fatalf("parseTextChapters: %v", err)
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
	chs, err := parseTextChapters(path)
	if err != nil || len(chs) != 1 || chs[0].Title != "全文" {
		t.Fatalf("无章节标记应归为全文一章: %v %v", chs, err)
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
