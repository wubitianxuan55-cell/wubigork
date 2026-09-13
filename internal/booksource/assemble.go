package booksource

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/PuerkitoBio/goquery"
)

// ── 组装导出（规格 §2.10：TXT 首页信息头 + 章序；GBK 兼容选项）─────────────

// BookInfo 书籍信息（详情页解析产物 + TXT 信息头）。
type BookInfo struct {
	Name           string `json:"name"`
	Author         string `json:"author"`
	Intro          string `json:"intro,omitempty"`
	Category       string `json:"category,omitempty"`
	CoverURL       string `json:"coverUrl,omitempty"`
	LatestChapter  string `json:"latestChapter,omitempty"`
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`
	Status         string `json:"status,omitempty"`
}

// BookDetail 解析书籍详情页。
func (e *Engine) BookDetail(ctx context.Context, detailURL string) (BookInfo, error) {
	doc, pageURL, err := e.pacedFetch(ctx, Request{URL: detailURL})
	if err != nil {
		return BookInfo{}, err
	}
	return e.parseBookDetail(doc, pageURL), nil
}

// parseBookDetail 从已抓取的文档解析详情（meta 里有就不用选择器，规格 §2.1）。
func (e *Engine) parseBookDetail(doc *goquery.Document, pageURL string) BookInfo {
	b := e.rule.Book
	info := BookInfo{}
	if b == nil {
		return info
	}
	meta := func(name string) string {
		v, _ := doc.Find(`meta[property="` + name + `"], meta[name="` + name + `"]`).First().Attr("content")
		return strings.TrimSpace(v)
	}
	fetch := func(sel string, attr string) string {
		if sel == "" {
			return ""
		}
		s, a := splitAttrSuffix(sel)
		if attr != "" && a == "" {
			a = attr
		}
		if a != "" {
			return attrOf(doc.Selection, s, a, e.base(pageURL))
		}
		return textOf(doc.Selection, s)
	}
	info.Name = firstNonBlank(fetch(b.BookName, ""), meta("og:novel:book_name"), meta("og:title"))
	info.Author = firstNonBlank(fetch(b.Author, ""), meta("og:novel:author"), meta("author"))
	info.Intro = firstNonBlank(fetch(b.Intro, ""), meta("og:description"), meta("description"))
	info.Category = firstNonBlank(fetch(b.Category, ""), meta("og:novel:category"))
	info.CoverURL = firstNonBlank(fetch(b.CoverURL, "src"), meta("og:image"))
	info.LatestChapter = firstNonBlank(fetch(b.LatestChapter, ""), meta("og:novel:latest_chapter_name"))
	info.LastUpdateTime = firstNonBlank(fetch(b.LastUpdateTime, ""), meta("og:novel:update_time"))
	info.Status = firstNonBlank(fetch(b.Status, ""), meta("og:novel:status"))
	return info
}

func firstNonBlank(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// AssembleTXT 组装整本 TXT。格式：
//
//	书名：… / 作者：… / 简介：…（空行）
//	第N章 标题（空行）段落…（章间空行）
//
// encoding 支持 "utf-8"（默认）与 "gbk"（旧设备兼容，规格 §2.10）。
func AssembleTXT(info BookInfo, chapters []ChapterText, encoding string) ([]byte, error) {
	if len(chapters) == 0 {
		return nil, errors.New("没有可组装的章节")
	}
	var b strings.Builder
	b.WriteString("书名：" + info.Name + "\n")
	b.WriteString("作者：" + info.Author + "\n")
	intro := info.Intro
	if intro == "" {
		intro = "暂无"
	}
	b.WriteString("简介：" + intro + "\n\n")
	for i, ch := range chapters {
		num := ch.Order
		if num <= 0 {
			num = i + 1
		}
		b.WriteString(fmt.Sprintf("第%d章 %s\n\n", num, ch.Title))
		b.WriteString(strings.Join(ch.Paragraphs, "\n"))
		b.WriteString("\n\n")
	}
	out := []byte(strings.TrimRight(b.String(), "\n") + "\n")
	switch strings.ToLower(encoding) {
	case "", "utf-8", "utf8":
		return out, nil
	case "gbk":
		return simplifiedchinese.GBK.NewEncoder().Bytes(out)
	default:
		return nil, fmt.Errorf("不支持的导出编码「%s」（utf-8 / gbk）", encoding)
	}
}
