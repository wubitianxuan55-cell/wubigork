package builtin

// websearch_html.go — websearch 的 HTML 结果解析（批 25 GA2-01 文件拆分，自
// websearch.go 原位搬移，零逻辑改动）：Bing/DDG Lite 无 key 回退引擎的 SERP
// 解析 + nethtml 节点工具（hasClass/attrValue/nodeText/firstChildElement/
// firstDescendantElement）。引擎侧 HTTP 扇出见 websearch_engine.go。

import (
	"bytes"
	"fmt"
	"strings"

	nethtml "golang.org/x/net/html"
)

// parseBingResults extracts organic results from a Bing SERP: <li class="b_algo">
// blocks with the title in <h2><a> and the snippet inside <div class="b_caption">.
func parseBingResults(body []byte, limit int) ([]SearchResult, error) {
	doc, err := nethtml.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse bing response: %w", err)
	}
	var results []SearchResult
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if len(results) >= limit {
			return
		}
		if n.Type == nethtml.ElementNode && n.Data == "li" && hasClass(n, "b_algo") {
			if r, ok := bingResultFromBlock(n); ok {
				results = append(results, r)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(results) == 0 {
		return nil, fmt.Errorf("no bing results parsed")
	}
	return results, nil
}

func bingResultFromBlock(li *nethtml.Node) (SearchResult, bool) {
	var title, href, snippet string
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if href != "" && snippet != "" {
			return
		}
		if n.Type == nethtml.ElementNode && n.Data == "h2" && href == "" {
			if a := firstChildElement(n, "a"); a != nil {
				href = strings.TrimSpace(attrValue(a, "href"))
				title = strings.TrimSpace(nodeText(a))
			}
		}
		if n.Type == nethtml.ElementNode && n.Data == "div" && hasClass(n, "b_caption") && snippet == "" {
			if p := firstDescendantElement(n, "p"); p != nil {
				snippet = strings.TrimSpace(nodeText(p))
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(li)
	if title == "" || href == "" || strings.HasPrefix(href, "javascript:") {
		return SearchResult{}, false
	}
	return SearchResult{Title: title, URL: href, Snippet: truncate(snippet, 300), Source: "bing"}, true
}

// parseDDGLiteResults extracts results from DuckDuckGo Lite: <a class="result-link">
// entries with snippets in <td class="result-snippet"> cells.
func parseDDGLiteResults(body []byte, limit int) ([]SearchResult, error) {
	doc, err := nethtml.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse duckduckgo response: %w", err)
	}
	var results []SearchResult
	var snippets []string
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode && n.Data == "a" && hasClass(n, "result-link") && len(results) < limit {
			href := strings.TrimSpace(attrValue(n, "href"))
			title := strings.TrimSpace(nodeText(n))
			if href != "" && title != "" {
				results = append(results, SearchResult{Title: title, URL: href, Source: "duckduckgo"})
			}
		}
		if n.Type == nethtml.ElementNode && n.Data == "td" && hasClass(n, "result-snippet") {
			snippets = append(snippets, strings.TrimSpace(nodeText(n)))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(results) == 0 {
		return nil, fmt.Errorf("no duckduckgo results parsed")
	}
	for i := range results {
		if i < len(snippets) {
			results[i].Snippet = truncate(snippets[i], 300)
		}
	}
	return results, nil
}

// --- HTML helpers ---

func hasClass(n *nethtml.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key != "class" {
			continue
		}
		for _, c := range strings.Fields(a.Val) {
			if c == class {
				return true
			}
		}
	}
	return false
}

func attrValue(n *nethtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func nodeText(n *nethtml.Node) string {
	var sb strings.Builder
	var walk func(*nethtml.Node)
	walk = func(m *nethtml.Node) {
		if m.Type == nethtml.TextNode {
			sb.WriteString(m.Data)
		}
		for c := m.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

func firstChildElement(n *nethtml.Node, tag string) *nethtml.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == nethtml.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

func firstDescendantElement(n *nethtml.Node, tag string) *nethtml.Node {
	if n.Type == nethtml.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if r := firstDescendantElement(c, tag); r != nil {
			return r
		}
	}
	return nil
}
