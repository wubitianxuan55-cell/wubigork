package booksource

import (
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ── 免规则解析（owllook 招牌机制，规格 §2.2/§2.3/§2.4）─────────────────────
//
// 对任意小说页凭文本形态认目录/翻页/标题；正文不猜（与上游同边界，
// chapter.content 仍必填）。

var (
	// 章节名形态：第? + 数字（含全角/汉字数字）1~6 位 + [章回卷节折篇幕集]
	// （上游 extract_chapters 正则的重推导版，规格 §4 D1）。
	chapterNameReg = regexp.MustCompile(`第?\s*[一二两三四五六七八九十○零百千万亿0-9０-９]{1,6}\s*[章回卷节折篇幕集]`)
	// 翻页/上下章锚点文本形态（上游 extract_pre_next_chapter judge_reg 同形）。
	nextPageTextReg = regexp.MustCompile(`[第上前下后][一]?[0-9]{0,6}[页张个篇章节步]`)
	// 翻页噪声词（上游 rm_list 同款语义）。
	nextNoiseWords = []string{"后一个", "天上掉下个"}
	// 页面 <title> 里的章节名提取：…第X章 xxx_书名_站名 → 取分组 1
	// （上游 cache.py title_reg 同路线，分隔符扩全角逗号）。
	titleInPageTitle = regexp.MustCompile(`(第?\s*[一二两三四五六七八九十○零百千万亿0-9０-９]{1,6}\s*[章回卷节折篇幕集][^_，,\-]*)[_，,\-]`)
	// URL 数字尾：路径末段（去扩展名）里的数字串（上游 int(尾段) 的防御化版，
	// 规格 §4 D3：非数字尾不 panic）。
	digitRun = regexp.MustCompile(`\d+`)
)

// GuessTocEntries 免规则目录：页面全部锚点里按文本形态认章节，URL 补全去重，
// 按 URL 数字尾**升序**（上游为追更取降序，gaea 是整本阅读取升序，规格 §4 D2）；
// 无数字尾的条目 index=0，稳定排序聚前并保持文档序。
func GuessTocEntries(doc *goquery.Document, pageURL string) []TocEntry {
	seen := map[string]bool{}
	var out []TocEntry
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(cleanInvisibleChars(s.Text()))
		if text == "" || !chapterNameReg.MatchString(text) {
			return
		}
		href, ok := s.Attr("href")
		if !ok || strings.TrimSpace(href) == "" {
			return
		}
		u := resolveLink(pageURL, href)
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, TocEntry{Title: text, URL: u})
	})
	for i := range out {
		out[i].Order = tocIndexFromURL(out[i].URL)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}

// tocIndexFromURL URL 数字尾：/b/57/57181/8845907.html → 8845907；
// /chapter_12/ → 12；全无数字 → 0。
func tocIndexFromURL(raw string) int {
	u, err := url.Parse(raw)
	if err != nil {
		return 0
	}
	seg := path.Base(u.Path)
	if i := strings.Index(seg, "."); i >= 0 {
		seg = seg[:i]
	}
	if all := digitRun.FindAllString(seg, -1); len(all) > 0 {
		n, err := strconv.Atoi(all[len(all)-1])
		if err == nil {
			return n
		}
	}
	return 0
}

// GuessNextPage 免规则翻页：锚点文本认「下一页/下页」型链接。
// 只认文本含「页」的锚点——上游翻页正则的字符类含「章」（下一章也命中），
// 那是给阅读器做章节导航用的；gaea 的翻页语义（目录多页/正文多段）跟错
// 「下一章」会把别章吞进来，故收窄（规格书源搜索 §4 D5 注）。
// 自链（与当前页相同）判末页（上游同语义）。
func GuessNextPage(doc *goquery.Document, pageURL string) (string, bool) {
	next := ""
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		if next != "" {
			return
		}
		text := strings.TrimSpace(cleanInvisibleChars(s.Text()))
		if text == "" || !strings.Contains(text, "页") || !nextPageTextReg.MatchString(text) {
			return
		}
		for _, w := range nextNoiseWords {
			if strings.Contains(text, w) {
				return
			}
		}
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		u := resolveLink(pageURL, href)
		if u == "" || u == pageURL { // 自链=末页
			return
		}
		next = u
	})
	return next, next != ""
}

// GuessChapterTitle 免规则标题三级回退（上游 cache.py 同路线）：
// <title> 正则提取 → h1 → <title> 全文。
func GuessChapterTitle(doc *goquery.Document) string {
	pageTitle := strings.TrimSpace(doc.Find("title").First().Text())
	if pageTitle != "" {
		if m := titleInPageTitle.FindStringSubmatch(pageTitle); len(m) >= 2 {
			return strings.TrimSpace(m[1])
		}
	}
	if h1 := strings.TrimSpace(doc.Find("h1").First().Text()); h1 != "" {
		return h1
	}
	return pageTitle
}
