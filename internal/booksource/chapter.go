package booksource

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// ── 正文面（规格 §2.7：分页三件套、末页判定、正文空=限流信号）──────────────

var ErrChapterUnsupported = errors.New("书源缺少 chapter 段")

var (
	// 通用末页启发式的按钮文本（规格 §2.7）。
	nextIsChapterEnd = regexp.MustCompile(`下一章|没有了|>>|书末页`)
	// 通用末页启发式的分页 URL 形态：以 -1.html / _12.html 结尾视为「仍是本章分页」。
	pageSuffix = regexp.MustCompile(`[-_]\d+\.html$`)
)

// maxChapterPages 单章分页上限：正常章节分页远小于此；触顶即判环形/超长分页，
// **显式报错**而非静默截断正文（规格 §2.7 的诚实报错原则）。
const maxChapterPages = 50

// ChapterText 单章清洗后的文本。
type ChapterText struct {
	Title      string   `json:"title"`
	Order      int      `json:"order"`
	Paragraphs []string `json:"paragraphs"`
}

// FailedChapter 下载失败的章节（重试穷尽后如实上报，不静默吞）。
type FailedChapter struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Err   string `json:"error"`
}

// Chapter 抓取并清洗单章：分页循环拼接正文 → 清洗链 → 段落归一。
func (e *Engine) Chapter(ctx context.Context, entry TocEntry) (ChapterText, error) {
	cr := e.rule.Chapter
	if cr == nil {
		return ChapterText{}, ErrChapterUnsupported
	}
	var htmls []string
	title := ""
	var firstDoc *goquery.Document
	cur := entry.URL
	// 单章分页上限：防站方环形链接把下载挂死（正常章节分页远小于 50）。
	for page := 0; ; page++ {
		if page >= maxChapterPages {
			return ChapterText{}, fmt.Errorf("本章分页超过 %d 页上限（疑似环形分页链接）", maxChapterPages)
		}
		doc, pageURL, err := e.pacedFetch(ctx, Request{URL: cur})
		if err != nil {
			return ChapterText{}, err
		}
		if firstDoc == nil {
			firstDoc = doc
		}
		if title == "" && strings.TrimSpace(cr.Title) != "" {
			title = textOf(doc.Selection, cr.Title)
		}
		content := htmlOf(doc.Selection, cr.Content)
		if strings.TrimSpace(invisibleChars.ReplaceAllString(content, "")) == "" {
			return ChapterText{}, ErrEmptyContent
		}
		htmls = append(htmls, content)

		next, last := e.nextChapterPage(doc, pageURL)
		if last {
			break
		}
		cur = next
	}
	// 免规则标题回退：<title> 正则 → h1 → <title>（规格书源搜索 §2.4）
	if title == "" && firstDoc != nil {
		title = GuessChapterTitle(firstDoc)
	}
	joined := cleanChapterHTML(strings.Join(htmls, "\n"), cr.FilterTxt, cr.FilterTag, title)
	paras := paragraphsFromHTML(joined, cr.ParagraphTagClosed, cr.ParagraphTag)
	if title == "" {
		title = entry.Title
	}
	return ChapterText{Title: normalizeTitle(title), Order: entry.Order, Paragraphs: paras}, nil
}

// nextChapterPage 解析下一分页：返回 (下一页 URL, 是否末页)。
// 末页判定：规则 endPattern 正则命中即末页；缺省走通用启发式——URL 非
// 「-_N.html」分页形态 且 按钮文本是「下一章/没有了/>>/书末页」（规格 §2.7）。
// nextPage 为空且 autoNext 开启时走免规则翻页（只认含「页」锚点，防吞下一章）；
// 无下一页元素或抽不到链接 = 末页（单页章节走这里）。
func (e *Engine) nextChapterPage(doc *goquery.Document, cur string) (string, bool) {
	cr := e.rule.Chapter
	if cr == nil || strings.TrimSpace(cr.NextPage) == "" {
		if cr != nil && cr.AutoNext {
			if next, ok := GuessNextPage(doc, e.base(cur)); ok {
				return next, false
			}
		}
		return "", true
	}
	nextEls := doc.Find(cr.NextPage)
	if nextEls.Length() == 0 {
		return "", true
	}
	href, _ := nextEls.First().Attr("href")
	next := resolveLink(e.base(cur), href)
	if next == "" || next == cur {
		return "", true
	}
	if cr.EndPattern != "" {
		if re, err := regexp.Compile(cr.EndPattern); err == nil && re.MatchString(next) {
			return "", true
		}
		return next, false
	}
	if !pageSuffix.MatchString(next) && nextIsChapterEnd.MatchString(nextEls.First().Text()) {
		return "", true
	}
	return next, false
}

// DownloadOptions 下载编排选项。
type DownloadOptions struct {
	Start, End int // 1 起闭区间；0 = 全本（透传 Toc）
	// OnProgress 每完成一章回调（done/total，done=成功数）；可为 nil。
	// 串行发射（v4.338 后非版本刀根修）：done 非降、末次=成功总数，
	// 并发 worker 不会同时进入回调——消费方无需自带同步，回调应保持轻量。
	OnProgress func(done, total int)
}

// DownloadReport 下载结果：成功章节按序 + 失败清单。
type DownloadReport struct {
	Chapters []ChapterText   `json:"chapters"`
	Failed   []FailedChapter `json:"failed,omitempty"`
}

// Download 抓取整本：目录 → 有界并发抓章 → 按章序归并（失败不占位）。
// 单章失败语义对齐上游：重试穷尽后跳过并记录，不中断整本。
func (e *Engine) Download(ctx context.Context, detailURL string, opt DownloadOptions) (DownloadReport, error) {
	toc, err := e.Toc(ctx, detailURL, opt.Start, opt.End)
	if err != nil {
		return DownloadReport{}, err
	}
	return e.fetchChapters(ctx, toc, opt)
}

// DownloadChapters 按显式清单抓章（失败章补下的引擎缝；t3）。清单由调用方
// 给定（导入 done 事件 Failed 原样回传），与 Download 共用有界并发/重试/
// 进度纪律；Start/End 在此路径无意义，仅 OnProgress 生效。
func (e *Engine) DownloadChapters(ctx context.Context, items []TocEntry, opt DownloadOptions) (DownloadReport, error) {
	if len(items) == 0 {
		return DownloadReport{}, errors.New("章节清单为空")
	}
	return e.fetchChapters(ctx, items, opt)
}

// fetchChapters 有界并发抓章并按给定顺序归并（Download/DownloadChapters 共用）。
func (e *Engine) fetchChapters(ctx context.Context, toc []TocEntry, opt DownloadOptions) (DownloadReport, error) {
	slots := make([]*ChapterText, len(toc))
	var (
		mu      sync.Mutex
		doneCnt int
		emitMu  sync.Mutex // OnProgress 发射互斥：锁外直接回调曾实测乱序 [0/3 2/3 1/3]
	)
	errs := e.runBounded(ctx, len(toc), e.cfg.Concurrency, func(i int) error {
		ch, err := e.Chapter(ctx, toc[i])
		if err == nil {
			mu.Lock()
			doneCnt++
			mu.Unlock()
			slots[i] = &ch
		}
		if opt.OnProgress != nil {
			// 发射互斥下现读计数：回调序列非降、末次=成功总数；且并发
			// worker 不同时进入回调，消费方免同步。
			emitMu.Lock()
			mu.Lock()
			d := doneCnt
			mu.Unlock()
			opt.OnProgress(d, len(toc))
			emitMu.Unlock()
		}
		return err
	})
	report := DownloadReport{Chapters: make([]ChapterText, 0, len(toc))}
	for i, s := range slots {
		if s != nil {
			report.Chapters = append(report.Chapters, *s)
			continue
		}
		f := FailedChapter{Title: toc[i].Title, URL: toc[i].URL}
		switch {
		case errs != nil && errs[i] != nil:
			f.Err = errs[i].Error()
		case ctx.Err() != nil:
			f.Err = ctx.Err().Error()
		default:
			f.Err = "未取到结果（原因未上报）"
		}
		report.Failed = append(report.Failed, f)
	}
	return report, nil
}
