package booksource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ── 泛搜索（owllook 蒸馏，规格 docs/gaea-sin-booksearch-distill-2026-09.md §2.1）──
//
// 路线=拿书名去通用搜索引擎检索，从 SERP 滤出小说站候选；与逐站搜索
// （Aggregate）互补：不依赖书源自带搜索端点，覆盖「无搜索接口」的站点。

// indexHtmlTail 目录页形态启发式：SERP 链接以 index.html 收尾时剥去
// （上游 url.replace('index.html',”) 的收口版）。
var indexHtmlTail = regexp.MustCompile(`[iI]ndex\.html/?$`)

// SearchEngineRule 通用搜索引擎规则（数据资产，随书源规则目录装载）。
// 引擎坏了对付法=换一份 JSON，不改代码（规格 §4 D5）。
type SearchEngineRule struct {
	Name string `json:"name"` // bing / ddg-html / so360 / 自定义
	URL  string `json:"url"`  // SERP 地址，"%s"=转义后的查询词
	// QueryFormat 查询塑形模板（上游「{书名} 小说 免费阅读」同款）；缺省只搜关键字。
	QueryFormat string `json:"queryFormat,omitempty"`
	Result      string `json:"result"`         // 结果条目选择器（如 bing .b_algo）
	Title       string `json:"title"`          // 标题链接选择器（如 h2 a）
	Link        string `json:"link,omitempty"` // 链接选择器；缺省=对 title 元素取 href
	Disabled    bool   `json:"disabled,omitempty"`
	Comment     string `json:"comment,omitempty"`
	// LinkParam 链接包在查询参数里时给出参数名（ddg html 版为 uddg）。
	LinkParam string `json:"linkParam,omitempty"`
	// RedirectHosts 需 HEAD 跟随解真实 URL 的宿主（如 www.baidu.com）。
	RedirectHosts []string   `json:"redirectHosts,omitempty"`
	Crawl         *CrawlRule `json:"crawl,omitempty"`
}

// Validate fail-closed 校验（与书源规则同一纪律：CSS only、禁 @js:）。
func (r *SearchEngineRule) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("搜索引擎规则缺少 name")
	}
	if strings.TrimSpace(r.URL) == "" {
		return errors.New("搜索 url 缺失")
	}
	if strings.Contains(r.URL, "@js:") {
		return errors.New("搜索引擎规则不支持 @js: 步骤")
	}
	for label, sel := range map[string]string{"result": r.Result, "title": r.Title, "link": r.Link} {
		if sel == "" {
			if label == "result" || label == "title" {
				return fmt.Errorf("搜索引擎规则缺少 %s 选择器", label)
			}
			continue
		}
		if strings.Contains(sel, "@js:") {
			return fmt.Errorf("搜索引擎规则 %s 不支持 @js: 步骤", label)
		}
		if _, _, err := compileSelector(sel); err != nil {
			return fmt.Errorf("搜索引擎规则 %s 选择器不可用: %w", label, err)
		}
	}
	if r.Crawl != nil && (r.Crawl.MinIntervalMs < 0 || r.Crawl.MaxIntervalMs < 0) {
		return errors.New("crawl 间隔不能为负")
	}
	return nil
}

// LoadEnginesFile 装载引擎规则文件（JSON 数组，逐条校验 fail-closed）。
func LoadEnginesFile(path string) ([]*SearchEngineRule, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadEnginesFileBytes(raw)
}

// LoadEnginesFileBytes 解析并校验引擎规则字节。
func LoadEnginesFileBytes(raw []byte) ([]*SearchEngineRule, error) {
	var list []*SearchEngineRule
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("引擎规则 JSON 解析失败: %w", err)
	}
	if len(list) == 0 {
		return nil, errors.New("引擎规则文件为空")
	}
	for i, e := range list {
		if e == nil {
			return nil, fmt.Errorf("引擎规则第 %d 条为空", i+1)
		}
		if err := e.Validate(); err != nil {
			return nil, fmt.Errorf("引擎规则「%s」: %w", e.Name, err)
		}
	}
	return list, nil
}

// WebCandidate 泛搜索候选（对应上游 SERP 条目 {title,url,netloc,is_parse}）。
type WebCandidate struct {
	Engine  string `json:"engine"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Host    string `json:"host"`
	HasRule bool   `json:"hasRule"` // 规则目录同 host 有书源（上游 is_parse 对应物）
}

// WebSearchOptions 泛搜索选项。
type WebSearchOptions struct {
	Limit int // 候选上限，缺省 10
	// IncludeDetailLinks 关闭「目录页形态启发式」。默认（false=启发式开）：
	// 剥 index.html 尾；链接路径仍含 .html 判详情页丢弃（上游同款启发式）。
	IncludeDetailLinks bool
	// SkipRedirectResolve 关闭 HEAD 重定向解包。默认（false=开）：对命中引擎
	// RedirectHosts 的链接做跟随解真实 URL（上游 baidu 路线，有门控才实际发生）。
	SkipRedirectResolve bool
	// BlackHosts 用户自备黑名单（不出厂策展数据，规格 §6）。
	BlackHosts []string
	// KnownRules 回调：该 host 是否已有书源规则（打标 HasRule）。
	KnownRules func(host string) bool
}

func (o WebSearchOptions) limit() int {
	if o.Limit > 0 {
		return o.Limit
	}
	return 10
}

// WebSearcher 泛搜索器：复用抓取纪律，结果带 TTL 缓存。
type WebSearcher struct {
	*crawler
	cache *TTLCache[[]WebCandidate]
}

// WebOptions WebSearcher 依赖（CacheTTL<=0 关缓存）。
type WebOptions struct {
	Options
	CacheTTL time.Duration
	Now      func() time.Time
}

func NewWebSearcher(opt WebOptions) *WebSearcher {
	defaults := Rule{}
	w := &WebSearcher{crawler: newCrawler(defaults.crawlConfig(), opt.Options)}
	if opt.CacheTTL > 0 {
		now := opt.Now
		if now == nil {
			now = time.Now
		}
		w.cache = NewTTLCache[[]WebCandidate](opt.CacheTTL, now)
	}
	return w
}

// DiscoverySearch 泛搜索：SERP 抓取 → 候选抽取净化 → 重定向解包 → 缓存。
func (w *WebSearcher) DiscoverySearch(ctx context.Context, eng *SearchEngineRule, keyword string, opt WebSearchOptions) ([]WebCandidate, error) {
	if eng == nil {
		return nil, errors.New("搜索引擎规则为空")
	}
	if err := eng.Validate(); err != nil {
		return nil, err
	}
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, errors.New("搜索关键字为空")
	}
	cacheKey := eng.Name + "\x00" + keyword
	if w.cache != nil {
		if hit, ok := w.cache.Get(cacheKey); ok {
			return hit, nil
		}
	}

	// 本地化 crawler：引擎 crawl 段只影响本次搜索（零共享竞态）
	c := *w.crawler
	if eng.Crawl != nil {
		tmp := Rule{Crawl: eng.Crawl}
		c.cfg = tmp.crawlConfig()
	}
	out, err := w.discover(ctx, &c, eng, keyword, opt)
	if err != nil {
		return nil, err
	}
	if w.cache != nil {
		w.cache.Set(cacheKey, out)
	}
	return out, nil
}

func (w *WebSearcher) discover(ctx context.Context, c *crawler, eng *SearchEngineRule, keyword string, opt WebSearchOptions) ([]WebCandidate, error) {
	query := url.QueryEscape(keyword)
	if eng.QueryFormat != "" {
		query = url.QueryEscape(strings.ReplaceAll(eng.QueryFormat, "%s", keyword))
	}
	doc, pageURL, err := c.pacedFetch(ctx, Request{URL: strings.ReplaceAll(eng.URL, "%s", query)})
	if err != nil {
		return nil, err
	}

	engineHost := hostOf(eng.URL)
	black := map[string]bool{}
	for _, h := range opt.BlackHosts {
		black[strings.ToLower(h)] = true
	}
	skipDetail := !opt.IncludeDetailLinks

	var out []WebCandidate
	seen := map[string]bool{}
	doc.Find(eng.Result).Each(func(_ int, row *goquery.Selection) {
		if len(out) >= opt.limit()*3 { // 解包/过滤前的采集上限，防异常 SERP 刷爆
			return
		}
		title := textOf(row, eng.Title)
		rawLink := ""
		if eng.Link != "" {
			sel, attr := splitAttrSuffix(eng.Link)
			if attr == "" {
				attr = "href"
			}
			rawLink = attrOf(row, sel, attr, pageURL)
		} else {
			rawLink = attrOf(row, eng.Title, "href", pageURL)
		}
		if title == "" || rawLink == "" {
			return
		}
		link := rawLink
		if eng.LinkParam != "" { // ddg uddg 式解包
			if u, err := url.Parse(link); err == nil {
				if v := u.Query().Get(eng.LinkParam); v != "" {
					link = v
				}
			}
		}
		if indexHtmlTail.MatchString(link) { // 目录页形态净化
			link = indexHtmlTail.ReplaceAllString(link, "")
		}
		u, err := url.Parse(link)
		if err != nil || u.Host == "" || u.Path == "/" {
			return
		}
		host := strings.ToLower(u.Host)
		if host == engineHost || black[host] {
			return
		}
		if skipDetail && strings.Contains(u.Path, ".html") {
			return // 详情/章节页不是目录候选（上游同款启发式）
		}
		clean := u.Scheme + "://" + u.Host + u.Path
		if clean == "" || seen[clean] {
			return
		}
		seen[clean] = true
		out = append(out, WebCandidate{Engine: eng.Name, Title: strings.TrimSpace(title), URL: clean, Host: host})
	})

	if !opt.SkipRedirectResolve && len(eng.RedirectHosts) > 0 {
		redirectors := map[string]bool{}
		for _, h := range eng.RedirectHosts {
			redirectors[strings.ToLower(h)] = true
		}
		_ = c.runBounded(ctx, len(out), maxPageConcurrency, func(i int) error {
			cand := out[i]
			if !redirectors[cand.Host] {
				return nil
			}
			real, ok := w.resolveURL(ctx, c, cand.URL)
			if !ok {
				out[i].URL = "" // 解包失败丢弃
				return nil
			}
			if ru, err := url.Parse(real); err != nil || ru.Path == "/" {
				out[i].URL = "" // 站点根=无效候选（上游同判）
				return nil
			}
			out[i].URL = indexHtmlTail.ReplaceAllString(real, "")
			out[i].Host = hostOf(out[i].URL)
			return nil
		})
		kept := out[:0]
		for _, cand := range out {
			if cand.URL != "" {
				kept = append(kept, cand)
			}
		}
		out = kept
		seen2 := map[string]bool{} // 解包后可能撞车，二次去重
		kept2 := out[:0]
		for _, cand := range out {
			if seen2[cand.URL] {
				continue
			}
			seen2[cand.URL] = true
			kept2 = append(kept2, cand)
		}
		out = kept2
	}
	if len(out) > opt.limit() {
		out = out[:opt.limit()]
	}
	for i := range out {
		if opt.KnownRules != nil {
			out[i].HasRule = opt.KnownRules(out[i].Host)
		}
	}
	return out, nil
}

// resolveURL HEAD 跟随解真实 URL（上游 baidu get_real_url 同路线；HTTPFetcher
// 带回重定向后的最终 URL）。
func (w *WebSearcher) resolveURL(ctx context.Context, c *crawler, u string) (string, bool) {
	if err := ctx.Err(); err != nil {
		return "", false
	}
	page, err := c.fetch.Fetch(ctx, Request{Method: "head", URL: u})
	if err != nil || page == nil || page.URL == "" {
		return "", false
	}
	return page.URL, true
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Host)
}
