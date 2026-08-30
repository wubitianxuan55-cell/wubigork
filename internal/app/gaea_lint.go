package app

// GaeaDocumentLint 对工作区文档做中文规范体检（v4.1c 红头第一刀 → v4.6.1
// 规范包机制化）：md/txt 直接读文本，docx 经 docmd 转 markdown 后校验；
// 检查器可插拔（standard.Registry：GB/T 9704 红头要素 + 造价工程表式）。
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/office/docmd"
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
	return standard.LintDocument(rel, head, body), nil
}
