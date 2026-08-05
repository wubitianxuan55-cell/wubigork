package proposal

import (
	"strings"
)

// ExportMarkdown 导出方案为 Markdown（自动编号装配），返回文件路径
func (s *Service) ExportMarkdown(proposalID string) (string, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString(Assemble(p))
	if p.Requirements != "" {
		sb.WriteString("\n---\n\n## 需求描述\n\n" + p.Requirements + "\n")
	}
	exportPath := filepathInDir(s.store.ExportDir(), p.ID+".md")
	if err := writeFile(exportPath, sb.String()); err != nil {
		return "", err
	}
	return exportPath, nil
}
