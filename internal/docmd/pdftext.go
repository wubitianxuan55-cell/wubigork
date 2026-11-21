package docmd

// pdftext.go — PDF 逐页文本抽取（2026-11-21 反馈 P0-2 静态目录页码回填）。
// doc_render 管线 = soffice→PDF + poppler 渲染；pdftotext 与 pdftoppm 同属
// poppler 套件，同一套三级查找。页码回填两遍法用它定位每个标题的渲染实测页。

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// pdftotextLookup 抽取前置查找 seam（与 pdftoppmLookup 同款：包级变量以便
// 测试堵住路径；生产走 findPdftotext 三级查找）。
var pdftotextLookup = findPdftotext

// findPdftotext 探测可用的 pdftotext（与 findPdftoppm 同目录同三级查找：
// GAEA_PDFTOTEXT → PATH .exe → codex-runtimes poppler）。
func findPdftotext() string {
	if p := os.Getenv("GAEA_PDFTOTEXT"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("pdftotext"); err == nil && strings.EqualFold(filepath.Ext(p), ".exe") {
		return p
	}
	base := filepath.Join(os.Getenv("USERPROFILE"), ".cache", "codex-runtimes")
	matches, _ := filepath.Glob(filepath.Join(base, "*", "dependencies", "native", "poppler", "Library", "bin", "pdftotext.exe"))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// PDFPageTexts 抽取 PDF 每页文本（pdftotext stdout 以 \f 分页）。返回切片
// 下标即 1-based 页码-1；文本只做空白归一以外的原样保留（匹配由调用方决定
// 归一口径）。
func PDFPageTexts(pdfPath string) ([]string, error) {
	exe := pdftotextLookup()
	if exe == "" {
		return nil, fmt.Errorf("未找到 pdftotext（poppler 套件，与 pdftoppm 同目录）；静态目录页码回填需要它")
	}
	out, err := exec.Command(exe, "-enc", "UTF-8", pdfPath, "-").Output()
	if err != nil {
		return nil, fmt.Errorf("pdftotext 执行失败: %w", err)
	}
	pages := strings.Split(string(out), "\f")
	if n := len(pages); n > 0 && strings.TrimSpace(pages[n-1]) == "" {
		pages = pages[:n-1] // 尾部孤立 \f 不算一页
	}
	return pages, nil
}
