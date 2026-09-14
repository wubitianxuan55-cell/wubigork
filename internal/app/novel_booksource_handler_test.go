package app

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/bookimport"
	"github.com/gaea/gaea/internal/booksource"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
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
	return bsWriteFixtureRules(t, t.TempDir())
}

// bsWriteFixtureRules 把夹具规则/引擎写进指定规则目录（工具测试要落在
// booksourceRulesDir() 的临时 HOME 下，不能自己另开 TempDir）。
func bsWriteFixtureRules(t *testing.T, dir string) string {
	t.Helper()
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

// ── 失败章补下（t3）──

func TestAppendProjectChapters_RenumbersAndAppendsOutline(t *testing.T) {
	dir := t.TempDir()
	base, err := createImportedProject(dir, "补下测试", "未分类", "默认", []importChapter{
		{Title: "第1章", Content: "一"}, {Title: "第2章", Content: "二"}, {Title: "第3章", Content: "三"},
	}, bookimport.Report{SplitStrategy: "booksource"})
	if err != nil {
		t.Fatalf("建基线项目: %v", err)
	}

	res, err := appendProjectChapters(base.Path, booksource.DownloadReport{
		Chapters: []booksource.ChapterText{
			{Title: "第4章", Order: 4, Paragraphs: []string{"四A", "四B"}},
			{Title: "第5章", Order: 5, Paragraphs: []string{"五"}},
		},
		Failed: []booksource.FailedChapter{{Title: "第6章", URL: "https://book.test/ch/6", Err: "boom"}},
	})
	if err != nil {
		t.Fatalf("补下: %v", err)
	}
	if res.Appended != 2 || res.TotalChapters != 5 || res.AddedWords != 6 || res.Title != "补下测试" {
		t.Fatalf("追加语义词不符: %+v", res)
	}
	if len(res.Failed) != 1 || res.Failed[0].Err != "boom" {
		t.Fatalf("失败清单应原样透出: %+v", res.Failed)
	}

	pm, err := project.Open(base.Path)
	if err != nil {
		t.Fatalf("重开项目: %v", err)
	}
	defer pm.Close()
	if _, err := pm.ReadChapter(4); err != nil {
		t.Fatalf("第4章应已落盘: %v", err)
	}
	if c, err := pm.ReadChapter(5); err != nil || c != "五" {
		t.Fatalf("第5章内容不符: %q %v", c, err)
	}
	of, err := pm.ReadOutlines()
	if err != nil || len(of.Nodes) != 5 {
		t.Fatalf("大纲应有 5 节点: %v %d", err, len(of.Nodes))
	}
	last := of.Nodes[4]
	if last.ID != "imp-005" || last.OrderIndex != 5 || last.Status != types.OutlineDone {
		t.Fatalf("追加节点应续编且 done: %+v", last)
	}
	for _, n := range of.Nodes[:3] { // 既有节点原样保留
		if n.ID != "imp-001" && n.ID != "imp-002" && n.ID != "imp-003" {
			t.Fatalf("既有节点被动过: %+v", n)
		}
	}

	// 全部失败：追加 0，大纲不动，失败清单如实透出
	res2, err := appendProjectChapters(base.Path, booksource.DownloadReport{
		Failed: []booksource.FailedChapter{{Title: "第6章", URL: "u6", Err: "boom"}},
	})
	if err != nil {
		t.Fatalf("全败补下不应报错: %v", err)
	}
	if res2.Appended != 0 || res2.TotalChapters != 5 || len(res2.Failed) != 1 {
		t.Fatalf("全败语义不符: %+v", res2)
	}
}

// ── 搜索引擎规则编辑（t3）──

func TestEnginesGetSave_RoundTripAndFailClosed(t *testing.T) {
	dir := t.TempDir()

	// Get：文件缺失=空清单+路径照返（不报错）
	got, err := enginesGet(dir)
	if err != nil || len(got.Rules) != 0 || got.Path == "" {
		t.Fatalf("缺失文件应空清单不报错: %+v %v", got, err)
	}

	// Save：合法两条（一条 disabled）→ 落盘 → Get 回读一致
	n, err := enginesSave(dir, `[{"name":"bing","url":"https://www.bing.com/search?q=%s","result":".b_algo","title":"h2 a"},
		{"name":"ddg","url":"https://html.duckduckgo.com/html/?q=%s","result":".result","title":".result__a","linkParam":"uddg","disabled":true}]`)
	if err != nil || n != 2 {
		t.Fatalf("保存: %v n=%d", err, n)
	}
	got, err = enginesGet(dir)
	if err != nil || len(got.Rules) != 2 || got.Rules[1].Name != "ddg" || !got.Rules[1].Disabled {
		t.Fatalf("回读不符: %+v %v", got, err)
	}

	// Get：装载剔除口径一致性——坏 JSON 显式报错不静默
	if err := os.WriteFile(filepath.Join(dir, enginesFileName), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := enginesGet(dir); err == nil {
		t.Fatal("坏文件应报错")
	}

	// Save fail-closed：空清单拒绝 / 缺 name 点名 / 重复名拒绝，且原文件不被破坏
	good, _ := os.ReadFile(filepath.Join(dir, enginesFileName))
	if _, err := enginesSave(dir, `[]`); err == nil {
		t.Fatal("空清单应拒绝")
	}
	if _, err := enginesSave(dir, `[{"name":"","url":"https://x.com/s?q=%s","result":".r","title":"a"}]`); err == nil {
		t.Fatal("缺 name 应拒绝")
	}
	if _, err := enginesSave(dir, `[{"name":"a","url":"https://x.com/s?q=%s","result":".r","title":"a"},{"name":"a","url":"https://y.com/s?q=%s","result":".r","title":"a"}]`); err == nil {
		t.Fatal("重复名应拒绝")
	}
	if _, err := enginesSave(dir, `[{"name":"a","url":"https://x.com/s?q=%s","result":".r","title":"a@js:x"}]`); err == nil {
		t.Fatal("@js: 应拒绝")
	}
	after, _ := os.ReadFile(filepath.Join(dir, enginesFileName))
	if string(after) != string(good) {
		t.Fatal("校验失败后原文件不得被破坏")
	}
}
