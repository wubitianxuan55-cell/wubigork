package export

import (
	"fmt"
	"html"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/types"
)

// ── HTML 导出 ────────────────────────────────────────────────

// CompileTemplate 编译模板定义
type CompileTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CSS         string `json:"css"`         // 自定义 CSS
	FontFamily  string `json:"font_family"` // 正文字体
	FontSize    string `json:"font_size"`   // 正文字号
	LineHeight  string `json:"line_height"` // 行高
	PageWidth   string `json:"page_width"`  // 页面宽度
	ChapterSep  string `json:"chapter_sep"` // 章节分隔符
}

// DefaultTemplates 内置编译模板
func DefaultTemplates() []CompileTemplate {
	return []CompileTemplate{
		{
			Name:        "网文阅读",
			Description: "适合屏幕阅读的宽版布局，深色背景",
			CSS:         webNovelCSS,
			FontFamily:  "'PingFang SC', 'Microsoft YaHei', sans-serif",
			FontSize:    "16px",
			LineHeight:  "1.8",
			PageWidth:   "800px",
			ChapterSep:  "<hr class='chapter-sep'/>",
		},
		{
			Name:        "出版审阅",
			Description: "仿纸质书排版，宋体，适合打印",
			CSS:         printCSS,
			FontFamily:  "'Songti SC', 'SimSun', 'Noto Serif CJK SC', serif",
			FontSize:    "14px",
			LineHeight:  "2.0",
			PageWidth:   "650px",
			ChapterSep:  "<div class='page-break'></div>",
		},
		{
			Name:        "极简纯净",
			Description: "极简白色背景，适合复制到其他平台",
			CSS:         minimalCSS,
			FontFamily:  "'Inter', 'PingFang SC', sans-serif",
			FontSize:    "15px",
			LineHeight:  "1.7",
			PageWidth:   "720px",
			ChapterSep:  "<p style='text-align:center;color:#ccc;'>* * *</p>",
		},
	}
}

// ExportHTML 导出为独立 HTML 文件
func (m *Manager) ExportHTML(outPath string, tmpl CompileTemplate) (string, error) {
	var body strings.Builder

	// 封面
	body.WriteString(fmt.Sprintf(`<header class="cover">
<h1>%s</h1>
<p class="meta">题材: %s · 文风: %s</p>
</header>`, html.EscapeString(m.pm.Meta.Title), html.EscapeString(m.pm.Meta.Genre), html.EscapeString(m.pm.Meta.Style)))

	// 世界观
	worldview, err := m.pm.ReadWorldview()
	if err == nil && worldview != "" {
		body.WriteString(`<section class="worldview"><h2>世界观</h2>`)
		body.WriteString(markdownToHTML(worldview))
		body.WriteString(`</section>`)
	}

	// 角色表
	chars, err := m.pm.ReadCharacters()
	if err == nil && chars != nil && len(chars.Characters) > 0 {
		body.WriteString(`<section class="characters"><h2>角色表</h2><div class="char-grid">`)
		for _, ch := range chars.Characters {
			body.WriteString(fmt.Sprintf(`<div class="char-card">
<strong>%s</strong> <em>[%s]</em>
<p>%s</p>
</div>`, html.EscapeString(ch.Name), html.EscapeString(ch.RoleType), html.EscapeString(ch.Personality)))
		}
		body.WriteString(`</div></section>`)
	}

	// 章节
	body.WriteString(`<main class="chapters">`)
	failed := 0
	for _, ch := range m.listChapters() {
		content, err := m.pm.ReadChapterBranch(ch.num, ch.branch)
		if err != nil {
			failed++
			slog.Warn("exportHTML: 读取章节失败，跳过", "num", ch.num, "branch", ch.branch, "error", err)
			continue
		}
		var summary *types.ChapterSummary
		if ch.branch != "" {
			summary, _ = m.pm.ReadChapterBranchSummary(ch.num, ch.branch)
		} else {
			summary, _ = m.pm.ReadChapterSummary(ch.num)
		}

		chapterTitle := fmt.Sprintf("第 %d 章", ch.num)
		if ch.branch != "" {
			chapterTitle += " " + ch.branch
		}
		if summary != nil && summary.Title != "" {
			chapterTitle = summary.Title
		}

		body.WriteString(fmt.Sprintf(`<article class="chapter">
<h2>%s</h2>
<div class="content">%s</div>
</article>`, html.EscapeString(chapterTitle), markdownToHTML(content)))

		body.WriteString(tmpl.ChapterSep)
	}
	m.FailedChapters = failed
	body.WriteString(`</main>`)

	// 组装完整 HTML
	fullHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
:root {
  --font-family: %s;
  --font-size: %s;
  --line-height: %s;
  --page-width: %s;
}
body {
  font-family: var(--font-family);
  font-size: var(--font-size);
  line-height: var(--line-height);
  max-width: var(--page-width);
  margin: 0 auto;
  padding: 2rem 1.5rem;
  color: #1a1a1a;
  background: #fafafa;
}
@media (prefers-color-scheme: dark) {
  body { color: #e5e5e5; background: #1a1a1a; }
}
%s
</style>
</head>
<body>
%s
<footer style="text-align:center;margin-top:4rem;opacity:0.4;font-size:0.8em;">
由 gaea 生成
</footer>
</body>
</html>`, html.EscapeString(m.pm.Meta.Title),
		tmpl.FontFamily, tmpl.FontSize, tmpl.LineHeight, tmpl.PageWidth,
		tmpl.CSS, body.String())

	return fullHTML, nil
}

// markdownBlockKind 行的 markdown 块类型（IN3-13：markdownToHTML 与
// docx.markdownHeading 共用同一判定，原两处各自维护 #/##/### 前缀 switch）。
type markdownBlock int

const (
	blockParagraph markdownBlock = iota // 普通正文
	blockBlank                          // 空行（段落边界）
	blockHeading1                       // "# " 一级标题
	blockHeading2                       // "## " 二级标题
	blockHeading3                       // "### " 三级标题
	blockListItem                       // "- " 列表项
)

// markdownBlockKind 判定一行（须已去首尾空白）的 markdown 块类型，返回类型与
// 剥掉前缀后的文本。前缀按 "### " → "## " → "# " 长者优先；"#### " 等四级
// 及更深前缀不构成标题（落入正文，与历史行为一致）。
func markdownBlockKind(trimmed string) (markdownBlock, string) {
	switch {
	case strings.HasPrefix(trimmed, "### "):
		return blockHeading3, strings.TrimPrefix(trimmed, "### ")
	case strings.HasPrefix(trimmed, "## "):
		return blockHeading2, strings.TrimPrefix(trimmed, "## ")
	case strings.HasPrefix(trimmed, "# "):
		return blockHeading1, strings.TrimPrefix(trimmed, "# ")
	case trimmed == "":
		return blockBlank, ""
	case strings.HasPrefix(trimmed, "- "):
		return blockListItem, strings.TrimPrefix(trimmed, "- ")
	default:
		return blockParagraph, trimmed
	}
}

// markdownToHTML 简单的 Markdown → HTML 转换
func markdownToHTML(md string) string {
	lines := strings.Split(md, "\n")
	var result []string
	inParagraph := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		kind, text := markdownBlockKind(trimmed)
		switch kind {
		case blockHeading3:
			if inParagraph {
				result = append(result, "</p>")
				inParagraph = false
			}
			result = append(result, "<h3>"+html.EscapeString(text)+"</h3>")
		case blockHeading2:
			if inParagraph {
				result = append(result, "</p>")
				inParagraph = false
			}
			result = append(result, "<h2>"+html.EscapeString(text)+"</h2>")
		case blockHeading1:
			if inParagraph {
				result = append(result, "</p>")
				inParagraph = false
			}
			result = append(result, "<h1>"+html.EscapeString(text)+"</h1>")
		case blockBlank:
			if inParagraph {
				result = append(result, "</p>")
				inParagraph = false
			}
		case blockListItem:
			if inParagraph {
				result = append(result, "</p>")
				inParagraph = false
			}
			result = append(result, "<li>"+html.EscapeString(text)+"</li>")
		default:
			if !inParagraph {
				result = append(result, "<p>")
				inParagraph = true
			}
			result = append(result, html.EscapeString(text)+" ")
		}
	}
	if inParagraph {
		result = append(result, "</p>")
	}

	return strings.Join(result, "\n")
}

// ── CSS 模板 ─────────────────────────────────────────────────

const webNovelCSS = `
.cover { text-align:center; padding:3rem 0; border-bottom:2px solid #e5e5e5; margin-bottom:2rem; }
.cover h1 { font-size:2em; margin-bottom:0.5rem; }
.meta { color:#888; font-size:0.9em; }
.worldview, .characters { margin-bottom:2rem; padding:1rem; background:rgba(0,0,0,0.02); border-radius:8px; }
.char-grid { display:grid; grid-template-columns:repeat(auto-fill,minmax(200px,1fr)); gap:0.8rem; }
.char-card { padding:0.8rem; border:1px solid #e5e5e5; border-radius:6px; }
.char-card p { margin:0.3rem 0 0; font-size:0.9em; color:#666; }
.chapter { margin-bottom:2rem; }
.chapter h2 { border-bottom:1px solid #e5e5e5; padding-bottom:0.5rem; }
.content p { text-indent:2em; margin:0.5em 0; }
.chapter-sep { border:none; border-top:3px double #ccc; margin:2rem 0; }
@media (prefers-color-scheme: dark) {
  .worldview, .characters { background:rgba(255,255,255,0.03); }
  .char-card { border-color:#333; }
  .char-card p { color:#999; }
  .cover { border-color:#333; }
  .chapter h2 { border-color:#333; }
}
`

const printCSS = `
.cover { text-align:center; padding:2rem 0; page-break-after:always; }
.cover h1 { font-size:1.8em; }
.worldview, .characters { margin-bottom:1.5rem; }
.char-grid { display:block; }
.char-card { margin-bottom:0.5rem; padding:0.3rem 0; border-bottom:1px dotted #ccc; }
.char-card p { display:inline; margin-left:0.5em; }
.chapter { page-break-before:always; }
.chapter h2 { text-align:center; }
.content p { text-indent:2em; margin:0.3em 0; }
.page-break { page-break-before:always; }
@media print {
  body { font-size:12pt; }
  .cover { page-break-after:always; }
}
`

const minimalCSS = `
.cover { text-align:center; padding:2rem 0; }
.cover h1 { font-size:1.6em; font-weight:600; }
.meta { color:#999; }
.worldview, .characters { margin:1.5rem 0; }
.char-grid { display:block; }
.char-card { display:inline-block; margin:0.2rem 1rem 0.2rem 0; }
.char-card p { display:none; }
.chapter h2 { font-weight:600; }
.content p { margin:0.6em 0; }
`
