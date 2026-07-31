package builtin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wubigork/wubigork/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(formatConvert{}) }

type formatConvert struct{}

func (formatConvert) Name() string { return "format_convert" }

func (formatConvert) Description() string {
	return "文档格式转换：将 docx/xlsx/pdf 文件转换为 Markdown。docx→md 保留标题层级和表格；xlsx→md 生成表格；pdf→md 提取文本（含 OCR 扫描件回退）。"
}

func (formatConvert) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"源文件路径（支持 .docx/.xlsx/.pdf）"},
  "output":{"type":"string","description":"输出 Markdown 文件路径（可选，不指定则返回文本）"},
  "pages":{"type":"string","description":"PDF页码范围，如\"1-5\"或\"1,3,5\"（仅PDF有效）"}
},
"required":["path"]
}`)
}

func (formatConvert) ReadOnly() bool { return true }

func (formatConvert) CompactDescription() string     { return compactDesc["format_convert"] }
func (formatConvert) CompactSchema() json.RawMessage { return compactSchema["format_convert"] }

type fcInput struct {
	Path   string `json:"path"`
	Output string `json:"output,omitempty"`
	Pages  string `json:"pages,omitempty"`
}

func (formatConvert) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p fcInput
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path 不能为空")
	}

	ext := strings.ToLower(filepath.Ext(p.Path))
	var md string
	var err error

	switch ext {
	case ".docx", ".doc":
		md, err = docxToMarkdown(p.Path)
	case ".xlsx", ".xls":
		md, err = xlsxToMarkdown(p.Path)
	case ".pdf":
		md, err = pdfToMarkdown(p.Path, p.Pages)
	default:
		return "", fmt.Errorf("不支持的文件格式: %s（支持 .docx/.xlsx/.pdf）", ext)
	}
	if err != nil {
		return "", fmt.Errorf("转换失败: %w", err)
	}

	md = fmt.Sprintf("# 文档转换: %s\n\n%s\n\n---\n*由 gaea format_convert 转换*", filepath.Base(p.Path), md)

	if p.Output != "" {
		if err := os.WriteFile(p.Output, []byte(md), 0644); err != nil {
			return "", fmt.Errorf("写入输出文件失败: %w", err)
		}
		return tool.WrapText(fmt.Sprintf("已转换并保存为 %s（%d 字符）", p.Output, len(md))), nil
	}
	return tool.WrapText(md), nil
}

// docxToMarkdown 提取 docx 为 Markdown（含标题和表格）
func docxToMarkdown(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("不是有效的 docx 文件: %w", err)
	}

	var docXML []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			docXML, _ = io.ReadAll(rc)
			rc.Close()
			break
		}
	}
	if docXML == nil {
		return "", fmt.Errorf("未找到 word/document.xml")
	}

	// 解析带命名空间的 XML
	type wDoc struct {
		Body struct {
			InnerXML string `xml:",innerxml"`
		} `xml:"body"`
	}
	var doc wDoc
	if err := xml.Unmarshal(docXML, &doc); err != nil {
		return "", fmt.Errorf("解析 XML 失败: %w", err)
	}

	// 手动解析段落和表格
	// 用字符串方式处理命名空间（<w:p> 代表段落，<w:tbl> 代表表格）
	bodyContent := doc.Body.InnerXML

	var mdParts []string
	pos := 0
	tblIdx := 0
	for pos < len(bodyContent) {
		// 检查表格
		tblStart := strings.Index(bodyContent[pos:], "<w:tbl>")
		pStart := strings.Index(bodyContent[pos:], "<w:p>")
		if tblStart < 0 && pStart < 0 {
			break
		}
		if tblStart >= 0 && (pStart < 0 || tblStart < pStart) {
			// 处理表格
			absStart := pos + tblStart
			tblEnd := strings.Index(bodyContent[absStart:], "</w:tbl>")
			if tblEnd < 0 {
				break
			}
			tblXML := bodyContent[absStart : absStart+tblEnd+8]
			tblMD := extractDocxTable(tblXML, &tblIdx)
			mdParts = append(mdParts, tblMD)
			pos = absStart + tblEnd + 8
		} else {
			// 处理段落
			absStart := pos + pStart
			pEnd := strings.Index(bodyContent[absStart:], "</w:p>")
			if pEnd < 0 {
				break
			}
			pXML := bodyContent[absStart : absStart+pEnd+6]
			pMD := extractDocxParagraph(pXML)
			if strings.TrimSpace(pMD) != "" {
				mdParts = append(mdParts, pMD)
			}
			pos = absStart + pEnd + 6
		}
	}

	return strings.Join(mdParts, "\n\n"), nil
}

func extractDocxParagraph(pXML string) string {
	// 提取段落属性中的样式
	style := ""
	if si := strings.Index(pXML, "<w:pStyle"); si >= 0 {
		sv := extractAttr(pXML[si:], "w:val")
		style = sv
	}

	// 提取所有 w:t 标签内的文本
	var texts []string
	remaining := pXML
	for {
		tStart := strings.Index(remaining, "<w:t")
		if tStart < 0 {
			break
		}
		// 跳过 <w:t ...> 到 >
		gt := strings.IndexByte(remaining[tStart:], '>')
		if gt < 0 {
			break
		}
		contentStart := tStart + gt + 1
		tEnd := strings.Index(remaining[contentStart:], "</w:t>")
		if tEnd < 0 {
			break
		}
		texts = append(texts, remaining[contentStart:contentStart+tEnd])
		remaining = remaining[contentStart+tEnd+6:]
	}
	text := strings.Join(texts, "")

	// 根据样式决定输出格式
	if style != "" {
		if style == "Title" || style == "title" {
			return "# " + text
		}
		if style == "Heading1" || style == "heading1" || style == "1" {
			return "# " + text
		}
		if style == "Heading2" || style == "heading2" || style == "2" {
			return "## " + text
		}
		if style == "Heading3" || style == "heading3" || style == "3" {
			return "### " + text
		}
		// 也检查 heading 前缀
		if strings.HasPrefix(style, "Heading") || strings.HasPrefix(style, "heading") {
			levelStr := strings.TrimLeft(style, "Headingheading ")
			level := 1
			if len(levelStr) == 1 && levelStr[0] >= '1' && levelStr[0] <= '9' {
				level = int(levelStr[0] - '0')
			}
			if level > 6 {
				level = 6
			}
			return strings.Repeat("#", level) + " " + text
		}
	}
	return text
}

func extractDocxTable(tblXML string, idx *int) string {
	*idx++
	var md strings.Builder
	fmt.Fprintf(&md, "**表 %d**\n\n", *idx)

	// 提取行
	var rows []string
	remaining := tblXML
	for {
		trStart := strings.Index(remaining, "<w:tr>")
		if trStart < 0 {
			break
		}
		trEnd := strings.Index(remaining[trStart:], "</w:tr>")
		if trEnd < 0 {
			break
		}
		rowXML := remaining[trStart : trStart+trEnd+7]
		remaining = remaining[trStart+trEnd+7:]

		// 提取单元格
		var cells []string
		rc := rowXML
		for {
			tcStart := strings.Index(rc, "<w:tc>")
			if tcStart < 0 {
				break
			}
			tcEnd := strings.Index(rc[tcStart:], "</w:tc>")
			if tcEnd < 0 {
				break
			}
			cellXML := rc[tcStart : tcStart+tcEnd+7]
			rc = rc[tcStart+tcEnd+7:]

			// 提取单元格内文本
			cellText := extractCellText(cellXML)
			cells = append(cells, strings.TrimSpace(cellText))
		}
		if len(cells) > 0 {
			rows = append(rows, strings.Join(cells, " | "))
		}
	}

	if len(rows) == 0 {
		return ""
	}
	// 表头行
	md.WriteString("| " + rows[0] + " |\n")
	// 分隔行
	colCount := len(strings.Split(rows[0], " | "))
	md.WriteString("|" + strings.Repeat(" --- |", colCount) + "\n")
	// 数据行
	for i := 1; i < len(rows); i++ {
		md.WriteString("| " + rows[i] + " |\n")
	}
	return md.String()
}

func extractCellText(cellXML string) string {
	var texts []string
	remaining := cellXML
	for {
		tStart := strings.Index(remaining, "<w:t")
		if tStart < 0 {
			break
		}
		gt := strings.IndexByte(remaining[tStart:], '>')
		if gt < 0 {
			break
		}
		cs := tStart + gt + 1
		tEnd := strings.Index(remaining[cs:], "</w:t>")
		if tEnd < 0 {
			break
		}
		texts = append(texts, remaining[cs:cs+tEnd])
		remaining = remaining[cs+tEnd+6:]
	}
	return strings.Join(texts, "")
}

func extractAttr(xml, attr string) string {
	attr = strings.ToLower(attr)
	lowXML := strings.ToLower(xml)
	idx := strings.Index(lowXML, attr+`="`)
	if idx < 0 {
		idx = strings.Index(lowXML, attr+`='`)
	}
	if idx < 0 {
		return ""
	}
	idx += len(attr) + 2
	end := strings.IndexByte(xml[idx:], '"')
	if end < 0 {
		end = strings.IndexByte(xml[idx:], '\'')
	}
	if end < 0 {
		return ""
	}
	return xml[idx : idx+end]
}

// xlsxToMarkdown 提取 xlsx 为 Markdown 表格
func xlsxToMarkdown(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("不是有效的 xlsx 文件: %w", err)
	}

	// 读 sharedStrings
	ssMap := make(map[int]string)
	for _, f := range r.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, _ := f.Open()
			ssXML, _ := io.ReadAll(rc)
			rc.Close()
			type ssDoc struct {
				Items []struct {
					Text string `xml:"t"`
				} `xml:"si"`
			}
			var ss ssDoc
			if xml.Unmarshal(ssXML, &ss) == nil {
				for i, si := range ss.Items {
					ssMap[i] = si.Text
				}
			}
			break
		}
	}

	// 读 workbook 获取 sheet 名
	type wbDoc struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
		} `xml:"sheets>sheet"`
	}
	var wb wbDoc
	for _, f := range r.File {
		if f.Name == "xl/workbook.xml" {
			rc, _ := f.Open()
			wbXML, _ := io.ReadAll(rc)
			rc.Close()
			xml.Unmarshal(wbXML, &wb)
			break
		}
	}

	var md strings.Builder
	for i := 1; ; i++ {
		sheetFile := fmt.Sprintf("xl/worksheets/sheet%d.xml", i)
		var sheetXML []byte
		found := false
		for _, f := range r.File {
			if f.Name == sheetFile {
				rc, _ := f.Open()
				sheetXML, _ = io.ReadAll(rc)
				rc.Close()
				found = true
				break
			}
		}
		if !found {
			break
		}

		sheetName := fmt.Sprintf("Sheet%d", i)
		if i-1 < len(wb.Sheets) {
			sheetName = wb.Sheets[i-1].Name
		}
		fmt.Fprintf(&md, "### %s\n\n", sheetName)

		// 解析 sheet XML
		type sheetData struct {
			Rows []struct {
				Cells []struct {
					Ref  string `xml:"r,attr"`
					Type string `xml:"t,attr"`
					Val  string `xml:"v"`
				} `xml:"c"`
			} `xml:"sheetData>row"`
		}
		var sd sheetData
		if xml.Unmarshal(sheetXML, &sd) != nil {
			continue
		}

		for ri, row := range sd.Rows {
			var vals []string
			for _, cell := range row.Cells {
				val := cell.Val
				if cell.Type == "s" {
					var idx int
					if _, serr := fmt.Sscanf(val, "%d", &idx); serr == nil {
						if s, ok := ssMap[idx]; ok {
							val = s
						}
					}
				}
				vals = append(vals, val)
			}
			if len(vals) == 0 {
				continue
			}
			if ri == 0 {
				md.WriteString("| " + strings.Join(vals, " | ") + " |\n")
				md.WriteString("|" + strings.Repeat(" --- |", len(vals)) + "\n")
			} else {
				md.WriteString("| " + strings.Join(vals, " | ") + " |\n")
			}
		}
		md.WriteString("\n")
	}
	if md.Len() == 0 {
		return "", fmt.Errorf("未找到工作表数据")
	}
	return md.String(), nil
}

// pdfToMarkdown 提取 PDF 文本（重用现有 pdf_extract 逻辑 + 分页支持）
func pdfToMarkdown(path string, pages string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	content := string(data)
	var texts []string

	// 解析 PDF 页数信息
	totalPages := 0
	pageRunes := []rune(content)
	for i := 0; i < len(pageRunes)-8; i++ {
		if string(pageRunes[i:i+8]) == "/Type /P" && i+14 <= len(pageRunes) && string(pageRunes[i+8:i+14]) == "age" {
			totalPages++
		}
	}
	if totalPages == 0 {
		totalPages = 1
	}

	// 提取 BT...ET 文本
	remaining := content
	pageNum := 1
	for {
		btIdx := strings.Index(remaining, "BT")
		if btIdx < 0 {
			break
		}
		remaining = remaining[btIdx+2:]
		etIdx := strings.Index(remaining, "ET")
		if etIdx < 0 {
			break
		}
		block := remaining[:etIdx]
		text := extractPDFText(block)
		if strings.TrimSpace(text) != "" {
			if pages != "" && !pageInRange(pageNum, pages) {
				pageNum++
				continue
			}
			texts = append(texts, text)
			pageNum++
		}
		remaining = remaining[etIdx+2:]
	}

	result := strings.TrimSpace(strings.Join(texts, "\n"))
	if result == "" {
		result = extractRawText(data)
	}
	if result == "" {
		// 文本提取失败 → 回退 OCR（扫描件 PDF）
		return ocrPDF(path, pages)
	}
	return result, nil
}

func pageInRange(page int, spec string) bool {
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			parts := strings.SplitN(part, "-", 2)
			start, end := 1, 9999
			if s, err := fmt.Sscanf(parts[0], "%d", &start); err != nil || s != 1 {
				continue
			}
			if len(parts) > 1 {
				if s, err := fmt.Sscanf(parts[1], "%d", &end); err != nil || s != 1 {
					end = start
				}
			}
			if page >= start && page <= end {
				return true
			}
		} else {
			var pn int
			if _, err := fmt.Sscanf(part, "%d", &pn); err == nil && pn == page {
				return true
			}
		}
	}
	return false
}

// ocrPDF 使用外部 OCR 引擎处理扫描件 PDF（pdftoppm → tesseract 流水线）
func ocrPDF(path string, pages string) (string, error) {
	// 检查外部工具是否可用
	tesseractPath, errT := exec.LookPath("tesseract")
	pdftoppmPath, errP := exec.LookPath("pdftoppm")
	if errT != nil || errP != nil {
		return "", fmt.Errorf("扫描件 PDF 需要 OCR 引擎，但未找到 tesseract 或 pdftoppm。" +
			"请安装：\n  - tesseract: https://github.com/tesseract-ocr/tesseract\n" +
			"  - poppler (pdftoppm): https://poppler.freedesktop.org\n\n" +
			"或者使用文本 PDF（非扫描件）")
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "gaea-ocr-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// pdftoppm: PDF → PNG（逐页）
	pngPrefix := filepath.Join(tmpDir, "page")
	args := []string{"-png", "-r", "300", path, pngPrefix}
	cmd := exec.Command(pdftoppmPath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("pdftoppm 执行失败: %w\n输出: %s", err, string(out))
	}

	// 收集生成的 PNG 文件并按页码排序
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return "", fmt.Errorf("读取临时目录失败: %w", err)
	}
	var pngFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".png") {
			pngFiles = append(pngFiles, filepath.Join(tmpDir, e.Name()))
		}
	}
	if len(pngFiles) == 0 {
		return "", fmt.Errorf("pdftoppm 未生成页面图片")
	}

	// 逐页 OCR
	var pageTexts []string
	pageNum := 1
	for _, pngPath := range pngFiles {
		if pages != "" && !pageInRange(pageNum, pages) {
			pageNum++
			continue
		}
		// tesseract: PNG → 文本
		cmd := exec.Command(tesseractPath, pngPath, "stdout", "-l", "chi_sim+eng", "--psm", "3")
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("tesseract OCR 第 %d 页失败: %w", pageNum, err)
		}
		text := strings.TrimSpace(string(out))
		if text != "" {
			pageTexts = append(pageTexts, text)
		}
		pageNum++
	}

	if len(pageTexts) == 0 {
		return "", fmt.Errorf("OCR 未能提取到任何文本")
	}
	result := strings.Join(pageTexts, "\n\n---\n\n")
	return fmt.Sprintf("（以下内容由 OCR 识别，可能存在误差）\n\n%s", result), nil
}
