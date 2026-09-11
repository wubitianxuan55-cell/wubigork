package app

// GaeaDocumentLint 对工作区文档做中文规范体检（v4.1c 红头第一刀 → v4.6.1
// 规范包机制化）：md/txt 直接读文本，docx 经 docmd 转 markdown 后校验；
// 检查器可插拔（standard.Registry：GB/T 9704 红头要素 + 造价工程表式）。
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/office/docxedit"
	"github.com/gaea/gaea/internal/office/standard"
)

func (a *App) GaeaDocumentLint(rel string) (standard.LintReport, error) {
	if rel == "" {
		return standard.LintReport{}, fmt.Errorf("缺少文件路径")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	if _, err := os.Stat(path); err != nil {
		return standard.LintReport{}, fmt.Errorf("文件不存在：%s", rel)
	}
	ext := strings.ToLower(filepath.Ext(path))
	text := ""
	var layout *standard.DocxLayout
	switch ext {
	case ".md", ".markdown", ".txt":
		raw, err := os.ReadFile(path)
		if err != nil {
			return standard.LintReport{}, err
		}
		text = string(raw)
	case ".docx":
		md, err := docmd.Convert(path, "")
		if err != nil {
			return standard.LintReport{}, fmt.Errorf("docx 提取失败：%w", err)
		}
		text = md
		// v4.223 排版细则：提取主文档 XML 的排版事实；解析失败不阻断文本层
		// 检查（layout=nil 时排版包跳过，诚实口径=其余规范包照常体检）。
		if xmlData, err := docxedit.ReadDocumentXML(path); err == nil {
			if l, err := standard.ParseDocxLayout(xmlData); err == nil {
				layout = &l
			}
		}
	default:
		return standard.LintReport{}, fmt.Errorf("暂支持 md/txt/docx（当前 %s）", ext)
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	head := ""
	body := ""
	if len(lines) > 3 {
		head = strings.Join(lines[:3], "\n")
		body = strings.Join(lines[3:], "\n")
	} else {
		head = text
	}
	return standard.LintDocumentWithLayout(rel, head, body, layout), nil
}
