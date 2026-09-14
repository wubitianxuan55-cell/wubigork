package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/booksource"
)

// ── 原罪书源 t2（假站夹具复用 novel_booksource_handler_test.go 同包件）──

func TestSinDownloadBook_WritesAssembledTXT(t *testing.T) {
	dir := bsFixtureDir(t)
	books := t.TempDir()
	res, err := sinDownloadBook(context.Background(), dir, books,
		bookImportParams{Source: "测试源", URL: "https://book.test/book/1/", Title: "测试之书"},
		bsTestOptions(bsTocStub(2, 0)), nil)
	if err != nil {
		t.Fatalf("sinDownloadBook: %v", err)
	}
	if res.Chapters != 2 || res.Title != "测试之书" || res.Words <= 0 {
		t.Fatalf("成书结果异常: %+v", res)
	}
	raw, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatalf("读成书: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "书名：测试之书") || !strings.Contains(text, "第1章 第1章") || !strings.Contains(text, "第2章 第2章") {
		t.Fatalf("TXT 应含书名头与两章: %q", text)
	}
	if filepath.Dir(res.Path) != books {
		t.Fatalf("成书应落指定 books 目录: %s", res.Path)
	}
}

func TestSinDownloadBook_DefaultTitleFallback(t *testing.T) {
	dir := bsFixtureDir(t)
	res, err := sinDownloadBook(context.Background(), dir, t.TempDir(),
		bookImportParams{Source: "测试源", URL: "https://book.test/book/1/"},
		bsTestOptions(bsTocStub(1, 0)), nil)
	if err != nil {
		t.Fatalf("sinDownloadBook: %v", err)
	}
	// 夹具 book 规则只有 url 正则无 bookName 选择器 → 详情名空 → 兜底名。
	if res.Title != "书源成书" || filepath.Base(res.Path) != "书源成书.txt" {
		t.Fatalf("兜底书名应生效: %+v", res)
	}
}

func TestSinDownloadBook_FailedChapterReported(t *testing.T) {
	res, err := sinDownloadBook(context.Background(), bsFixtureDir(t), t.TempDir(),
		bookImportParams{Source: "测试源", URL: "https://book.test/book/1/", Title: "测试之书"},
		bsTestOptions(bsTocStub(3, 2)), nil)
	if err != nil {
		t.Fatalf("单章失败不应中断成书: %v", err)
	}
	if res.Chapters != 2 || len(res.Failed) != 1 {
		t.Fatalf("失败章应如实带出: %+v", res)
	}
}

func TestSinDownloadBook_RulelessRefused(t *testing.T) {
	_, err := sinDownloadBook(context.Background(), bsFixtureDir(t), t.TempDir(),
		bookImportParams{Source: "不存在的源", URL: "https://x.test/book/1/"},
		booksource.Options{}, nil)
	if err == nil || !strings.Contains(err.Error(), "不存在或未启用") {
		t.Fatalf("免规则来源下载应 fail-closed: %v", err)
	}
}

func TestSinDownloadBook_NameCollisionSuffix(t *testing.T) {
	dir := bsFixtureDir(t)
	books := t.TempDir()
	params := bookImportParams{Source: "测试源", URL: "https://book.test/book/1/", Title: "同名书"}
	first, err := sinDownloadBook(context.Background(), dir, books, params, bsTestOptions(bsTocStub(1, 0)), nil)
	if err != nil {
		t.Fatalf("首次成书: %v", err)
	}
	second, err := sinDownloadBook(context.Background(), dir, books, params, bsTestOptions(bsTocStub(1, 0)), nil)
	if err != nil {
		t.Fatalf("二次成书: %v", err)
	}
	if filepath.Base(first.Path) != "同名书.txt" || filepath.Base(second.Path) != "同名书 (2).txt" {
		t.Fatalf("同名应追加序号: %s / %s", first.Path, second.Path)
	}
}

func TestSinBooksList_EmptyMissingAndSortedNewestFirst(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "books")
	books, err := sinBooksList(missing)
	if err != nil || len(books) != 0 {
		t.Fatalf("目录不存在应空清单不报错: %v %v", books, err)
	}
	dir := t.TempDir()
	old := filepath.Join(dir, "旧书.txt")
	new := filepath.Join(dir, "新书.txt")
	for _, p := range []string{old, new, filepath.Join(dir, "忽略.epub")} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("写夹具: %v", err)
		}
	}
	base := time.Now().UTC()
	if err := os.Chtimes(old, base.Add(-time.Hour), base.Add(-time.Hour)); err != nil {
		t.Fatalf("设时间: %v", err)
	}
	books, err = sinBooksList(dir)
	if err != nil {
		t.Fatalf("sinBooksList: %v", err)
	}
	if len(books) != 2 || books[0].Title != "新书" || books[1].Title != "旧书" {
		t.Fatalf("应按修改时间新→旧且忽略非 txt: %+v", books)
	}
}

func TestSinBookDeleteAt_GuardsPath(t *testing.T) {
	dir := t.TempDir()
	inside := filepath.Join(dir, "书.txt")
	if err := os.WriteFile(inside, []byte("x"), 0o644); err != nil {
		t.Fatalf("写夹具: %v", err)
	}
	if err := sinBookDeleteAt(dir, filepath.Join(t.TempDir(), "外面.txt")); err == nil {
		t.Fatalf("目录外路径应拒绝")
	}
	if err := sinBookDeleteAt(dir, filepath.Join(dir, "非文本.epub")); err == nil {
		t.Fatalf("非 txt 应拒绝")
	}
	if err := sinBookDeleteAt(dir, inside); err != nil {
		t.Fatalf("目录内 txt 应可删: %v", err)
	}
	if _, err := os.Stat(inside); !os.IsNotExist(err) {
		t.Fatalf("删除后不应存在")
	}
}

func TestSinBookSourceDownload_SyncRulelessRefused(t *testing.T) {
	a := bsTestApp(t)
	if _, err := a.SinBookSourceDownload("不存在的源", "https://x.test/book/1/", 0, 0, "x"); err == nil {
		t.Fatalf("起跑前应同步拒绝无规则来源")
	}
}
