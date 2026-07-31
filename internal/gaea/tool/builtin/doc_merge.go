package builtin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wubigork/wubigork/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(docMerge{}) }

type docMerge struct{}

func (docMerge) Name() string { return "doc_merge" }

func (docMerge) Description() string {
	return "合并多个 Word (.docx) 文档为一个文档。将多个 docx 文件的正文内容按序合并，保留格式。适合将多份报告片段组装为完整报告。"
}

func (docMerge) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "files":{"type":"array","items":{"type":"string"},"description":"待合并的 docx 文件路径列表（至少2个）"},
  "output":{"type":"string","description":"输出文件路径（.docx）"},
  "add_page_breaks":{"type":"boolean","description":"每个文档后是否加分页符","default":false}
},
"required":["files","output"]
}`)
}

func (docMerge) ReadOnly() bool { return false }

func (docMerge) CompactDescription() string     { return compactDesc["doc_merge"] }
func (docMerge) CompactSchema() json.RawMessage { return compactSchema["doc_merge"] }

func (docMerge) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Files         []string `json:"files"`
		Output        string   `json:"output"`
		AddPageBreaks bool     `json:"add_page_breaks,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if len(p.Files) < 2 {
		return "", fmt.Errorf("至少需要2个文件进行合并")
	}
	if p.Output == "" {
		return "", fmt.Errorf("output 不能为空")
	}

	// 读取每个 docx 的 document.xml
	type docEntry struct {
		rels   map[string]string
		docXML []byte
	}
	var entries []docEntry

	for _, f := range p.Files {
		data, err := os.ReadFile(f)
		if err != nil {
			return "", fmt.Errorf("读取 %s 失败: %w", f, err)
		}

		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return "", fmt.Errorf("%s 不是有效的 docx: %w", f, err)
		}

		entry := docEntry{rels: make(map[string]string)}
		for _, zf := range zr.File {
			if zf.Name == "word/document.xml" {
				rc, _ := zf.Open()
				entry.docXML, _ = io.ReadAll(rc)
				rc.Close()
			}
		}
		if entry.docXML == nil {
			return "", fmt.Errorf("%s 中未找到 word/document.xml", f)
		}
		entries = append(entries, entry)
	}

	// 以第一个文档为基准，追加其他文档的 body 子元素
	baseDoc := entries[0].docXML
	baseStr := string(baseDoc)

	// 找到 </w:body> 位置
	bodyEnd := strings.Index(baseStr, "</w:body>")
	if bodyEnd < 0 {
		return "", fmt.Errorf("第一个文档格式无效（未找到 body 结束标签）")
	}

	// 构建合并后的 XML
	var merged bytes.Buffer
	merged.WriteString(baseStr[:bodyEnd])

	for i := 1; i < len(entries); i++ {
		// 提取 body 内容（去掉 <w:body> 和 </w:body>）
		docStr := string(entries[i].docXML)
		bodyStart := strings.Index(docStr, "<w:body>")
		bodyEnd2 := strings.LastIndex(docStr, "</w:body>")
		if bodyStart < 0 || bodyEnd2 < 0 {
			continue
		}
		inner := docStr[bodyStart+8 : bodyEnd2]

		if p.AddPageBreaks {
			merged.WriteString(`<w:p><w:r><w:br w:type="page"/></w:r></w:p>`)
		}
		merged.WriteString(inner)
	}
	merged.WriteString("</w:body></w:document>")

	// 读取第一个文档的 ZIP 结构，替换 document.xml
	srcData, _ := os.ReadFile(p.Files[0])
	zr, _ := zip.NewReader(bytes.NewReader(srcData), int64(len(srcData)))

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, zf := range zr.File {
		rc, _ := zf.Open()
		content, _ := io.ReadAll(rc)
		rc.Close()

		w, _ := zw.Create(zf.Name)
		if zf.Name == "word/document.xml" {
			w.Write(merged.Bytes())
		} else {
			w.Write(content)
		}
	}
	zw.Close()

	if err := os.WriteFile(p.Output, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("写入输出文件失败: %w", err)
	}

	return tool.WrapText(fmt.Sprintf("✅ 已合并 %d 个文档为: %s", len(p.Files), p.Output)), nil
}

// 保留 xml 包导入（用于未来可能的扩展）
