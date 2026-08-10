// Package docmd converts office documents (docx/xlsx/pdf) to Markdown.
// Shared by the format_convert agent tool and the desktop file preview panel.
package docmd

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// DefaultMaxPDFPages caps PDF conversion/preview at this many pages per call
// (mirrors the industry norm: Gemini/Kimi/MinerU cap PDFs at 500-1000 pages
// and stream or chunk the rest). 0 disables the cap.
const DefaultMaxPDFPages = 500

// Convert renders a docx/xlsx/pdf file as Markdown. pages only applies to PDFs
// ("1-5" or "1,3,5"); unsupported extensions return an error.
func Convert(path, pages string) (string, error) {
	md, _, _, err := ConvertLimit(path, pages, 0)
	return md, err
}

// ConvertLimit is like Convert but caps PDF output at maxPages pages
// (maxPages <= 0 disables the cap). It returns the markdown, the PDF's total
// page count (0 for non-PDFs), whether the cap dropped pages, and any error.
// Callers use the total/truncated values to surface an honest "已截断" notice.
func ConvertLimit(path, pages string, maxPages int) (md string, total int, truncated bool, err error) {
	return ConvertLimitProgress(path, pages, maxPages, nil)
}

// ConvertLimitProgress is ConvertLimit with an optional per-page progress
// callback (done, total) fired while OCR-ing a scanned PDF page-by-page.
// nil disables callbacks; the OCR loop never fires one for in-memory text PDFs.
func ConvertLimitProgress(path, pages string, maxPages int, progress func(done, total int)) (md string, total int, truncated bool, err error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx", ".doc":
		md, err := docxToMarkdown(path)
		return md, 0, false, err
	case ".xlsx", ".xls":
		md, err := xlsxToMarkdown(path)
		return md, 0, false, err
	case ".pdf":
		return pdfToMarkdownLimit(path, pages, maxPages, progress)
	default:
		return "", 0, false, fmt.Errorf("不支持的文件格式: %s（支持 .docx/.xlsx/.pdf）", filepath.Ext(path))
	}
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

// pdfToMarkdown 提取 PDF 文本（含分页支持与 OCR 扫描件回退）。
func pdfToMarkdown(path string, pages string) (string, error) {
	md, _, _, err := pdfToMarkdownLimit(path, pages, 0, nil)
	return md, err
}

// pdfToMarkdownLimit 提取 PDF 文本；maxPages > 0 时按页数上限截断（配合扫描件
// 分页 OCR，避免整本超大 PDF 一次渲染/识别）。返回 markdown、总页数、是否被
// 上限截断、错误。progress 在逐页 OCR 时回调 (done, total)。
func pdfToMarkdownLimit(path, pages string, maxPages int, progress func(done, total int)) (string, int, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, false, err
	}

	content := string(data)
	// 剔除压缩流/图像/ICC 等非文本 stream 块，避免二进制里的假 BT/ET 与可打印垃圾
	// 混入正文（真实文本流含 BT，会被保留）。
	content = stripNonTextStreams(content)
	var texts []string

	// 解析 PDF 页数信息：/Type /Page 每出现一次即一页。
	totalPages := strings.Count(content, "/Type /Page")
	if totalPages == 0 {
		totalPages = 1
	}

	// 页数上限保护：把 pages 规格收敛到 maxPages 内（空规格 → 1-maxPages），
	// 超限部分不再进入正文；OCR 路径用同一规格只渲染需要的页。
	effSpec, truncated, err := capPageSpec(pages, maxPages, totalPages)
	if err != nil {
		return "", totalPages, false, err
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
			if effSpec != "" && !pageInRange(pageNum, effSpec) {
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
		// 文本提取失败 → 回退 OCR（扫描件 PDF），只渲染 effSpec 覆盖的页
		first, last := pageBounds(effSpec, totalPages)
		md, ocrErr := ocrPDFRange(path, effSpec, first, last, totalPages, progress)
		if ocrErr != nil {
			return "", totalPages, false, ocrErr
		}
		return md, totalPages, truncated, nil
	}
	return result, totalPages, truncated, nil
}

// capPageSpec 应用 maxPages 上限（<=0 不限）到页码范围规格，返回收敛后的规格
// 与是否截断。给定规格整体超出上限时返回错误，避免静默输出空文档。
func capPageSpec(spec string, maxPages, total int) (string, bool, error) {
	if maxPages <= 0 {
		return spec, false, nil
	}
	last := total
	if spec != "" {
		if f, l, ok := parsePageSpecBounds(spec); ok {
			last = l
			if f > maxPages {
				return "", false, fmt.Errorf("请求页码超出转换上限（最大 %d 页）", maxPages)
			}
		}
	}
	if last <= maxPages {
		return spec, false, nil
	}
	if spec == "" {
		return fmt.Sprintf("1-%d", maxPages), true, nil
	}
	return clampPageSpec(spec, maxPages), true, nil
}

// parsePageSpecBounds 解析 "1-5"/"1,3,5"/"3" 得到覆盖的 (first,last)。
func parsePageSpecBounds(spec string) (first, last int, ok bool) {
	first, last = 1<<30, 0
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			parts := strings.SplitN(part, "-", 2)
			var s, e int
			if _, err := fmt.Sscanf(parts[0], "%d", &s); err != nil {
				continue
			}
			e = s
			if len(parts) > 1 {
				if _, err := fmt.Sscanf(parts[1], "%d", &e); err != nil {
					e = s
				}
			}
			if s < first {
				first = s
			}
			if e > last {
				last = e
			}
			ok = true
		} else {
			var p int
			if _, err := fmt.Sscanf(part, "%d", &p); err != nil {
				continue
			}
			if p < first {
				first = p
			}
			if p > last {
				last = p
			}
			ok = true
		}
	}
	if !ok {
		return 1, 0, false
	}
	return first, last, true
}

// clampPageSpec 把 "1-3,7-9,11" 这类规格裁到 max 页以内，丢弃越界段。
func clampPageSpec(spec string, max int) string {
	var parts []string
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			ps := strings.SplitN(part, "-", 2)
			var s, e int
			if _, err := fmt.Sscanf(ps[0], "%d", &s); err != nil {
				continue
			}
			e = s
			if len(ps) > 1 {
				if _, err := fmt.Sscanf(ps[1], "%d", &e); err != nil {
					e = s
				}
			}
			if s > max {
				continue
			}
			if e > max {
				e = max
			}
			if s == e {
				parts = append(parts, fmt.Sprintf("%d", s))
			} else {
				parts = append(parts, fmt.Sprintf("%d-%d", s, e))
			}
		} else {
			var p int
			if _, err := fmt.Sscanf(part, "%d", &p); err != nil {
				continue
			}
			if p <= max {
				parts = append(parts, fmt.Sprintf("%d", p))
			}
		}
	}
	return strings.Join(parts, ",")
}

// pageBounds 返回规格覆盖的 (first,last) 页边界（OCR 渲染范围用）。
func pageBounds(spec string, total int) (int, int) {
	if spec == "" {
		return 1, total
	}
	if f, l, ok := parsePageSpecBounds(spec); ok {
		return f, l
	}
	return 1, total
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

// ovisOCRPrompt 是 OvisOCR2 文档解析的固定提示词（与 pdf 技能保持一致）。
const ovisOCRPrompt = "Extract all readable content from the image in natural human reading order and output the result as a single Markdown document. Preserve the original text without translation."

// ovisServerBase 返回常驻 OvisOCR2 llama-server 的 base URL；未安装/无法拉起时返回 ""，
// 调用方退回 tesseract。优先复用已在跑的实例（pdf 技能可能已拉起），否则按需静默拉起一次。
func ovisServerBase() string {
	base := strings.TrimRight(os.Getenv("GAEA_OCR_URL"), "/")
	if base == "" {
		port := os.Getenv("GAEA_OCR_PORT")
		if port == "" {
			port = "8137"
		}
		base = "http://127.0.0.1:" + port
	}
	client := &http.Client{Timeout: 3 * time.Second}
	if ovisServerHealthy(client, base) {
		return base
	}
	if startOvisServer(client, base) {
		return base
	}
	return ""
}

func ovisServerHealthy(c *http.Client, base string) bool {
	resp, err := c.Get(base + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return strings.Contains(string(body), "ok")
}

// startOvisServer 按 GAEA_OCR_* 环境变量或默认 C:\AI\gaea-ocr 拉起 llama-server
// （隐藏窗口，Vulkan），等待就绪（≤60s）。失败返回 false。
func startOvisServer(c *http.Client, base string) bool {
	dir := os.Getenv("GAEA_OCR_DIR")
	if dir == "" {
		dir = `C:\AI\gaea-ocr`
	}
	exe := os.Getenv("GAEA_OCR_LLAMA")
	if exe == "" {
		exe = filepath.Join(dir, "llama", "llama-server.exe")
	}
	model := os.Getenv("GAEA_OCR_MODEL")
	if model == "" {
		model = filepath.Join(dir, "models", "OvisOCR2-Q5_K_M.gguf")
	}
	mmproj := os.Getenv("GAEA_OCR_MMPROJ")
	if mmproj == "" {
		mmproj = filepath.Join(dir, "models", "mmproj-F16.gguf")
	}
	for _, p := range []string{exe, model, mmproj} {
		if _, err := os.Stat(p); err != nil {
			return false
		}
	}
	port := os.Getenv("GAEA_OCR_PORT")
	if port == "" {
		port = "8137"
	}
	var logf *os.File
	if f, err := os.OpenFile(filepath.Join(dir, "llama-server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		logf = f
	}
	cmd := exec.Command(exe, "-m", model, "--mmproj", mmproj, "--port", port,
		"-c", "8192", "-ngl", "99", "--jinja", "--host", "127.0.0.1")
	if logf != nil {
		cmd.Stdout = logf
		cmd.Stderr = logf
	}
	proc.HideWindow(cmd)
	if err := cmd.Start(); err != nil {
		if logf != nil {
			logf.Close()
		}
		return false
	}
	go func() { _ = cmd.Wait() }()
	if logf != nil {
		logf.Close() // 子进程已持有文件句柄，父进程可立即释放
	}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(time.Second)
		if ovisServerHealthy(c, base) {
			return true
		}
	}
	return false
}

// ovisPageOCR 把一页 PNG 发给常驻 OvisOCR2 服务，返回识别文本。
func ovisPageOCR(base, pngPath string) (string, error) {
	data, err := os.ReadFile(pngPath)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"model": "ovis",
		"messages": []map[string]any{{
			"role": "user",
			"content": []map[string]any{
				{"type": "text", "text": ovisOCRPrompt},
				{"type": "image_url", "image_url": map[string]string{
					"url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(data),
				}},
			},
		}},
		"temperature": 0.0,
		"max_tokens":  1024,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Post(base+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OvisOCR2 服务返回 %d", resp.StatusCode)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("OvisOCR2 无返回")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

// OCRImageText 识别单张图片中的文字（常驻 OvisOCR2 服务）。服务未安装/无法拉起时
// 返回明确错误，方便上层提示安装路径。
func OCRImageText(path string) (string, error) {
	base := ovisServerBase()
	if base == "" {
		return "", fmt.Errorf("OvisOCR2 本地 OCR 不可用（未安装或服务无法拉起），请检查 C:\\AI\\gaea-ocr 或 GAEA_OCR_DIR")
	}
	return ovisPageOCR(base, path)
}

// ocrPDF 处理扫描件 PDF：本地 OvisOCR2（常驻 llama-server）优先，
// 未安装/不可用时退回 tesseract（pdftoppm → tesseract 流水线）。
func ocrPDFRange(path, pages string, first, last, total int, progress func(done, totalN int)) (string, error) {
	pdftoppmPath := findPdftoppm()
	if pdftoppmPath == "" {
		return "", fmt.Errorf("扫描件 PDF 需要 poppler 渲染（pdftoppm），但未找到。" +
			"请安装 poppler：https://poppler.freedesktop.org\n\n或者使用文本 PDF（非扫描件）")
	}
	ovisBase := ovisServerBase() // 可能为空 → 退回 tesseract
	tesseractPath, _ := exec.LookPath("tesseract")
	if ovisBase == "" && tesseractPath == "" {
		return "", fmt.Errorf("扫描件 PDF 需要 OCR 引擎，但未找到 OvisOCR2 或 tesseract。" +
			"请安装其一：\n  - OvisOCR2（本地推荐，见 pdf 技能：C:\\AI\\gaea-ocr）\n" +
			"  - tesseract: https://github.com/tesseract-ocr/tesseract\n\n" +
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
	args := []string{"-png", "-r", "300"}
	if first != 1 {
		args = append(args, "-f", strconv.Itoa(first))
	}
	if last < total {
		args = append(args, "-l", strconv.Itoa(last))
	}
	args = append(args, path, pngPrefix)
	cmd := exec.Command(pdftoppmPath, args...)
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
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
	totalOCR := last - first + 1
	if totalOCR < 1 {
		totalOCR = 0
	}
	done := 0
	for _, pngPath := range pngFiles {
		if pages != "" && !pageInRange(pageNum, pages) {
			pageNum++
			continue
		}
		// OvisOCR2 常驻服务优先；不可用时退回 tesseract。
		text := ""
		if ovisBase != "" {
			if t, err := ovisPageOCR(ovisBase, pngPath); err == nil {
				text = t
			}
		}
		if text == "" && tesseractPath != "" {
			cmd := exec.Command(tesseractPath, pngPath, "stdout", "-l", "chi_sim+eng", "--psm", "3")
			proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
			out, err := cmd.Output()
			if err != nil {
				return "", fmt.Errorf("tesseract OCR 第 %d 页失败: %w", pageNum, err)
			}
			text = strings.TrimSpace(string(out))
		}
		if text != "" {
			pageTexts = append(pageTexts, text)
		}
		pageNum++
		done++
		if progress != nil && totalOCR > 0 {
			progress(done, totalOCR)
		}
	}

	if len(pageTexts) == 0 {
		return "", fmt.Errorf("OCR 未能提取到任何文本")
	}
	result := strings.Join(pageTexts, "\n\n---\n\n")
	return fmt.Sprintf("（以下内容由 OCR 识别，可能存在误差）\n\n%s", result), nil
}

// findPdftoppm 探测可用的 pdftoppm：优先 GAEA_PDFTOPM 显式路径，其次 PATH 里的
// pdftoppm.exe，再回退到 codex 运行时自带的 poppler（本机 PATH 里的 .cmd 包装器
// 指向不存在的路径，直接执行会失败）。
func findPdftoppm() string {
	if p := os.Getenv("GAEA_PDFTOPM"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("pdftoppm"); err == nil && strings.EqualFold(filepath.Ext(p), ".exe") {
		return p
	}
	base := filepath.Join(os.Getenv("USERPROFILE"), ".cache", "codex-runtimes")
	matches, _ := filepath.Glob(filepath.Join(base, "*", "dependencies", "native", "poppler", "Library", "bin", "pdftoppm.exe"))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// extractPDFText 从 BT...ET 块中提取文本
func extractPDFText(block string) string {
	var parts []string

	// 处理括号文本: (text) Tj
	remaining := block
	for {
		// 查找 ( 后跟文本 )
		parenStart := -1
		for i, c := range remaining {
			if c == '(' {
				// 检查前面不是反斜杠
				if i == 0 || remaining[i-1] != '\\' {
					parenStart = i
					break
				}
			}
		}
		if parenStart < 0 {
			break
		}

		// 查找匹配的 )
		depth := 1
		parenEnd := -1
		for i := parenStart + 1; i < len(remaining); i++ {
			if remaining[i] == '\\' {
				i++ // skip escaped char
				continue
			}
			if remaining[i] == '(' {
				depth++
			} else if remaining[i] == ')' {
				depth--
				if depth == 0 {
					parenEnd = i
					break
				}
			}
		}
		if parenEnd < 0 {
			break
		}

		text := remaining[parenStart+1 : parenEnd]
		// 检查后面是否有 Tj 操作符
		tail := remaining[parenEnd+1:]
		tail = strings.TrimSpace(tail)
		if strings.HasPrefix(tail, "Tj") || strings.HasPrefix(tail, "'") || strings.HasPrefix(tail, "\"") {
			parts = append(parts, text)
		}

		remaining = remaining[parenEnd+1:]
	}

	// 处理 TJ 数组: [(text1) num (text2)] TJ
	remaining = block
	for {
		brStart := strings.Index(remaining, "[(")
		brStartHex := strings.Index(remaining, "[<")
		if brStart < 0 || (brStartHex >= 0 && brStartHex < brStart) {
			brStart = brStartHex
		}
		if brStart < 0 {
			break
		}
		brEnd := strings.Index(remaining[brStart:], "] TJ")
		if brEnd < 0 {
			break
		}
		arrContent := remaining[brStart+1 : brStart+brEnd]
		remaining = remaining[brStart+brEnd+4:]

		var arrParts []string
		for {
			op := strings.Index(arrContent, "(")
			hx := strings.IndexByte(arrContent, '<')
			if op < 0 && hx < 0 {
				break
			}
			if hx >= 0 && (op < 0 || hx < op) {
				ce := strings.IndexByte(arrContent[hx+1:], '>')
				if ce < 0 {
					break
				}
				arrParts = append(arrParts, decodePDFHex(arrContent[hx+1:hx+1+ce]))
				arrContent = arrContent[hx+1+ce+1:]
				continue
			}
			cp := strings.Index(arrContent[op+1:], ")")
			if cp < 0 {
				break
			}
			arrParts = append(arrParts, arrContent[op+1:op+1+cp])
			arrContent = arrContent[op+1+cp+1:]
		}
		if len(arrParts) > 0 {
			parts = append(parts, strings.Join(arrParts, ""))
		}
	}

	// 解码转义字符
	var decoded []string
	for _, p := range parts {
		p = strings.ReplaceAll(p, "\\(", "(")
		p = strings.ReplaceAll(p, "\\)", ")")
		p = strings.ReplaceAll(p, "\\n", "\n")
		p = strings.ReplaceAll(p, "\\r", "\r")
		p = strings.ReplaceAll(p, "\\\\", "\\")
		decoded = append(decoded, p)
	}

	return strings.Join(decoded, " ")
}

// decodePDFHex 解析 PDF 十六进制字符串 <...>：CJK 文本多为 UTF-16BE（常带 FEFF
// BOM），按 UTF-16BE 解码失败时退回原始字节（Latin-1 语义）。
func decodePDFHex(hexStr string) string {
	hexStr = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, hexStr)
	if hexStr == "" {
		return ""
	}
	if len(hexStr)%2 == 1 {
		hexStr += "0"
	}
	raw, err := hex.DecodeString(hexStr)
	if err != nil {
		return hexStr
	}
	if len(raw) >= 2 {
		// 带 FEFF BOM → 一定是 UTF-16BE；否则先按 Latin-1，含控制字符再试 UTF-16BE。
		if raw[0] == 0xFE && raw[1] == 0xFF {
			return string(utf16.Decode(toUint16BE(raw[2:])))
		}
		if !hasControlByte(string(raw)) {
			return string(raw)
		}
		u := utf16.Decode(toUint16BE(raw))
		if !hasReplacementRune(u) {
			return string(u)
		}
	}
	return string(raw)
}

func toUint16BE(raw []byte) []uint16 {
	out := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		out = append(out, uint16(raw[i])<<8|uint16(raw[i+1]))
	}
	return out
}

func hasReplacementRune(rs []rune) bool {
	for _, r := range rs {
		if r == '\uFFFD' {
			return true
		}
	}
	return false
}

func hasControlByte(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < 0x20 || b >= 0x7F && b <= 0x9F {
			return true
		}
	}
	return false
}

// extractRawText 从 PDF 二进制中提取可读文本
func extractRawText(data []byte) string {
	// 移除 stream 和 endstream 之间的二进制内容
	content := stripNonTextStreams(string(data))

	// 提取所有可打印ASCII和中文
	var buf bytes.Buffer
	runes := []rune(content)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r >= 32 && r <= 126 {
			buf.WriteRune(r)
		} else if r >= 0x4E00 && r <= 0x9FFF {
			buf.WriteRune(r)
		} else if r == '\n' || r == '\r' || r == '\t' {
			buf.WriteRune(r)
		}
	}

	text := buf.String()
	// 移除PDF关键行
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 跳过PDF内部指令
		if isPDFKeyword(line) || isPDFNoiseLine(line) {
			continue
		}
		lines = append(lines, line)
	}

	out := strings.Join(lines, "\n")
	if !meaningfulText(out) {
		// 只剩 PDF 结构垃圾（对象字典/trailer 等）说明是扫描件，返回空让上层走 OCR。
		return ""
	}
	return out
}

// stripNonTextStreams 移除 PDF 中所有 stream...endstream 块，仅保留包含 "BT"
// 的文本内容流（未压缩文本流由 BT/ET 提取器读取）。压缩流、图像流、ICC 配置文件等
// 二进制块一旦被剔除，就不会再污染原始文本提取，也不会产生假 BT/ET 命中。
func stripNonTextStreams(s string) string {
	var b strings.Builder
	pos := 0
	for pos < len(s) {
		i := strings.Index(s[pos:], "stream")
		if i < 0 {
			break
		}
		i += pos
		// 真正的 stream 关键字：前面最近的非空白字符是对象字典结尾 '>'，
		// 后面跟换行。图像/ICC 二进制里按字节出现的 "stream" 不满足该模式。
		pre := i
		for pre > 0 && (s[pre-1] == ' ' || s[pre-1] == '\t' || s[pre-1] == '\r' || s[pre-1] == '\n') {
			pre--
		}
		if pre == 0 || s[pre-1] != '>' {
			pos = i + len("stream")
			continue
		}
		bodyStart := i + len("stream")
		for bodyStart < len(s) && (s[bodyStart] == '\r' || s[bodyStart] == '\n') {
			bodyStart++
		}
		// 找独立的 endstream 关键字（后面是换行/空白/文件尾）。
		end := -1
		search := bodyStart
		for search < len(s) {
			j := strings.Index(s[search:], "endstream")
			if j < 0 {
				break
			}
			j += search
			after := j + len("endstream")
			if after >= len(s) || s[after] == '\r' || s[after] == '\n' || s[after] == ' ' || s[after] == '\t' {
				end = j
				break
			}
			search = j + len("endstream")
		}
		if end < 0 {
			break
		}
		if isTextStreamBody(s[bodyStart:end]) {
			// 文本内容流：整段保留给 BT/ET 提取
			b.WriteString(s[pos : end+len("endstream")])
			pos = end + len("endstream")
			continue
		}
		b.WriteString(s[pos:i])
		pos = end + len("endstream")
	}
	b.WriteString(s[pos:])
	return b.String()
}

// isTextStreamBody 判断流内容是否为文本内容流：含 BT/ET 文本块与 Tj/TJ 文本操作符，
// 且体积在合理范围内（排除大型图像/ICC 等二进制流）。
func isTextStreamBody(body string) bool {
	if len(body) > 1<<20 {
		return false
	}
	return strings.Contains(body, "BT") && strings.Contains(body, "ET") &&
		(strings.Contains(body, "Tj") || strings.Contains(body, "TJ"))
}

func isPDFKeyword(s string) bool {
	keywords := []string{"endstream", "stream", "obj", "endobj", "xref", "trailer",
		"BT", "ET", "Tj", "TJ", "Td", "Tm", "cm", "Do", "gs", "rg", "RG", "k", "K",
		"w", "J", "j", "M", "d", "ri", "sh", "EI", "BDC", "BMC", "EMC", "MP", "DP"}
	for _, kw := range keywords {
		if s == kw || strings.HasPrefix(s, kw+" ") || strings.HasSuffix(s, " "+kw) {
			return true
		}
	}
	return false
}

// pdfNoisePrefixes 是 PDF 结构行的特征前缀：对象字典、交叉引用表、流声明等，
// 出现在扫描件 PDF 的二进制里，会被 extractRawText 误当文本。
var pdfNoisePrefixes = []string{
	"/Type", "/Subtype", "/Length", "/Filter", "/Width", "/Height",
	"/BitsPerComponent", "/ColorSpace", "/Root", "/Info", "/ID", "/Size",
	"/Pages", "/Count", "/MediaBox", "/Kids", "/Resources", "/Font",
	"/ProcSet", "/XObject", "/Image", "/DCTDecode", "/FlateDecode", "/Name",
	"/Parent", "/CropBox", "/Rotate", "/StructTreeRoot", "startxref", "%%EOF",
	"endstream", "stream", "trailer", "xref",
}

func isPDFNoiseLine(line string) bool {
	// PDF 对象字典 <</...>> 与交叉引用表偏移行（0000000172 00000 n）
	if strings.Contains(line, "<<") || strings.Contains(line, ">>") || isXrefEntry(line) {
		return true
	}
	for _, p := range pdfNoisePrefixes {
		if strings.HasPrefix(line, p) || strings.Contains(line, " "+p) {
			return true
		}
	}
	// 纯数字/十六进制偏移（xref 表行）也是结构噪声。
	if isDigitsOrHex(line) {
		return true
	}
	return false
}

func isXrefEntry(s string) bool {
	digits := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits++
			continue
		}
		if r == ' ' || r == '\t' || r == 'n' || r == 'f' {
			continue
		}
		return false
	}
	return digits >= 4
}

func isDigitsOrHex(s string) bool {
	if s == "" {
		return false
	}
	digits := 0
	for _, r := range s {
		if r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F' || r == 'x' || r == '<' || r == '>' {
			if r >= '0' && r <= '9' {
				digits++
			}
			continue
		}
		return false
	}
	return digits >= 4
}

// meaningfulText 判断提取结果是否包含真实可读内容（中文字符或成段英文），
// 避免把 PDF 结构/二进制垃圾当成正文。词按空白切分并要求是常规字母词
// （拒绝高熵符号串），只有真正成句的内容才能通过。
func meaningfulText(s string) bool {
	cjk := 0
	totalWords := 0
	proseLines := 0
	for _, line := range strings.Split(s, "\n") {
		for _, r := range line {
			if r >= 0x4E00 && r <= 0x9FFF {
				cjk++
			}
		}
		n := alphaWordCount(line)
		totalWords += n
		if n >= 2 {
			proseLines++
		}
	}
	return cjk >= 5 || (totalWords >= 8 && proseLines >= 2)
}

// alphaWordCount 统计一行里由空白分隔的常规字母词数量（词只含字母与少量标点，
// 至少 2 个字母），过滤 PDF 二进制噪声。
func alphaWordCount(line string) int {
	n := 0
	for _, tok := range strings.Fields(line) {
		alpha := 0
		ok := true
		for _, r := range tok {
			switch {
			case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z':
				alpha++
			case r == '-' || r == '\'' || r == '.' || r == ',' || r == ':' || r == '(' || r == ')':
				// 常规词内标点
			default:
				ok = false
			}
			if !ok {
				break
			}
		}
		if ok && alpha >= 2 {
			n++
		}
	}
	return n
}
