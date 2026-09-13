// Package booksource 原罪板块「书源」的**纯规则引擎**（零 AI、零 app 依赖，
// Fetcher/Sleeper/Rand 全部注入，可夹具单测）。
//
// 规格来源：docs/gaea-sin-booksource-distill-2026-09.md（so-novel 蒸馏）。
// 上游 so-novel 为 AGPL-3.0、本仓库为私有版权——**机制取道、代码与规则数据
// 一行不搬**，相似度加权/抖动区间等常数均为重推导（规格 §4 D1）。
//
// 规则文件是用户自备数据（<用户配置目录>/gaea/sin/rules/*.json），出厂只内置
// rule-template.json 模板；不支持特性（@js: 步骤、XPath）在校验层 fail-closed。
package booksource

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/andybalholm/cascadia"
)

// ── 抓取默认值（对齐规格 §2.3；上游 config.ini [crawl] 同档常数）──────────

const (
	DefaultConcurrency   = 50
	MaxConcurrency       = 100
	DefaultMinIntervalMs = 200
	DefaultMaxIntervalMs = 400
	DefaultMaxRetries    = 3
	DefaultRetryMinMs    = 2000
	DefaultRetryMaxMs    = 4000
	DefaultSearchLimit   = 30
	DefaultTimeout       = 20 // 秒，每请求
	maxPageConcurrency   = 5  // 目录/搜索分页的抓取并发（上游 TocParser 同值）

	// 相似度过滤阈值：低于则视为低相关丢弃；全被滤空时回退到 >0（规格 §2.5）。
	similarityKeep = 0.25
)

// ── 规则 schema（字段语义对齐上游 rule-template，命名按 gaea 口径，规格 §3.2）──

type Rule struct {
	Name     string       `json:"name"`
	BaseURI  string       `json:"baseUri,omitempty"`
	Comment  string       `json:"comment,omitempty"`
	Disabled bool         `json:"disabled,omitempty"`
	Search   *SearchRule  `json:"search,omitempty"`
	Book     *BookRule    `json:"book,omitempty"`
	Toc      *TocRule     `json:"toc,omitempty"`
	Chapter  *ChapterRule `json:"chapter,omitempty"`
	Crawl    *CrawlRule   `json:"crawl,omitempty"`
}

type SearchRule struct {
	URL            string            `json:"url"`
	Method         string            `json:"method,omitempty"` // get（默认）| post
	Data           map[string]string `json:"data,omitempty"`   // 值为 "%s" 的槽位填关键字
	Cookies        map[string]string `json:"cookies,omitempty"`
	Result         string            `json:"result"`
	BookName       string            `json:"bookName"`
	Link           string            `json:"link,omitempty"` // 缺省=对 bookName 元素取 href
	Author         string            `json:"author,omitempty"`
	Category       string            `json:"category,omitempty"`
	LatestChapter  string            `json:"latestChapter,omitempty"`
	LastUpdateTime string            `json:"lastUpdateTime,omitempty"`
	Status         string            `json:"status,omitempty"`
	WordCount      string            `json:"wordCount,omitempty"`
	NextPage       string            `json:"nextPage,omitempty"`
	Limit          int               `json:"limit,omitempty"` // 缺省 30
}

type BookRule struct {
	URL            string `json:"url,omitempty"` // 从详情 URL 正则捕获书 id（组 1）
	BookName       string `json:"bookName,omitempty"`
	Author         string `json:"author,omitempty"`
	Intro          string `json:"intro,omitempty"`
	Category       string `json:"category,omitempty"`
	CoverURL       string `json:"coverUrl,omitempty"`
	LatestChapter  string `json:"latestChapter,omitempty"`
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`
	Status         string `json:"status,omitempty"`
}

type TocRule struct {
	URL  string `json:"url,omitempty"` // 详情/目录分页时必填，"%s"=书 id
	List string `json:"list,omitempty"`
	// Item 章节链接选择器；缺省走**免规则路线**（owllook 猜目录，
	// 规格书源搜索 docs/gaea-sin-booksearch-distill-2026-09.md §2.2）。
	Item     string `json:"item,omitempty"`
	Reverse  bool   `json:"reverse,omitempty"` // 上游 isDesc 同义
	NextPage string `json:"nextPage,omitempty"`
}

type ChapterRule struct {
	// Title 章节名选择器；缺省走免规则标题回退（<title> 正则 → h1 → title，
	// 规格书源搜索 §2.4）。
	Title              string `json:"title,omitempty"`
	Content            string `json:"content"`
	ParagraphTagClosed bool   `json:"paragraphTagClosed,omitempty"`
	ParagraphTag       string `json:"paragraphTag,omitempty"` // 非闭合源的切段正则（如 "<br>+"）
	FilterTxt          string `json:"filterTxt,omitempty"`    // 广告正则（| 连缀单条）
	FilterTag          string `json:"filterTag,omitempty"`    // 元素连内容整删（"div, script"）
	NextPage           string `json:"nextPage,omitempty"`
	// AutoNext nextPage 为空时启用免规则翻页（「下一页」锚点启发式，
	// 只认含「页」文本防把下一章吞成本章续页，规格书源搜索 §2.3）。
	AutoNext   bool   `json:"autoNext,omitempty"`
	EndPattern string `json:"endPattern,omitempty"` // 末页 URL 正则；缺省走通用启发式
}

type CrawlRule struct {
	Concurrency   int `json:"concurrency,omitempty"`
	MinIntervalMs int `json:"minIntervalMs,omitempty"`
	MaxIntervalMs int `json:"maxIntervalMs,omitempty"`
	MaxRetries    int `json:"maxRetries,omitempty"`
	RetryMinMs    int `json:"retryMinMs,omitempty"`
	RetryMaxMs    int `json:"retryMaxMs,omitempty"`
}

// CrawlConfig 抓取纪律的生效值（书源 crawl 段覆盖默认，并发钳到 [1,100]）。
type CrawlConfig struct {
	Concurrency   int
	MinIntervalMs int
	MaxIntervalMs int
	MaxRetries    int
	RetryMinMs    int
	RetryMaxMs    int
}

func (r *Rule) crawlConfig() CrawlConfig {
	c := CrawlConfig{
		Concurrency:   DefaultConcurrency,
		MinIntervalMs: DefaultMinIntervalMs,
		MaxIntervalMs: DefaultMaxIntervalMs,
		MaxRetries:    DefaultMaxRetries,
		RetryMinMs:    DefaultRetryMinMs,
		RetryMaxMs:    DefaultRetryMaxMs,
	}
	if cr := r.Crawl; cr != nil {
		if cr.Concurrency > 0 {
			c.Concurrency = cr.Concurrency
		}
		if cr.MinIntervalMs > 0 {
			c.MinIntervalMs = cr.MinIntervalMs
		}
		if cr.MaxIntervalMs > 0 {
			c.MaxIntervalMs = cr.MaxIntervalMs
		}
		if cr.MaxRetries > 0 {
			c.MaxRetries = cr.MaxRetries
		}
		if cr.RetryMinMs > 0 {
			c.RetryMinMs = cr.RetryMinMs
		}
		if cr.RetryMaxMs > 0 {
			c.RetryMaxMs = cr.RetryMaxMs
		}
	}
	if c.MinIntervalMs > c.MaxIntervalMs {
		c.MinIntervalMs = c.MaxIntervalMs
	}
	if c.RetryMinMs > c.RetryMaxMs {
		c.RetryMinMs = c.RetryMaxMs
	}
	if c.Concurrency > MaxConcurrency {
		c.Concurrency = MaxConcurrency
	}
	return c
}

// ── 校验（fail-closed：不支持即报错并点名，规格 §3.2）────────────────────

var (
	// XPath 前缀判定（规格 §2.2 的 (/|//|( 三类，// 以 / 开头）；v1 CSS only。
	xpathPrefix = regexp.MustCompile(`^(/|\()`)
	attrSuffix  = regexp.MustCompile(`@(href|src)$`)
	titleNumber = regexp.MustCompile(`^(\d+)\s*\.\s*(.+)$`)
)

// Validate 校验规则；返回首个错误的可读原因。
func (r *Rule) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("书源缺少 name")
	}
	if r.Search == nil && r.Book == nil && r.Toc == nil && r.Chapter == nil {
		return errors.New("书源至少需要 search/book/toc/chapter 之一")
	}
	unsupported := func(where string) error {
		return fmt.Errorf("%s 不支持 @js: 步骤（规格 §4 D4）", where)
	}
	xpath := func(where, v string) error {
		return fmt.Errorf("%s 暂不支持 XPath 选择器「%s」（CSS only，规格 §4 D4）", where, v)
	}
	checkSel := func(where, q string, required bool) error {
		q = strings.TrimSpace(q)
		if q == "" {
			if required {
				return fmt.Errorf("%s 缺少选择器", where)
			}
			return nil
		}
		if strings.Contains(q, "@js:") {
			return unsupported(where)
		}
		if xpathPrefix.MatchString(q) {
			return xpath(where, q)
		}
		if _, _, err := compileSelector(q); err != nil {
			return fmt.Errorf("%s 选择器不可用: %w", where, err)
		}
		return nil
	}
	checkRegex := func(where, pattern string) error {
		if pattern == "" {
			return nil
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("%s 正则不可用: %w", where, err)
		}
		return nil
	}
	// 只对「真是正则」的字段做正则校验：book.url（捕获书 id）、filterTxt、
	// paragraphTag、endPattern。search.url / toc.url 是带 %s 槽位的 **URL 模板**，
	// 合法 URL 里会出现 []、( 等正则元字符（如 ?tags[]=1），当正则校验会误拒合法
	// 规则——fail-closed 只针对真不支持的语法。

	if s := r.Search; s != nil {
		for _, f := range []string{s.URL, s.Result, s.BookName} {
			if strings.Contains(f, "@js:") {
				return unsupported("search")
			}
		}
		if strings.TrimSpace(s.URL) == "" {
			return errors.New("search.url 缺失")
		}
		switch strings.ToLower(s.Method) {
		case "", "get", "post":
		default:
			return fmt.Errorf("search.method 仅支持 get/post，得到「%s」", s.Method)
		}
		if err := checkSel("search.result", s.Result, true); err != nil {
			return err
		}
		if err := checkSel("search.bookName", s.BookName, true); err != nil {
			return err
		}
		for _, w := range []struct{ label, sel string }{
			{"search.link", s.Link}, {"search.author", s.Author}, {"search.category", s.Category},
			{"search.latestChapter", s.LatestChapter}, {"search.lastUpdateTime", s.LastUpdateTime},
			{"search.status", s.Status}, {"search.wordCount", s.WordCount}, {"search.nextPage", s.NextPage},
		} {
			if err := checkSel(w.label, w.sel, false); err != nil {
				return err
			}
		}
		if s.Limit < 0 {
			return errors.New("search.limit 不能为负")
		}
	}
	if b := r.Book; b != nil {
		if b.URL != "" {
			re, err := regexp.Compile(b.URL)
			if err != nil {
				return fmt.Errorf("book.url 正则不可用: %w", err)
			}
			if re.NumSubexp() < 1 {
				return errors.New("book.url 正则必须含捕获组（组 1 = 书 id）")
			}
		}
		for _, w := range []struct{ label, sel string }{
			{"book.bookName", b.BookName}, {"book.author", b.Author}, {"book.intro", b.Intro},
			{"book.category", b.Category}, {"book.coverUrl", b.CoverURL},
			{"book.latestChapter", b.LatestChapter}, {"book.lastUpdateTime", b.LastUpdateTime},
			{"book.status", b.Status},
		} {
			if err := checkSel(w.label, w.sel, false); err != nil {
				return err
			}
		}
	}
	if t := r.Toc; t != nil {
		if err := checkSel("toc.item", t.Item, false); err != nil {
			return err
		}
		if err := checkSel("toc.list", t.List, false); err != nil {
			return err
		}
		if err := checkSel("toc.nextPage", t.NextPage, false); err != nil {
			return err
		}
	}
	if c := r.Chapter; c != nil {
		if err := checkSel("chapter.title", c.Title, false); err != nil {
			return err
		}
		if err := checkSel("chapter.content", c.Content, true); err != nil {
			return err
		}
		if err := checkSel("chapter.nextPage", c.NextPage, false); err != nil {
			return err
		}
		if err := checkRegex("chapter.filterTxt", c.FilterTxt); err != nil {
			return err
		}
		if err := checkRegex("chapter.paragraphTag", c.ParagraphTag); err != nil {
			return err
		}
		if err := checkRegex("chapter.endPattern", c.EndPattern); err != nil {
			return err
		}
		if err := checkSel("chapter.filterTag", c.FilterTag, false); err != nil {
			return err
		}
	}
	if cr := r.Crawl; cr != nil {
		for _, it := range []struct {
			label string
			val   int
		}{
			{"crawl.concurrency", cr.Concurrency},
			{"crawl.minIntervalMs", cr.MinIntervalMs},
			{"crawl.maxIntervalMs", cr.MaxIntervalMs},
			{"crawl.maxRetries", cr.MaxRetries},
			{"crawl.retryMinMs", cr.RetryMinMs},
			{"crawl.retryMaxMs", cr.RetryMaxMs},
		} {
			if it.val < 0 {
				return fmt.Errorf("%s 不能为负", it.label)
			}
		}
	}
	return nil
}

// compileSelector 解出 "@href/@src" 属性后缀并编译 CSS（校验与运行时共用一条路）。
func compileSelector(q string) (cascadia.Selector, string, error) {
	sel := q
	attr := ""
	if m := attrSuffix.FindString(sel); m != "" {
		attr = m[1:]
		sel = strings.TrimSuffix(sel, m)
	}
	if strings.HasPrefix(sel, "meta[") {
		return nil, "content", nil
	}
	if xpathPrefix.MatchString(sel) {
		return nil, "", fmt.Errorf("XPath 不支持: %s", sel)
	}
	c, err := cascadia.Compile(sel)
	if err != nil {
		return nil, "", err
	}
	return c, attr, nil
}

// ── 规则装载 ─────────────────────────────────────────────────────────────

// LoadFile 读入并校验单个规则文件（fail-closed，坏文件返回错误）。
func LoadFile(path string) (*Rule, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(raw)
}

// Parse 解析并校验规则字节。
func Parse(raw []byte) (*Rule, error) {
	if err := rejectUnsupportedFields(raw); err != nil {
		return nil, err
	}
	var r Rule
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("规则 JSON 解析失败: %w", err)
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

// unsupportedFields 上游规则里 gaea v1 明确不做、但必须**如实点名**的字段
// （规格 §4 D4）。schema 结构体里没有这些字段，json.Unmarshal 默认静默丢弃——
// 静默的代价是「分页抓不到 → 章节被截断」，属静默失真，故在校验层 fail-closed。
var unsupportedFields = []struct{ Section, Field, Why string }{
	{"chapter", "nextPageInJs", "需 JS 步骤引擎，gaea v1 不做（规格 §4 D4）"},
}

// rejectUnsupportedFields 在 schema 之外检查明确不支持的字段名。
func rejectUnsupportedFields(raw []byte) error {
	var sections map[string]json.RawMessage
	if err := json.Unmarshal(raw, &sections); err != nil {
		return nil // 结构层面的错误交给主解析路径报，避免掩盖真因
	}
	for _, f := range unsupportedFields {
		rawSec, ok := sections[f.Section]
		if !ok || len(rawSec) == 0 || rawSec[0] != '{' {
			continue
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(rawSec, &fields); err != nil {
			continue
		}
		if _, hit := fields[f.Field]; hit {
			return fmt.Errorf("%s.%s 不支持：%s", f.Section, f.Field, f.Why)
		}
	}
	return nil
}

// Loaded LoadDir 的逐文件结果（坏文件不致命，错误随条目上报）。
type Loaded struct {
	File string
	Rule *Rule
	Err  error
}

// LoadDir 装载目录下全部 *.json（按文件名序）；单文件失败记录在 Loaded.Err。
func LoadDir(dir string) []Loaded {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []Loaded{{File: dir, Err: err}}
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".json") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)
	out := make([]Loaded, 0, len(files))
	for _, f := range files {
		path := filepath.Join(dir, f)
		r, err := LoadFile(path)
		out = append(out, Loaded{File: f, Rule: r, Err: err})
	}
	return out
}
