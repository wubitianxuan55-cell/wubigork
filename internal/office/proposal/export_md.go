package proposal

import (
	"fmt"
	"strings"
)

// ExportMarkdown 导出方案为 Markdown 文件，返回文件路径
func (s *Service) ExportMarkdown(proposalID string) (string, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", p.Title))
	sb.WriteString(fmt.Sprintf("> 类型：%s | 状态：%s | 更新：%s\n\n", p.Template, p.Status, p.UpdatedAt))
	if p.Requirements != "" {
		sb.WriteString(fmt.Sprintf("## 需求描述\n\n%s\n\n", p.Requirements))
	}
	var walk func(ss []ProposalSection, level int)
	walk = func(ss []ProposalSection, level int) {
		for _, sec := range ss {
			sb.WriteString(strings.Repeat("#", level) + " " + sec.Title + "\n\n")
			if sec.Content != "" {
				sb.WriteString(sec.Content + "\n\n")
			} else {
				sb.WriteString("（待撰写）\n\n")
			}
			walk(sec.Children, level+1)
		}
	}
	walk(p.Sections, 2)
	exportPath := filepathInDir(s.store.ExportDir(), p.ID+".md")
	if err := writeFile(exportPath, sb.String()); err != nil {
		return "", err
	}
	return exportPath, nil
}
