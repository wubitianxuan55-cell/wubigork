package app

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/booksource"
	"github.com/gaea/gaea/internal/config"
)

// ── 假站与夹具（引擎 Fetcher/Sleeper 注入，零真网络）──

// bsStubFetcher 按 URL 出静态页；前缀路由给搜索/泛搜索这类带查询串的地址；
// 未命中报错暴露夹具缺口（与 booksource 包验收测试同思路）。
type bsStubFetcher struct {
	pages  map[string]string
	prefix []bsPrefixRoute
}

type bsPrefixRoute struct{ prefix, body string }

func (f *bsStubFetcher) Fetch(_ context.Context, req booksource.Request) (*booksource.Page, error) {
	if body, ok := f.pages[req.URL]; ok {
		return &booksource.Page{URL: req.URL, Body: []byte(body)}, nil
	}
	for _, r := range f.prefix {
		if strings.HasPrefix(req.URL, r.prefix) {
			return &booksource.Page{URL: req.URL, Body: []byte(r.body)}, nil
		}
	}
	return nil, context.DeadlineExceeded // 未夹具的 URL：超时形态报错（重试穷尽后进 Failed）
}

func bsInstantSleeper(context.Context, time.Duration) error { return nil }

func bsTestOptions(fetch booksource.Fetcher) booksource.Options {
	return booksource.Options{Fetcher: fetch, Sleeper: bsInstantSleeper}
}

// bsFixtureDir 临时规则目录：一本源规则 + 一份泛搜索引擎文件。
func bsFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	rule := `{
		"name": "测试源",
		"baseUri": "https://book.test/",
		"search": {"url": "https://book.test/search?q=%s", "result": ".item", "bookName": ".t a", "link": ".t a@href", "author": ".a"},
		"book": {"url": "https://book\\.test/book/(\\d+)/"},
		"toc": {"item": "#list a"},
		"chapter": {"title": "h1", "content": "#content", "paragraphTagClosed": true},
		"crawl": {"concurrency": 2, "minIntervalMs": 0, "maxIntervalMs": 0, "maxRetries": 1, "retryMinMs": 0, "retryMaxMs": 0}
	}`
	engines := `[{"name": "测试引擎", "url": "https://eng.test/s?q=%s", "result": ".r", "title": "a.t"}]`
	if err := os.WriteFile(filepath.Join(dir, "测试源.json"), []byte(rule), 0o644); err != nil {
		t.Fatalf("写规则: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, enginesFileName), []byte(engines), 0o644); err != nil {
		t.Fatalf("写引擎: %v", err)
	}
	return dir
}

func bsSearchPage(items ...string) string {
	var b strings.Builder
	b.WriteString("<html><body>")
	for _, it := range items {
		b.WriteString("<div class=\"item\"><div class=\"t\"><a href=\"" + it + "\">测试之书</a></div><div class=\"a\">测试作者</div></div>")
	}
	b.WriteString("</body></html>")
	return b.String()
}

func bsTocPage(urls ...string) string {
	var b strings.Builder
	b.WriteString("<html><body><div id=\"list\">")
	for i, u := range urls {
		n := strconv.Itoa(i + 1)
		b.WriteString("<a href=\"" + u + "\">第" + n + "章 标题" + n + "</a>")
	}
	b.WriteString("</div></body></html>")
	return b.String()
}

func bsChapterPage(title, body string) string {
	return "<html><body><h1>" + title + "</h1><div id=\"content\"><p>" + body + "</p><p>第二段。</p></div></body></html>"
}

func bsSearchStub() *bsStubFetcher {
	return &bsStubFetcher{
		prefix: []bsPrefixRoute{{prefix: "https://book.test/search", body: bsSearchPage("https://book.test/book/1/")}},
		pages: map[string]string{
			"https://eng.test/s?q=%E6%B5%8B%E8%AF%95": "<html><body>" +
				"<div class=\"r\"><a class=\"t\" href=\"https://book.test/book/1/\">测试之书（泛搜索同名）</a></div>" +
				"<div class=\"r\"><a class=\"t\" href=\"https://other.test/novel/9/\">测试之书（别站）</a></div>" +
				"</body></html>",
		},
	}
}

// ── 规则装载 ──

func TestLoadBookSourceRules_SkipsEnginesBrokenAndDisabled(t *testing.T) {
	dir := bsFixtureDir(t)
	broken := `{"name": "坏源", "search": }`
	if err := os.WriteFile(filepath.Join(dir, "坏.json"), []byte(broken), 0o644); err != nil {
		t.Fatalf("写坏规则: %v", err)
	}
	disabled := `{"name": "停用源", "disabled": true, "search": {"url": "https://x.test/s?q=%s", "result": ".r", "bookName": ".t"}}`
	if err := os.WriteFile(filepath.Join(dir, "停用.json"), []byte(disabled), 0o644); err != nil {
		t.Fatalf("写停用规则: %v", err)
	}
	rules := loadBookSourceRules(dir)
	if len(rules) != 1 || rules[0].Name != "测试源" {
		t.Fatalf("应只装载启用中的「测试源」: %d 条", len(rules))
	}
}

// ── 搜索：合并去重 + HasRule 打标 + 告警透出 ──

func TestSearchBookSources_MergesRuleAndWeb_WithDedup(t *testing.T) {
	dir := bsFixtureDir(t)
	res, err := searchBookSources(context.Background(), dir, "测试", bsTestOptions(bsSearchStub()))
	if err != nil {
		t.Fatalf("searchBookSources: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("去重后应剩 2 条（书源命中与泛搜索同 URL 合一 + 别站 1 条）: %+v", res.Candidates)
	}
	first := res.Candidates[0]
	if first.Kind != "rule" || !first.HasRule || first.Host != "book.test" || first.Author != "测试作者" {
		t.Fatalf("书源命中应排前且 HasRule: %+v", first)
	}
	for _, c := range res.Candidates {
		if c.URL == "https://other.test/novel/9/" && c.HasRule {
			t.Fatalf("别站候选不应带 HasRule: %+v", c)
		}
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("不应有告警: %v", res.Warnings)
	}
}

func TestSearchBookSources_EmptyKeywordRefused(t *testing.T) {
	if _, err := searchBookSources(context.Background(), bsFixtureDir(t), "  ", booksource.Options{}); err == nil {
		t.Fatalf("空关键字应拒绝")
	}
}

// ── 目录预览：总数 + 截断样例 ──

func bsTocStub(n int, missing int) *bsStubFetcher {
	pages := map[string]string{"https://book.test/book/1/": bsTocPageFunc(n)}
	for i := 1; i <= n; i++ {
		if i == missing {
			continue // 模拟单章失败
		}
		pages["https://book.test/c/"+strconv.Itoa(i)+"/"] = bsChapterPage("第"+strconv.Itoa(i)+"章", "第"+strconv.Itoa(i)+"章正文内容。")
	}
	return &bsStubFetcher{pages: pages}
}

func bsTocPageFunc(n int) string {
	urls := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		urls = append(urls, "https://book.test/c/"+strconv.Itoa(i)+"/")
	}
	return bsTocPage(urls...)
}

func TestTocBookSource_TotalAndTruncatedSample(t *testing.T) {
	dir := bsFixtureDir(t)
	// 5 章：不截断
	pv, err := tocBookSource(context.Background(), dir, "测试源", "https://book.test/book/1/", bsTestOptions(bsTocStub(5, 0)))
	if err != nil {
		t.Fatalf("tocBookSource: %v", err)
	}
	if pv.Total != 5 || len(pv.Sample) != 5 || pv.Truncated {
		t.Fatalf("5 章应全样例: %+v", pv)
	}
	// 20 章：首 8 + 末 4 截断
	pv, err = tocBookSource(context.Background(), dir, "测试源", "https://book.test/book/1/", bsTestOptions(bsTocStub(20, 0)))
	if err != nil {
		t.Fatalf("tocBookSource: %v", err)
	}
	if pv.Total != 20 || len(pv.Sample) != 12 || !pv.Truncated {
		t.Fatalf("20 章应首8+末4: %+v", pv)
	}
	if pv.Sample[0].Title != "第1章 标题1" || pv.Sample[11].Title != "第20章 标题20" {
		t.Fatalf("样例应首末对齐: %+v", pv.Sample)
	}
}

func TestTocBookSource_RulelessRefused(t *testing.T) {
	_, err := tocBookSource(context.Background(), bsFixtureDir(t), "不存在的源", "https://x.test/book/1/", booksource.Options{})
	if err == nil || !strings.Contains(err.Error(), "不存在或未启用") {
		t.Fatalf("免规则来源应诚实拒绝: %v", err)
	}
}

// ── 在线导入：落库 + 范围重编号 + Failed 清单 + 无规则拒绝 ──

func TestImportBookSource_HappyPathLandsProject(t *testing.T) {
	dir := bsFixtureDir(t)
	novels := t.TempDir()
	res, failed, err := importBookSource(context.Background(), dir, novels,
		bookImportParams{Source: "测试源", URL: "https://book.test/book/1/", Title: "测试之书"},
		bsTestOptions(bsTocStub(3, 0)), nil)
	if err != nil {
		t.Fatalf("importBookSource: %v", err)
	}
	if len(failed) != 0 {
		t.Fatalf("不应有失败章: %+v", failed)
	}
	if res.ChapterCount != 3 || res.SplitStrategy != "booksource" || res.Title != "测试之书" {
		t.Fatalf("导入结果异常: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(res.Path, "chapters", "003.md")); err != nil {
		t.Fatalf("章节应落盘: %v", err)
	}
}

func TestImportBookSource_FailedChapterReported(t *testing.T) {
	dir := bsFixtureDir(t)
	res, failed, err := importBookSource(context.Background(), dir, t.TempDir(),
		bookImportParams{Source: "测试源", URL: "https://book.test/book/1/", Title: "测试之书"},
		bsTestOptions(bsTocStub(3, 2)), nil)
	if err != nil {
		t.Fatalf("单章失败不应中断整本: %v", err)
	}
	if res.ChapterCount != 2 || len(failed) != 1 || failed[0].URL != "https://book.test/c/2/" {
		t.Fatalf("失败章应如实上报: res=%+v failed=%+v", res, failed)
	}
}

func TestImportBookSource_RangeRenumberedFromOne(t *testing.T) {
	dir := bsFixtureDir(t)
	res, _, err := importBookSource(context.Background(), dir, t.TempDir(),
		bookImportParams{Source: "测试源", URL: "https://book.test/book/1/", Title: "测试之书", Start: 2, End: 3},
		bsTestOptions(bsTocStub(3, 0)), nil)
	if err != nil {
		t.Fatalf("importBookSource: %v", err)
	}
	if res.ChapterCount != 2 {
		t.Fatalf("范围 [2,3] 应得 2 章: %+v", res)
	}
	for _, num := range []string{"001.md", "002.md"} {
		if _, err := os.Stat(filepath.Join(res.Path, "chapters", num)); err != nil {
			t.Fatalf("项目章号应从 1 重编号（缺 %s）: %v", num, err)
		}
	}
}

func TestImportBookSource_AllChaptersFailed(t *testing.T) {
	dir := bsFixtureDir(t)
	_, _, err := importBookSource(context.Background(), dir, t.TempDir(),
		bookImportParams{Source: "测试源", URL: "https://book.test/book/1/", Title: "测试之书"},
		bsTestOptions(&bsStubFetcher{pages: map[string]string{"https://book.test/book/1/": bsTocPageFunc(2)}}), nil)
	if err == nil || !strings.Contains(err.Error(), "全部未取到") {
		t.Fatalf("全败应报「全部未取到」: %v", err)
	}
}

func TestImportBookSource_RulelessRefused(t *testing.T) {
	_, _, err := importBookSource(context.Background(), bsFixtureDir(t), t.TempDir(),
		bookImportParams{Source: "不存在的源", URL: "https://x.test/book/1/", Title: "x"},
		booksource.Options{}, nil)
	if err == nil || !strings.Contains(err.Error(), "不存在或未启用") {
		t.Fatalf("免规则来源导入应 fail-closed: %v", err)
	}
}

// ── 绑定层：取消登记簿 + 起跑前规则预检 ──

func bsTestApp(t *testing.T) *App {
	t.Helper()
	c := &core{cfg: &config.Config{NovelsDir: t.TempDir()}}
	return &App{core: c, writingState: &writingState{core: c}}
}

func TestNovelBookSourceImportCancel_UnknownFalseRegisteredTrue(t *testing.T) {
	a := bsTestApp(t)
	if a.NovelBookSourceImportCancel("bsi_none") {
		t.Fatalf("未知 job 应返回 false")
	}
	bookImportMu.Lock()
	bookImportRuns["bsi_test"] = func() {}
	bookImportMu.Unlock()
	defer func() {
		bookImportMu.Lock()
		delete(bookImportRuns, "bsi_test")
		bookImportMu.Unlock()
	}()
	if !a.NovelBookSourceImportCancel("bsi_test") {
		t.Fatalf("已登记 job 应可取消")
	}
	bookImportMu.Lock()
	_, ok := bookImportRuns["bsi_test"]
	bookImportMu.Unlock()
	if ok {
		t.Fatalf("取消后应移出登记簿")
	}
}

func TestNovelBookSourceImport_RulelessSyncRefused(t *testing.T) {
	a := bsTestApp(t)
	if _, err := a.NovelBookSourceImport("不存在的源", "https://x.test/book/1/", 0, 0, "x", "", ""); err == nil {
		t.Fatalf("起跑前应同步拒绝无规则来源")
	}
}

// ── 文件导入重构回归：ImportNovelBook 行为零变化（规格 t1 验收）──

func TestImportNovelBook_RefactorRegression(t *testing.T) {
	a := bsTestApp(t)
	src := filepath.Join(t.TempDir(), "book.txt")
	content := "第一章 初遇\n\n雨夜，她推开茶馆的门。\n\n第二章 风波\n\n城外的消息传开了。"
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatalf("写源文件: %v", err)
	}
	res, err := a.ImportNovelBook(src, "重构回归书", "", "")
	if err != nil {
		t.Fatalf("ImportNovelBook: %v", err)
	}
	if res.ChapterCount != 2 || res.SplitStrategy != "strong" || res.Encoding != "utf-8" {
		t.Fatalf("重构后文件导入行为应零变化: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(res.Path, "chapters", "002.md")); err != nil {
		t.Fatalf("章节应落盘: %v", err)
	}
}
