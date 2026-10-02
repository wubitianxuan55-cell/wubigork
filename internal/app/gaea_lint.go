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
	"github.com/gaea/gaea/internal/gaea/wspath"
	"github.com/gaea/gaea/internal/office/standard"
)

func (a *App) GaeaDocumentLint(rel string) (standard.LintReport, error) {
	if rel == "" {
		return standard.LintReport{}, fmt.Errorf("缺少文件路径")
	}
	// 审计 2026-10-02 AP5-09 漏点修复：此前 rel 非绝对时直接 Join(工作区根, rel)，
	// 既无 Clean 也无包含性检查——`../../x` 被 Join 清洗成工作区外路径后，
	// os.Stat / os.ReadFile / docmd.Convert 全部作用于逃逸目标（工作区外文件
	// 可被读进体检报告）。现走唯一原语 wspath.ResolveRelWithin（归一后必须在
	// 工作区根内）。绝对路径分支**保留原样**：本绑定是设置面板「规范体检」里
	// 手输路径的读原语（OfficePanel 的 lintPath 输入框），用户可粘贴工作区外
	// 的合法绝对文档路径，收紧会直接咬到桌面用法。
	path := rel
	if !filepath.IsAbs(rel) {
		p, err := wspath.ResolveRelWithin(gaeaCwd(), rel)
		if err != nil {
			return standard.LintReport{}, fmt.Errorf("非法工作区相对路径: %s", rel)
		}
		path = p
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
