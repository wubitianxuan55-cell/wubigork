package booksource

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// ── 搜索面（规格 §2.4 单源 / §2.5 相似度过滤排序）─────────────────────────

var ErrSearchUnsupported = errors.New("书源不支持搜索（缺 search 段或已禁用）")

// SearchResult 单条搜索结果；Similarity 由聚合排序填充。
type SearchResult struct {
	Source         string  `json:"source"`
	SourceDisabled bool    `json:"-"`
	URL            string  `json:"url"`
	BookName       string  `json:"bookName"`
	Author         string  `json:"author,omitempty"`
	Category       string  `json:"category,omitempty"`
	LatestChapter  string  `json:"latestChapter,omitempty"`
	LastUpdateTime string  `json:"lastUpdateTime,omitempty"`
	Status         string  `json:"status,omitempty"`
	WordCount      string  `json:"wordCount,omitempty"`
	Similarity     float64 `json:"-"`
}

// splitAttrSuffix 拆「selector@attr」后缀（Link 等字段用）。
func splitAttrSuffix(q string) (string, string) {
	if m := attrSuffix.FindString(q); m != "" {
		return strings.TrimSuffix(q, m), m[1:]
	}
	return q, ""
}

func (e *Engine) searchLimit() int {
	if s := e.rule.Search; s != nil && s.Limit > 0 {
		return s.Limit
	}
	return DefaultSearchLimit
}

// Search 单源搜索：URL %s 填关键字 → 抓取解析 → 分页集合一次性抓全 → 截 limit。
// GET 关键字做 URL 转义（比上游 fmt.Sprintf 直填更稳：关键字含 &/% 不炸 URL）。
func (e *Engine) Search(ctx context.Context, keyword string) ([]SearchResult, error) {
	s := e.rule.Search
	if s == nil || e.rule.Disabled {
		return nil, ErrSearchUnsupported
	}
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, errors.New("搜索关键字为空")
	}
	isPost := strings.ToLower(s.Method) == "post"
	// GET：关键字做 URL 转义（比上游 fmt.Sprintf 直填更稳，关键字含 &/% 不炸 URL）；
	// POST：URL 槽位填原文，表单值由表单编码转义。
	searchURL := strings.ReplaceAll(s.URL, "%s", url.QueryEscape(keyword))
	req := Request{URL: searchURL, Cookies: s.Cookies}
	if isPost {
		searchURL = strings.ReplaceAll(s.URL, "%s", keyword)
		req.URL = searchURL
		req.Method = "post"
		form := url.Values{}
		for k, v := range s.Data {
			if v == "%s" {
				v = keyword
			}
			form.Set(k, v)
		}
		req.Form = form
	}
	doc, pageURL, err := e.pacedFetch(ctx, req)
	if err != nil {
		return nil, err
	}
	results := e.parseSearchPage(doc, pageURL)

	// 分页：锚点/下拉集合一次性取全（首页已抓，不含首页），有界并发按序解析。
	if s.NextPage != "" {
		if pages := absLinks(doc.Selection, s.NextPage, pageURL); len(pages) > 0 {
			extra := make([][]SearchResult, len(pages))
			_ = e.runBounded(ctx, len(pages), maxPageConcurrency, func(i int) error {
				pdoc, purl, err := e.pacedFetch(ctx, Request{URL: pages[i], Cookies: s.Cookies})
				if err != nil {
					return nil // 单页失败降级，不拖垮整源（上游 parallelStream 同语义）
				}
				extra[i] = e.parseSearchPage(pdoc, purl)
				return nil
			})
			for _, list := range extra {
				results = append(results, list...)
			}
		}
	}
	if len(results) > e.searchLimit() {
		results = results[:e.searchLimit()]
	}
	return results, nil
}

// parseSearchPage 解析一页搜索结果；结果行全空且详情书名可解析时，按「完全
// 匹配直接跳详情页」构造单条结果（规格 §2.4）。
func (e *Engine) parseSearchPage(doc *goquery.Document, pageURL string) []SearchResult {
	s := e.rule.Search
	rows := doc.Find(s.Result)
	var out []SearchResult
	rows.Each(func(_ int, row *goquery.Selection) {
		name := textOf(row, s.BookName)
		if name == "" {
			return
		}
		sr := SearchResult{Source: e.rule.Name, BookName: name, URL: searchLink(row, s, e.base(pageURL))}
		if s.Author != "" {
			sr.Author = textOf(row, s.Author)
		}
		if s.Category != "" {
			sr.Category = textOf(row, s.Category)
		}
		if s.LatestChapter != "" {
			sr.LatestChapter = textOf(row, s.LatestChapter)
		}
		if s.LastUpdateTime != "" {
			sr.LastUpdateTime = textOf(row, s.LastUpdateTime)
		}
		if s.Status != "" {
			sr.Status = textOf(row, s.Status)
		}
		if s.WordCount != "" {
			sr.WordCount = textOf(row, s.WordCount)
		}
		out = append(out, sr)
	})
	if len(out) == 0 && e.rule.Book != nil && e.rule.Book.BookName != "" {
		if b := e.parseBookDetail(doc, pageURL); b.Name != "" {
			out = append(out, SearchResult{
				Source:   e.rule.Name,
				URL:      e.base(pageURL),
				BookName: b.Name,
				Author:   b.Author,
				Category: b.Category,
			})
		}
	}
	return out
}

// searchLink 详情链接：显式 link 优先，缺省对 bookName 元素取 href（上游同语义）。
func searchLink(row *goquery.Selection, s *SearchRule, pageURL string) string {
	if s.Link != "" {
		sel, attr := splitAttrSuffix(s.Link)
		if attr == "" {
			attr = "href"
		}
		return attrOf(row, sel, attr, pageURL)
	}
	return attrOf(row, s.BookName, "href", pageURL)
}

// ── 相似度过滤排序（规格 §2.5；算法与加权常数重推导，规格 §4 D1）────────────

// similarity = 1 − Levenshtein/maxLen（rune 计）。
func similarity(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 && len(rb) == 0 {
		return 1
	}
	if len(ra) == 0 || len(rb) == 0 {
		return 0
	}
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := 0; j <= len(rb); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, min(cur[j-1]+1, prev[j-1]+cost))
		}
		prev, cur = cur, prev
	}
	m := len(ra)
	if len(rb) > m {
		m = len(rb)
	}
	return 1 - float64(prev[len(rb)])/float64(m)
}

// similarityBoost 短关键字（≤4 字）对完全命中重加权、长关键字（≥10 字）降权：
// 「搜作者名」这种短精确串必须压过一堆书名近似命中（机制对齐，常数重推导）。
func similarityBoost(s float64, kwLen int) float64 {
	if s >= 1 {
		switch {
		case kwLen <= 4:
			return 12
		case kwLen >= 10:
			return 3
		default:
			return 6
		}
	}
	return s
}

// FilterSortBySimilarity 聚合结果的过滤排序：自动判别关键字是书名还是作者
// （两边相似度总量各加权取胜方），按胜方相似度降序，>0.25 过滤，全滤空回退
// >0；并列按另一字段字典序。就地填充 Similarity。
func FilterSortBySimilarity(results []SearchResult, keyword string) []SearchResult {
	if len(results) == 0 {
		return results
	}
	kw := strings.TrimSpace(keyword)
	kwLen := len([]rune(kw))
	bookW, authorW := 0.0, 0.0
	for i := range results {
		results[i].Similarity = similarity(kw, results[i].BookName)
		bookW += similarityBoost(results[i].Similarity, kwLen)
		authorW += similarityBoost(similarity(kw, results[i].Author), kwLen)
	}
	authorSearch := authorW > bookW
	chosen := func(r SearchResult) float64 {
		if authorSearch {
			return similarity(kw, r.Author)
		}
		return r.Similarity
	}
	other := func(r SearchResult) string {
		if authorSearch {
			return r.BookName
		}
		return r.Author
	}
	sort.SliceStable(results, func(i, j int) bool {
		si, sj := chosen(results[i]), chosen(results[j])
		if si != sj {
			return si > sj
		}
		return other(results[i]) < other(results[j])
	})
	filtered := make([]SearchResult, 0, len(results))
	for _, r := range results {
		if chosen(r) > similarityKeep {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) > 0 {
		copy(results, filtered)
		return results[:len(filtered)]
	}
	kept := results[:0]
	for _, r := range results {
		if chosen(r) > 0 {
			kept = append(kept, r)
		}
	}
	return kept
}

// ── 聚合搜索（规格 §2.4：书源并发、单源失败降级）──────────────────────────

// SourceError 单书源失败（聚合时单独上报，不拖垮整体）。
type SourceError struct {
	Source string
	Err    error
}

// Aggregate 全书源并发搜索 + 相似度过滤排序。
func Aggregate(ctx context.Context, opt Options, keyword string, rules ...*Rule) ([]SearchResult, []SourceError) {
	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		out  []SearchResult
		errs []SourceError
	)
	for _, r := range rules {
		if r == nil || r.Disabled || r.Search == nil {
			continue
		}
		wg.Add(1)
		go func(r *Rule) {
			defer wg.Done()
			res, err := New(r, opt).Search(ctx, keyword)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, SourceError{Source: r.Name, Err: err})
				return
			}
			out = append(out, res...)
		}(r)
	}
	wg.Wait()
	return FilterSortBySimilarity(out, keyword), errs
}
