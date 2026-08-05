// Package proposal — 文档分页文本提取（供来源定位使用）
package proposal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carmel/gooxml/document"
	"github.com/ledongthuc/pdf"
)

// PageText 单页文本
type PageText struct {
	Page int    `json:"page"`
	Text string `json:"text"`
}

// ExtractPageText 提取文件分页文本。
// PDF 按页返回真实页码；DOCX/TXT 无可靠分页信息，统一返回 Page=0。
func ExtractPageText(filePath string) ([]PageText, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		return extractPdfPages(filePath)
	case ".docx":
		text, err := extractDocxText(filePath)
		if err != nil {
			return nil, err
		}
		return []PageText{{Page: 0, Text: text}}, nil
	default:
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return []PageText{{Page: 0, Text: string(data)}}, nil
	}
}

func extractPdfPages(filePath string) ([]PageText, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开 PDF 失败: %w", err)
	}
	defer f.Close()
	var out []PageText
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		out = append(out, PageText{Page: i, Text: text})
	}
	return out, nil
}

func extractDocxText(filePath string) (string, error) {
	doc, err := document.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 DOCX 失败: %w", err)
	}
	var sb strings.Builder
	for _, para := range doc.Paragraphs() {
		for _, run := range para.Runs() {
			sb.WriteString(run.Text())
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
