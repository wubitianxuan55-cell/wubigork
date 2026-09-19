package app

// ── 原罪导出：EPUB 电子书（图文成品出口）──
//
// 与 SinExportMarkdown 同一数据源（统一聊天存储 + extra.illustrations），两种
// 成品格式各司其职：Markdown 是图文源文件（插图以本地路径引用），EPUB 是
// 阅读成品（插图内嵌进书，可传阅读器离线看）。
//
// 结构：每条助手回合一小节（中文序数标题，目录读起来不像日志），用户指令以
// 引块置节首（导出物含用户的每一轮，与 Markdown 导出同口径）；插图标记就地
// 内嵌，未生成/无法内嵌的保留画面描述占位（不吞）。第一张成功内嵌的插图兼作
// 封面。
//
// 落点与 sin_export 工具同一目录（sin/exports），同名不覆盖（-2/-3）；
// 写出失败即删半截产物（与书源线 EPUB 导出同一纪律）。

import (
	"fmt"
	"html"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bmaupin/go-epub"
)

// sinEpubImgExts EPUB 可内嵌的图片扩展名（go-epub 支持集；webp 等不在其列，
// 遇到即保留占位，不硬塞）。
var sinEpubImgExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true,
}

// SinExportEpub 把故事导出为 EPUB 电子书（插图内嵌），落原罪导出目录并返回
// 文件路径；重复导出生成 -2/-3 新文件（用户的历史导出不被顶掉）。
func (a *App) SinExportEpub(topicID string) (string, error) {
	if err := a.sinTopicGuard(topicID); err != nil {
		return "", err
	}
	topic, err := a.chatStore.GetTopic(topicID)
	if err != nil {
		return "", err
	}
	msgs, err := a.chatStore.ListMessages(topicID)
	if err != nil {
		return "", err
	}

	book := epub.NewEpub(topic.Title)
	book.SetAuthor("原罪 · 图文故事")

	imgSeq := 0
	coverSet := false
	// embedIllu 内嵌一张已生成插图，返回可直接用于 <img src> 的书内路径；
	// 文件缺失/扩展名不支持/内嵌失败 = 空串（调用方落占位，导出不整单失败）。
	embedIllu := func(path string) string {
		ext := strings.ToLower(filepath.Ext(path))
		if ext == "" || !sinEpubImgExts[ext] {
			return ""
		}
		if _, err := os.Stat(path); err != nil {
			return ""
		}
		imgSeq++
		internal, err := book.AddImage(path, fmt.Sprintf("illu-%d%s", imgSeq, ext))
		if err != nil {
			slog.Warn("原罪 EPUB 内嵌插图失败，保留占位", "path", path, "error", err)
			return ""
		}
		if !coverSet {
			coverSet = true
			book.SetCover(internal, "")
		}
		return internal
	}

	rounds := 0
	directive := ""
	for _, m := range msgs {
		if m.Role != "assistant" {
			directive = m.Content // 最近的用户指令，随下一条助手回合入书
			continue
		}
		rounds++
		body := sinSectionHTML(m.Content, directive, sinIllustrationMap(m.Extra), embedIllu)
		directive = ""
		if _, err := book.AddSection(body, sinSectionTitle(rounds), "", ""); err != nil {
			return "", fmt.Errorf("写入第 %d 回失败: %w", rounds, err)
		}
	}
	if rounds == 0 {
		return "", fmt.Errorf("故事还没有内容可导出")
	}

	name, err := sinExportFileName(topic.Title, topicID, time.Now())
	if err != nil {
		return "", err
	}
	dir := sinExportsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建原罪导出目录失败: %w", err)
	}
	path, err := sinExportUniquePath(dir, name, "epub")
	if err != nil {
		return "", err
	}
	if err := book.Write(path); err != nil {
		_ = os.Remove(path) // 半截产物不落第
		return "", fmt.Errorf("写出 EPUB 失败: %w", err)
	}
	return path, nil
}

// sinSectionTitle EPUB 小节标题（回合序数的中文序数）。
func sinSectionTitle(n int) string {
	return "第" + sinCnNum(n) + "回"
}

// sinCnNum 中文序数（1~99 用汉字，更大回退阿拉伯数字——真到 99 回再说）。
func sinCnNum(n int) string {
	digits := []string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九"}
	switch {
	case n <= 0:
		return "零"
	case n < 10:
		return digits[n]
	case n < 100:
		t, r := n/10, n%10
		s := ""
		if t > 1 {
			s += digits[t]
		}
		s += "十"
		if r > 0 {
			s += digits[r]
		}
		return s
	default:
		return strconv.Itoa(n)
	}
}

// sinSectionHTML 一条助手回合 → EPUB 小节 HTML：用户指令引块置首，正文按行
// 成段，插图标记就地替换（已生成 → 内嵌 <img>；未生成 → 画面描述占位）。
// cue 按消息内出现次序编键（sinCueKey），与 Markdown 导出/前端解析同一规则；
// 标记旁的残留文字照常成段（协议要求标记独占一行，这里兜底不吞字）。
func sinSectionHTML(content, directive string, arts map[string]string, embed func(string) string) string {
	var b strings.Builder
	if d := strings.TrimSpace(directive); d != "" {
		b.WriteString("<blockquote><p>我：" + html.EscapeString(d) + "</p></blockquote>\n")
	}
	cueIdx := 0
	for _, line := range strings.Split(content, "\n") {
		if seg := sinLineHTML(strings.TrimSpace(line), arts, embed, &cueIdx); seg != "" {
			b.WriteString(seg)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// sinLineHTML 单行 → HTML：文本段逃逸成 <p>，标记段替换为插图块/占位。
func sinLineHTML(line string, arts map[string]string, embed func(string) string, cueIdx *int) string {
	if line == "" {
		return ""
	}
	var b strings.Builder
	flush := func(seg string) {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return
		}
		b.WriteString("<p>" + html.EscapeString(seg) + "</p>\n")
	}
	rest := line
	for {
		start := strings.Index(rest, sinIllustrationCueOpen)
		if start < 0 {
			break
		}
		end := strings.Index(rest[start+len(sinIllustrationCueOpen):], sinIllustrationCueClose)
		if end < 0 {
			break // 未闭合标记：当普通文本，不当指令解析
		}
		flush(rest[:start])
		prompt := strings.TrimSpace(rest[start+len(sinIllustrationCueOpen) : start+len(sinIllustrationCueOpen)+end])
		key := sinCueKey(*cueIdx)
		*cueIdx++
		path := arts[key]
		if path == "" {
			path = arts[prompt] // 与 Markdown 导出同口径：次序键优先，画面描述键兜底
		}
		if path == "" {
			b.WriteString("<p><em>（插图未生成：" + html.EscapeString(prompt) + "）</em></p>\n")
		} else if internal := embed(path); internal != "" {
			b.WriteString("<div><img src=\"" + html.EscapeString(internal) + "\" alt=\"" + html.EscapeString(prompt) + "\"/></div>\n")
		} else {
			b.WriteString("<p><em>（插图未能内嵌：" + html.EscapeString(prompt) + "）</em></p>\n")
		}
		rest = rest[start+len(sinIllustrationCueOpen)+end+len(sinIllustrationCueClose):]
	}
	flush(rest)
	return strings.TrimRight(b.String(), "\n")
}
