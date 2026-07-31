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
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(docxRead{}) }

type docxRead struct{}

func (docxRead) Name() string { return "docx_read" }

func (docxRead) Description() string {
	return "读取 Word (.docx) 文件文本内容。docx 本质是 ZIP 包，解析其 XML 提取正文段落文本。"
}

func (docxRead) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"docx文件路径"}
},
"required":["path"]
}`)
}

func (docxRead) ReadOnly() bool { return true }

func (docxRead) CompactDescription() string     { return compactDesc["docx_read"] }
func (docxRead) CompactSchema() json.RawMessage { return compactSchema["docx_read"] }

// docx XML 结构
type docxDocument struct {
	Body docxBody `xml:"body"`
}
type docxBody struct {
	Paragraphs []docxParagraph `xml:"p"`
}
type docxParagraph struct {
	Runs []docxRun `xml:"r"`
}
type docxRun struct {
	Text string `xml:"t"`
}

func (docxRead) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path 不能为空")
	}

	data, err := os.ReadFile(p.Path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("不是有效的 docx 文件: %w", err)
	}

	var docXML []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("打开 document.xml 失败: %w", err)
			}
			docXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", fmt.Errorf("读取 document.xml 失败: %w", err)
			}
			break
		}
	}
	if docXML == nil {
		return "", fmt.Errorf("未找到 word/document.xml（无效的 docx 文件）")
	}

	var doc docxDocument
	if err := xml.Unmarshal(docXML, &doc); err != nil {
		return "", fmt.Errorf("解析 document.xml 失败: %w", err)
	}

	var lines []string
	for _, p := range doc.Body.Paragraphs {
		var textParts []string
		for _, r := range p.Runs {
			textParts = append(textParts, r.Text)
		}
		line := strings.Join(textParts, "")
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	return tool.WrapText(strings.Join(lines, "\n")), nil
}

func init() { tool.RegisterBuiltin(docxWrite{}) }

type docxWrite struct{}

func (docxWrite) Name() string { return "docx_write" }

func (docxWrite) Description() string {
	return "创建 Word (.docx) 文件：输入 Markdown 文本，生成 docx 文档。支持标题（#/##/###）、无序/有序列表、表格（Markdown 管道语法）、加粗和斜体。"
}

func (docxWrite) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"输出文件路径"},
  "title":{"type":"string","description":"文档标题（居中大标题）"},
  "content":{"type":"string","description":"文档正文（Markdown 格式：## 标题、- 列表、| 表格 |）"}
},
"required":["path","content"]
}`)
}

func (docxWrite) ReadOnly() bool { return false }

func (docxWrite) CompactDescription() string     { return compactDesc["docx_write"] }
func (docxWrite) CompactSchema() json.RawMessage { return compactSchema["docx_write"] }

func (docxWrite) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path    string `json:"path"`
		Title   string `json:"title,omitempty"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Path == "" || p.Content == "" {
		return "", fmt.Errorf("path 和 content 不能为空")
	}

	var docXML bytes.Buffer
	docXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	docXML.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` + "\n")
	docXML.WriteString(`<w:body>` + "\n")

	// 标题
	if p.Title != "" {
		docXML.WriteString(`<w:p><w:pPr><w:pStyle w:val="Title"/><w:jc w:val="center"/></w:pPr>`)
		docXML.WriteString(fmt.Sprintf(`<w:r><w:t>%s</w:t></w:r>`, xmlEscape(p.Title)))
		docXML.WriteString(`</w:p>` + "\n")
	}

	lines := strings.Split(p.Content, "\n")
	inTable := false
	tableCols := 0

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			if inTable {
				docXML.WriteString(`</w:tbl>` + "\n")
				inTable = false
			}
			continue
		}

		// 检测表格行
		if strings.HasPrefix(line, "|") {
			cols := strings.Split(line, "|")
			var cells []string
			for _, c := range cols {
				c = strings.TrimSpace(c)
				if c != "" || len(cells) > 0 {
					cells = append(cells, c)
				}
			}
			// 分隔线行
			if len(cells) > 0 && strings.TrimLeft(cells[0], "-: ") == "" {
				continue
			}
			if !inTable {
				tableCols = len(cells)
				docXML.WriteString(`<w:tbl>` + "\n")
				docXML.WriteString(`<w:tblPr><w:tblW w:w="5000" w:type="pct"/><w:tblBorders>` +
					`<w:top w:val="single" w:sz="4" w:space="0" w:color="auto"/>` +
					`<w:bottom w:val="single" w:sz="4" w:space="0" w:color="auto"/>` +
					`<w:insideH w:val="single" w:sz="4" w:space="0" w:color="auto"/>` +
					`<w:insideV w:val="single" w:sz="4" w:space="0" w:color="auto"/>` +
					`</w:tblBorders></w:tblPr>` + "\n")
				for ci := 0; ci < tableCols; ci++ {
					colW := 9000 / tableCols
					docXML.WriteString(fmt.Sprintf(`<w:tblGrid><w:gridCol w:w="%d"/></w:tblGrid>`+"\n", colW))
				}
				inTable = true
			}
			docXML.WriteString(`<w:tr>` + "\n")
			for _, cell := range cells {
				docXML.WriteString(`<w:tc><w:p><w:r><w:t>`)
				docXML.WriteString(xmlEscape(cell))
				docXML.WriteString(`</w:t></w:r></w:p></w:tc>` + "\n")
			}
			docXML.WriteString(`</w:tr>` + "\n")
			continue
		}

		if inTable {
			docXML.WriteString(`</w:tbl>` + "\n")
			inTable = false
		}

		// 标题
		if strings.HasPrefix(line, "### ") {
			text := strings.TrimPrefix(line, "### ")
			docXML.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading3"/></w:pPr>`)
			docXML.WriteString(writeFormattedText(text))
			docXML.WriteString(`</w:p>` + "\n")
			continue
		}
		if strings.HasPrefix(line, "## ") {
			text := strings.TrimPrefix(line, "## ")
			docXML.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr>`)
			docXML.WriteString(writeFormattedText(text))
			docXML.WriteString(`</w:p>` + "\n")
			continue
		}
		if strings.HasPrefix(line, "# ") {
			text := strings.TrimPrefix(line, "# ")
			docXML.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr>`)
			docXML.WriteString(writeFormattedText(text))
			docXML.WriteString(`</w:p>` + "\n")
			continue
		}

		// 无序列表
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			text := strings.TrimPrefix(line, "- ")
			text = strings.TrimPrefix(text, "* ")
			docXML.WriteString(`<w:p><w:pPr><w:pStyle w:val="ListParagraph"/>` +
				`<w:ind w:left="720" w:hanging="360"/><w:numPr><w:numId w:val="1"/></w:numPr></w:pPr>`)
			docXML.WriteString(writeFormattedText(text))
			docXML.WriteString(`</w:p>` + "\n")
			continue
		}

		// 有序列表
		if matched := matchOrderedList(line); matched != "" {
			docXML.WriteString(`<w:p><w:pPr><w:pStyle w:val="ListParagraph"/>` +
				`<w:ind w:left="720" w:hanging="360"/><w:numPr><w:numId w:val="2"/></w:numPr></w:pPr>`)
			docXML.WriteString(writeFormattedText(matched))
			docXML.WriteString(`</w:p>` + "\n")
			continue
		}

		// 普通段落
		docXML.WriteString(`<w:p>`)
		docXML.WriteString(writeFormattedText(line))
		docXML.WriteString(`</w:p>` + "\n")
	}

	if inTable {
		docXML.WriteString(`</w:tbl>` + "\n")
	}

	docXML.WriteString(`</w:body>` + "\n")
	docXML.WriteString(`</w:document>` + "\n")

	// 创建 ZIP
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	writeZipEntry(zw, "[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
<Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>
</Types>`)

	writeZipEntry(zw, "_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)

	writeZipEntry(zw, "word/document.xml", docXML.String())
	writeZipEntry(zw, "word/styles.xml", docxStyles)
	writeZipEntry(zw, "word/numbering.xml", docxNumbering)

	zw.Close()

	if err := os.WriteFile(p.Path, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return tool.WrapText(fmt.Sprintf("已创建 docx 文件：%s", p.Path)), nil
}

// writeFormattedText 解析行内加粗 **text** 和斜体 *text*
func writeFormattedText(text string) string {
	if !strings.Contains(text, "**") && !strings.Contains(text, "*") {
		return fmt.Sprintf(`<w:r><w:t>%s</w:t></w:r>`, xmlEscape(text))
	}
	var b strings.Builder
	runes := []rune(text)
	i := 0
	for i < len(runes) {
		if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
			end := findClose(runes, i+2, []rune("**"))
			if end > i+2 {
				inner := string(runes[i+2 : end])
				b.WriteString(fmt.Sprintf(`<w:r><w:rPr><w:b/></w:rPr><w:t>%s</w:t></w:r>`, xmlEscape(inner)))
				i = end + 2
				continue
			}
		}
		if runes[i] == '*' {
			end := findClose(runes, i+1, []rune("*"))
			if end > i+1 && (end+1 >= len(runes) || runes[end+1] != '*') {
				inner := string(runes[i+1 : end])
				b.WriteString(fmt.Sprintf(`<w:r><w:rPr><w:i/></w:rPr><w:t>%s</w:t></w:r>`, xmlEscape(inner)))
				i = end + 1
				continue
			}
		}
		start := i
		for i < len(runes) && runes[i] != '*' {
			i++
		}
		if i > start {
			b.WriteString(fmt.Sprintf(`<w:r><w:t>%s</w:t></w:r>`, xmlEscape(string(runes[start:i]))))
		} else {
			i++
		}
	}
	return b.String()
}

func findClose(runes []rune, start int, delim []rune) int {
	for i := start; i < len(runes); i++ {
		if runes[i] == delim[0] {
			if len(delim) == 1 || (i+1 < len(runes) && runes[i+1] == delim[1]) {
				return i
			}
		}
	}
	return -1
}

func matchOrderedList(line string) string {
	if len(line) < 3 {
		return ""
	}
	runes := []rune(line)
	j := 0
	for j < len(runes) && runes[j] >= '0' && runes[j] <= '9' {
		j++
	}
	if j == 0 || j >= len(runes) {
		return ""
	}
	if runes[j] == '.' || runes[j] == ')' {
		if j+1 < len(runes) && runes[j+1] == ' ' {
			return string(runes[j+2:])
		}
	}
	return ""
}

// docxStyles 增强的 styles.xml（含 Heading3、ListParagraph）
const docxStyles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:pPr><w:jc w:val="center"/><w:spacing w:after="200"/></w:pPr><w:rPr><w:sz w:val="48"/><w:b/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:pPr><w:spacing w:before="360" w:after="120"/></w:pPr><w:rPr><w:sz w:val="32"/><w:b/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:pPr><w:spacing w:before="240" w:after="80"/></w:pPr><w:rPr><w:sz w:val="28"/><w:b/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading3"><w:name w:val="heading 3"/><w:pPr><w:spacing w:before="180" w:after="60"/></w:pPr><w:rPr><w:sz w:val="24"/><w:b/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="ListParagraph"><w:name w:val="List Paragraph"/><w:pPr><w:ind w:left="720"/></w:pPr><w:rPr><w:sz w:val="24"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Normal"><w:name w:val="Normal"/><w:rPr><w:sz w:val="24"/></w:rPr></w:style>
</w:styles>`

// docxNumbering 列表编号定义：numId=1 无序（bullet），numId=2 有序（decimal）
const docxNumbering = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:abstractNum w:abstractNumId="0">
  <w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="bullet"/><w:lvlText w:val="●"/><w:lvlJc w:val="left"/><w:pPr><w:ind w:left="720" w:hanging="360"/></w:pPr></w:lvl>
</w:abstractNum>
<w:abstractNum w:abstractNumId="1">
  <w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/><w:lvlJc w:val="left"/><w:pPr><w:ind w:left="720" w:hanging="360"/></w:pPr></w:lvl>
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>
<w:num w:numId="2"><w:abstractNumId w:val="1"/></w:num>
</w:numbering>`
