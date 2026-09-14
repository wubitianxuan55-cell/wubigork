package app

// sin_tool_book_test.go — 原罪书源工具（book_search / book_download，t4）。
// sinTestHome 把 sinRoot/规则目录重定向到临时 HOME（sin_tool_impl_test.go 同款）；
// 网络通道经 sinBookToolOpts 注入假站（夹具复用 novel_booksource_handler_test.go）。

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/booksource"
)

func withStubBookToolOpts(t *testing.T, fetch *bsStubFetcher) {
	t.Helper()
	sinBookToolOpts = bsTestOptions(fetch)
	t.Cleanup(func() { sinBookToolOpts = booksource.Options{} })
}

func sinToolExecute(t *testing.T, name string, args string) (string, error) {
	t.Helper()
	tool := sinToolByName((&App{}).sinToolSet("sin_1_1"), name)
	if tool == nil {
		t.Fatalf("工具 %s 未注册", name)
	}
	return tool.Execute(context.Background(), json.RawMessage(args))
}

func TestSinBookSearchTool_ReturnsCompactCandidates(t *testing.T) {
	sinTestHome(t)
	withStubBookToolOpts(t, bsSearchStub())
	if err := os.MkdirAll(booksourceRulesDir(), 0o755); err != nil {
		t.Fatalf("建规则目录: %v", err)
	}
	bsWriteFixtureRules(t, booksourceRulesDir())

	out, err := sinToolExecute(t, sinToolBookSearch, `{"keyword":"测试"}`)
	if err != nil {
		t.Fatalf("book_search: %v", err)
	}
	if !strings.Contains(out, "《测试之书》") || !strings.Contains(out, "可下载") {
		t.Fatalf("应含候选与可下载标: %q", out)
	}
	if !strings.Contains(out, "https://book.test/book/1/") {
		t.Fatalf("应含候选链接: %q", out)
	}
}

func TestSinBookSearchTool_EmptyHonestMessage(t *testing.T) {
	sinTestHome(t)
	withStubBookToolOpts(t, &bsStubFetcher{})
	out, err := sinToolExecute(t, sinToolBookSearch, `{"keyword":"测试"}`)
	if err != nil {
		t.Fatalf("book_search: %v", err)
	}
	if !strings.Contains(out, "没有搜到候选") || !strings.Contains(out, "rule-template.json") {
		t.Fatalf("空态应如实说明规则目录与模板: %q", out)
	}
}

func TestSinBookDownloadTool_WritesBookAndReturnsExcerpt(t *testing.T) {
	sinTestHome(t)
	withStubBookToolOpts(t, bsTocStub(2, 0))
	if err := os.MkdirAll(booksourceRulesDir(), 0o755); err != nil {
		t.Fatalf("建规则目录: %v", err)
	}
	bsWriteFixtureRules(t, booksourceRulesDir())

	out, err := sinToolExecute(t, sinToolBookDownload,
		`{"source":"测试源","url":"https://book.test/book/1/","title":"测试之书"}`)
	if err != nil {
		t.Fatalf("book_download: %v", err)
	}
	if !strings.Contains(out, "已下载《测试之书》") || !strings.Contains(out, "2 章") || !strings.Contains(out, "【正文节选") {
		t.Fatalf("应含摘要与节选: %q", out)
	}
	books, err := sinBooksList(sinBooksDir())
	if err != nil || len(books) != 1 || books[0].Title != "测试之书" {
		t.Fatalf("成书应落 sin 数据面: %v %+v", err, books)
	}
}

func TestSinBookDownloadTool_RulelessRefused(t *testing.T) {
	sinTestHome(t)
	withStubBookToolOpts(t, &bsStubFetcher{})
	_, err := sinToolExecute(t, sinToolBookDownload, `{"source":"不存在的源","url":"https://x.test/book/1/"}`)
	if err == nil || !strings.Contains(err.Error(), "可下载") {
		t.Fatalf("免规则来源应拒绝并引导改选: %v", err)
	}
}
