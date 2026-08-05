// Package proposal — 文档导出（Word/PDF）
package proposal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carmel/gooxml/document"
	"github.com/carmel/gooxml/measurement"
	"github.com/carmel/gooxml/schema/soo/wml"
	"github.com/ledongthuc/pdf"
)

// ExportDocx 导出方案为 Word (.docx) 文件，返回文件路径
func (s *Service) ExportDocx(proposalID string) (string, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return "", err
	}

	doc := document.New()

	// 封面标题
	titlePara := doc.AddParagraph()
	titlePara.Properties().SetAlignment(wml.ST_JcCenter)
	titleRun := titlePara.AddRun()
	titleRun.AddText(p.Title)
	titleRun.Properties().SetSize(28 * measurement.Point)
	titleRun.Properties().SetBold(true)

	// 元信息
	metaPara := doc.AddParagraph()
	metaPara.Properties().SetAlignment(wml.ST_JcCenter)
	metaPara.AddRun().AddText(fmt.Sprintf("类型：%s | 状态：%s | 更新：%s", p.Template, p.Status, p.UpdatedAt))
	doc.AddParagraph()

	if p.Requirements != "" {
		docxAddHeading(doc, "需求描述", 1)
		docxAddBody(doc, p.Requirements)
	}

	if p.BidSummary != nil {
		docxAddHeading(doc, "投标要点摘要", 1)
		if p.BidSummary.Duration != "" {
			docxAddBody(doc, "工期："+p.BidSummary.Duration)
		}
		if p.BidSummary.Overview != "" {
			docxAddBody(doc, "项目概况："+p.BidSummary.Overview)
		}
		if len(p.BidSummary.TechScoring) > 0 {
			docxAddHeading(doc, "评分标准", 2)
			for _, sc := range p.BidSummary.TechScoring {
				docxAddBody(doc, fmt.Sprintf("• %s（%s分）：%s", sc.Name, sc.MaxScore, sc.Requirement))
			}
		}
		if len(p.BidSummary.RedLines) > 0 {
			docxAddHeading(doc, "废标条款", 2)
			for _, rl := range p.BidSummary.RedLines {
				docxAddBody(doc, "⚠ "+rl)
			}
		}
		doc.AddParagraph()
	}

	var walk func(ss []ProposalSection, level int)
	walk = func(ss []ProposalSection, level int) {
		for _, sec := range ss {
			docxAddHeading(doc, sec.Title, level)
			if sec.Content != "" {
				docxAddMarkdown(doc, sec.Content)
			} else {
				docxAddBody(doc, "（待撰写）")
			}
			doc.AddParagraph()
			walk(sec.Children, level+1)
		}
	}
	walk(p.Sections, 1)

	exportDir := s.store.ExportDir()
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", fmt.Errorf("创建导出目录失败: %w", err)
	}
	exportPath := filepathInDir(exportDir, p.ID+".docx")
	if err := doc.SaveToFile(exportPath); err != nil {
		return "", fmt.Errorf("保存 docx 失败: %w", err)
	}
	return exportPath, nil
}

func docxAddHeading(doc *document.Document, text string, level int) {
	para := doc.AddParagraph()
	para.Properties().SetAlignment(wml.ST_JcLeft)
	run := para.AddRun()
	run.AddText(text)
	pr := run.Properties()
	pr.SetBold(true)
	switch level {
	case 1:
		pr.SetSize(20 * measurement.Point)
	case 2:
		pr.SetSize(16 * measurement.Point)
	default:
		pr.SetSize(14 * measurement.Point)
	}
}

func docxAddBody(doc *document.Document, text string) {
	para := doc.AddParagraph()
	para.AddRun().AddText(text)
}

func docxAddMarkdown(doc *document.Document, md string) {
	lines := strings.Split(md, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "### ") {
			docxAddHeading(doc, strings.TrimPrefix(line, "### "), 3)
		} else if strings.HasPrefix(line, "## ") {
			docxAddHeading(doc, strings.TrimPrefix(line, "## "), 2)
		} else if strings.HasPrefix(line, "# ") {
			docxAddHeading(doc, strings.TrimPrefix(line, "# "), 1)
		} else if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			docxAddBody(doc, "• "+strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* "))
		} else if strings.HasPrefix(line, "```") {
			continue
		} else {
			clean := strings.ReplaceAll(line, "**", "")
			clean = strings.ReplaceAll(clean, "__", "")
			clean = strings.ReplaceAll(clean, "*", "")
			docxAddBody(doc, clean)
		}
	}
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
