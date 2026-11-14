package builtin

// webfetch_readability.go — readability 正文主提取（P2，2026-11-14；Obscura
// LP.getMarkdown 思路蒸馏，零新依赖）：Mozilla Readability 简化版启发式，从整页
// HTML 选出主内容容器，连同 <title> 重序列化后复用 htmlToText 单一渲染路径。
// 计分=长 <p> 给双亲计分（parent 全额/grandparent 半额）+ class/id 关键词
// 修正 + article/main 加成；判据不足（<3 段、<200 字、提取无净增益 ≥95%）或
// 找不到容器时回退整页旧路径——旧行为永远兜底，只升信噪比不改有则。
// mode=auto（默认；web_fetch 与 web_search 直抓同享）/mode=full 显式整页。

import (
	stdhtml "html"
	"regexp"
	"strings"

	nethtml "golang.org/x/net/html"
)

const (
	readabilityModeAuto = "auto"
	readabilityModeFull = "full"
)

// htmlToReadableText HTML→可读文本统一入口：auto 走 readability 主提取
// （判据不足自动回退整页），full 走整页旧行为。
func htmlToReadableText(s, mode string) string {
	if mode != readabilityModeFull {
		if frag, ok := readabilityExtract(s); ok {
			return htmlToText(frag)
		}
	}
	return htmlToText(s)
}

var (
	readabilityPositive = regexp.MustCompile(`(?i)(article|body|content|entry|hentry|main|page|post|text|blog|story)`)
	readabilityNegative = regexp.MustCompile(`(?i)(banner|combx|comment|contact|foot|masthead|media|meta|outbrain|promo|related|scroll|shoutbox|sidebar|sponsor|shopping|tags|tool|widget|hidden|modal|popup|nav|menu)`)
)

// readabilityStripTags 主容器渲染前摘除的噪声元素（保守清单：不摘 header/
// footer——正文容器内的 <header> 常承载文章标题，摘了丢内容）。
var readabilityStripTags = map[string]bool{
	"script": true, "style": true, "noscript": true, "nav": true, "aside": true,
	"form": true, "iframe": true, "svg": true, "select": true, "button": true,
}

// readabilityExtract 尝试从整页 HTML 提取主内容片段；ok=false 表示判据不足，
// 调用方回退整页渲染。返回片段形如 <html><head><title>…</title></head><body>…。
func readabilityExtract(page string) (string, bool) {
	doc, err := nethtml.Parse(strings.NewReader(page))
	if err != nil {
		return "", false
	}

	scores := map[*nethtml.Node]float64{}
	pageText := 0 // 全部文本节点字数（nav/footer 等噪声通常无 <p>，必须全口径计）
	// 计分趟：长 <p> 给双亲计分（原版 Readability 简化）。
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.TextNode {
			pageText += len([]rune(n.Data))
		} else if n.Type == nethtml.ElementNode && n.Data == "p" {
			r := len([]rune(nodeText(n)))
			if r >= 20 && n.Parent != nil && n.Parent.Type == nethtml.ElementNode {
				scores[n.Parent] += float64(r)
				if gp := n.Parent.Parent; gp != nil && gp.Type == nethtml.ElementNode {
					scores[gp] += float64(r) / 2
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// 选优：排除 html/body（包裹级永远满分但无信息量），关键词修正 + 标签加成。
	best := (*nethtml.Node)(nil)
	bestScore := 0.0
	for n, s := range scores {
		if n.Data == "body" || n.Data == "html" {
			continue
		}
		kw := classIDOf(n)
		if readabilityNegative.MatchString(kw) {
			s *= 0.2
		} else if readabilityPositive.MatchString(kw) {
			s *= 1.5
		}
		if n.Data == "article" || n.Data == "main" {
			s += 500
		}
		if s > bestScore {
			best, bestScore = n, s
		}
	}
	if best == nil {
		return "", false
	}

	// 验收：主容器须有真实正文（≥3 段且 ≥200 字），且提取有净增益（主容器
	// 文本 <95% 页文本——整页包裹容器不值得提，回退旧行为）。
	bestP, bestText := 0, 0
	var count func(*nethtml.Node)
	count = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode && n.Data == "p" {
			bestP++
			bestText += len([]rune(nodeText(n)))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			count(c)
		}
	}
	count(best)
	if bestP < 3 || bestText < 200 || bestText*20 > pageText*19 {
		return "", false
	}

	stripUnwanted(best)

	var b strings.Builder
	b.WriteString("<html><head><title>")
	b.WriteString(stdhtml.EscapeString(readabilityTitle(doc)))
	b.WriteString("</title></head><body>")
	if err := nethtml.Render(&b, best); err != nil {
		return "", false
	}
	b.WriteString("</body></html>")
	return b.String(), true
}

// stripUnwanted 主容器内递归摘除噪声元素（先收集后摘除，避免遍历中改链表）。
func stripUnwanted(n *nethtml.Node) {
	var doomed []*nethtml.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == nethtml.ElementNode && readabilityStripTags[c.Data] {
			doomed = append(doomed, c)
			continue
		}
		stripUnwanted(c)
	}
	for _, d := range doomed {
		if d.Parent != nil {
			d.Parent.RemoveChild(d)
		}
	}
}

// nodeText/attrValue 复用 websearch_html.go 既有 helper（收集文本与属性读取），
// 不重复造轮子。

// classIDOf 元素 class+id 归一关键词串（小写，供正则子串匹配）。
func classIDOf(n *nethtml.Node) string {
	return strings.ToLower(attrValue(n, "class") + " " + attrValue(n, "id"))
}

// readabilityTitle 取 <title> 文本（无则空串）。
func readabilityTitle(doc *nethtml.Node) string {
	var find func(*nethtml.Node) *nethtml.Node
	find = func(n *nethtml.Node) *nethtml.Node {
		if n.Type == nethtml.ElementNode && n.Data == "title" {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if t := find(c); t != nil {
				return t
			}
		}
		return nil
	}
	if t := find(doc); t != nil {
		return strings.TrimSpace(nodeText(t))
	}
	return ""
}
