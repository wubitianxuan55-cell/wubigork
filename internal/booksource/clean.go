package booksource

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ── 清洗链（规格 §2.8，顺序敏感）+ 排版归一（§2.9）────────────────────────

var (
	// 不可见字符：控制/格式/私有区 PUA/行区/段分隔/ZWSP/BOM。
	// PUA 私有区字符是盗版站正文「中文乱码」的根源，必须先清（规格 §2.8 第 1 条）。
	invisibleChars = regexp.MustCompile(`[\p{C}\p{Cf}\p{Co}\p{Zl}\p{Zp}\x{200B}\x{FEFF}]`)
	// HTML 实体引用全删（阅读器兼容，上游同语义：宁缺勿错）。
	htmlEntity = regexp.MustCompile(`&[^;]+;`)
	// 空白串（「去空格标题」的重复标题剥离：全删，对齐上游 cleanBlank 语义）。
	blankAll = regexp.MustCompile(`\s+`)
)

// cleanInvisibleChars 不可见字符清理。
func cleanInvisibleChars(text string) string {
	return invisibleChars.ReplaceAllString(text, "")
}

// removeFilterTags 元素连内容整删（上游 jsoup .remove() 同语义）。
func removeFilterTags(html, filterTag string) string {
	if strings.TrimSpace(filterTag) == "" {
		return html
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}
	doc.Find(filterTag).Remove()
	return bodyHTML(doc, html)
}

// voidElements 自闭合/无内容的结构标签，空内容清删时必须豁免（<br> 是段落结构）。
var voidElements = map[string]bool{
	"br": true, "img": true, "hr": true, "input": true, "meta": true, "link": true, "area": true, "source": true,
}

// removeEmptyTags 清删无内容的非空转义标签（如 <p></p>、<span></span>）。
func removeEmptyTags(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}
	changed := false
	doc.Find("*").Each(func(_ int, el *goquery.Selection) {
		if voidElements[goquery.NodeName(el)] {
			return
		}
		if el.Children().Length() == 0 && strings.TrimSpace(el.Text()) == "" {
			el.Remove()
			changed = true
		}
	})
	if !changed {
		return html
	}
	return bodyHTML(doc, html)
}

// bodyHTML 序列化 body 内容；失败回落原串。
func bodyHTML(doc *goquery.Document, fallback string) string {
	body := doc.Find("body").First()
	if body.Length() == 0 {
		return fallback
	}
	h, err := body.Html()
	if err != nil {
		return fallback
	}
	return h
}

// stripLeadingTitle 剥离正文开头的重复标题。标题先 QuoteMeta——章节名里的
// 「[ ( .」等元字符不得破坏正则（规格 §2.8 第 4 条）。捕获组 1 = 开头的空白/标签
// 前缀，替换保留前缀丢弃标题本体。
func stripLeadingTitle(content, title string) string {
	if title == "" {
		return content
	}
	spaceless := blankAll.ReplaceAllString(title, "")
	re, err := regexp.Compile(`^((?:\s|<[^>]+>)*)(?:` + regexp.QuoteMeta(title) + `|` + regexp.QuoteMeta(spaceless) + `)`)
	if err != nil {
		return content
	}
	return re.ReplaceAllString(content, "$1")
}

// normalizeTitle 「12.章节名」→「第12章 章节名」（阅读器目录解析兼容，规格 §2.8 第 4 条）。
func normalizeTitle(title string) string {
	if m := titleNumber.FindStringSubmatch(strings.TrimSpace(title)); m != nil {
		return "第" + m[1] + "章 " + strings.TrimSpace(m[2])
	}
	return strings.TrimSpace(title)
}

// cleanChapterHTML 清洗链主入口（顺序对齐规格 §2.8）：
// 不可见字符 → 实体 → 广告正则 → 整删标签 → 开头标题剥离（原题 + 去序号变体）→ 空标签。
func cleanChapterHTML(contentHTML, filterTxt, filterTag, title string) string {
	out := cleanInvisibleChars(contentHTML)
	out = htmlEntity.ReplaceAllString(out, "")
	if filterTxt != "" {
		if re, err := regexp.Compile(filterTxt); err == nil { // 校验层已把关，这里防御
			out = re.ReplaceAllString(out, "")
		}
	}
	out = removeFilterTags(out, filterTag)
	out = stripLeadingTitle(out, title)
	// 页面标题带「1.」序号而正文开头不带时，再剥去序号变体（如
	// 「1.第一章 起身」→ 剥「第一章 起身」）。
	if m := titleNumber.FindStringSubmatch(strings.TrimSpace(title)); m != nil {
		out = stripLeadingTitle(out, strings.TrimSpace(m[2]))
	}
	return removeEmptyTags(out)
}

// paragraphsFromHTML 排版归一（规格 §2.9）并输出段落文本行。
// 闭合标签源（paragraphTagClosed=true）：内容根里已有 <p> 直接逐段取文本；
// 没有 <p> 的（<div>/<span> 当段落用）按直接子级逐块取文本——等价于上游
// 「非 p 闭合标签统一改名 <p>」的文本效果。
// 非闭合源：按 paragraphTag 正则切段逐段成段。
// 偏离上游（规格 §4 D3）：Go RE2 无反向引用，不用上游的正则改名路线。
func paragraphsFromHTML(contentHTML string, closed bool, paragraphTag string) []string {
	var paras []string
	if closed {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader("<div>" + contentHTML + "</div>"))
		if err != nil {
			return nil
		}
		root := doc.Find("div").First()
		ps := root.Find("p")
		if ps.Length() > 0 {
			ps.Each(func(_ int, s *goquery.Selection) {
				if t := strings.TrimSpace(s.Text()); t != "" {
					paras = append(paras, t)
				}
			})
			return paras
		}
		// 无 <p> 的闭合源：按上游「非 <p> 闭合标签统一改名 <p>」的等价做法，把直接
		// 子级（**含文本节点**——只有 Contents 含非元素节点）按文档序各自成段。只认
		// Children() 的 #text 会把 <div> 逐段成句的源压成一整段（正文黏连）。
		root.Contents().Each(func(_ int, s *goquery.Selection) {
			switch goquery.NodeName(s) {
			case "#comment", "script", "style":
				return
			}
			for _, line := range strings.Split(s.Text(), "\n") {
				if t := strings.TrimSpace(line); t != "" {
					paras = append(paras, t)
				}
			}
		})
		if len(paras) > 0 {
			return paras
		}
		// 整段纯文本（无块级结构）：整体成段。
		if t := strings.TrimSpace(root.Text()); t != "" {
			for _, line := range strings.Split(t, "\n") {
				if line = strings.TrimSpace(line); line != "" {
					paras = append(paras, line)
				}
			}
		}
		return paras
	}
	if strings.TrimSpace(paragraphTag) == "" {
		paragraphTag = `<br\s*/?>`
	}
	re, err := regexp.Compile(paragraphTag)
	if err != nil { // 校验层已把关，这里防御
		re = regexp.MustCompile(`<br\s*/?>`)
	}
	for _, seg := range re.Split(contentHTML, -1) {
		t := seg
		if strings.Contains(t, "<") { // 段内残留标签（无 <br> 的规则失配页）：取文本防泄漏
			if d, err := goquery.NewDocumentFromReader(strings.NewReader(t)); err == nil {
				t = d.Text()
			}
		}
		t = strings.TrimSpace(cleanInvisibleChars(htmlEntity.ReplaceAllString(t, "")))
		if t != "" {
			paras = append(paras, t)
		}
	}
	return paras
}
