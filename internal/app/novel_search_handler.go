package app

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	// novelSearchTotalCap 返回命中条数总上限（total_hits 仍统计全量，不受此限制）。
	novelSearchTotalCap = 300
	// novelSearchPerChapterCap 单章返回命中条数上限。
	novelSearchPerChapterCap = 20
	// snippetMargin snippet 中命中词前后保留的上下文字数（按 rune）。
	snippetMargin = 40
)

// NovelSearchHit 全文检索命中。
// 结构演进（全文搜索升级）：位置字段与汇总字段为新增，旧字段（node_id/title/
// chapter_num/branch/snippet/title_hit）名称与语义均不变。
// 说明：绑定面 NovelSearch 返回扁平切片（签名冻结），全书汇总（total_hits/
// chapter_count）冗余填充在每一行上，前端取首行读取即可。
type NovelSearchHit struct {
	NodeID     string `json:"node_id"`
	Title      string `json:"title"`
	ChapterNum int    `json:"chapter_num"`
	Branch     string `json:"branch,omitempty"`
	Snippet    string `json:"snippet"`
	TitleHit   bool   `json:"title_hit"`

	MatchIndex     int `json:"match_index"`     // 本章内命中序次（1-based；标题命中为 1）
	ParagraphIndex int `json:"paragraph_index"` // 正文段落索引（0-based，按空行分段；标题命中为 -1）
	CharOffset     int `json:"char_offset"`     // 段内命中起始 rune 偏移（标题命中为 -1）
	MatchLen       int `json:"match_len"`       // 命中词 rune 长度
	TotalHits      int `json:"total_hits"`      // 全书总命中数（不受返回条数上限影响；每行冗余）
	ChapterCount   int `json:"chapter_count"`   // 命中章节总数（每行冗余）
}

// novelParaSplitRe 段落切分：与前端阅读渲染器的 scene.split(/\n\s*\n/) 对齐，
// 保证 paragraph_index 可直接映射为 .novel-reading-p 的 DOM 序号。
var novelParaSplitRe = regexp.MustCompile(`\n\s*\n`)

// splitParagraphs 按空行切段并去除首尾空白、丢弃空段（与前端 trim + filter(Boolean) 一致）。
func splitParagraphs(content string) []string {
	parts := novelParaSplitRe.Split(content, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// NovelSearch 全文检索：按大纲顺序扫描各章（标题 + 正文，大小写不敏感）。
// 每章返回全部命中（单章上限 20 条，返回总上限 300 条）；total_hits/chapter_count
// 为全量统计（不受上限影响），冗余填充在每条命中上。缺失/损坏章节跳过。
func (a *writingState) NovelSearch(query string) ([]NovelSearchHit, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	if a.outlineAgent == nil {
		return nil, fmt.Errorf("大纲未初始化")
	}
	of := a.outlineAgent.GetOutlines()
	if of == nil {
		return nil, fmt.Errorf("读取大纲失败")
	}
	lowerQ := strings.ToLower(q)
	qRunes := utf8.RuneCountInString(q)
	var hits []NovelSearchHit
	totalHits, chapterCount := 0, 0
	for _, node := range of.Nodes {
		if node.OrderIndex <= 0 {
			continue // 卷/分组节点不参与检索
		}
		title := strings.TrimSpace(node.Title)
		chapterMatches := 0
		if strings.Contains(strings.ToLower(title), lowerQ) {
			// 标题命中：语义不变（不再扫描正文），位置信息不适用（-1）。
			chapterMatches = 1
			if len(hits) < novelSearchTotalCap {
				hits = append(hits, NovelSearchHit{
					NodeID: node.ID, Title: title, ChapterNum: node.OrderIndex,
					Branch: node.Branch, Snippet: "章节标题命中", TitleHit: true,
					MatchIndex: 1, ParagraphIndex: -1, CharOffset: -1, MatchLen: qRunes,
				})
			}
		} else {
			var content string
			var err error
			if node.Branch != "" {
				content, err = pm.ReadChapterBranch(node.OrderIndex, node.Branch)
			} else {
				content, err = pm.ReadChapter(node.OrderIndex)
			}
			if err != nil {
				continue
			}
			for pi, para := range splitParagraphs(content) {
				lower := strings.ToLower(para)
				for start := 0; ; {
					rel := strings.Index(lower[start:], lowerQ)
					if rel < 0 {
						break
					}
					abs := start + rel
					chapterMatches++
					if len(hits) < novelSearchTotalCap && chapterMatches <= novelSearchPerChapterCap {
						hits = append(hits, NovelSearchHit{
							NodeID: node.ID, Title: title, ChapterNum: node.OrderIndex,
							Branch:         node.Branch,
							Snippet:        snippetAround(para, abs, qRunes),
							MatchIndex:     chapterMatches,
							ParagraphIndex: pi,
							CharOffset:     utf8.RuneCountInString(para[:abs]),
							MatchLen:       qRunes,
						})
					}
					start = abs + len(lowerQ)
				}
			}
		}
		if chapterMatches > 0 {
			totalHits += chapterMatches
			chapterCount++
		}
	}
	// 汇总字段冗余回填到每一行（签名冻结，无法返回顶层统计结构）。
	for i := range hits {
		hits[i].TotalHits = totalHits
		hits[i].ChapterCount = chapterCount
	}
	return hits, nil
}

// snippetAround 截取命中位置前后各 snippetMargin 字的上下文片段（按 rune，省略号补齐）。
func snippetAround(s string, byteIdx, queryRunes int) string {
	runes := []rune(s)
	runeStart := utf8.RuneCountInString(s[:byteIdx])
	start := runeStart - snippetMargin
	if start < 0 {
		start = 0
	}
	end := runeStart + queryRunes + snippetMargin
	if end > len(runes) {
		end = len(runes)
	}
	out := string(runes[start:end])
	if start > 0 {
		out = "…" + out
	}
	if end < len(runes) {
		out += "…"
	}
	return out
}
