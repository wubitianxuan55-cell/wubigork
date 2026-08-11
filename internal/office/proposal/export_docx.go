// Package proposal — docx 排版引擎
package proposal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carmel/gooxml/document"
	"github.com/carmel/gooxml/measurement"
	"github.com/carmel/gooxml/schema/soo/wml"
)

// ExportOptions 导出选项
type ExportOptions struct {
	IncludeCover bool
	IncludeTOC   bool
	DarkRuleID   string
	FontSize     float64 // 正文磅值，0=默认 12
	LineSpacing  float64 // 行距倍数，0=默认 1.5
}

// DefaultExportOptions 默认导出选项
func DefaultExportOptions() ExportOptions {
	return ExportOptions{IncludeCover: true, IncludeTOC: true, FontSize: 12, LineSpacing: 1.5}
}

// renderDocxToFile 渲染方案为 docx 并保存到 exports 目录
func renderDocxToFile(p *Proposal, opts ExportOptions, exportDir string) (string, error) {
	return renderDocxToFileWithRule(p, opts, exportDir, nil)
}

// renderDocxToFileWithRule 按暗标规则渲染
func renderDocxToFileWithRule(p *Proposal, opts ExportOptions, exportDir string, darkRule *DarkRule) (string, error) {
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(exportDir, p.ID+".docx")
	doc, err := buildDocx(p, opts, darkRule)
	if err != nil {
		return "", err
	}
	if err := doc.SaveToFile(path); err != nil {
		return "", err
	}
	return path, nil
}

func buildDocx(p *Proposal, opts ExportOptions, darkRule *DarkRule) (*document.Document, error) {
	doc := document.New()
	doc.BodySection().SetPageMargins(
		2.54*measurement.Centimeter, 2.54*measurement.Centimeter,
		2.54*measurement.Centimeter, 2.54*measurement.Centimeter,
		1.5*measurement.Centimeter, 1.5*measurement.Centimeter,
		measurement.Zero,
	)
	addHeaderFooter(doc, p.Title)
	if opts.IncludeCover {
		addCover(doc, p)
	}
	if opts.IncludeTOC {
		addTOC(doc, p)
	}
	fontSize := opts.FontSize
	if fontSize <= 0 {
		fontSize = 12
	}
	renderSections(doc, p.Sections, "", 0, opts, darkRule)
	return doc, nil
}

func renderSections(doc *document.Document, ss []ProposalSection, prefix string, chapter int, opts ExportOptions, darkRule *DarkRule) {
	idx := 0
	for _, sec := range ss {
		idx++
		num, markdownLevel := numbering(sec, prefix, chapter, idx)
		content := sec.Content
		if darkRule != nil {
			content = darkRule.Apply(content)
		}
		heading := doc.AddParagraph()
		run := heading.AddRun()
		run.AddText(num + " " + sec.Title)
		pr := run.Properties()
		pr.SetBold(darkRule == nil || !darkRule.Options.NoBold)
		pr.SetSize(measurement.Distance(markdownSize(markdownLevel, opts.FontSize)))
		pr.SetFontFamily("宋体")
		for _, block := range parseMarkdownBlocks(content) {
			renderBlock(doc, block, opts, darkRule)
		}
		childChapter := chapter
		if sec.Level == 1 {
			childChapter = idx
		}
		renderSections(doc, sec.Children, num, childChapter, opts, darkRule)
	}
}

func addHeaderFooter(doc *document.Document, title string) {
	hdr := doc.AddHeader()
	hp := hdr.AddParagraph()
	hr := hp.AddRun()
	hr.AddText(title)
	hr.Properties().SetSize(10 * measurement.Point)
	doc.BodySection().SetHeader(hdr, wml.ST_HdrFtrDefault)

	ftr := doc.AddFooter()
	fp := ftr.AddParagraph()
	fp.Properties().SetAlignment(wml.ST_JcCenter)
	fr := fp.AddRun()
	fr.AddText("第 ")
	fr.AddField("PAGE")
	fr.AddText(" 页")
	doc.BodySection().SetFooter(ftr, wml.ST_HdrFtrDefault)
}

func addCover(doc *document.Document, p *Proposal) {
	para := doc.AddParagraph()
	para.Properties().SetAlignment(wml.ST_JcCenter)
	run := para.AddRun()
	run.AddText(p.Title)
	run.Properties().SetSize(28 * measurement.Point)
	run.Properties().SetBold(true)
	run.Properties().SetFontFamily("宋体")
	meta := doc.AddParagraph()
	meta.Properties().SetAlignment(wml.ST_JcCenter)
	mr := meta.AddRun()
	mr.AddText(fmt.Sprintf("类型：%s | 更新：%s", p.Template, p.UpdatedAt))
	mr.Properties().SetSize(12 * measurement.Point)
	doc.AddParagraph().AddRun().AddPageBreak()
}

func addTOC(doc *document.Document, p *Proposal) {
	title := doc.AddParagraph()
	tr := title.AddRun()
	tr.AddText("目 录")
	tr.Properties().SetBold(true)
	tr.Properties().SetSize(16 * measurement.Point)
	doc.AddParagraph()
	renderTOCSections(doc, p.Sections, "", 0)
	doc.AddParagraph().AddRun().AddPageBreak()
}

func renderTOCSections(doc *document.Document, ss []ProposalSection, prefix string, chapter int) {
	idx := 0
	for _, sec := range ss {
		idx++
		num, _ := numbering(sec, prefix, chapter, idx)
		para := doc.AddParagraph()
		run := para.AddRun()
		run.AddText(num + " " + sec.Title)
		run.Properties().SetSize(12 * measurement.Point)
		childChapter := chapter
		if sec.Level == 1 {
			childChapter = idx
		}
		renderTOCSections(doc, sec.Children, num, childChapter)
	}
}

func numbering(sec ProposalSection, prefix string, chapter, idx int) (string, int) {
	level := sec.Level
	if level <= 0 {
		if prefix == "" {
			level = 1
		} else {
			level = 2
		}
	}
	switch level {
	case 1:
		return fmt.Sprintf("第%d章", idx), 1
	case 2:
		return fmt.Sprintf("%d.%d", chapter, idx), 2
	default:
		return fmt.Sprintf("%s.%d", prefix, idx), 3
	}
}

func markdownSize(level int, base float64) float64 {
	switch level {
	case 1:
		return 22
	case 2:
		return 16
	default:
		return 14
	}
}

func renderBlock(doc *document.Document, block mdBlock, opts ExportOptions, darkRule *DarkRule) {
	switch block.kind {
	case "heading":
		para := doc.AddParagraph()
		run := para.AddRun()
		run.AddText(block.text)
		run.Properties().SetBold(darkRule == nil || !darkRule.Options.NoBold)
		run.Properties().SetSize(measurement.Distance(markdownSize(block.level, opts.FontSize)))
		run.Properties().SetFontFamily("宋体")
	case "list":
		para := doc.AddParagraph()
		run := para.AddRun()
		run.AddText("• " + block.text)
		run.Properties().SetSize(measurement.Distance(opts.FontSize))
	case "table":
		if len(block.rows) == 0 {
			return
		}
		table := doc.AddTable()
		for _, row := range block.rows {
			r := table.AddRow()
			for _, cell := range row {
				c := r.AddCell()
				p := c.AddParagraph()
				cr := p.AddRun()
				cr.AddText(cell)
				cr.Properties().SetSize(measurement.Distance(opts.FontSize))
			}
		}
		doc.AddParagraph()
	case "code":
		para := doc.AddParagraph()
		run := para.AddRun()
		run.AddText(strings.TrimSpace(block.text))
		run.Properties().SetFontFamily("Consolas")
		run.Properties().SetSize(measurement.Distance(opts.FontSize - 1))
	default: // para
		para := doc.AddParagraph()
		run := para.AddRun()
		run.AddText(block.text)
		run.Properties().SetSize(measurement.Distance(opts.FontSize))
		run.Properties().SetFontFamily("宋体")
	}
}

// getDarkRule 从暗标规则库取规则（ID 为空或未找到返回 nil）
func getDarkRule(s *Service, id string) *DarkRule {
	if id == "" || s == nil || s.store == nil {
		return nil
	}
	rules, err := s.store.ListDarkRules()
	if err != nil {
		return nil
	}
	for i := range rules {
		if rules[i].ID == id && rules[i].Enabled {
			return &rules[i]
		}
	}
	return nil
}
