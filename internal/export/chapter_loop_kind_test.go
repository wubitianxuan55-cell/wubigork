package export

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// ── IN3-13：markdownBlockKind 共用判定 ─────────────────────────

// TestMarkdownBlockKind_Table 钉死块类型判定：前缀按 "### "→"## "→"# " 长者优先，
// 四级及以上不构成标题；输入须为已去首尾空白的行（html/docx 两调用方均如此）。
func TestMarkdownBlockKind_Table(t *testing.T) {
	cases := []struct {
		line string
		kind markdownBlock
		text string
	}{
		{"### 三级", blockHeading3, "三级"},
		{"## 二级", blockHeading2, "二级"},
		{"# 一级", blockHeading1, "一级"},
		{"", blockBlank, ""},
		{"- 列表项", blockListItem, "列表项"},
		{"正文段落。", blockParagraph, "正文段落。"},
		{"#### 四级不算标题", blockParagraph, "#### 四级不算标题"},
		{"#无空格不算", blockParagraph, "#无空格不算"},
		{"#", blockParagraph, "#"},
		{"-", blockParagraph, "-"},
		{"###", blockParagraph, "###"},
		{"##", blockParagraph, "##"},
		{"# 一级 # 尾部井号", blockHeading1, "一级 # 尾部井号"},
	}
	for _, c := range cases {
		kind, text := markdownBlockKind(c.line)
		if kind != c.kind || text != c.text {
			t.Errorf("markdownBlockKind(%q) = (%v, %q), want (%v, %q)",
				c.line, kind, text, c.kind, c.text)
		}
	}
}

// TestMarkdownHeading_MatchesDocmdEmission 对照钉：internal/docmd/office.go 的
// extractDocxParagraph 将 Word 样式 Title/Heading1/2/3 分别发射为 "# "/"## "
// "/### " 前缀行（Heading4+ 钳到 ≤6 级井号）。export.markdownHeading 只回收
// 1-3 级；本用例把 docmd 的发射词汇逐一喂给 markdownHeading，钉住「导出侧写、
// 导入侧读」的双向契约（docmd 属另一并行线足迹，本批不改其源码，仅对照）。
func TestMarkdownHeading_MatchesDocmdEmission(t *testing.T) {
	for _, emitted := range []string{"# 书名", "## 二级", "### 三级"} {
		if text, ok := markdownHeading(emitted); !ok || text == "" {
			t.Errorf("markdownHeading(%q) = (%q, %v), 应识别为标题", emitted, text, ok)
		}
	}
	for _, para := range []string{"#### 四级", "正文行", "- 列表", ""} {
		if _, ok := markdownHeading(para); ok {
			t.Errorf("markdownHeading(%q) 不应识别为标题（docmd Heading4+ 发射为多井号，导出侧按正文段落回收）", para)
		}
	}
}

// ── IN3-12：forEachChapter 循环骨架 ────────────────────────────

// TestForEachChapter_CountsLabelsOrderAndNoFailedWrite 计数、label 拼接、
// 排序（主线在前分支按字母序）与「不代写 m.FailedChapters」（EPUB 中途失败
// 保留旧计数的历史行为由调用方赋值保障）。
func TestForEachChapter_CountsLabelsOrderAndNoFailedWrite(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "循环", Genre: "玄幻", Style: "热血"},
		map[string]string{
			"002.md":  "第二章。",
			"001.md":  "第一章。",
			"002a.md": "分支甲。",
		}, "")
	link := filepath.Join(pm.Dir, "chapters", "003b.md")
	if err := os.Symlink(filepath.Join(pm.Dir, "chapters", "003b-missing.md"), link); err != nil {
		t.Skipf("当前环境无法创建符号链接，跳过失败分支用例: %v", err)
	}
	m := New(pm)
	var labels []string
	var contents []string
	failed, err := m.forEachChapter("exportPROBE", func(ch chapterEntry, label, content string) error {
		labels = append(labels, label)
		contents = append(contents, content)
		return nil
	})
	if err != nil {
		t.Fatalf("forEachChapter: %v", err)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1", failed)
	}
	if got, want := strings.Join(labels, "|"), "第 1 章|第 2 章|第 2 章 a"; got != want {
		t.Errorf("labels = %q, want %q（主线在前、分支按字母序）", got, want)
	}
	if contents[0] != "第一章。" {
		t.Errorf("content[0] = %q, want 第一章。", contents[0])
	}
	if m.FailedChapters != 0 {
		t.Errorf("forEachChapter 不得代写 m.FailedChapters，got %d", m.FailedChapters)
	}
}

// TestForEachChapter_CallbackErrorAborts 回调错误立即中止并透传。
func TestForEachChapter_CallbackErrorAborts(t *testing.T) {
	pm := newTestProject(t, types.ProjectMeta{Title: "中止", Genre: "都市", Style: "写实"},
		map[string]string{"001.md": "一。", "002.md": "二。", "003.md": "三。"}, "")
	m := New(pm)
	sentinel := errors.New("sentinel")
	calls := 0
	failed, err := m.forEachChapter("exportPROBE", func(chapterEntry, string, string) error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want sentinel", err)
	}
	if calls != 1 {
		t.Errorf("回调应首章即止，calls = %d", calls)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0（首章即成功后中止）", failed)
	}
}
