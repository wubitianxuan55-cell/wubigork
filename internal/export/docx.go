package export

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/carmel/gooxml/document"
)

// ── DOCX 导出 ────────────────────────────────────────────────
//
// 基于 gooxml（github.com/carmel/gooxml，AGPL，仓库既有依赖）从零生成简单
// Word 文档：书名（Title 样式）+ 元信息行 + 世界观 + 章节标题（Heading1）+
// 正文段落（每行一段，markdown 标题行映射 Heading2）。不设复杂排版/图片。

// ExportDOCX 导出为 Word 文档（.docx）
func (m *Manager) ExportDOCX(outPath string) (string, error) {
	doc := document.New()

	addPara := func(style, text string) {
		p := doc.AddParagraph()
		if style != "" {
			p.SetStyle(style)
		}
		if text != "" {
			p.AddRun().AddText(text)
		}
	}

	addPara("Title", m.pm.Meta.Title)
	addPara("", fmt.Sprintf("题材: %s  文风: %s", m.pm.Meta.Genre, m.pm.Meta.Style))

	// 世界观（与 TXT / Markdown / EPUB 对齐）
	worldview, err := m.pm.ReadWorldview()
	if err != nil {
		slog.Warn("exportDOCX: 读取世界观失败", "error", err)
	}
	if worldview != "" {
		addPara("Heading1", "世界观")
		for _, line := range contentLines(worldview) {
			addPara("", line)
		}
	}

	// 各章节（章节遍历与分支过滤同 TXT/Markdown/EPUB）
	failed := 0
	for _, ch := range m.listChapters() {
		content, err := m.pm.ReadChapterBranch(ch.num, ch.branch)
		if err != nil {
			failed++
			slog.Warn("exportDOCX: 读取章节失败，跳过", "num", ch.num, "branch", ch.branch, "error", err)
			continue
		}
		label := fmt.Sprintf("第 %d 章", ch.num)
		if ch.branch != "" {
			label += " " + ch.branch
		}
		addPara("Heading1", label)
		for _, line := range contentLines(content) {
			if text, ok := markdownHeading(line); ok {
				addPara("Heading2", text)
			} else {
				addPara("", line)
			}
		}
	}
	m.FailedChapters = failed

	if err := ensureParentDir(outPath); err != nil {
		return "", fmt.Errorf("创建导出目录失败: %w", err)
	}
	if err := doc.SaveToFile(outPath); err != nil {
		return "", fmt.Errorf("写入 DOCX 失败: %w", err)
	}
	return outPath, nil
}

// contentLines 返回文本中逐行去空白后的非空行，每行对应一个 DOCX 段落
// （正文以空行分段时与 markdownToHTML 的段落语义一致）。
func contentLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// markdownHeading 识别 markdown 标题行（#/##/### 带空格前缀，与 markdownToHTML
// 的判定一致），返回去掉井号后的标题文本。列表行等其余内容按正文段落处理。
func markdownHeading(line string) (string, bool) {
	switch {
	case strings.HasPrefix(line, "### "):
		return strings.TrimPrefix(line, "### "), true
	case strings.HasPrefix(line, "## "):
		return strings.TrimPrefix(line, "## "), true
	case strings.HasPrefix(line, "# "):
		return strings.TrimPrefix(line, "# "), true
	}
	return "", false
}
