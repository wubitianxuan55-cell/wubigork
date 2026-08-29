package app

// GaeaDocumentLint 对工作区文档做 v4.1c 中文规范体检（GB/T 9704 红头要素
// lint 第一刀）：md/txt 直接读文本，docx 经 docmd 转 markdown 后校验。
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
	return standard.LintText(rel, head, body), nil
}
