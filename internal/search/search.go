package search

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// Result 搜索结果
type Result struct {
	File    string `json:"file"`
	Context string `json:"context"` // 匹配行前后各 40 字符
}

// Searcher 全文搜索器
type Searcher struct {
	pm *project.Manager
}

// New 创建搜索器
func New(pm *project.Manager) *Searcher {
	return &Searcher{pm: pm}
}

// Search 在项目所有文件中搜索关键词
func (s *Searcher) Search(query string) ([]Result, error) {
	if query == "" {
		return nil, nil
	}
	lower := strings.ToLower(query)

	var results []Result

	// 搜索 worldview.md
	wv, err := s.pm.ReadWorldview()
	if err != nil {
		slog.Warn("搜索: 读取世界观失败", "error", err)
	}
	results = append(results, searchInText("世界观", wv, lower)...)

	// 搜索所有章节
	for i := 1; ; i++ {
		content, err := s.pm.ReadChapter(i)
		if err != nil {
			break
		}
		results = append(results, searchInText(
			fmt.Sprintf("第%d章", i), content, lower)...)
	}

	return results, nil
}

// SearchAll 搜索所有文件和 JSON 字段
func (s *Searcher) SearchAll(query string) (map[string][]Result, error) {
	result := make(map[string][]Result)

	results := []Result{}
	lower := strings.ToLower(query)

	// 世界观
	wv, err := s.pm.ReadWorldview()
	if err != nil {
		slog.Warn("搜索: 读取世界观失败", "error", err)
	}
	results = append(results, searchInText("世界观", wv, lower)...)
	result["worldview"] = results

	// 章节
	chapterResults := []Result{}
	for i := 1; ; i++ {
		content, err := s.pm.ReadChapter(i)
		if err != nil {
			break
		}
		chapterResults = append(chapterResults, searchInText(
			fmt.Sprintf("第%d章", i), content, lower)...)
	}
	result["chapters"] = chapterResults

	// 角色
	charResults := []Result{}
	chars, err := s.pm.ReadCharacters()
	if err != nil {
		slog.Warn("搜索: 读取角色失败", "error", err)
	}
	if chars != nil {
		for _, ch := range chars.Characters {
			if strings.Contains(strings.ToLower(ch.Name), lower) ||
				strings.Contains(strings.ToLower(ch.Personality), lower) ||
				strings.Contains(strings.ToLower(ch.Background), lower) {
				charResults = append(charResults, Result{
					File:    fmt.Sprintf("角色: %s [%s]", ch.Name, ch.RoleType),
					Context: fmt.Sprintf("%s / %s", ch.Personality, ch.Background),
				})
			}
		}
	}
	result["characters"] = charResults

	// 大纲
	outlineResults := []Result{}
	outlines, err := s.pm.ReadOutlines()
	if err != nil {
		slog.Warn("搜索: 读取大纲失败", "error", err)
	}
	if outlines != nil {
		for _, node := range outlines.Nodes {
			searchOutlineNode(&node, lower, &outlineResults)
		}
	}
	result["outlines"] = outlineResults

	return result, nil
}

func searchOutlineNode(node *types.OutlineNode, lower string, results *[]Result) {
	if strings.Contains(strings.ToLower(node.Title), lower) ||
		strings.Contains(strings.ToLower(node.Summary), lower) {
		*results = append(*results, Result{
			File:    fmt.Sprintf("大纲: %s", node.Title),
			Context: node.Summary,
		})
	}
	for i := range node.Children {
		searchOutlineNode(&node.Children[i], lower, results)
	}
}

func searchInText(file, text, lower string) []Result {
	var results []Result
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), lower) {
			ctx := line
			if len([]rune(ctx)) > 120 {
				ctx = string([]rune(ctx)[:120]) + "..."
			}
			results = append(results, Result{
				File:    file,
				Context: ctx,
			})
		}
	}
	return results
}

// ── 版本备份 ─────────────────────────────────────────────────

