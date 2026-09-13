package booksource

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ── 泛搜索（owllook 蒸馏）测试 ──────────────────────────────────────────

func TestDiscoverySearchParseAndFilters(t *testing.T) {
	m := newMapServer(t)
	var gotQuery string
	m.headers = func(r *http.Request) {
		if r.URL.Path == "/serp" {
			gotQuery = r.URL.Query().Get("q")
		}
	}
	opt, _ := testOptions(m)
	w := NewWebSearcher(WebOptions{Options: opt})
	cands, err := w.DiscoverySearch(context.Background(), serpEngine(m), "大道朝天", WebSearchOptions{
		BlackHosts: []string{"www.blacklist.com"},
		KnownRules: func(host string) bool { return host == "site-a.com" },
	})
	if err != nil {
		t.Fatalf("泛搜索失败: %v", err)
	}
	if gotQuery != "大道朝天 小说 免费阅读" {
		t.Fatalf("queryFormat 塑形应生效: %q", gotQuery)
	}
	if len(cands) != 2 {
		t.Fatalf("应得 2 条候选（.html/重复/黑名单/站点根剔除），得到 %+v", cands)
	}
	if cands[0].URL != "https://site-a.com/book/55/" || !cands[0].HasRule {
		t.Fatalf("候选 0 不符: %+v", cands[0])
	}
	if cands[1].URL != "https://site-b.com/book/66/" || cands[1].HasRule {
		t.Fatalf("index.html 应剥离且无规则标记: %+v", cands[1])
	}
}

func TestDiscoverySearchLinkParamUnwrap(t *testing.T) {
	m := newMapServer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<div class="r"><h2><a href="/x?uddg=https%3A%2F%2Fsite-e.com%2Fbook%2F9%2F">大道朝天</a></h2></div>
</body></html>`))
	})
	m.srv.Config.Handler = mux
	opt, _ := testOptions(m)
	w := NewWebSearcher(WebOptions{Options: opt})
	eng := &SearchEngineRule{Name: "ddg式", URL: m.url("/") + "?q=%s", Result: ".r", Title: "h2 a", LinkParam: "uddg"}
	cands, err := w.DiscoverySearch(context.Background(), eng, "大道朝天", WebSearchOptions{})
	if err != nil || len(cands) != 1 {
		t.Fatalf("uddg 解包失败: %v %+v", err, cands)
	}
	if cands[0].URL != "https://site-e.com/book/9/" {
		t.Fatalf("uddg 应解出真实链接: %+v", cands[0])
	}
}

func TestDiscoverySearchRedirectResolve(t *testing.T) {
	// 重定向宿主独立成站：/jump/1 → 302 → mapServer /book/77/（引擎宿主≠跳转宿主）
	m := newMapServer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/book/77/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html><body>ok</body></html>"))
	})
	mux.HandleFunc("/serp3", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<div class="r"><h2><a href="` + redirSrv.URL + `/jump/1">大道朝天（跳转）</a></h2></div>
</body></html>`))
	})
	m.srv.Config.Handler = mux
	redirSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, m.srv.URL+"/book/77/", http.StatusFound)
	}))
	defer redirSrv.Close()

	opt, _ := testOptions(m)
	w := NewWebSearcher(WebOptions{Options: opt})
	eng := &SearchEngineRule{
		Name: "需解包", URL: m.url("/serp3") + "?q=%s", Result: ".r", Title: "h2 a",
		RedirectHosts: []string{hostOf(redirSrv.URL)},
	}
	cands, err := w.DiscoverySearch(context.Background(), eng, "大道朝天", WebSearchOptions{})
	if err != nil || len(cands) != 1 {
		t.Fatalf("重定向解包失败: %v %+v", err, cands)
	}
	if cands[0].URL != m.srv.URL+"/book/77/" {
		t.Fatalf("应解出真实目录址: %+v", cands[0])
	}
}

// redirSrv 供 SERP 夹具字符串拼接引用（测试内赋值）。
var redirSrv *httptest.Server

func TestDiscoverySearchCache(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	now := time.Unix(1000000, 0)
	w := NewWebSearcher(WebOptions{Options: opt, CacheTTL: time.Minute, Now: func() time.Time { return now }})
	for i := 0; i < 3; i++ {
		if _, err := w.DiscoverySearch(context.Background(), serpEngine(m), "大道朝天", WebSearchOptions{}); err != nil {
			t.Fatalf("第 %d 次搜索失败: %v", i+1, err)
		}
	}
	if m.count("/serp") != 1 {
		t.Fatalf("TTL 内应命中缓存只抓一次，实际 %d", m.count("/serp"))
	}
	now = now.Add(2 * time.Minute) // 越过 TTL
	if _, err := w.DiscoverySearch(context.Background(), serpEngine(m), "大道朝天", WebSearchOptions{}); err != nil {
		t.Fatal(err)
	}
	if m.count("/serp") != 2 {
		t.Fatalf("过期后应重抓，实际 %d", m.count("/serp"))
	}
}

func TestSearchEngineRuleValidate(t *testing.T) {
	if err := (&SearchEngineRule{Name: "x"}).Validate(); err == nil {
		t.Fatal("缺 url 应报错")
	}
	e := &SearchEngineRule{Name: "x", URL: "https://e.com/s?q=%s", Result: ".r", Title: "h2 a"}
	if err := e.Validate(); err != nil {
		t.Fatalf("合法引擎不应报错: %v", err)
	}
	e.Title = "//h2/a"
	if err := e.Validate(); err == nil || !strings.Contains(err.Error(), "XPath") {
		t.Fatalf("XPath 应 fail-closed: %v", err)
	}
}

func TestEnginesTemplateLoadsAndValidates(t *testing.T) {
	list, err := LoadEnginesFileBytes(EnginesTemplateJSON)
	if err != nil {
		t.Fatalf("出厂引擎模板必须始终可装载（防模板与 schema 漂移）: %v", err)
	}
	if len(list) < 2 {
		t.Fatalf("模板应含多个引擎示例: %d", len(list))
	}
}

// ── 免规则解析（owllook 蒸馏）测试 ──────────────────────────────────────

func TestGuessTocEntriesOrderAndDedup(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageGuessToc1))
	if err != nil {
		t.Fatal(err)
	}
	entries := GuessTocEntries(doc, "https://x.com/gtoc1")
	// 期望：数字尾升序；全角数字第３章可识别；重复 URL 去重；
	// 「关于我们/本页」等非章节锚点与「上一页」剔除。
	if len(entries) != 3 {
		t.Fatalf("应得 3 条（去重+剔除后），得到 %d: %+v", len(entries), entries)
	}
	// 无数字尾条目 Order=0 稳定聚前（URL 路径按 url.Parse 规则百分号编码）
	if entries[0].Order != 0 || entries[0].Title != "第３章 试剑" {
		t.Fatalf("无数字尾条目应 Order=0 聚前: %+v", entries[0])
	}
	if entries[1].Order != 8845907 || entries[2].Order != 8845908 {
		t.Fatalf("数字尾应升序: %+v", entries[1:])
	}
	if entries[1].Title != "第一章 起身" || entries[2].Title != "第一卷 风起（序）" {
		t.Fatalf("数字尾相同时应稳定保序: %+v", entries[1:])
	}
}

func TestGuessTocSortsByNumericTail(t *testing.T) {
	html := `<html><body>
<a href="/b/9/999.html">第九章 尾</a><a href="/b/9/100.html">第三章 头</a><a href="/b/9/500.html">第五章 中</a>
</body></html>`
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	entries := GuessTocEntries(doc, "https://x.com/b/9/")
	if len(entries) != 3 || entries[0].URL != "https://x.com/b/9/100.html" || entries[2].URL != "https://x.com/b/9/999.html" {
		t.Fatalf("应按数字尾升序: %+v", entries)
	}
}

func TestGuessNextPageFilters(t *testing.T) {
	html := `<html><body>
<a href="/gtoc1">上一页</a><a href="/hou.html">后一个</a>
<a href="/ch/999.html">下一章</a><a href="/gtoc2">下一页</a><a href="/gtoc2b">下页</a>
</body></html>`
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	next, ok := GuessNextPage(doc, "https://x.com/gtoc1")
	if !ok || next != "https://x.com/gtoc2" {
		t.Fatalf("应取首个「页」型锚点（下一章/后一个不得混入）: %q %v", next, ok)
	}
	self, _ := goquery.NewDocumentFromReader(strings.NewReader(`<html><body><a href="/here">下一页</a></body></html>`))
	if _, ok := GuessNextPage(self, "https://x.com/here"); ok {
		t.Fatal("自链应判末页")
	}
}

func TestGuessChapterTitleFallbacks(t *testing.T) {
	t1, _ := goquery.NewDocumentFromReader(strings.NewReader(`<html><head><title>第一章 起身_大道朝天_假站</title></head><body></body></html>`))
	if got := GuessChapterTitle(t1); got != "第一章 起身" {
		t.Fatalf("title 正则提取不符: %q", got)
	}
	t2, _ := goquery.NewDocumentFromReader(strings.NewReader(`<html><head><title>无关标题</title></head><body><h1>第二章 施展</h1></body></html>`))
	if got := GuessChapterTitle(t2); got != "第二章 施展" {
		t.Fatalf("h1 回退不符: %q", got)
	}
	t3, _ := goquery.NewDocumentFromReader(strings.NewReader(`<html><head><title>纯标题页</title></head><body></body></html>`))
	if got := GuessChapterTitle(t3); got != "纯标题页" {
		t.Fatalf("title 全文回退不符: %q", got)
	}
}

func TestTocGuessRouteMultiPage(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Toc = &TocRule{URL: m.url("/gtoc1")} // Item 留空 → 免规则路线
	e := New(r, opt)
	toc, err := e.Toc(context.Background(), m.url("/gtoc1"), 0, 0)
	if err != nil {
		t.Fatalf("免规则目录失败: %v", err)
	}
	if len(toc) != 4 {
		t.Fatalf("两页目录应合并 4 章（页1 三条 + 页2 一条），得到 %d: %+v", len(toc), toc)
	}
	// 页1 内无数字尾的「第３章」稳定聚前，数字尾升序随后；页2 的第四章收尾
	if toc[0].Title != "第３章 试剑" || toc[1].Title != "第一章 起身" || toc[2].Title != "第一卷 风起（序）" || toc[3].Title != "第四章 收官" || toc[3].Order != 4 {
		t.Fatalf("跨页拼接与重编号不符: %+v", toc)
	}
}

func TestChapterAutoNextAndTitleGuess(t *testing.T) {
	m := newMapServer(t)
	opt, _ := testOptions(m)
	r := baseRule(m)
	r.Chapter = &ChapterRule{Content: "#content", ParagraphTagClosed: true, AutoNext: true} // 无 title 选择器、无 nextPage
	e := New(r, opt)
	ch, err := e.Chapter(context.Background(), TocEntry{Title: "", URL: m.url("/gch1"), Order: 1})
	if err != nil {
		t.Fatalf("免规则章节失败: %v", err)
	}
	if ch.Title != "第一章 起身" {
		t.Fatalf("标题应走 <title> 正则回退: %q", ch.Title)
	}
	// 「下一页」跟随拼装；第二页的「下一章」不得吞入本章
	if len(ch.Paragraphs) != 2 || ch.Paragraphs[0] != "甲。" || ch.Paragraphs[1] != "乙。" {
		t.Fatalf("AutoNext 分页拼装不符: %+v", ch.Paragraphs)
	}
}
