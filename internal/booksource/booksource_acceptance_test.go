package booksource

// ── 规格 §7 验收缺口补测（独立文件，不动既有夹具）─────────────────────────
//
// 依据：docs/gaea-sin-booksource-distill-2026-09.md §7 验收清单。
// 既有 booksource_test.go 覆盖了主链路；本文件补齐其未触达的验收点，并对
// 「静默失真类」缺陷立回归锁（红色先于修复，见 .gaea/reports 对拍报告）。

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// instantSleeper 零等待注入（补测不真睡）。
func instantSleeper(context.Context, time.Duration) error { return nil }

// staticFetcher 按 URL 出静态页面；未命中的 URL 直接报错，暴露夹具缺口。
type staticFetcher struct {
	mu    sync.Mutex
	pages map[string]string
	calls []string
}

func (s *staticFetcher) Fetch(_ context.Context, req Request) (*Page, error) {
	s.mu.Lock()
	s.calls = append(s.calls, req.URL)
	s.mu.Unlock()
	body, ok := s.pages[req.URL]
	if !ok {
		return nil, fmt.Errorf("测试夹具缺页面: %s", req.URL)
	}
	return &Page{URL: req.URL, Body: []byte(body)}, nil
}

func (s *staticFetcher) count(u string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.calls {
		if c == u {
			n++
		}
	}
	return n
}

// ── 抽取层（规格 §7「抽取：text/html/@href/absURL 补全」）─────────────────

func TestExtractHelpersAndAbsURLResolution(t *testing.T) {
	const page = `<html><body>
<div id="a"><a href="../p/1">甲</a></div>
<div id="b"><em>x</em>尾巴</div>
<div id="c"><span>无链接</span></div>
<select id="opt"><option value="/p/2">2</option><option>3</option><option value="/p/2">dup</option></select>
</body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	const base = "https://site.example/dir/page.html"
	if got := textOf(doc.Selection, "#a a"); got != "甲" {
		t.Fatalf("text 抽取不符: %q", got)
	}
	if got := textOf(doc.Selection, "#missing"); got != "" {
		t.Fatalf("未命中应得空串: %q", got)
	}
	if got := htmlOf(doc.Selection, "#b"); !strings.Contains(got, "<em>x</em>") {
		t.Fatalf("html 抽取应保留标签: %q", got)
	}
	if got := attrOf(doc.Selection, "#a a", "", base); got != "https://site.example/p/1" {
		t.Fatalf("attr 缺省取 href + 相对补全不符: %q", got)
	}
	if got := attrOf(doc.Selection, "#c span", "", base); got != "" {
		t.Fatalf("元素无该属性应为空: %q", got)
	}
	if got := attrOf(doc.Selection, "#a a", "title", base); got != "" {
		t.Fatalf("缺属性应为空: %q", got)
	}
	// 链接集合：href 缺失回落 value、绝对化、去重保序
	links := absLinks(doc.Selection, "#opt option", base)
	if len(links) != 1 || links[0] != "https://site.example/p/2" {
		t.Fatalf("链接集合应去重保序且回落 value: %#v", links)
	}
	if got := resolveLink(base, "//cdn.example/x.js"); got != "https://cdn.example/x.js" {
		t.Fatalf("协议相对链接补全不符: %q", got)
	}
	if got := resolveLink("", "/rel"); got != "/rel" {
		t.Fatalf("无基准应原样返回: %q", got)
	}
}

// baseUri 是规格 §3.2 声明的「相对链接补全基准」，不得是死字段：
// 注入的 Fetcher 未回报页址时（Page.URL 为空）仍要能补全相对链接。
func TestBaseURIUsedWhenPageURLAbsent(t *testing.T) {
	const page = `<html><body><div class="r"><span class="t"><a href="/book/9/">书名</a></span></div></body></html>`
	empty := &emptyURLFetcher{body: page} // 回报空页址的通道
	rule := &Rule{
		Name:    "无页址源",
		BaseURI: "https://base.example/",
		Search:  &SearchRule{URL: "https://base.example/search?q=%s", Result: ".r", BookName: ".t a"},
	}
	res, err := New(rule, Options{Fetcher: empty, Sleeper: instantSleeper}).Search(context.Background(), "书名")
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("应解析 1 条结果: %+v", res)
	}
	if res[0].URL != "https://base.example/book/9/" {
		t.Fatalf("Page.URL 为空时应用 baseUri 补全: %q", res[0].URL)
	}
}

// emptyURLFetcher 回报空页址的假通道（模拟不回报 URL 的 Fetcher 实现）。
type emptyURLFetcher struct{ body string }

func (f *emptyURLFetcher) Fetch(_ context.Context, _ Request) (*Page, error) {
	return &Page{Body: []byte(f.body)}, nil
}

// ── 校验器（规格 §7「坏规则逐类报错」+ §4 D4「如实报不支持」）────────────

// 合法 URL 模板里的「[]」「(」不是正则元字符错误——fail-closed 只针对真不支持特性。
func TestValidateAcceptsLegitURLTemplates(t *testing.T) {
	r := &Rule{
		Name:   "方括号源",
		Search: &SearchRule{URL: "https://x.example/search?tags[]=1&q=%s", Result: ".r", BookName: ".t"},
		Toc:    &TocRule{URL: "https://x.example/list(1)/%s", Item: "#l a"},
		Chapter: &ChapterRule{
			Title: "h1", Content: "#c",
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("合法 URL 模板不得被当作坏正则拒收: %v", err)
	}
}

// §4 D4：nextPageInJs 不做，校验器必须如实点名，不得静默忽略（静默=分页抓不到 → 章节被截断）。
func TestParseRejectsNextPageInJs(t *testing.T) {
	raw := []byte(`{"name":"JS 源","chapter":{"title":"h1","content":"#c","nextPageInJs":"next()"}}`)
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("nextPageInJs 必须 fail-closed 报不支持")
	}
	if !strings.Contains(err.Error(), "nextPageInJs") {
		t.Fatalf("报错应点名字段: %v", err)
	}
}

// ── 抓取纪律（规格 §7「抖动区间与重试线性放大、并发钳制」）────────────────

func TestCrawlConfigDefaultsOverrideAndClamp(t *testing.T) {
	c := (&Rule{Name: "默认"}).crawlConfig()
	if c.Concurrency != DefaultConcurrency || c.MinIntervalMs != DefaultMinIntervalMs ||
		c.MaxIntervalMs != DefaultMaxIntervalMs || c.MaxRetries != DefaultMaxRetries ||
		c.RetryMinMs != DefaultRetryMinMs || c.RetryMaxMs != DefaultRetryMaxMs {
		t.Fatalf("缺省抓取参数不符: %+v", c)
	}
	r := &Rule{Name: "覆盖", Crawl: &CrawlRule{
		Concurrency: 5, MinIntervalMs: 1000, MaxIntervalMs: 2000,
		MaxRetries: 1, RetryMinMs: 50, RetryMaxMs: 60,
	}}
	if c := r.crawlConfig(); c.Concurrency != 5 || c.MinIntervalMs != 1000 || c.MaxIntervalMs != 2000 ||
		c.MaxRetries != 1 || c.RetryMinMs != 50 || c.RetryMaxMs != 60 {
		t.Fatalf("书源 crawl 段应覆盖默认: %+v", c)
	}
	// 并发上限全局钳制 100（规格 §2.3）
	r2 := &Rule{Name: "超大并发", Crawl: &CrawlRule{Concurrency: 500}}
	if c := r2.crawlConfig(); c.Concurrency != MaxConcurrency || MaxConcurrency != 100 {
		t.Fatalf("并发应钳到 %d: %+v", MaxConcurrency, c)
	}
	// 区间倒置收敛（min 不得大于 max）
	r3 := &Rule{Name: "倒置", Crawl: &CrawlRule{MinIntervalMs: 900, MaxIntervalMs: 100, RetryMinMs: 700, RetryMaxMs: 200}}
	c3 := r3.crawlConfig()
	if c3.MinIntervalMs != c3.MaxIntervalMs || c3.MinIntervalMs != 100 ||
		c3.RetryMinMs != c3.RetryMaxMs || c3.RetryMinMs != 200 {
		t.Fatalf("倒置区间应收敛: %+v", c3)
	}
}

func TestJitterUsesCrawlOverrideAndDefaults(t *testing.T) {
	fixed := New(&Rule{Name: "定值", Crawl: &CrawlRule{MinIntervalMs: 1000, MaxIntervalMs: 1000}},
		Options{Fetcher: &staticFetcher{pages: map[string]string{}}, Sleeper: instantSleeper})
	for i := 0; i < 5; i++ {
		if d := fixed.jitter(); d != time.Second {
			t.Fatalf("抖动应取书源覆盖区间: %v", d)
		}
	}
	def := New(&Rule{Name: "默认"}, Options{Sleeper: instantSleeper})
	for i := 0; i < 40; i++ {
		d := def.jitter()
		if d < 200*time.Millisecond || d > 400*time.Millisecond {
			t.Fatalf("默认抖动应落 [200,400]ms: %v", d)
		}
	}
}

func TestRunBoundedEnforcesConcurrencyLimit(t *testing.T) {
	e := New(&Rule{Name: "并发", Crawl: &CrawlRule{Concurrency: 2}}, Options{Sleeper: instantSleeper})
	peak := func(n, limit int) (int, []error) {
		var mu sync.Mutex
		cur, max := 0, 0
		errs := e.runBounded(context.Background(), n, limit, func(int) error {
			mu.Lock()
			cur++
			if cur > max {
				max = cur
			}
			mu.Unlock()
			time.Sleep(2 * time.Millisecond)
			mu.Lock()
			cur--
			mu.Unlock()
			return nil
		})
		return max, errs
	}
	max, errs := peak(24, 3)
	if max > 3 || max < 2 {
		t.Fatalf("显式 limit=3 应生效: peak=%d", max)
	}
	if len(errs) != 24 {
		t.Fatalf("errs 应与任务数对齐: %d", len(errs))
	}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("无错任务不得回填错误: errs[%d]=%v", i, err)
		}
	}
	// limit<=0 → 落书源 crawl.concurrency（钳制后的 2）
	if max2, _ := peak(16, 0); max2 > 2 {
		t.Fatalf("limit<=0 应用 cfg.Concurrency: peak=%d", max2)
	}
	// ctx 取消：先占满信号量（两个任务阻塞在 fn 内），其余槽位只能走 ctx 分支
	ctx, cancel := context.WithCancel(context.Background())
	release := make(chan struct{})
	var calls int32
	done := make(chan []error, 1)
	go func() {
		done <- e.runBounded(ctx, 20, 2, func(int) error {
			atomic.AddInt32(&calls, 1)
			<-release
			return nil
		})
	}()
	for atomic.LoadInt32(&calls) < 2 {
		time.Sleep(time.Millisecond)
	}
	cancel()
	time.Sleep(50 * time.Millisecond) // 让等待槽位的 goroutine 观察到取消
	close(release)
	errs = <-done
	canceled := 0
	for _, err := range errs {
		if err == nil {
			continue
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("只应回填 ctx 取消错误: %v", err)
		}
		canceled++
	}
	if canceled != 18 || atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("取消后等待槽位应记 ctx 错误且超限任务不得起跑: canceled=%d calls=%d",
			canceled, atomic.LoadInt32(&calls))
	}
}

// ── 目录（规格 §7「list 二跳 / 分页集合去重保序」）──────────────────────

func TestTocListContainerTwoHopParsing(t *testing.T) {
	const page = `<html><body><div id="wrap"><div id="inner">
<a href="/ch/1">第一章</a><a href="/ch/2">第二章</a>
</div></div></body></html>`
	f := &staticFetcher{pages: map[string]string{"https://s.example/book/1/": page}}
	rule := &Rule{Name: "二跳", Toc: &TocRule{List: "#wrap", Item: "#inner a"}}
	toc, err := New(rule, Options{Fetcher: f, Sleeper: instantSleeper}).
		Toc(context.Background(), "https://s.example/book/1/", 0, 0)
	if err != nil {
		t.Fatalf("list 二跳解析失败: %v", err)
	}
	if len(toc) != 2 || toc[0].Title != "第一章" || toc[1].URL != "https://s.example/ch/2" {
		t.Fatalf("容器二跳结果不符: %+v", toc)
	}
}

func TestTocPaginationDedupesFirstPage(t *testing.T) {
	const first = "https://s.example/toc/1.html"
	const second = "https://s.example/toc/2.html"
	p1 := `<html><body><div id="list"><a href="/ch/a">A 章</a></div>
<select id="page"><option value="` + second + `">2</option><option value="` + first + `">1</option></select></body></html>`
	p2 := `<html><body><div id="list"><a href="/ch/b">B 章</a></div></body></html>`
	f := &staticFetcher{pages: map[string]string{first: p1, second: p2}}
	rule := &Rule{Name: "去重", Toc: &TocRule{Item: "#list a", NextPage: "#page option"}}
	toc, err := New(rule, Options{Fetcher: f, Sleeper: instantSleeper}).
		Toc(context.Background(), first, 0, 0)
	if err != nil {
		t.Fatalf("目录失败: %v", err)
	}
	if len(toc) != 2 {
		t.Fatalf("下拉里重复出现首页不得丢章/重复章: %+v", toc)
	}
	// 「去重保序、后写覆盖」：首页被挪到队尾 → B 在前、A 在后
	if toc[0].Title != "B 章" || toc[1].Title != "A 章" {
		t.Fatalf("后写覆盖语义不符: %+v", toc)
	}
	if n := f.count(first); n != 1 {
		t.Fatalf("首页已抓过的文档应复用，不应重复抓取: %d 次", n)
	}
}

// ── 搜索（规格 §7「空书名行跳过 / 单源守卫」）────────────────────────────

func TestSearchGuardsAndEmptyNameRows(t *testing.T) {
	const page = `<html><body>
<div class="r"><span class="t"></span></div>
<div class="r"><span class="t"><a href="/book/1/">有书名</a></span></div>
</body></html>`
	f := &staticFetcher{pages: map[string]string{"https://s.example/s?q=%E4%B9%A6": page}}
	rule := &Rule{Name: "守卫", Search: &SearchRule{URL: "https://s.example/s?q=%s", Result: ".r", BookName: ".t a"}}
	e := New(rule, Options{Fetcher: f, Sleeper: instantSleeper})
	res, err := e.Search(context.Background(), "书")
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if len(res) != 1 || res[0].BookName != "有书名" {
		t.Fatalf("空书名行应跳过: %+v", res)
	}
	if _, err := e.Search(context.Background(), "   "); err == nil {
		t.Fatal("空关键字应显式报错")
	}
	disabled := &Rule{Name: "禁用源", Disabled: true, Search: rule.Search}
	if _, err := New(disabled, Options{Fetcher: f, Sleeper: instantSleeper}).
		Search(context.Background(), "书"); !errors.Is(err, ErrSearchUnsupported) {
		t.Fatalf("禁用源应报 ErrSearchUnsupported: %v", err)
	}
	noSearch := &Rule{Name: "无搜索段"}
	if _, err := New(noSearch, Options{Fetcher: f, Sleeper: instantSleeper}).
		Search(context.Background(), "书"); !errors.Is(err, ErrSearchUnsupported) {
		t.Fatalf("缺 search 段应报 ErrSearchUnsupported: %v", err)
	}
}

// ── 相似度（规格 §7「加权/tie-break」）──────────────────────────────────

func TestSimilarityBoostBranchesAndTieBreak(t *testing.T) {
	if got := similarityBoost(1, 3); got != 12 {
		t.Fatalf("短关键字完全命中应重加权 12: %v", got)
	}
	if got := similarityBoost(1, 6); got != 6 {
		t.Fatalf("中长关键字完全命中应加权 6: %v", got)
	}
	if got := similarityBoost(1, 12); got != 3 {
		t.Fatalf("长关键字完全命中应降权 3: %v", got)
	}
	if got := similarityBoost(0.5, 3); got != 0.5 {
		t.Fatalf("非完全命中不得加权: %v", got)
	}
	// 并列时按「另一字段」字典序（作者搜 → 另一字段 = 书名）
	in := []SearchResult{
		{BookName: "B 书", Author: "猫腻"},
		{BookName: "A 书", Author: "猫腻"},
	}
	out := FilterSortBySimilarity(in, "猫腻")
	if len(out) != 2 || out[0].BookName != "A 书" || out[1].BookName != "B 书" {
		t.Fatalf("并列应按另一字段字典序: %+v", out)
	}
	// 书名搜 → 另一字段 = 作者
	inAuthor := []SearchResult{
		{BookName: "大道朝天", Author: "B 某人"},
		{BookName: "大道朝天", Author: "A 某人"},
	}
	outAuthor := FilterSortBySimilarity(inAuthor, "大道朝天")
	if len(outAuthor) != 2 || outAuthor[0].Author != "A 某人" {
		t.Fatalf("书名搜并列应按作者字典序: %+v", outAuthor)
	}
	// 0.25 是严格下界：恰好 0.25 应被过滤（有其它命中项时）
	exact := SearchResult{BookName: "abcd", Author: ""}
	edge := SearchResult{BookName: "abcdXXXXXXXXXXXX", Author: ""} // 1−12/16 = 0.25
	out2 := FilterSortBySimilarity([]SearchResult{edge, exact}, "abcd")
	if len(out2) != 1 || out2[0].BookName != "abcd" {
		t.Fatalf("0.25 应被过滤（严格 > 阈值）: %+v", out2)
	}
}

// ── 排版归一（规格 §7「两路排版归一」+ §4 D3 直接子级改名路线）──────────

func TestParagraphsClosedRouteNonPBlocks(t *testing.T) {
	got := paragraphsFromHTML("<div>甲段</div>\n<div>乙段</div>", true, "")
	if len(got) != 2 || got[0] != "甲段" || got[1] != "乙段" {
		t.Fatalf("闭合路线的非 <p> 块级子级应各自成段（上游「改名 <p>」等价）: %#v", got)
	}
	mixed := paragraphsFromHTML("正文前<div>块中</div>正文后", true, "")
	if len(mixed) != 3 || mixed[0] != "正文前" || mixed[1] != "块中" || mixed[2] != "正文后" {
		t.Fatalf("文本节点与块级子级应按文档序成段: %#v", mixed)
	}
	ps := paragraphsFromHTML("<p>一</p><p>二</p>", true, "")
	if len(ps) != 2 {
		t.Fatalf("标准 <p> 路线不得回退: %#v", ps)
	}
}

func TestParagraphsNonClosedBrRoute(t *testing.T) {
	// 缺省切段正则（paragraphTag 留空 → <br> 系）
	got := paragraphsFromHTML("<p>一</p><br><br><p>二</p>", false, "")
	if len(got) != 2 || got[0] != "一" || got[1] != "二" {
		t.Fatalf("非闭合路线缺省应按 <br> 切段: %#v", got)
	}
	got2 := paragraphsFromHTML("一<br>二<br/><br>三", false, `(<br\s*/?>\s*)+`)
	if len(got2) != 3 || got2[2] != "三" {
		t.Fatalf("非闭合路线自定义切段正则不符: %#v", got2)
	}
}

// ── 正文分页上限（不得静默截断）────────────────────────────────────────

func TestChapterPaginationCapIsExplicitError(t *testing.T) {
	// 递增 URL 链：每页都给出「下一页」且 URL 各不相同（自环会被 next==cur 守卫
	// 收敛，测不到上限），文本「下一页」不在通用末页词表里 → 只能靠上限兜底。
	var pageNo int32
	f := &funcFetcher{fn: func(_ context.Context, _ Request) (*Page, error) {
		n := atomic.AddInt32(&pageNo, 1)
		body := fmt.Sprintf(`<html><body><h1>标题</h1><div id="content"><p>正文%d</p></div>
<a class="nav" href="/p/%d.html">下一页</a></body></html>`, n, n+1)
		return &Page{URL: fmt.Sprintf("https://s.example/p/%d.html", n), Body: []byte(body)}, nil
	}}
	rule := &Rule{Name: "环形分页", Chapter: &ChapterRule{Title: "h1", Content: "#content", NextPage: ".nav"}}
	_, err := New(rule, Options{Fetcher: f, Sleeper: instantSleeper}).
		Chapter(context.Background(), TocEntry{Title: "标题", URL: "https://s.example/c/1.html", Order: 1})
	if err == nil {
		t.Fatal("环形/超长分页必须显式报错，不得静默截断正文")
	}
	if !strings.Contains(err.Error(), "上限") {
		t.Fatalf("报错应说明分页上限: %v", err)
	}
	if n := atomic.LoadInt32(&pageNo); n > maxChapterPages {
		t.Fatalf("触顶后不得继续请求: %d 页", n)
	}
}

func TestDownloadCancelRecordsFailuresWithoutPanic(t *testing.T) {
	const tocPage = `<html><body><div id="list">
<a href="/ch/1">一</a><a href="/ch/2">二</a><a href="/ch/3">三</a>
</div></body></html>`
	ctx, cancel := context.WithCancel(context.Background())
	var once sync.Once
	f := &funcFetcher{fn: func(_ context.Context, req Request) (*Page, error) {
		if req.URL == "https://s.example/toc" {
			return &Page{URL: req.URL, Body: []byte(tocPage)}, nil
		}
		once.Do(cancel) // 首章抓取即取消整轮
		return nil, context.Canceled
	}}
	rule := &Rule{Name: "取消", Toc: &TocRule{Item: "#list a"}, Chapter: &ChapterRule{Title: "h1", Content: "#content"}}
	rep, err := New(rule, Options{Fetcher: f, Sleeper: instantSleeper}).
		Download(ctx, "https://s.example/toc", DownloadOptions{})
	if err != nil {
		t.Fatalf("取消应回报告而非错误: %v", err)
	}
	if len(rep.Chapters) != 0 || len(rep.Failed) != 3 {
		t.Fatalf("取消后未取到的章节应全进 Failed: %+v", rep)
	}
	for _, fc := range rep.Failed {
		if strings.TrimSpace(fc.Err) == "" {
			t.Fatalf("失败原因不得为空串: %+v", fc)
		}
	}
}

// funcFetcher 函数式假通道（需要按调用序做副作用的用例）。
type funcFetcher struct {
	fn func(context.Context, Request) (*Page, error)
}

func (f *funcFetcher) Fetch(ctx context.Context, req Request) (*Page, error) { return f.fn(ctx, req) }

// ── 详情页与组装（规格 §7「TXT 信息头」）────────────────────────────────

func TestBookDetailMetaFallbacksAndCoverSrc(t *testing.T) {
	const detail = `<html><head>
<meta property="og:novel:book_name" content="元数据书名">
<meta property="og:novel:author" content="元数据作者">
<meta property="og:description" content="元数据简介">
<meta property="og:image" content="https://img.example/meta.jpg">
</head><body><div id="bn">选择器书名</div><img id="cover" src="/img/cover.jpg"></body></html>`
	f := &staticFetcher{pages: map[string]string{"https://s.example/book/1/": detail}}
	rule := &Rule{Name: "详情", Book: &BookRule{BookName: "#bn", Author: "#none", CoverURL: "#cover@src"}}
	info, err := New(rule, Options{Fetcher: f, Sleeper: instantSleeper}).
		BookDetail(context.Background(), "https://s.example/book/1/")
	if err != nil {
		t.Fatalf("详情失败: %v", err)
	}
	if info.Name != "选择器书名" {
		t.Fatalf("选择器优先于 meta: %q", info.Name)
	}
	if info.Author != "元数据作者" || info.Intro != "元数据简介" {
		t.Fatalf("选择器缺失应回落 meta: %+v", info)
	}
	if info.CoverURL != "https://s.example/img/cover.jpg" {
		t.Fatalf("@src 应抽属性并补全: %q", info.CoverURL)
	}
	noBook := &Rule{Name: "无详情段"}
	empty, err := New(noBook, Options{Fetcher: f, Sleeper: instantSleeper}).
		BookDetail(context.Background(), "https://s.example/book/1/")
	if err != nil || empty.Name != "" {
		t.Fatalf("缺 book 段应返回零值不报错: %v %+v", err, empty)
	}
	// 页址缺失（注入式 Fetcher 不回报 URL）→ 封面等相对链接回落 baseUri
	baseRule := &Rule{Name: "无页址详情", BaseURI: "https://base.example/",
		Book: &BookRule{BookName: "#bn", CoverURL: "#cover@src"}}
	noURL := &emptyURLFetcher{body: `<html><body><div id="bn">书名</div><img id="cover" src="/img/c.jpg"></body></html>`}
	info3, err := New(baseRule, Options{Fetcher: noURL, Sleeper: instantSleeper}).
		BookDetail(context.Background(), "https://base.example/book/1/")
	if err != nil || info3.CoverURL != "https://base.example/img/c.jpg" {
		t.Fatalf("页址缺失时封面应回落 baseUri: %v %+v", err, info3)
	}
}

func TestAssembleTXTIntroAndOrderFallbacks(t *testing.T) {
	out, err := AssembleTXT(BookInfo{Name: "书", Author: "人"},
		[]ChapterText{{Title: "第一章", Order: 0, Paragraphs: []string{"正文"}}}, "")
	if err != nil {
		t.Fatalf("组装失败: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "简介：暂无") {
		t.Fatalf("简介缺失应回落「暂无」:\n%s", s)
	}
	if !strings.Contains(s, "第1章 第一章") {
		t.Fatalf("Order<=0 应按序回填章号:\n%s", s)
	}
}

// ── 生产 Fetcher（规格 §7「抓取纪律」生产实现分支）──────────────────────

func TestHTTPFetcherPostCookieAndNon200(t *testing.T) {
	var gotMethod, gotCT, gotForm, gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		gotCookie = r.Header.Get("Cookie")
		_ = r.ParseForm()
		gotForm = r.PostFormValue("searchkey")
		_, _ = w.Write([]byte("<html><body>ok</body></html>"))
	}))
	defer srv.Close()
	f := &HTTPFetcher{Client: srv.Client(), Timeout: 2 * time.Second}
	page, err := f.Fetch(context.Background(), Request{
		Method:  "post",
		URL:     srv.URL + "/s",
		Form:    url.Values{"searchkey": {"大道"}},
		Cookies: map[string]string{"night": "1"},
	})
	if err != nil {
		t.Fatalf("POST 抓取失败: %v", err)
	}
	if !strings.Contains(string(page.Body), "ok") {
		t.Fatalf("响应体不符: %q", page.Body)
	}
	if gotMethod != http.MethodPost || !strings.Contains(gotCT, "application/x-www-form-urlencoded") {
		t.Fatalf("POST 请求形态不符: method=%q ct=%q", gotMethod, gotCT)
	}
	if gotForm != "大道" {
		t.Fatalf("表单槽位未编码: %q", gotForm)
	}
	if !strings.Contains(gotCookie, "night=1") {
		t.Fatalf("cookies 未随请求: %q", gotCookie)
	}
	if _, err := f.Fetch(context.Background(), Request{URL: srv.URL + "/fail"}); err == nil {
		t.Fatal("非 200 应显式报错")
	}
}
