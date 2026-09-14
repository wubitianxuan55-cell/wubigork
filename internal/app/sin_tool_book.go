package app

// ── 原罪工具：书源取书（book_search / book_download，sin 书源线 t4）──
//
// 故事进行中拉素材：搜书返回紧凑候选清单（书源聚合 + 泛搜索同表，HasRule 口径
// 与小说侧一致）；下载成书落 sin 数据面（sinRoot()/books，硬隔离不变）并回带
// 开头若干章的**正文节选**——模型没有文件工具，节选才是它能直接化进描写的部分，
// 整本在成书清单里供用户自取。免规则候选正文不可解析（引擎 ErrChapterUnsupported），
// 下载工具对无规则来源 fail-closed 如实拒绝。
//
// 网络通道经 sinBookToolOpts 注入（生产零值=真网络；测试注入假站）。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/booksource"
)

const (
	sinToolBookSearch   = "book_search"
	sinToolBookDownload = "book_download"
)

// sinBookToolOpts 书源工具网络通道注入点（测试假站；生产保持零值）。
var sinBookToolOpts = booksource.Options{}

func init() {
	registerSinTool(sinToolBookSearch, func(sinToolContext) sinTool { return sinBookSearchTool{} })
	registerSinTool(sinToolBookDownload, func(sinToolContext) sinTool { return sinBookDownloadTool{} })
}

// ── book_search ──

type sinBookSearchTool struct{}

func (sinBookSearchTool) Name() string { return sinToolBookSearch }

func (sinBookSearchTool) Description() string {
	return "按书名/作者搜可取的小说素材（书源站点聚合 + 网页泛搜索同表）。返回候选清单，「可下载」的才能用 book_download 取正文。只用来拉故事素材（桥段/文笔/情节参考），查到的化进描写，不要在正文里罗列来源。"
}

func (sinBookSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"keyword":{"type":"string","description":"书名或作者关键字"}},"required":["keyword"]}`)
}

func (sinBookSearchTool) ReadOnly() bool { return true }

func (sinBookSearchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var a struct {
		Keyword string `json:"keyword"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	dir := booksourceRulesDir()
	if err := booksource.EnsureTemplate(dir); err != nil {
		return "", fmt.Errorf("书源模板初始化失败: %w", err)
	}
	cctx, cancel := context.WithTimeout(ctx, booksourceSearchTimeout)
	defer cancel()
	res, err := searchBookSources(cctx, dir, a.Keyword, sinBookToolOpts)
	if err != nil {
		return "", err
	}
	if len(res.Candidates) == 0 {
		msg := "没有搜到候选（书源规则目录为空或无命中）。规则目录：" + dir +
			"，可照目录内 rule-template.json 自编目标站点规则后再搜。"
		if len(res.Warnings) > 0 {
			msg += "\n另外有 " + fmt.Sprint(len(res.Warnings)) + " 条来源失败被略过。"
		}
		return msg, nil
	}
	var b strings.Builder
	// 可下载候选排前（模型下一步只关心这些），总量收敛在 12 行内防结果膨胀。
	shown := 0
	for _, c := range res.Candidates {
		if shown >= 12 {
			break
		}
		tag := "仅参考"
		if c.HasRule {
			tag = "可下载"
		}
		author := c.Author
		if author != "" {
			author = " 作者：" + author
		}
		fmt.Fprintf(&b, "《%s》%s · 来源 %s（%s）\n%s\n", c.Title, author, c.Source, tag, c.URL)
		shown++
	}
	if rest := len(res.Candidates) - shown; rest > 0 {
		fmt.Fprintf(&b, "（其余 %d 条略）\n", rest)
	}
	if len(res.Warnings) > 0 {
		fmt.Fprintf(&b, "（%d 条来源失败被略过）\n", len(res.Warnings))
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// ── book_download ──

type sinBookDownloadTool struct{}

func (sinBookDownloadTool) Name() string { return sinToolBookDownload }

func (sinBookDownloadTool) Description() string {
	return "从书源下载整本（或指定章节范围）的小说素材：正文落盘进「书源·成书清单」（用户可自取），" +
		"工具回带开头若干章的节选供当下化用。只接受 book_search 结果里标「可下载」的候选；" +
		"长书建议给 start/end 范围拉需要的章节，别整本等。素材化进描写，不要整段照搬。"
}

func (sinBookDownloadTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"source":{"type":"string","description":"书源规则名（book_search 结果括号里的来源名）"},` +
		`"url":{"type":"string","description":"书目详情页链接"},` +
		`"title":{"type":"string","description":"成书标题，缺省取详情页书名"},` +
		`"start":{"type":"integer","description":"起始章，默认 1"},` +
		`"end":{"type":"integer","description":"结束章，默认末章"}},"required":["source","url"]}`)
}

func (sinBookDownloadTool) ReadOnly() bool { return false } // 落盘成书文件

func (sinBookDownloadTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var a struct {
		Source string `json:"source"`
		URL    string `json:"url"`
		Title  string `json:"title"`
		Start  int    `json:"start"`
		End    int    `json:"end"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	// fail-closed 先查规则（与绑定层同口径，模型误传免规则候选时当场教它改选）
	if findRuleByName(loadBookSourceRules(booksourceRulesDir()), a.Source) == nil {
		return "", fmt.Errorf("书源规则不存在或未启用：%s（免规则来源正文不可解析；请改选 book_search 结果里标「可下载」的候选）", a.Source)
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	res, err := sinDownloadBook(cctx, booksourceRulesDir(), sinBooksDir(), bookImportParams{
		Source: a.Source, URL: a.URL, Title: a.Title, Start: a.Start, End: a.End,
	}, sinBookToolOpts, nil)
	if err != nil {
		return "", err
	}
	summary := fmt.Sprintf("已下载《%s》：%d 章 / %d 字，已存进成书清单（%s）。", res.Title, res.Chapters, res.Words, res.Path)
	if len(res.Failed) > 0 {
		summary += fmt.Sprintf("另有 %d 章失败未入库。", len(res.Failed))
	}
	return summary + "\n\n【正文节选（供化用）】\n" + sinBookExcerpt(res), nil
}

// sinBookExcerpt 成书开头节选（自截 ~5000 rune，工具循环外层还有 6000 总闸）。
func sinBookExcerpt(res SinBookSourceDownloadResult) string {
	raw, err := os.ReadFile(res.Path)
	if err != nil {
		return "（节选读取失败，但整本已在成书清单）"
	}
	const limit = 5000
	r := []rune(string(raw))
	if len(r) > limit {
		return string(r[:limit]) + "\n……（节选到此，整本在成书清单）"
	}
	return string(r)
}
