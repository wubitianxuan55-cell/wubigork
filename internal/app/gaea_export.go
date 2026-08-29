package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/spaces"
)

// ExportDeliverableInput 是统一交付出口的入参：受控 Markdown + 目标格式。
type ExportDeliverableInput struct {
	Markdown string `json:"markdown"`
	Format   string `json:"format"` // docx | pptx | xlsx | md | pdf
	Title    string `json:"title"`
	Template string `json:"template"` // docx 模板：通用 | 公文 | 报告 | 合同
	Cover    bool   `json:"cover"`
	TOC      bool   `json:"toc"`
	Header   string `json:"header"`
	Footer   string `json:"footer"`
}

// ExportDeliverableResult 是交付结果（工作区相对路径）。
type ExportDeliverableResult struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Format string `json:"format"`
	Size   int64  `json:"size"`
}

var headingRe = regexp.MustCompile(`^#{1,3}\s+(.*)$`)

// GaeaExportDeliverable 统一交付出口：事实底座/对话成果的受控 Markdown →
// docx / pptx / xlsx / md / pdf，一稿多用、多形态一致。
func (a *App) GaeaExportDeliverable(in ExportDeliverableInput) (ExportDeliverableResult, error) {
	if strings.TrimSpace(in.Markdown) == "" {
		return ExportDeliverableResult{}, fmt.Errorf("交付内容为空")
	}
	format := strings.ToLower(strings.TrimPrefix(in.Format, "."))
	switch format {
	case "docx", "pptx", "xlsx", "md", "pdf":
	default:
		return ExportDeliverableResult{}, fmt.Errorf("不支持的交付格式 %q（docx/pptx/xlsx/md/pdf）", in.Format)
	}
	if in.Template == "" {
		in.Template = "通用"
	}
	switch in.Template {
	case "通用", "公文", "报告", "合同":
	default:
		return ExportDeliverableResult{}, fmt.Errorf("不支持的模板 %q（通用/公文/报告/合同）", in.Template)
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		if m := headingRe.FindStringSubmatch(in.Markdown); m != nil {
			title = strings.TrimSpace(m[1])
		} else {
			title = "gaea 交付物"
		}
	}
	if in.Footer == "" {
		in.Footer = "第 {page} 页"
	}

	// S4 产物路径分区：work 恒 .gaea/exports（现状不动），play 落 .gaea/play/exports。
	exportsDir := spaces.ExportsDir(gaeaCwd(), gaeaEffectiveSpace())
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return ExportDeliverableResult{}, err
	}
	stamp := time.Now().Format("20060102-150405")
	base := safeDeliverableName(title)
	outPath := filepath.Join(exportsDir, base+"-"+stamp+"."+format)

	switch format {
	case "md":
		if err := os.WriteFile(outPath, []byte(in.Markdown), 0o644); err != nil {
			return ExportDeliverableResult{}, err
		}
	case "docx":
		if err := exportDocx(in, title, outPath); err != nil {
			return ExportDeliverableResult{}, err
		}
	case "pptx":
		if err := exportPptx(in, title, outPath); err != nil {
			return ExportDeliverableResult{}, err
		}
	case "xlsx":
		if err := exportXlsx(in, outPath); err != nil {
			return ExportDeliverableResult{}, err
		}
	case "pdf":
		// Markdown → docx（临时，复用交付模板链路）→ LibreOffice PDF
		if err := markdownToPdf(in, title, outPath); err != nil {
			return ExportDeliverableResult{}, err
		}
	}

	info, err := os.Stat(outPath)
	if err != nil {
		return ExportDeliverableResult{}, err
	}
	rel, _ := filepath.Rel(gaeaCwd(), outPath)
	return ExportDeliverableResult{
		Path:   filepath.ToSlash(rel),
		Name:   filepath.Base(outPath),
		Format: format,
		Size:   info.Size(),
	}, nil
}

// exportDocx 经 create_docx.py（python-docx）从 Markdown 排版 Word。
func exportDocx(in ExportDeliverableInput, title, outPath string) error {
	script := findSkillScript("docx", "scripts", "create_docx.py")
	if script == "" {
		return fmt.Errorf("未找到 create_docx.py（.gaea/skills/docx/scripts）")
	}
	mdPath := strings.TrimSuffix(outPath, ".docx") + ".md"
	if err := os.WriteFile(mdPath, []byte(in.Markdown), 0o644); err != nil {
		return err
	}
	defer os.Remove(mdPath)

	args := []string{script, mdPath, outPath, "--title", title}
	if in.Cover {
		args = append(args, "--cover")
	}
	if in.TOC {
		args = append(args, "--toc")
	}
	if in.Header != "" {
		args = append(args, "--header", in.Header)
	}
	if in.Footer != "" {
		args = append(args, "--footer", in.Footer)
	}
	args = append(args, "--template", in.Template)
	if err := runPython(args, 120); err != nil {
		return fmt.Errorf("Word 导出失败: %w", err)
	}
	return nil
}

// exportPptx 把 Markdown 转成简单 slides spec，经 create_pptx.py 出 16:9 演示文稿。
func exportPptx(in ExportDeliverableInput, title, outPath string) error {
	script := findSkillScript("pptx", "scripts", "create_pptx.py")
	if script == "" {
		return fmt.Errorf("未找到 create_pptx.py（.gaea/skills/pptx/scripts）")
	}
	spec := markdownToSlides(in.Markdown, title)
	specPath := strings.TrimSuffix(outPath, ".pptx") + ".json"
	b, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(specPath, b, 0o644); err != nil {
		return err
	}
	defer os.Remove(specPath)
	if err := runPython([]string{script, specPath, outPath}, 120); err != nil {
		return fmt.Errorf("PPT 导出失败: %w", err)
	}
	return nil
}

// markdownToSlides 简单转换：# 开头开启新幻灯片，- / 数字列表与段落作为要点。
func markdownToSlides(md, title string) map[string]interface{} {
	type slide struct {
		Title  string        `json:"title"`
		Points []interface{} `json:"points"`
	}
	slides := []slide{}
	var cur *slide
	flush := func() {
		if cur != nil {
			if cur.Points == nil {
				cur.Points = []interface{}{}
			}
			if len(cur.Points) > 0 || cur.Title != "" {
				slides = append(slides, *cur)
			}
		}
	}
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			continue
		}
		if m := headingRe.FindStringSubmatch(line); m != nil {
			flush()
			cur = &slide{Title: strings.TrimSpace(m[1]), Points: []interface{}{}}
			continue
		}
		if strings.HasPrefix(line, "|") {
			// 表格行并入要点（去管道符与分隔行）
			if strings.Contains(line, "---") {
				continue
			}
			cells := strings.Split(strings.Trim(line, "|"), "|")
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			text := strings.Join(cells, " ｜ ")
			if text != "" {
				if cur == nil {
					cur = &slide{Points: []interface{}{}}
				}
				cur.Points = append(cur.Points, text)
			}
			continue
		}
		text := strings.TrimSpace(line)
		text = strings.TrimLeft(text, "-*123456789. ")
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if cur == nil {
			cur = &slide{Points: []interface{}{}}
		}
		cur.Points = append(cur.Points, text)
	}
	flush()
	if len(slides) == 0 {
		slides = append(slides, slide{Title: title})
	}
	return map[string]interface{}{
		"title":    title,
		"subtitle": "由 gaea 基于事实底座生成",
		"author":   "gaea",
		"slides":   slides,
	}
}

// exportXlsx 把 Markdown 中的表格提取为工作表（无表格时正文写入 Sheet1）。
func exportXlsx(in ExportDeliverableInput, outPath string) error {
	f := excelize.NewFile()
	defer f.Close()
	f.SetSheetName("Sheet1", "摘要")
	f.SetCellValue("摘要", "A1", firstLine(in.Title))

	tableIdx := 0
	var curTable []string
	flushTable := func() {
		if len(curTable) == 0 {
			return
		}
		tableIdx++
		sheet := "表" + strconv.Itoa(tableIdx)
		if tableIdx == 1 {
			sheet = "数据"
			f.SetSheetName("摘要", sheet)
		} else {
			f.NewSheet(sheet)
		}
		for ri, line := range curTable {
			line = strings.Trim(line, "|")
			cells := strings.Split(line, "|")
			for ci, c := range cells {
				axis, _ := excelize.CoordinatesToCellName(ci+1, ri+1)
				f.SetCellValue(sheet, axis, strings.TrimSpace(c))
			}
		}
		curTable = nil
	}

	bodyRow := 2
	for _, line := range strings.Split(in.Markdown, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "|") {
			if strings.Contains(line, "---") {
				continue
			}
			curTable = append(curTable, line)
			continue
		}
		flushTable()
		text := strings.TrimSpace(line)
		if text != "" && !strings.HasPrefix(text, "#") {
			f.SetCellValue("数据", "A"+strconv.Itoa(bodyRow), text)
			bodyRow++
		}
	}
	flushTable()
	if tableIdx == 0 {
		f.SetSheetName("摘要", "数据")
	}
	return f.SaveAs(outPath)
}

func runPython(args []string, timeoutSec int) error {
	return runProcess("python", args, timeoutSec)
}

// findSkillScript 定位技能脚本（工作区 .gaea/skills 或用户 skills 镜像）。
func findSkillScript(skill, sub, name string) string {
	candidates := []string{
		filepath.Join(gaeaCwd(), ".gaea", "skills", skill, sub, name),
		filepath.Join(gaeaCwd(), "skills", skill, sub, name),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".codex", "skills", skill, sub, name),
			filepath.Join(home, ".gaea", "skills", skill, sub, name),
		)
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

// safeDeliverableName 生成交付物文件名（保留中文，去掉非法路径字符）。
func safeDeliverableName(s string) string {
	re := regexp.MustCompile(`[\\/:*?"<>|]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.TrimSpace(s)
	if len(s) > 60 {
		s = s[:60]
	}
	if s == "" {
		s = "deliverable"
	}
	return s
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
