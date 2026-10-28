// pdf.go — CAD 图纸的 PDF 页图通道：复用 docmd 的 poppler pdftoppm 渲染
// （与文件预览/视觉 diff 同一探测源），产出与 DWG/DXF 同构的 OpenResult。
// 一期口径：PDF 支持「看（页图翻页）+ 问图纸（页图喂识图）」；矢量文本
// 清单仅 DWG/DXF（GaeaCadTexts 对 PDF 报明确错误）。
package cad

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// PDFDpiDefault PDF 页图默认渲染 DPI（110：A3 图纸文字可读与耗时的折中）。
const PDFDpiDefault = 110

// RenderPDFPage 单页渲染（ask 档）：第 page 页（1 基）按 dpi 渲为 PNG，返回落盘
// 路径。与 OpenPDFContext 的整本渲染不同，本函数只渲一页——「问图纸」与 agent
// cad_ask 的单页高清通道（高清档 DPI 由调用方定，对话侧 200、面板默认档 110）。
func RenderPDFPage(ctx context.Context, path, cacheDir string, page, dpi int) (string, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("创建缓存目录失败: %w", err)
	}
	pngs, err := docmd.RenderPDFPageRange(ctx, path, filepath.Join(cacheDir, "ask"), dpi, page, page)
	if err != nil {
		if strings.Contains(err.Error(), "pdftoppm") {
			return "", fmt.Errorf("PDF 渲染需要 poppler（pdftoppm），当前环境未安装: %w", err)
		}
		return "", err
	}
	if len(pngs) == 0 {
		return "", fmt.Errorf("PDF 第 %d 页渲染失败：无输出", page)
	}
	return pngs[0], nil
}

// PDFMaxPages 一期页数上限（超出截断并警告；防超大 PDF 首开卡顿）。
const PDFMaxPages = 50

// OpenPDF 渲染 PDF 全部页（≤maxPages，0=PDFMaxPages）为 PNG 落盘 cacheDir，
// 产出与 DWG/DXF 同构的 OpenResult（Sheets=页序，PassRate=-1 无此口径）。
func OpenPDF(path, cacheDir string, dpi, maxPages int) (*OpenResult, error) {
	return OpenPDFContext(context.Background(), path, cacheDir, dpi, maxPages)
}

// OpenPDFContext renders only the requested leading page range and kills
// poppler's process tree when ctx is cancelled.
func OpenPDFContext(ctx context.Context, path, cacheDir string, dpi, maxPages int) (*OpenResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	kind, _, err := ValidateInput(path)
	if err != nil {
		return nil, err
	}
	if kind != KindPDF {
		return nil, fmt.Errorf("不是 PDF 图纸: %s", path)
	}
	if dpi <= 0 {
		dpi = PDFDpiDefault
	}
	if maxPages <= 0 {
		maxPages = PDFMaxPages
	}
	if maxPages > PDFMaxPages {
		maxPages = PDFMaxPages
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建缓存目录失败: %w", err)
	}
	pages, err := docmd.RenderPDFPageRange(ctx, path, filepath.Join(cacheDir, "page"), dpi, 1, maxPages)
	if err != nil {
		if strings.Contains(err.Error(), "pdftoppm") {
			return nil, fmt.Errorf("PDF 渲染需要 poppler（pdftoppm），当前环境未安装: %w", err)
		}
		return nil, err
	}
	res := &OpenResult{Summary: Summary{
		Kind: KindPDF, Layers: []string{}, Warnings: []string{},
	}}
	if len(pages) > maxPages {
		pages = pages[:maxPages]
	}
	res.warn(fmt.Sprintf("PDF 本期仅渲染前 %d 页；更长文档请拆分后重试", maxPages))
	for i, p := range pages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		dst := filepath.Join(cacheDir, fmt.Sprintf("sheet-%d.png", i+1))
		if p != dst {
			data, err := os.ReadFile(p)
			if err != nil {
				return nil, fmt.Errorf("读取页图失败: %w", err)
			}
			if err := fileutil.AtomicWrite(dst, data, 0o644); err != nil {
				return nil, fmt.Errorf("落盘页图失败: %w", err)
			}
			_ = os.Remove(p)
		}
		res.Sheets = append(res.Sheets, SheetOut{
			Sheet: Sheet{Name: fmt.Sprintf("第 %d 页", i+1), PassRate: -1},
			PNG:   dst,
		})
	}
	return res, nil
}

func (o *OpenResult) warn(msg string) { o.Warnings = append(o.Warnings, msg) }
