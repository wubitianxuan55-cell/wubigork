package booksource

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ── 目录面（规格 §2.6：分页下拉/锚点一次性取全、按页序归并、倒序源、范围）──

var ErrEmptyToc = errors.New("章节目录为空（源站可能反爬或规则失配）")

// TocEntry 目录条目；Order 为 1 起的章节序。
type TocEntry struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Order int    `json:"order"`
}

// bookID 从详情 URL 捕获书 id（book.url 正则组 1，规格 §2.6）。
func (e *Engine) bookID(detailURL string) string {
	b := e.rule.Book
	if b == nil || b.URL == "" {
		return ""
	}
	re, err := regexp.Compile(b.URL) // 校验层已把关，这里防御
	if err != nil {
		return ""
	}
	if m := re.FindStringSubmatch(detailURL); len(m) >= 2 {
		return m[1]
	}
	return ""
}

// tocPageURLs 目录分页集合：首页 + next 选择器命中的下拉/锚点集合。
// 后写入的 URL 覆盖既有位置并保持顺序（上游 TocParser 同语义：个别书源
// toc.url 不一定是首个 option，规格 §2.6）。首页文档同步返回（分页发现顺带
// 取回，免重复抓取）；其余页文档槽位为 nil，由调用方并发补抓。
func (e *Engine) tocPageURLs(ctx context.Context, firstURL string) ([]string, []*goquery.Document, error) {
	pages := []string{firstURL}
	seen := map[string]int{firstURL: 0}
	docs := []*goquery.Document{nil}
	doc, pageURL, err := e.pacedFetch(ctx, Request{URL: firstURL})
	if err != nil {
		return nil, nil, err
	}
	docs[0] = doc
	if t := e.rule.Toc; t != nil && t.NextPage != "" {
		for _, u := range absLinks(doc.Selection, t.NextPage, pageURL) {
			if i, ok := seen[u]; ok {
				// 已存在：挪到队尾（后写覆盖语义）
				pages = append(pages[:i], pages[i+1:]...)
				docs = append(docs[:i], docs[i+1:]...)
			}
			seen[u] = len(pages)
			pages = append(pages, u)
			docs = append(docs, nil)
		}
	}
	return pages, docs, nil
}

// tocItemsOf 单页目录的章节条目（list 容器二跳解析，规格 §2.6）。
func (e *Engine) tocItemsOf(doc *goquery.Document, pageURL string) []TocEntry {
	t := e.rule.Toc
	if t == nil {
		return nil
	}
	scope := doc.Selection
	if t.List != "" {
		h := htmlOf(doc.Selection, t.List)
		if h == "" {
			return nil
		}
		inner, err := goquery.NewDocumentFromReader(strings.NewReader("<div>" + h + "</div>"))
		if err != nil {
			return nil
		}
		scope = inner.Find("div").First()
	}
	var out []TocEntry
	scope.Find(t.Item).Each(func(_ int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Text())
		href, _ := s.Attr("href")
		u := resolveLink(pageURL, href)
		if title == "" || u == "" {
			return
		}
		out = append(out, TocEntry{Title: title, URL: u})
	})
	return out
}

// Toc 解析目录；start/end 为 1 起闭区间（end<=0 表示到末卷）。
func (e *Engine) Toc(ctx context.Context, detailURL string, start, end int) ([]TocEntry, error) {
	t := e.rule.Toc
	if t == nil || strings.TrimSpace(t.Item) == "" {
		return nil, errors.New("书源缺少 toc 段或 toc.item")
	}
	firstURL := detailURL
	if id := e.bookID(detailURL); id != "" && t.URL != "" {
		firstURL = strings.ReplaceAll(t.URL, "%s", id)
	}
	pageURLs, docs, err := e.tocPageURLs(ctx, firstURL)
	if err != nil {
		return nil, err
	}
	if len(pageURLs) > 1 {
		idx := make([]int, 0, len(pageURLs))
		for i, u := range pageURLs {
			_ = u
			if docs[i] == nil {
				idx = append(idx, i)
			}
		}
		_ = e.runBounded(ctx, len(idx), maxPageConcurrency, func(k int) error {
			i := idx[k]
			doc, _, err := e.pacedFetch(ctx, Request{URL: pageURLs[i]})
			if err != nil {
				return nil // 单页失败降级（上游同语义），页序占位保持
			}
			docs[i] = doc
			return nil
		})
	}
	var toc []TocEntry
	for i, u := range pageURLs {
		if docs[i] == nil {
			continue
		}
		toc = append(toc, e.tocItemsOf(docs[i], u)...)
	}
	if t.Reverse { // 倒序源反转后仍是「第 1 章在前」
		for i, j := 0, len(toc)-1; i < j; i, j = i+1, j-1 {
			toc[i], toc[j] = toc[j], toc[i]
		}
	}
	if len(toc) == 0 {
		return nil, ErrEmptyToc
	}
	if start < 1 {
		start = 1
	}
	if end <= 0 || end > len(toc) {
		end = len(toc)
	}
	if start > end {
		return nil, fmt.Errorf("章节范围 [%d,%d] 越界（共 %d 章）", start, end, len(toc))
	}
	toc = toc[start-1 : end]
	for i := range toc {
		toc[i].Order = i + 1
	}
	return toc, nil
}
