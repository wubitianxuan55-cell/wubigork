// Package proposal — 工具函数
package proposal

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/util"
)

func filepathInDir(dir, filename string) string {
	return filepath.Join(dir, filename)
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}
	// 兜底：定位第一个 { 到最后一个 }，兼容无代码块包裹的输出
	return util.ExtractJSON(strings.TrimSpace(s))
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}

// flattenSections 返回章节树中所有节点的指针（保持深度优先顺序），
// 用于跨层级查找并原地更新内容。
func flattenSections(ss []ProposalSection) []*ProposalSection {
	var r []*ProposalSection
	for i := range ss {
		r = append(r, &ss[i])
		r = append(r, flattenSections(ss[i].Children)...)
	}
	return r
}

// countSections 统计章节树中的节点总数
func countSections(ss []ProposalSection) int {
	n := 0
	for _, s := range ss {
		n += 1 + countSections(s.Children)
	}
	return n
}
