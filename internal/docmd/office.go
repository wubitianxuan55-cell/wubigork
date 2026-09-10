package docmd

// office.go — Office 文档（docx/xlsx）→ Markdown 转换实现（docmd.go 拆分）。
// 职责：解包 docx/xlsx（zip）并解析其 XML 为 Markdown，含标题样式、表格与
// 单元格文本提取（docxToMarkdown/xlsxToMarkdown 及其辅助函数）。
// 不含 PDF 解析（pdf.go）、OCR 编排（ocr.go）与页规格工具（pagespec.go）。

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

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
		// 单趟扫描取最早的段落/表格标签，避免每次迭代在整个剩余正文里重复搜索。
		tagIdx, tag := nextOpenTag(bodyContent[pos:], "w:tbl", "w:p")
		if tagIdx < 0 {
			break
		}
		if tag == "w:tbl" {
			absStart := pos + tagIdx
			tblEnd := strings.Index(bodyContent[absStart:], "</w:tbl>")
			if tblEnd < 0 {
				break
			}
			tblXML := bodyContent[absStart : absStart+tblEnd+8]
			tblMD := extractDocxTable(tblXML, &tblIdx)
			mdParts = append(mdParts, tblMD)
			pos = absStart + tblEnd + 8
		} else {
			absStart := pos + tagIdx
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

// nextOpenTag returns the earliest open tag whose name is one of tags
// (e.g. "w:p", "w:tbl"), or -1. A tag matches only when the name ends at
// '>', ' ', '/', '\t', '\n' or '\r', so <w:pPr> is never mistaken for <w:p>.
// A single left-to-right pass over '<' characters keeps this O(n) instead of
// O(n^2) on documents where the attribute form of a tag is rare or absent.
func nextOpenTag(s string, tags ...string) (int, string) {
	offset := 0
	for {
		i := strings.IndexByte(s[offset:], '<')
		if i < 0 {
			return -1, ""
		}
		abs := offset + i
		for _, tag := range tags {
			if strings.HasPrefix(s[abs+1:], tag) {
				after := abs + 1 + len(tag)
				if after >= len(s) {
					return abs, tag
				}
				switch s[after] {
				case '>', ' ', '/', '\t', '\n', '\r':
					return abs, tag
				}
			}
		}
		offset = abs + 1
	}
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
		tStart := findWt(remaining)
		if tStart < 0 {
			break
		}
		// 自闭合 <w:t/> 无文本，跳过
		if isSelfClosing(remaining[tStart:]) {
			remaining = remaining[tStart+4:]
			continue
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
		tStart := findWt(remaining)
		if tStart < 0 {
			break
		}
		if isSelfClosing(remaining[tStart:]) {
			remaining = remaining[tStart+4:]
			continue
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

// findWt locates a real <w:t> / <w:t ...> text tag, skipping sibling tags that
// share the "<w:t" prefix (w:tc, w:tbl, w:tabs, w:tr, ...). Returns -1 when
// none remains.
func findWt(s string) int {
	offset := 0
	for {
		i := strings.Index(s, "<w:t")
		if i < 0 {
			return -1
		}
		rest := s[i+4:]
		if len(rest) > 0 {
			c := rest[0]
			// tag name ends at '>' (open), ' ' (attrs), '/' (self-close),
			// or whitespace. Anything else (e.g. 'c' in w:tcPr, 'r' in w:tr)
			// means a sibling tag — skip past this '<' and keep scanning.
			if c == '>' || c == ' ' || c == '/' || c == '\t' || c == '\n' || c == '\r' {
				return offset + i
			}
		}
		offset += i + 1
		s = s[i+1:]
	}
}

// isSelfClosing reports whether s starts with a self-closing tag like <w:t/>.
func isSelfClosing(s string) bool {
	gt := strings.IndexByte(s, '>')
	if gt < 0 {
		return false
	}
	if gt < 4 {
		// s does not start with a full <w:t tag — not a text node; treat as
		// non-self-closing so the caller skips it safely.
		return false
	}
	inner := s[4:gt]
	return strings.HasSuffix(inner, "/")
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

	// 工作表文件按名建索引，避免每个 sheet 都全量扫描 zip 条目。
	sheetByName := make(map[string]*zip.File, len(r.File))
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			sheetByName[f.Name] = f
		}
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
		zf, ok := sheetByName[sheetFile]
		if !ok {
			break
		}
		rc, _ := zf.Open()
		sheetXML, _ := io.ReadAll(rc)
		rc.Close()

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
					Is   struct {
						T string `xml:"t"`
					} `xml:"is"`
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
				switch cell.Type {
				case "inlineStr":
					val = cell.Is.T
				case "s":
					if idx, err := strconv.Atoi(val); err == nil {
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
