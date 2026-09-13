package booksource

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ── 声明式抽取（规格 §2.2；goquery/cascadia，CSS only）───────────────────

// docPageURL 页面基准 URL：goquery 从 reader 构造时 Url 为空，取 fetch 时的 URL。
func resolveLink(pageURL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if pageURL == "" {
		return ref
	}
	base, err := url.Parse(pageURL)
	if err != nil {
		return ref
	}
	r, err := base.Parse(ref)
	if err != nil {
		return ref
	}
	return r.String()
}

// textOf 取首个命中文本的收拢文本（对齐上游 ContentType.TEXT）。
func textOf(s *goquery.Selection, sel string) string {
	return strings.TrimSpace(s.Find(sel).First().Text())
}

// htmlOf 取首个命中的子 HTML（对齐 ContentType.HTML，正文抽取用它保留段落结构）。
func htmlOf(s *goquery.Selection, sel string) string {
	h, _ := s.Find(sel).First().Html()
	return h
}

// attrOf 取首个命中的属性并相对 pageURL 补全；attr 缺省时对元素取 href
// （搜索结果详情链接缺省路线，对齐上游「bookName 元素取 href」）。
func attrOf(s *goquery.Selection, sel, attr, pageURL string) string {
	el := s.Find(sel).First()
	if el.Length() == 0 {
		return ""
	}
	if attr == "" {
		attr = "href"
	}
	v, ok := el.Attr(attr)
	if !ok {
		return ""
	}
	return resolveLink(pageURL, v)
}

// absLinks 收拢选择器命中的全部链接（绝对化、去重保序、可选覆盖基准），
// 目录/搜索分页「锚点集合一次性取全」的公共底座（规格 §2.4/§2.6）。
func absLinks(s *goquery.Selection, sel, pageURL string) []string {
	seen := map[string]bool{}
	var out []string
	s.Find(sel).Each(func(_ int, el *goquery.Selection) {
		href, ok := el.Attr("href")
		if !ok {
			href, _ = el.Attr("value") // 目录分页下拉菜单 option 的常见承载
		}
		u := resolveLink(pageURL, href)
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, u)
	})
	return out
}
