package booksource

import (
	"context"
	"errors"
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
	cur := entry.URL
	// 单章分页上限：防站方环形链接把下载挂死（正常章节分页远小于 50）。
	for page := 0; ; page++ {
		if page > 50 {
			break
		}
		doc, _, err := e.pacedFetch(ctx, Request{URL: cur})
		if err != nil {
			return ChapterText{}, err
		}
		if title == "" {
			title = textOf(doc.Selection, cr.Title)
		}
		content := htmlOf(doc.Selection, cr.Content)
		if strings.TrimSpace(invisibleChars.ReplaceAllString(content, "")) == "" {
			return ChapterText{}, ErrEmptyContent
		}
		htmls = append(htmls, content)

		next, last := e.nextChapterPage(doc, cur)
		if last {
			break
		}
		cur = next
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
// 无下一页元素或抽不到链接 = 末页（单页章节走这里）。
func (e *Engine) nextChapterPage(doc *goquery.Document, cur string) (string, bool) {
	cr := e.rule.Chapter
	if cr == nil || strings.TrimSpace(cr.NextPage) == "" {
		return "", true
	}
	nextEls := doc.Find(cr.NextPage)
	if nextEls.Length() == 0 {
		return "", true
	}
	href, _ := nextEls.First().Attr("href")
	next := resolveLink(cur, href)
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
	// OnProgress 每完成一章回调（done/total）；可为 nil。
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
	slots := make([]*ChapterText, len(toc))
	var (
		mu      sync.Mutex
		doneCnt int
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
			mu.Lock()
			d := doneCnt
			mu.Unlock()
			opt.OnProgress(d, len(toc))
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
		if errs != nil && errs[i] != nil {
			f.Err = errs[i].Error()
		} else {
			f.Err = ctx.Err().Error()
		}
		report.Failed = append(report.Failed, f)
	}
	return report, nil
}
