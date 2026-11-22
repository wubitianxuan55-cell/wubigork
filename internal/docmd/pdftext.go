package docmd

// pdftext.go — PDF 逐页文本抽取（2026-11-21 反馈 P0-2 静态目录页码回填）。
// doc_render 管线 = soffice→PDF + poppler 渲染；pdftotext 与 pdftoppm 同属
// poppler 套件，同一套三级查找。页码回填两遍法用它定位每个标题的渲染实测页。

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// PDFImageInfo pdfimages -list 的一行（版面机检 P1-6：图片实际插入尺寸与
// 有效 ppi——「图片糊不糊」「图占了多少版面」的确定性数据源）。
type PDFImageInfo struct {
	Page   int
	Width  int     // 像素宽
	Height int     // 像素高
	XPPI   float64 // 有效水平 ppi（插入尺寸折算）
	YPPI   float64
}

// pdfimagesLookup 同款三级查找 seam。
var pdfimagesLookup = findPdfimages

// findPdfimages 探测 pdfimages（poppler 套件，与 pdftoppm/pdftotext 同目录）。
func findPdfimages() string {
	if p := os.Getenv("GAEA_PDFIMAGES"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("pdfimages"); err == nil && strings.EqualFold(filepath.Ext(p), ".exe") {
		return p
	}
	base := filepath.Join(os.Getenv("USERPROFILE"), ".cache", "codex-runtimes")
	matches, _ := filepath.Glob(filepath.Join(base, "*", "dependencies", "native", "poppler", "Library", "bin", "pdfimages.exe"))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// PDFImagesInfo 列 PDF 内嵌图片（页/像素尺寸/有效 ppi）。pdfimages -list 输出
// 定宽表：page num type width height color comp bpc enc interp object ID x-ppi y-ppi size ratio——按空白切列取第 1/4/5/13/14 列（列数随版本稳定在这些位）。
func PDFImagesInfo(pdfPath string) ([]PDFImageInfo, error) {
	exe := pdfimagesLookup()
	if exe == "" {
		return nil, fmt.Errorf("未找到 pdfimages（poppler 套件，与 pdftoppm 同目录）；版面机检图片面需要它")
	}
	out, err := exec.Command(exe, "-list", pdfPath).Output()
	if err != nil {
		return nil, fmt.Errorf("pdfimages 执行失败: %w", err)
	}
	var imgs []PDFImageInfo
	for _, ln := range strings.Split(string(out), "\n") {
		fields := strings.Fields(ln)
		// 表头/分隔行以字母开头；数据行以数字开头且列数充足。
		if len(fields) < 14 {
			continue
		}
		page, err1 := parseIntSafe(fields[0])
		w, err2 := parseIntSafe(fields[3])
		h, err3 := parseIntSafe(fields[4])
		xppi, err4 := parseFloatSafe(fields[12])
		yppi, err5 := parseFloatSafe(fields[13])
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
			continue
		}
		imgs = append(imgs, PDFImageInfo{Page: page, Width: w, Height: h, XPPI: xppi, YPPI: yppi})
	}
	return imgs, nil
}

func parseIntSafe(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func parseFloatSafe(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%g", &f)
	return f, err
}

// PDFPageText pdftotext -bbox 的一页（版面机检 P1-6 文本越界判定）。
type PDFPageText struct {
	Width  float64 // PDF 单位（pt）
	Height float64
	Words  []PDFWord
}

// PDFWord 一个词的 bbox（pt）。
type PDFWord struct {
	XMin, XMax float64
	Text       string
}

// PDFTextBBoxes pdftotext -bbox 输出解析（页尺寸+逐词坐标）。regex 解析固定
// 形态的 XHTML（<page width=".." height=".."> / <word xMin=".." yMin=".."
// xMax=".." yMax="..">text</word>），不引 XML 依赖。
func PDFTextBBoxes(pdfPath string) ([]PDFPageText, error) {
	exe := pdftotextLookup()
	if exe == "" {
		return nil, fmt.Errorf("未找到 pdftotext（poppler 套件）；版面机检文本面需要它")
	}
	out, err := exec.Command(exe, "-bbox", "-enc", "UTF-8", pdfPath, "-").Output()
	if err != nil {
		return nil, fmt.Errorf("pdftotext -bbox 执行失败: %w", err)
	}
	var pages []PDFPageText
	rePage := regexp.MustCompile(`<page width="([0-9.]+)" height="([0-9.]+)"`)
	reWord := regexp.MustCompile(`<word xMin="([0-9.eE+-]+)" yMin="[0-9.eE+-]+" xMax="([0-9.eE+-]+)" yMax="[0-9.eE+-]+">([^<]*)</word>`)
	for _, block := range strings.Split(string(out), "<page ") {
		if block == string(out) {
			continue // 首段（<page 前的 head）无页
		}
		pm := rePage.FindStringSubmatch("<page " + firstLine(block))
		if pm == nil {
			continue
		}
		w, _ := strconvParseFloat(pm[1])
		h, _ := strconvParseFloat(pm[2])
		page := PDFPageText{Width: w, Height: h}
		for _, wm := range reWord.FindAllStringSubmatch(block, -1) {
			xmin, _ := strconvParseFloat(wm[1])
			xmax, _ := strconvParseFloat(wm[2])
			page.Words = append(page.Words, PDFWord{XMin: xmin, XMax: xmax, Text: wm[3]})
		}
		pages = append(pages, page)
	}
	return pages, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '>'); i >= 0 {
		return s[:i+1]
	}
	return s
}

func strconvParseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%g", &f)
	return f, err
}
