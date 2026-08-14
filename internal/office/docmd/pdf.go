package docmd

// pdf.go — PDF 文本提取与页数/分页解析实现（docmd.go 拆分）。
// 职责：页对象计数与页码归类（countPDFPages/findPageType/pdfPageTexts）、
// BT..ET 文本提取与十六进制串解码（extractPDFText/decodePDFHex）、原始文本
// 兜底（extractRawText）与非文本流剔除（stripNonTextStreams）。
// 扫描件 OCR 回退见 ocr.go；页码范围规格工具见 pagespec.go。

import (
	"bytes"
	"encoding/hex"
	"os"
	"strings"
	"unicode/utf16"
)

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

	// 解析 PDF 页数信息：/Type /Page 每出现一次即一页。精确匹配排除 /Type /Pages
	//（页树数组对象声明），避免总页数恒多 ≥1；压缩对象流不在本实现支持范围内，
	// 与文本提取（仅未压缩文本流）保持一致，不额外解流。
	totalPages := countPDFPages(content)
	if totalPages == 0 {
		totalPages = 1
	}

	// 页数上限保护：把 pages 规格收敛到 maxPages 内（空规格 → 1-maxPages），
	// 超限部分不再进入正文；OCR 路径用同一规格只渲染需要的页。
	effSpec, truncated, err := capPageSpec(pages, maxPages, totalPages)
	if err != nil {
		return "", totalPages, false, err
	}

	// 提取 BT...ET 文本，按 /Type /Page 页对象归类页码：页码由页对象决定，
	// 不再由 BT 块自增（页内可有多个 BT..ET 块，BT 与页对象也非一一对应），
	// 页范围过滤因此不会错位。
	pageTexts := pdfPageTexts(content)
	var texts []string
	for pageIdx, text := range pageTexts {
		pageNum := pageIdx + 1
		if effSpec != "" && !pageInRange(pageNum, effSpec) {
			continue
		}
		if strings.TrimSpace(text) != "" {
			texts = append(texts, text)
		}
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

// countPDFPages 统计 PDF 页对象数：/Type /Page 精确匹配（排除 /Type /Pages 数组
// 对象声明，避免总页数恒多 ≥1）。只统计未压缩对象字典；压缩对象流（ObjStm）不在
// 本实现支持范围内，与文本提取（仅未压缩文本流）保持一致。
func countPDFPages(content string) int {
	n := 0
	for pos := 0; ; {
		i := findPageType(content[pos:])
		if i < 0 {
			break
		}
		n++
		pos += i + 1
	}
	return n
}

// findPageType 返回 s 中下一个页对象声明 /Type /Page 的位置（-1 表示没有）。
// 精确匹配规则：
//   - /Type 必须独立成 name（前面不是字母/数字/#，避免 /FontType 之类更长 name 的子串）
//   - 后面允许任意空白或直接跟 /Page（兼容 /Type/Page 无空格写法）
//   - /Page 后面不能是 name 字符（排除 /Pages、/PageMode 等更长 name）
func findPageType(s string) int {
	pos := 0
	for pos < len(s) {
		i := strings.Index(s[pos:], "/Type")
		if i < 0 {
			return -1
		}
		i += pos
		if i > 0 && isPDFNameChar(s[i-1]) {
			pos = i + len("/Type")
			continue
		}
		k := i + len("/Type")
		for k < len(s) && isPDFSpace(s[k]) {
			k++
		}
		if strings.HasPrefix(s[k:], "/Page") {
			after := k + len("/Page")
			if after >= len(s) || !isPDFNameChar(s[after]) {
				return i
			}
		}
		pos = i + len("/Type")
	}
	return -1
}

// isPDFNameChar 判断字节是否为 PDF name 字符（字母/数字/# 十六进制转义符）。
// 用于排除 /Type、/Page 作为更长 name（如 /Pages、/PageMode）子串的情况。
func isPDFNameChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '#'
}

// isPDFSpace 判断字节是否为 PDF 空白字符。
func isPDFSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f'
}

// pdfPageTexts 按页对象提取文本：扫描时每遇到一个 /Type /Page 声明就推进当前页码，
// 其后出现的 BT..ET 块归入该页（页码由页对象决定，不再由 BT 块自增）。返回切片
// 下标 = 页码-1；无文本的页对应空串；未声明页对象时文本归入第 1 页。
func pdfPageTexts(content string) []string {
	pages := make([]string, 0, 16)
	curPage := -1
	pos := 0
	for {
		relBT := strings.Index(content[pos:], "BT")
		relPG := findPageType(content[pos:])
		// 页对象先于 BT 出现（或没有 BT）→ 推进页码后继续。
		if relPG >= 0 && (relBT < 0 || relPG < relBT) {
			curPage++
			pos += relPG + 1
			continue
		}
		if relBT < 0 {
			break
		}
		afterBT := pos + relBT + len("BT")
		etIdx := strings.Index(content[afterBT:], "ET")
		if etIdx < 0 {
			break
		}
		etPos := afterBT + etIdx
		text := extractPDFText(content[afterBT:etPos])
		if strings.TrimSpace(text) != "" {
			if curPage < 0 {
				curPage = 0 // 页对象缺失 → 文本归入第 1 页
			}
			for len(pages) <= curPage {
				pages = append(pages, "")
			}
			if pages[curPage] == "" {
				pages[curPage] = text
			} else {
				pages[curPage] += "\n" + text
			}
		}
		pos = etPos + len("ET")
	}
	return pages
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
