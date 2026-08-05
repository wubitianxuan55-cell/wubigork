// Package proposal — 文档导出（Word/PDF）
package proposal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExportDocx 导出方案为 Word（默认排版选项）
func (s *Service) ExportDocx(proposalID string) (string, error) {
	return s.ExportDocxWithOptions(proposalID, DefaultExportOptions())
}

// ExportDocxWithOptions 按选项导出方案
func (s *Service) ExportDocxWithOptions(proposalID string, opts ExportOptions) (string, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return "", err
	}
	path, err := renderDocxToFileWithRule(p, opts, s.store.ExportDir(), getDarkRule(s, opts.DarkRuleID))
	if err != nil {
		return "", err
	}
	p.advanceStage(StageFormat)
	p.UpdatedAt = now()
	_ = s.store.Update(p)
	return path, nil
}

// ExportSectionDocx 导出单章节（含子章节）
func (s *Service) ExportSectionDocx(proposalID, sectionID string, opts ExportOptions) (string, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return "", err
	}
	sec := findSectionByID(p.Sections, sectionID)
	if sec == nil {
		return "", fmt.Errorf("章节未找到: %s", sectionID)
	}
	mini := &Proposal{ID: p.ID + "-" + sectionID, Title: sec.Title, Template: p.Template, Sections: []ProposalSection{*sec}}
	path, err := renderDocxToFile(mini, opts, s.store.ExportDir())
	if err != nil {
		return "", err
	}
	p.advanceStage(StageFormat)
	p.UpdatedAt = now()
	_ = s.store.Update(p)
	return path, nil
}

// ReadPdfFile 读取 PDF 文件文本内容
func ReadPdfFile(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 PDF 失败: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	totalPage := r.NumPage()
	for i := 1; i <= totalPage; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n")
		if buf.Len() > 100000 {
			buf.WriteString("\n…（内容过长已截断）")
			break
		}
	}
	result := buf.String()
	if result == "" {
		return "", fmt.Errorf("PDF 无可提取文本（可能是扫描件或图片型 PDF）")
	}
	return result, nil
}

// ReadTextFile 读取文本/PDF/Word 文件内容
func ReadTextFile(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		return ReadPdfFile(filePath)
	case ".docx", ".doc":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		text := string(data)
		if len(text) > 50000 {
			return "", fmt.Errorf("Word 文件较大，建议先转换为文本再粘贴")
		}
		return text, nil
	default:
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}
