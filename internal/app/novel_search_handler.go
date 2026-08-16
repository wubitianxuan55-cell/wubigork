package app

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// NovelSearchHit 全文检索命中（一章最多一个命中：标题命中优先，否则正文首次出现）。
type NovelSearchHit struct {
	NodeID     string `json:"node_id"`
	Title      string `json:"title"`
	ChapterNum int    `json:"chapter_num"`
	Branch     string `json:"branch,omitempty"`
	Snippet    string `json:"snippet"`
	TitleHit   bool   `json:"title_hit"`
}

// NovelSearch 全文检索：按大纲顺序扫描各章（标题 + 正文，大小写不敏感）。
// 正文命中返回首次出现位置的上下文片段；上限 100 条，缺失/损坏章节跳过。
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
	var hits []NovelSearchHit
	for _, node := range of.Nodes {
		if node.OrderIndex <= 0 {
			continue // 卷/分组节点不参与检索
		}
		title := strings.TrimSpace(node.Title)
		if strings.Contains(strings.ToLower(title), lowerQ) {
			hits = append(hits, NovelSearchHit{
				NodeID: node.ID, Title: title, ChapterNum: node.OrderIndex,
				Branch: node.Branch, Snippet: "章节标题命中", TitleHit: true,
			})
			continue
		}
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
		if idx := strings.Index(strings.ToLower(content), lowerQ); idx >= 0 {
			hits = append(hits, NovelSearchHit{
				NodeID: node.ID, Title: title, ChapterNum: node.OrderIndex,
				Branch: node.Branch, Snippet: snippetAround(content, idx, utf8.RuneCountInString(q)),
			})
		}
		if len(hits) >= 100 {
			break
		}
	}
	return hits, nil
}

// snippetAround 截取命中位置前后各 40 字的上下文片段（按 rune，省略号补齐）。
func snippetAround(s string, byteIdx, queryRunes int) string {
	runes := []rune(s)
	runeStart := utf8.RuneCountInString(s[:byteIdx])
	const margin = 40
	start := runeStart - margin
	if start < 0 {
		start = 0
	}
	end := runeStart + queryRunes + margin
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
