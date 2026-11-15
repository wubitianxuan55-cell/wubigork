package app

// 刀3 章节协议测试：标记扫描/切章/落盘续编/幂等合并 + sin_chapter 工具三动作。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
)

func TestSinScanChapterMarks(t *testing.T) {
	text := "回合旁白，不属于任何章。\n\n@@章节|雨夜相遇@@\n\n雨夜的开场正文。\n\n@@章节|车厢对峙@@\n\n对峙正文。"
	marks := sinScanChapterMarks(text)
	if len(marks) != 2 {
		t.Fatalf("marks = %d, want 2", len(marks))
	}
	if marks[0].Title != "雨夜相遇" || marks[1].Title != "车厢对峙" {
		t.Fatalf("titles = %q / %q", marks[0].Title, marks[1].Title)
	}
	// 非独占行不识别（协议硬边界）
	if len(sinScanChapterMarks("正文里提到@@章节|伪标记@@不换行")) != 0 {
		t.Fatal("行中标记不应识别")
	}
}

func TestSinSplitChapters(t *testing.T) {
	text := "旁白\n@@章节|第一章@@\n\n甲正文\n@@章节|第二章@@\n\n乙正文"
	chs := sinSplitChapters(text)
	if len(chs) != 2 {
		t.Fatalf("chapters = %d", len(chs))
	}
	if chs[0][0] != "第一章" || !strings.HasPrefix(chs[0][1], "甲正文") {
		t.Fatalf("ch0 = %v", chs[0])
	}
	if chs[1][0] != "第二章" || !strings.HasPrefix(chs[1][1], "乙正文") {
		t.Fatalf("ch1 = %v", chs[1])
	}
	if sinSplitChapters("没有标记的回合") != nil {
		t.Fatal("无标记应返回 nil")
	}
}

func TestSinPersistChapters(t *testing.T) {
	a := newSinEngineTestApp(t)
	book := newShelfBook(t, a)

	// 两回合：编号续接（003 起——书架书可能已有 000-002？newShelfBook 无既有章，
	// 首回合落 001/002，次回合 003）
	w1 := a.sinPersistChapters(book, "旁白\n@@章节|开篇@@\n\n第一回正文。\n@@章节|对峙@@\n\n第二回正文。")
	if len(w1) != 2 {
		t.Fatalf("w1 = %+v", w1)
	}
	if w1[0].Num != 1 || w1[1].Num != 2 {
		t.Fatalf("编号不续接: %+v", w1)
	}
	raw, err := os.ReadFile(filepath.Join(book, "chapters", "001.md"))
	if err != nil || strings.TrimSpace(string(raw)) != "第一回正文。" {
		t.Fatalf("001.md = %q, err=%v", string(raw), err)
	}
	// outline 节点：标题/文件/状态
	m, err := project.Open(book)
	if err != nil {
		t.Fatal(err)
	}
	of, err := m.ReadOutlines()
	if err != nil || len(of.Nodes) != 2 {
		t.Fatalf("outline nodes = %+v, err=%v", of, err)
	}
	if of.Nodes[0].Title != "开篇" || of.Nodes[0].ChapterFile != "001.md" || of.Nodes[0].Status != "done" {
		t.Fatalf("node0 = %+v", of.Nodes[0])
	}

	// 次回合续编 003；同文件重扫幂等（不堆积节点）
	w2 := a.sinPersistChapters(book, "@@章节|第三回@@\n\n第三回正文。")
	if len(w2) != 1 || w2[0].Num != 3 {
		t.Fatalf("w2 = %+v", w2)
	}
	a.sinPersistChapters(book, "@@章节|第三回@@\n\n第三回正文。")
	of2, _ := m.ReadOutlines()
	count := 0
	for _, n := range of2.Nodes {
		if n.ChapterFile == "003.md" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("003 节点堆积: %d", count)
	}

	// 未绑书 = 零动作不落盘
	if got := a.sinPersistChapters("", "@@章节|x@@\n\n正文"); got != nil {
		t.Fatal("未绑书应零动作")
	}
}

func TestSinChapterTool(t *testing.T) {
	a := newSinEngineTestApp(t)
	book := newShelfBook(t, a)
	a.sinPersistChapters(book, "@@章节|开篇@@\n\n开篇正文第一段。")

	tl := &sinChapterTool{app: a, bookPath: book}
	// list
	out, err := tl.Execute(t.Context(), []byte(`{"action":"list"}`))
	if err != nil || !strings.Contains(out, "开篇") || !strings.Contains(out, "001.md") {
		t.Fatalf("list = %q, err=%v", out, err)
	}
	// read
	out, err = tl.Execute(t.Context(), []byte(`{"action":"read","num":1}`))
	if err != nil || !strings.Contains(out, "开篇正文") {
		t.Fatalf("read = %q, err=%v", out, err)
	}
	// write（新章 2 + 标题进 outline）
	out, err = tl.Execute(t.Context(), []byte(`{"action":"write","num":2,"title":"第二回","content":"第二回正文。"}`))
	if err != nil || !strings.Contains(out, "第 2 章") {
		t.Fatalf("write = %q, err=%v", out, err)
	}
	// write 重写已有章 → 备份落 .history/
	if _, err := tl.Execute(t.Context(), []byte(`{"action":"write","num":2,"title":"第二回改","content":"改后正文。"}`)); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if _, err := os.Stat(filepath.Join(book, "chapters", ".history", "002-")); err != nil && len(sinListHistory(book, "002")) == 0 {
		t.Fatal("重写备份缺失")
	}
	m, _ := project.Open(book)
	if raw, _ := m.ReadChapter(2); strings.TrimSpace(raw) != "改后正文。" {
		t.Fatalf("rewrite content = %q", raw)
	}
	// 未绑书拒绝
	tn := &sinChapterTool{app: a}
	if _, err := tn.Execute(t.Context(), []byte(`{"action":"list"}`)); err == nil {
		t.Fatal("未绑书应拒绝")
	}
	// 空 content 拒绝
	if _, err := tl.Execute(t.Context(), []byte(`{"action":"write","num":3}`)); err == nil {
		t.Fatal("空 content 应拒绝")
	}
}

// sinListHistory 列备份目录中指定章的备份文件数。
func sinListHistory(book, num string) []string {
	entries, _ := os.ReadDir(filepath.Join(book, "chapters", ".history"))
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), num+"-") {
			out = append(out, e.Name())
		}
	}
	return out
}
