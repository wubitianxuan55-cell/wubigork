// Package fileindex 工作区文件语义索引：扫描可提取文本的文档
// （md/txt/csv/docx/xlsx/pdf），提取正文供 bge-m3 向量化（复用
// semantic_vectors，kind=file，id=相对路径），实现资料语义检索。
package fileindex

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/semantic"
	"github.com/gaea/gaea/internal/office/docmd"
)

const (
	// MaxFiles 单次索引的文件数上限（防误扫超大工作区）。
	MaxFiles = 300
	// MaxFileBytes 超过该体积的文件跳过（不索引）。
	MaxFileBytes = 2 << 20
	// MaxIndexChars 向量化文本上限（bge-m3 上限内截断）。
	MaxIndexChars = 20000
)

// FileDoc 是一个可索引文件及其提取文本。
type FileDoc struct {
	Path string // 工作区相对路径（/ 分隔）
	Text string
}

// skipDirs 扫描时跳过的目录（内部状态/构建产物/依赖）。
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "dist": true, "build": true,
	".venv": true, "vendor": true, ".gaea": true, "releases": true,
}

// Supported 报告扩展名是否支持文本提取。
func Supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".txt", ".text", ".csv", ".tsv", ".json", ".xml", ".html", ".htm",
		".docx", ".xlsx", ".xlsm", ".pdf", ".et", ".ods", ".pptx":
		return true
	default:
		return false
	}
}

// Scan 遍历 root（工作区）收集支持的文本文件；返回文档列表与跳过数量。
func Scan(root string) ([]FileDoc, int, error) {
	var docs []FileDoc
	skipped := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if len(docs) >= MaxFiles {
			skipped++
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		if !Supported(path) {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil || info.Size() > MaxFileBytes {
			skipped++
			return nil
		}
		text, xerr := Extract(path)
		if xerr != nil || strings.TrimSpace(text) == "" {
			skipped++
			return nil
		}
		docs = append(docs, FileDoc{Path: filepath.ToSlash(rel), Text: text})
		return nil
	})
	return docs, skipped, err
}

// Extract 提取文件正文（docx/xlsx/pdf 走 docmd；其余直接读取），超长截断。
func Extract(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx", ".xlsx", ".xlsm", ".pdf", ".et", ".ods":
		md, _, _, err := docmd.ConvertLimit(path, "", 30)
		if err != nil {
			return "", err
		}
		return capRunes(md, MaxIndexChars), nil
	case ".pptx":
		text, err := extractPPTXText(path)
		if err != nil {
			return "", err
		}
		return capRunes(text, MaxIndexChars), nil
	default:
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return capRunes(string(b), MaxIndexChars), nil
	}
}

// extractPPTXText 从 pptx（zip + slide XML）提取 <a:t> 文本，无第三方依赖。
func extractPPTXText(path string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	var parts []string
	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		if !strings.HasPrefix(name, "ppt/slides/slide") || !strings.HasSuffix(name, ".xml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		text, terr := slideXMLText(rc)
		_ = rc.Close()
		if terr == nil && strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, "\n"), nil
}

// slideXMLText 提取 slide XML 中所有 <a:t> 元素的文本。
func slideXMLText(r io.Reader) (string, error) {
	dec := xml.NewDecoder(r)
	var b strings.Builder
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "t" {
			continue
		}
		// 读取直到 </t>
		for {
			inner, ierr := dec.Token()
			if ierr != nil {
				break
			}
			if end, ok := inner.(xml.EndElement); ok && end.Name.Local == "t" {
				break
			}
			if cd, ok := inner.(xml.CharData); ok {
				b.Write(cd)
			}
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

func capRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max])
}

// Doc 把 FileDoc 转为语义索引文档（text 同时用于向量化与命中预览，
// 额外截短预览避免 IPC/上下文失控）。
func Doc(f FileDoc) semantic.Doc {
	preview := capRunes(f.Text, 2000)
	return semantic.Doc{ID: f.Path, Text: preview}
}
