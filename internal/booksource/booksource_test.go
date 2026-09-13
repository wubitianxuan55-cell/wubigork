package booksource

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/bookimport"
)

// ── 测试基建：假站点 + 记录型依赖（零真实网络、零真实睡眠）─────────────────

// mapServer 路径→HTML 的假站点；hits/fails 支持命中计数与「先失败 N 次」。
type mapServer struct {
	srv     *httptest.Server
	mu      sync.Mutex
	hits    map[string]int
	fails   map[string]int
	headers func(r *http.Request)
}

func newMapServer(t *testing.T) *mapServer {
	t.Helper()
	m := &mapServer{hits: map[string]int{}, fails: map[string]int{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.hits[r.URL.Path]++
		if n := m.fails[r.URL.Path]; n > 0 {
			m.fails[r.URL.Path]--
			m.mu.Unlock()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		m.mu.Unlock()
		if m.headers != nil {
			m.headers(r)
		}
		var body string
		switch {
		case strings.HasPrefix(r.URL.Path, "/search2"):
			body = pageSearchPage2
		case strings.HasPrefix(r.URL.Path, "/search"):
			body = pageSearch
		case strings.HasPrefix(r.URL.Path, "/book/123"):
			body = pageBookDetail
		case strings.HasPrefix(r.URL.Path, "/toc1"):
			body = pageTocP1
		case strings.HasPrefix(r.URL.Path, "/toc2"):
			body = pageTocP2
		case strings.HasPrefix(r.URL.Path, "/ch/1"):
			body = pageChapter1
		case strings.HasPrefix(r.URL.Path, "/ch/2_1"):
			body = pageChapter2P2
		case strings.HasPrefix(r.URL.Path, "/ch/2"):
			body = pageChapter2
		case strings.HasPrefix(r.URL.Path, "/ch/3"):
			body = pageChapter3 // 正文空（限流信号）
		case strings.HasPrefix(r.URL.Path, "/serp"):
			body = pageSERP
		case strings.HasPrefix(r.URL.Path, "/gtoc2"):
			body = pageGuessToc2
		case strings.HasPrefix(r.URL.Path, "/gtoc1"):
			body = pageGuessToc1
		case strings.HasPrefix(r.URL.Path, "/gch1_p2"):
			body = pageGuessChapterP2
		case strings.HasPrefix(r.URL.Path, "/gch1"):
			body = pageGuessChapter1
		case strings.HasPrefix(r.URL.Path, "/cf"):
			body = "<html><head><title>Just a moment...</title></head><body></body></html>"
		case strings.HasPrefix(r.URL.Path, "/echo"):
			hj, _ := json.Marshal(map[string]string{
				"ua": r.Header.Get("User-Agent"), "referer": r.Header.Get("Referer"),
				"cookie": r.Header.Get("Cookie"), "method": r.Method,
				"body": r.FormValue("searchkey"),
			})
			w.Header().Set("Content-Type", "application/json")
			body = "<html><body><pre id='echo'>" + string(hj) + "</pre></body></html>"
		default:
			body = "<html><body>missing</body></html>"
		}
		_, _ = w.Write([]byte(body))
	})
	m.srv = httptest.NewServer(mux)
	t.Cleanup(m.srv.Close)
	return m
}

func (m *mapServer) url(path string) string { return m.srv.URL + path }
func (m *mapServer) count(path string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hits[path]
}
func (m *mapServer) failFirst(path string, n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fails[path] = n
}

// recordingSleeper 记录等待时长，不真睡。
type recordingSleeper struct {
	mu    sync.Mutex
	spent []time.Duration
}

func (s *recordingSleeper) Sleep(_ context.Context, d time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spent = append(s.spent, d)
	return nil
}

func testOptions(m *mapServer) (Options, *recordingSleeper) {
	s := &recordingSleeper{}
	return Options{Fetcher: &HTTPFetcher{Client: m.srv.Client(), Timeout: 2 * time.Second}, Sleeper: s.Sleep}, s
}

// baseRule 指向假站点的完整规则（各测试按需覆盖字段）。
func baseRule(m *mapServer) *Rule {
	return &Rule{
		Name:    "假源",
		BaseURI: m.url("/") + "/",
		Search: &SearchRule{
			URL:           m.url("/search") + "?q=%s",
			Result:        ".result",
			BookName:      ".title a",
			Author:        ".author",
			LatestChapter: ".latest",
			NextPage:      ".pager a",
			Limit:         10,
		},
		Book: &BookRule{
			URL:      regexp.QuoteMeta(m.url("/book/")) + `(\d+)/`,
			BookName: "#bn", Author: "#au", Intro: "#intro",
		},
		Toc: &TocRule{URL: m.url("/toc1/%s"), Item: "#list a"},
		Chapter: &ChapterRule{
			Title: "h1", Content: "#content", ParagraphTagClosed: true,
			FilterTxt: `本站广告|\(本章完\)`, FilterTag: "script",
		},
		Crawl: &CrawlRule{MinIntervalMs: 1, MaxIntervalMs: 2, RetryMinMs: 10, RetryMaxMs: 10},
	}
}

// ── 夹具页面 ─────────────────────────────────────────────────────────────

const pageSearch = `<html><head><title>搜索</title></head><body>
<div class="result"><span class="title"><a href="/book/123/">大道朝天</a></span><span class="author">猫腻</span><span class="latest">第100章</span></div>
<div class="result"><span class="title"><a href="/book/456/">择天记</a></span><span class="author">完全不同的某人</span><span class="latest">第55章</span></div>
<div class="pager"><a href="/search2">2</a></div>
</body></html>`

const pageSearchPage2 = `<html><head><title>搜索</title></head><body>
<div class="result"><span class="title"><a href="/book/789/">间客</a></span><span class="author">猫腻</span><span class="latest">第200章</span></div>
</body></html>`

const pageBookDetail = `<html><head><title>详情</title></head><body>
<div id="bn">大道朝天</div><div id="au">猫腻</div><div id="intro">千里杀一人，十步不愿行。</div>
</body></html>`

const pageTocP1 = `<html><body><div id="list">
<a href="/ch/1">第一章 起身</a><a href="/ch/2">第二章 施展</a>
</div><select id="pages"><option value="/toc2">2</option></select></body></html>`

const pageTocP2 = `<html><body><div id="list">
<a href="/ch/3">第三章 空章</a>
</div></body></html>`

// 第一章夹具故意压进：正文开头重复标题、私有区 PUA 字符（\uE0B5）、广告行、
// script 标签、空段落、「1.标题」形态——清洗链全量素材（规格 §2.8）。
const pageChapter1 = "<html><body><h1>1.第一章 起身</h1><div id=\"content\">\n" +
	"<p>第一章 起身</p><p>朝闻道，夕死可矣。</p><p>\uE0B5剑出鞘。</p><p>本站广告</p>" +
	"<script>evil()</script><p></p>\n</div></body></html>"

const pageChapter2 = `<html><body><h1>第二章 施展</h1><div id="content">
<p>剑气纵横。</p><a class="nav" href="/ch/2_1.html">下一页</a>
</div></body></html>`

const pageChapter2P2 = `<html><body><div id="content">
<p>剑落无声。</p><a class="nav" href="/ch/3">下一章</a>
</div></body></html>`

const pageChapter3 = `<html><body><h1>第三章 空章</h1><div id="content"></div></body></html>`

// ── 泛搜索与免规则夹具（owllook 蒸馏）───────────────────────────────────

// 假 SERP：合法候选 / index.html 剥离 / .html 详情拒绝 / 重复去重 / 黑名单 /
// 站点根拒绝（引擎自家域过滤由 redirect 测试覆盖）。
const pageSERP = `<html><body>
<div class="b_algo"><h2><a href="https://site-a.com/book/55/">大道朝天 - siteA</a></h2></div>
<div class="b_algo"><h2><a href="https://site-b.com/book/66/index.html">大道朝天 - siteB</a></h2></div>
<div class="b_algo"><h2><a href="https://site-c.com/book/77/8845907.html">大道朝天 第一章</a></h2></div>
<div class="b_algo"><h2><a href="https://site-a.com/book/55/">大道朝天 - siteA 复制</a></h2></div>
<div class="b_algo"><h2><a href="https://www.blacklist.com/book/8/">大道朝天 - 黑名单</a></h2></div>
<div class="b_algo"><h2><a href="https://site-d.com/">站点根</a></h2></div>
</body></html>`

// 免规则目录两页：页内含卷锚点/全角数字/噪声锚点/重复 URL；页间「下一页」串联。
const pageGuessToc1 = `<html><body>
<div class="nav"><a href="/">关于我们</a><a href="/gtoc1">本页</a></div>
<a href="/ch/8845907.html">第一章 起身</a>
<a href="/ch/8845908.html">第一卷 风起（序）</a>
<a href="/ch/第３章.html">第３章 试剑</a>
<a href="/ch/8845908.html">第二章 重复URL</a>
<a href="/gtoc2">下一页</a><a href="/gtoc1">上一页</a>
</body></html>`

const pageGuessToc2 = `<html><body>
<a href="/ch/8845909.html">第四章 收官</a>
<a href="/gtoc2">下一页</a>
</body></html>`

// 免规则正文：<title> 带章节名，AutoNext 翻「下一页」，「下一章」不得吞入。
const pageGuessChapter1 = `<html><head><title>第一章 起身_大道朝天_假站</title></head><body>
<div id="content"><p>甲。</p></div>
<a href="/gch1_p2">下一页</a>
</body></html>`

const pageGuessChapterP2 = `<html><head><title>正文页</title></head><body>
<div id="content"><p>乙。</p></div>
<a href="/ch/9999999.html">下一章</a>
</body></html>`

func serpEngine(m *mapServer) *SearchEngineRule {
	return &SearchEngineRule{
		Name:        "假引擎",
		URL:         m.url("/serp") + "?q=%s",
		QueryFormat: "%s 小说 免费阅读",
		Result:      ".b_algo",
		Title:       "h2 a",
	}
}

// ── 规则与校验 ──────────────────────────────────────────────────────────

func TestTemplateParsesAndValidates(t *testing.T) {
	r, err := Parse(TemplateJSON)
	if err != nil {
		t.Fatalf("出厂模板必须始终可通过校验（防模板与 schema 漂移）: %v", err)
	}
	if r.Name == "" || r.Search == nil || r.Toc == nil || r.Chapter == nil {
		t.Fatalf("模板段缺失: %+v", r)
	}
	if !r.Disabled {
		t.Fatal("模板必须 disabled=true（不得被聚合搜索拾取）")
	}
}

func TestValidateFailClosed(t *testing.T) {
	m := newMapServer(t)
	cases := []struct {
		name    string
		mutate  func(*Rule)
		wantSub string
	}{
		{"缺name", func(r *Rule) { r.Name = "" }, "name"},
		{"空段", func(r *Rule) { r.Search, r.Book, r.Toc, r.Chapter = nil, nil, nil, nil }, "至少需要"},
		{"js步骤", func(r *Rule) { r.Search.URL = "https://x.com/s@js:code" }, "@js:"},
		{"xpath", func(r *Rule) { r.Toc.Item = "//div/a" }, "XPath"},
		{"坏正则", func(r *Rule) { r.Chapter.FilterTxt = "((" }, "正则"},
		{"书id无捕获组", func(r *Rule) { r.Book.URL = regexp.QuoteMeta(m.url("/book/")) + "\\d+" }, "捕获组"},
		{"负并发", func(r *Rule) { r.Crawl.Concurrency = -1 }, "负"},
		{"坏method", func(r *Rule) { r.Search.Method = "put" }, "get/post"},
		{"坏item", func(r *Rule) { r.Toc.Item = "//a" }, "XPath"},
		{"坏css", func(r *Rule) { r.Chapter.Content = "##" }, "选择器"},
	}
	for _, c := range cases {
		r := baseRule(m)
		c.mutate(r)
		err := r.Validate()
		if err == nil {
			t.Fatalf("%s: 期望 fail-closed 报错", c.name)
		}
		if !strings.Contains(err.Error(), c.wantSub) {
			t.Fatalf("%s: 报错应含「%s」，得到 %v", c.name, c.wantSub, err)
		}
	}
}

func TestLoadDirReportsPerFileErrors(t *testing.T) {
	dir := t.TempDir()
	good := baseRule(newMapServer(t))
	gb, _ := json.Marshal(good)
	if err := os.WriteFile(filepath.Join(dir, "b-good.json"), gb, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a-bad.json"), []byte(`{"name":""}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c-ignored.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded := LoadDir(dir)
	if len(loaded) != 2 {
		t.Fatalf("只应收 *.json，得到 %d", len(loaded))
	}
	if loaded[0].File != "a-bad.json" || loaded[0].Err == nil {
		t.Fatalf("坏文件应带错误且按文件名序: %+v", loaded[0])
	}
	if loaded[1].Err != nil || loaded[1].Rule.Name != good.Name {
		t.Fatalf("好文件应装载成功: %+v", loaded[1])
	}
}

func TestValidateAllowsGuessRoutes(t *testing.T) {
	// 免规则路线（owllook 蒸馏）：toc.item 与 chapter.title 可空，正文仍必填
	m := newMapServer(t)
	r := baseRule(m)
	r.Toc.Item = ""
	r.Chapter.Title = ""
	if err := r.Validate(); err != nil {
		t.Fatalf("免规则路线应通过校验: %v", err)
	}
	r.Chapter.Content = ""
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "chapter.content") {
		t.Fatalf("正文选择器仍必须 fail-closed: %v", err)
	}
}

// ── 抓取纪律 ────────────────────────────────────────────────────────────

func TestFetchDisciplineHeaders(t *testing.T) {
	m := newMapServer(t)
	var got map[string]string
	var mu sync.Mutex
	m.headers = func(r *http.Request) {
		if r.URL.Path != "/search" { // 只捕获搜索主请求（分页页会覆盖）
			return
		}
		hj, _ := json.Marshal(map[string]string{
			"ua": r.Header.Get("User-Agent"), "referer": r.Header.Get("Referer"),
			"cookie": r.Header.Get("Cookie"), "method": r.Method,
			"body": r.FormValue("searchkey"),
		})
		mu.Lock()
		defer mu.Unlock()
		_ = json.Unmarshal(hj, &got)
	}
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Search.Method = "post"
	r.Search.Data = map[string]string{"searchkey": "%s"}
	r.Search.Cookies = map[string]string{"night": "1"}
	e := New(r, opt)
	if _, err := e.Search(context.Background(), "大道"); err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if got["ua"] == "" {
		t.Fatal("缺 User-Agent（随机 UA 池）")
	}
	if !strings.HasPrefix(got["referer"], m.srv.URL) {
		t.Fatalf("Referer 应为本站根: %q", got["referer"])
	}
	if !strings.Contains(got["cookie"], "night=1") {
		t.Fatalf("规则 cookies 未随请求: %q", got["cookie"])
	}
	if got["method"] != "POST" || got["body"] != "大道" {
		t.Fatalf("POST 表单槽位应填关键字: %+v", got)
	}
}

func TestFetchDisciplineRetryBackoffLinear(t *testing.T) {
	m := newMapServer(t)
	m.failFirst("/ch/1", 2) // 前两次 500，第三次成功 → 恰好两条重试间隔
	opt, sleeper := testOptions(m)
	r := baseRule(m)
	e := New(r, opt)
	toc, err := e.Toc(context.Background(), m.url("/book/123/"), 1, 1)
	if err != nil || len(toc) != 1 {
		t.Fatalf("目录失败: %v %+v", err, toc)
	}
	if _, err := e.Chapter(context.Background(), toc[0]); err != nil {
		t.Fatalf("重试后应成功: %v", err)
	}
	if m.count("/ch/1") != 3 {
		t.Fatalf("应请求 3 次（1+2 重试），实际 %d", m.count("/ch/1"))
	}
	// 首请求前 = 抖动；两次重试 = retry(10ms)×1、×2 线性放大（规格 §2.3）。
	sleeper.mu.Lock()
	defer sleeper.mu.Unlock()
	chapterSleeps := sleeper.spent[len(sleeper.spent)-3:]
	if chapterSleeps[1] != 10*time.Millisecond || chapterSleeps[2] != 20*time.Millisecond {
		t.Fatalf("重试间隔应 10ms→20ms 线性放大: %v", chapterSleeps)
	}
}

func TestCancelledCtxZeroCalls(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	e := New(baseRule(m), opt)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.Search(ctx, "大道"); !errors.Is(err, context.Canceled) {
		t.Fatalf("期望 ctx 取消错误，得到 %v", err)
	}
	if m.count("/search") != 0 {
		t.Fatalf("ctx 取消后必须零调用，实际 %d", m.count("/search"))
	}
}

func TestCloudflareDetectedNotBypassed(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Search.URL = m.url("/cf") + "?q=%s"
	r.Search.NextPage = ""
	e := New(r, opt)
	_, err := e.Search(context.Background(), "大道")
	if !errors.Is(err, ErrCloudflare) {
		t.Fatalf("搜索首页命中 CF 应报 ErrCloudflare，得到 %v", err)
	}
}

// ── 搜索 ────────────────────────────────────────────────────────────────

func TestSearchParseLimitAndPagination(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Search.Limit = 2 // 首页 2 条 + 第二页 1 条 → 合并后截 2
	e := New(r, opt)
	res, err := e.Search(context.Background(), "大道朝天")
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("limit 截断应得 2 条，得到 %d", len(res))
	}
	if res[0].BookName != "大道朝天" || res[0].Author != "猫腻" || res[0].LatestChapter != "第100章" {
		t.Fatalf("字段抽取不符: %+v", res[0])
	}
	if !strings.HasPrefix(res[0].URL, m.srv.URL) || !strings.HasSuffix(res[0].URL, "/book/123/") {
		t.Fatalf("详情链接应相对补全: %q", res[0].URL)
	}
	if m.count("/search2") == 0 {
		t.Fatal("nextPage 分页集合应被抓取")
	}
}

func TestSearchExactMatchFallsBackToDetail(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Search.URL = m.url("/book/123/") // 「搜索」直接跳详情页的站点形态
	r.Search.Result = ".none"
	r.Search.NextPage = ""
	e := New(r, opt)
	res, err := e.Search(context.Background(), "大道朝天")
	if err != nil || len(res) != 1 {
		t.Fatalf("完全匹配应构造单条结果: %v %+v", err, res)
	}
	if res[0].BookName != "大道朝天" || res[0].Author != "猫腻" || res[0].URL != m.url("/book/123/") {
		t.Fatalf("详情构造不符: %+v", res[0])
	}
}

func TestAggregateDegradedAndSorted(t *testing.T) {
	m := newMapServer(t)
	m.failFirst("/badsearch", 999) // 始终 500 → 重试穷尽后单源降级
	opt, _ := testOptions(m)
	good := baseRule(m)
	bad := baseRule(m)
	bad.Name = "坏源"
	bad.Search.URL = m.url("/badsearch") + "?q=%s"
	disabled := baseRule(m)
	disabled.Name = "禁用源"
	disabled.Disabled = true
	res, errs := Aggregate(context.Background(), opt, "大道朝天", good, bad, disabled)
	if len(errs) != 1 || errs[0].Source != "坏源" {
		t.Fatalf("坏源应单独降级上报: %+v", errs)
	}
	if len(res) == 0 {
		t.Fatal("好源结果不应丢失")
	}
	for _, r := range res {
		if r.Source == "禁用源" {
			t.Fatal("禁用源不得参与聚合")
		}
	}
	if res[0].BookName != "大道朝天" || res[0].Similarity != 1 {
		t.Fatalf("完全命中应排首位且相似度 1: %+v", res[0])
	}
}

// ── 相似度（纯函数）──────────────────────────────────────────────────────

func TestSimilarityBasics(t *testing.T) {
	if similarity("大道朝天", "大道朝天") != 1 {
		t.Fatal("全等应得 1")
	}
	if similarity("", "") != 1 {
		t.Fatal("双空全等")
	}
	if similarity("abc", "") != 0 {
		t.Fatal("单侧空为 0")
	}
	if s := similarity("大道朝天", "大道朝"); s <= 0.7 || s >= 1 {
		t.Fatalf("前缀近似应落 (0.7,1): %v", s)
	}
}

func TestFilterSortDetectsAuthorQuery(t *testing.T) {
	in := []SearchResult{
		{BookName: "完全不同的书", Author: "猫腻"},
		{BookName: "大道朝天", Author: "猫腻"},
		{BookName: "近似的书名啊", Author: "别人"},
	}
	out := FilterSortBySimilarity(in, "猫腻")
	if len(out) != 2 || out[0].Author != "猫腻" {
		t.Fatalf("作者搜应识别并过滤书名近似项: %+v", out)
	}
	// 书名搜：完全命中排前
	in2 := []SearchResult{
		{BookName: "大道朝天（转载）", Author: "某人"},
		{BookName: "大道朝天", Author: "猫腻"},
	}
	out2 := FilterSortBySimilarity(in2, "大道朝天")
	if out2[0].BookName != "大道朝天" {
		t.Fatalf("完全命中应排前: %+v", out2)
	}
	// 全部 ≤0.25 → 回退保留 >0（规格 §2.5）。「风」对 8 字关键字仅 0.125，
	// 「逆风而行」为 0：过滤路线全空，回退只留 >0 者并保持降序。
	in3 := []SearchResult{{BookName: "逆风而行", Author: "甲"}, {BookName: "风", Author: "乙"}}
	out3 := FilterSortBySimilarity(in3, "风起云涌龙腾虎跃")
	if len(out3) != 1 || out3[0].BookName != "风" {
		t.Fatalf("过滤空应回退 >0 并保持降序: %+v", out3)
	}
}

// ── 目录 ────────────────────────────────────────────────────────────────

func TestTocPaginationMergedInOrderAndRange(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Toc.NextPage = "option" // 下拉菜单一次性取全（value 属性承载）
	e := New(r, opt)
	toc, err := e.Toc(context.Background(), m.url("/book/123/"), 0, 0)
	if err != nil {
		t.Fatalf("目录失败: %v", err)
	}
	if len(toc) != 3 {
		t.Fatalf("两页目录应合并 3 章，得到 %d: %+v", len(toc), toc)
	}
	if toc[0].Title != "第一章 起身" || toc[2].Title != "第三章 空章" {
		t.Fatalf("页序归并错乱: %+v", toc)
	}
	if toc[0].Order != 1 || toc[2].Order != 3 {
		t.Fatalf("Order 应重排: %+v", toc)
	}
	if m.count("/toc2") == 0 {
		t.Fatal("下拉分页应被抓取")
	}
	// 范围 [2,3]：从第二章起、重新编号
	toc2, err := e.Toc(context.Background(), m.url("/book/123/"), 2, 3)
	if err != nil || len(toc2) != 2 || toc2[0].Title != "第二章 施展" || toc2[0].Order != 1 {
		t.Fatalf("范围解析不符: %v %+v", err, toc2)
	}
}

func TestTocReverseAndEmpty(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Toc.Reverse = true
	r.Toc.NextPage = ""
	e := New(r, opt)
	toc, err := e.Toc(context.Background(), m.url("/book/123/"), 0, 0)
	if err != nil || toc[0].Title != "第二章 施展" {
		t.Fatalf("reverse 应反转章序: %v %+v", err, toc)
	}
	r.Toc.Item = ".none"
	if _, err := e.Toc(context.Background(), m.url("/book/123/"), 0, 0); !errors.Is(err, ErrEmptyToc) {
		t.Fatalf("空目录应报 ErrEmptyToc: %v", err)
	}
}

// ── 正文 ────────────────────────────────────────────────────────────────

func TestChapterCleanChainAndTitleNormalize(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	r := baseRule(m)
	e := New(r, opt)
	toc, err := e.Toc(context.Background(), m.url("/book/123/"), 1, 1)
	if err != nil || len(toc) != 1 {
		t.Fatalf("目录失败: %v %+v", err, toc)
	}
	ch, err := e.Chapter(context.Background(), toc[0])
	if err != nil {
		t.Fatalf("章节失败: %v", err)
	}
	if ch.Title != "第1章 第一章 起身" {
		t.Fatalf("「1.标题」应归一为「第1章 …」: %q", ch.Title)
	}
	want := []string{"朝闻道，夕死可矣。", "剑出鞘。"}
	if len(ch.Paragraphs) != 2 || ch.Paragraphs[0] != want[0] || ch.Paragraphs[1] != want[1] {
		t.Fatalf("清洗链产物不符: got %#v want %#v", ch.Paragraphs, want)
	}
}

func TestCleanChainUnits(t *testing.T) {
	// 不可见字符：PUA/ZWSP 清除、普通空白保留（规格 §2.8 第 1 条）
	if got := cleanInvisibleChars("a\u200bB\uE0B5 c"); got != "aB c" {
		t.Fatalf("不可见字符清理不符: %q", got)
	}
	// 实体全删（阅读器兼容，上游同语义；孤立的尾分号不在实体形态内）
	if got := htmlEntity.ReplaceAllString("x&nbsp;y&amp;z;", ""); got != "xyz;" {
		t.Fatalf("实体清理不符: %q", got)
	}
	// 开头标题剥离：含正则元字符的标题不得破坏剥离（QuoteMeta）
	if got := stripLeadingTitle("[ ( .第一章) 开始", "[ ( .第一章)"); got != " 开始" {
		t.Fatalf("元字符标题剥离不符: %q", got)
	}
	// 「12.章节名」归一（阅读器目录兼容）
	if normalizeTitle("12.武器库") != "第12章 武器库" {
		t.Fatalf("标题归一不符: %q", normalizeTitle("12.武器库"))
	}
	if normalizeTitle("第三章 施展") != "第三章 施展" {
		t.Fatal("已是章号形态不得改写")
	}
	// 空标签清删（<br> 由 x/net/html 序列化为 <br/>）
	if got := removeEmptyTags("<p></p>留<p> </p><br><span></span>下"); !strings.Contains(got, "留") || !strings.Contains(got, "<br") || strings.Contains(got, "<span>") {
		t.Fatalf("空标签清理不符: %q", got)
	}
}

func TestChapterNonClosedParagraphRoute(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Chapter.ParagraphTagClosed = false
	r.Chapter.ParagraphTag = `<br\s*>+`
	e := New(r, opt)
	ch, err := e.Chapter(context.Background(), TocEntry{Title: "第一章 起身", URL: m.url("/ch/1"), Order: 1})
	if err != nil {
		t.Fatalf("章节失败: %v", err)
	}
	// 非闭合路线：清洗后的 HTML 以 <br>（隐式）分隔——夹具无 <br>，退化为整段；
	// 验证段落不空且无标签残留即可。
	if len(ch.Paragraphs) == 0 {
		t.Fatal("非闭合路线应产出段落")
	}
	for _, p := range ch.Paragraphs {
		if strings.ContainsAny(p, "<>") {
			t.Fatalf("段落不得残留标签: %q", p)
		}
	}
}

func TestChapterPaginatedEndHeuristics(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Chapter.NextPage = ".nav"
	e := New(r, opt)
	toc, _ := e.Toc(context.Background(), m.url("/book/123/"), 2, 2)
	ch, err := e.Chapter(context.Background(), toc[0])
	if err != nil {
		t.Fatalf("分页章节失败: %v", err)
	}
	got := strings.Join(ch.Paragraphs, "|")
	// /ch/2_1.html 以 _1.html 结尾 → 仍是本章分页；下一按钮指向 /ch/3 且文本
	// 「下一章」+ 非 _N.html 形态 → 通用末页启发式收尾（规格 §2.7）。
	if !strings.Contains(got, "剑气纵横") || !strings.Contains(got, "剑落无声") {
		t.Fatalf("两页正文应拼装: %q", got)
	}
}

func TestChapterEndPatternRule(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Chapter.NextPage = ".nav"
	r.Chapter.EndPattern = `/ch/2(_\d+)?\.html$` // 命中即末页：只取第一页
	e := New(r, opt)
	ch, err := e.Chapter(context.Background(), TocEntry{Title: "第二章 施展", URL: m.url("/ch/2"), Order: 2})
	if err != nil {
		t.Fatalf("章节失败: %v", err)
	}
	if got := strings.Join(ch.Paragraphs, "|"); strings.Contains(got, "剑落无声") {
		t.Fatalf("endPattern 命中后不应继续翻页: %q", got)
	}
}

func TestChapterEmptyContentIsExplicitError(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	e := New(baseRule(m), opt)
	_, err := e.Chapter(context.Background(), TocEntry{Title: "第三章 空章", URL: m.url("/ch/3"), Order: 3})
	if !errors.Is(err, ErrEmptyContent) {
		t.Fatalf("正文空应显式报错（限流信号）: %v", err)
	}
}

// ── 整本编排 + 组装 ─────────────────────────────────────────────────────

func TestDownloadSkipsFailedKeepsOrder(t *testing.T) {
	m := newMapServer(t)
	m.failFirst("/ch/2", 99) // 第二章始终失败 → 进 Failed，不占位
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Toc.NextPage = "option" // 两页目录共 3 章
	e := New(r, opt)
	var lastDone, lastTotal int
	report, err := e.Download(context.Background(), m.url("/book/123/"), DownloadOptions{
		OnProgress: func(done, total int) { lastDone, lastTotal = done, total },
	})
	if err != nil {
		t.Fatalf("下载失败: %v", err)
	}
	if len(report.Chapters) != 1 || report.Chapters[0].Title != "第1章 第一章 起身" {
		t.Fatalf("仅第一章成功: %+v", report.Chapters)
	}
	// 第二章恒 500；第三章正文抽空（ErrEmptyContent 限流信号）——双双如实上报
	if len(report.Failed) != 2 || report.Failed[0].Title != "第二章 施展" || report.Failed[1].Title != "第三章 空章" {
		t.Fatalf("失败章应如实上报: %+v", report.Failed)
	}
	if lastDone != 1 || lastTotal != 3 {
		t.Fatalf("进度回调应到 (1,3): (%d,%d)", lastDone, lastTotal)
	}
}

func TestAssembleTXTAndGBKRoundtrip(t *testing.T) {
	info := BookInfo{Name: "大道朝天", Author: "猫腻", Intro: "千里杀一人。"}
	chapters := []ChapterText{
		{Title: "第一章 起身", Order: 1, Paragraphs: []string{"朝闻道。", "夕死可矣。"}},
		{Title: "第二章 施展", Order: 2, Paragraphs: []string{"剑气纵横。"}},
	}
	utf8Out, err := AssembleTXT(info, chapters, "utf-8")
	if err != nil {
		t.Fatalf("UTF-8 组装失败: %v", err)
	}
	s := string(utf8Out)
	if !strings.Contains(s, "书名：大道朝天") || !strings.Contains(s, "第1章 第一章 起身") || !strings.Contains(s, "朝闻道。\n夕死可矣。") {
		t.Fatalf("TXT 结构不符:\n%s", s)
	}
	gbkOut, err := AssembleTXT(info, chapters, "GBK")
	if err != nil {
		t.Fatalf("GBK 编码失败: %v", err)
	}
	back, enc := bookimport.Decode(gbkOut)
	// GBK ⊂ GB18030：五级链的 gb18030 档先命中并完整解出（行为正确，档名取前者）
	if enc != "gb18030" || !strings.Contains(back, "剑气纵横。") {
		t.Fatalf("GBK 回环失败: enc=%s body=%q", enc, back)
	}
	if _, err := AssembleTXT(info, nil, ""); err == nil {
		t.Fatal("空章节应报错")
	}
	if _, err := AssembleTXT(info, chapters, "big5"); err == nil {
		t.Fatal("不支持的编码应报错")
	}
}

// ── 模板落盘 ────────────────────────────────────────────────────────────

func TestEnsureTemplateIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureTemplate(dir); err != nil {
		t.Fatalf("写模板失败: %v", err)
	}
	if err := EnsureTemplate(dir); err != nil {
		t.Fatalf("重复写应幂等: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "rule-template.json"))
	if err != nil {
		t.Fatalf("读模板失败: %v", err)
	}
	if _, err := Parse(raw); err != nil {
		t.Fatalf("落盘模板应可解析: %v", err)
	}
	// 引擎规则模板（owllook 泛搜索蒸馏）同随落盘
	engRaw, err := os.ReadFile(filepath.Join(dir, "websearch-engines.json"))
	if err != nil {
		t.Fatalf("读引擎模板失败: %v", err)
	}
	if _, err := LoadEnginesFileBytes(engRaw); err != nil {
		t.Fatalf("落盘引擎模板应可装载: %v", err)
	}
}

// v4.283.1 走查补刀回归：v4.283.0 实机 NovelBookSourceSearch 空指针 panic——
// newCrawler 只兜底 Sleeper/Rand，零注入 Fetcher 时 DiscoverySearch 首请求即
// nil 接口解引用（测试全走假站注入，故发布前未暴露）。
func TestNewCrawlerZeroOptionsMustNotNilFetcher(t *testing.T) {
	c := newCrawler(CrawlConfig{}, Options{})
	if c.fetch == nil {
		t.Fatal("零注入 Options 的 crawler.fetch 必须回落生产 HTTPFetcher，不得为 nil")
	}
	if c.sleep == nil || c.rand == nil {
		t.Fatal("Sleeper/Rand 缺省兜底不应被回归破坏")
	}
	// Engine / WebSearcher 两条构造路径同源（Toc/Import/Search 四绑定全走这里）
	if e := New(&Rule{Name: "x"}, Options{}); e == nil || e.crawler == nil || e.crawler.fetch == nil {
		t.Fatal("New 零注入必须有可用 fetch 通道")
	}
	w := NewWebSearcher(WebOptions{})
	if w == nil || w.crawler == nil || w.crawler.fetch == nil {
		t.Fatal("NewWebSearcher 零注入必须有可用 fetch 通道")
	}
}

// 零值 HTTPFetcher（Client/Timeout 全缺省）对本地 httptest 站点真实取回一页，
// 证明缺省生产通道端到端可用（仅回环，零真网络）。
func TestHTTPFetcherZeroValueLocalServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Errorf("应携带随机 UA")
		}
		_, _ = w.Write([]byte(`<html><title>t</title><body><p>正文段落</p></body></html>`))
	}))
	defer srv.Close()
	f := &HTTPFetcher{}
	page, err := f.Fetch(context.Background(), Request{URL: srv.URL})
	if err != nil {
		t.Fatalf("零值 HTTPFetcher 本地取页失败: %v", err)
	}
	if page == nil || !strings.Contains(string(page.Body), "正文段落") {
		t.Fatalf("取回内容不符: %+v", page)
	}
}
