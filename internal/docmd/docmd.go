// Package docmd converts office documents (docx/xlsx/pdf) to Markdown.
// Shared by the format_convert agent tool and the desktop file preview panel.
package docmd

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DefaultMaxPDFPages caps PDF conversion/preview at this many pages per call
// (mirrors the industry norm: Gemini/Kimi/MinerU cap PDFs at 500-1000 pages
// and stream or chunk the rest). 0 disables the cap.
const DefaultMaxPDFPages = 500

// Convert renders a docx/xlsx/pdf file as Markdown. pages only applies to PDFs
// ("1-5" or "1,3,5"); unsupported extensions return an error.
func Convert(path, pages string) (string, error) {
	md, _, _, err := ConvertLimit(path, pages, 0)
	return md, err
}

// ConvertLimit is like Convert but caps PDF output at maxPages pages
// (maxPages <= 0 disables the cap). It returns the markdown, the PDF's total
// page count (0 for non-PDFs), whether the cap dropped pages, and any error.
// Callers use the total/truncated values to surface an honest "已截断" notice.
func ConvertLimit(path, pages string, maxPages int) (md string, total int, truncated bool, err error) {
	return ConvertLimitProgress(path, pages, maxPages, nil)
}

// ConvertLimitProgress is ConvertLimit with an optional per-page progress
// callback (done, total) fired while OCR-ing a scanned PDF page-by-page.
// nil disables callbacks; the OCR loop never fires one for in-memory text PDFs.
func ConvertLimitProgress(path, pages string, maxPages int, progress func(done, total int)) (md string, total int, truncated bool, err error) {
	md, total, truncated, _, err = ConvertLimitProgressReport(path, pages, maxPages, progress)
	return md, total, truncated, err
}

// ConvertLimitProgressReport 是 ConvertLimitProgress 加回 docx 结构报告：
// docx 走过重建路线或体检时给出指标（非 docx 为 nil）。回执的结构完整性指标
// （列数一致率/合并展开/目录剔除）从这里出——沿江包1 复盘 P0-2。
func ConvertLimitProgressReport(path, pages string, maxPages int, progress func(done, total int)) (md string, total int, truncated bool, rep *DocxRebuildReport, err error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx", ".doc":
		if strings.EqualFold(filepath.Ext(path), ".docx") {
			// 路由（沿江包1 复盘 T1 根修）：带合并网格/折行碎片/目录残表签名的
			// 脏文档走重建路线就地修表格；干净文档仍走 markitdown（保真优先）；
			// markitdown 不可用时重建产物兜底（旧路退 docxToMarkdown，网格照散）。
			if md, rep, err := convertDocxSmart(path, false); err == nil {
				return md, 0, false, rep, nil
			}
		}
		if converted, convErr := convertViaMarkItDown(path); convErr == nil {
			return converted, 0, false, nil, nil
		}
		md, err := docxToMarkdown(path)
		return md, 0, false, nil, err
	case ".xlsx", ".xls":
		md, err := xlsxToMarkdown(path)
		return md, 0, false, nil, err
	case ".pptx", ".ppt":
		converted, convErr := convertViaMarkItDown(path)
		if convErr != nil {
			return "", 0, false, nil, fmt.Errorf("不支持的文件格式: %s（pptx 转换需要 markitdown: pip install markitdown）", filepath.Ext(path))
		}
		return converted, 0, false, nil, nil
	case ".pdf":
		md, total, truncated, err := pdfToMarkdownLimit(path, pages, maxPages, progress)
		return md, total, truncated, nil, err
	default:
		return "", 0, false, nil, fmt.Errorf("不支持的文件格式: %s（支持 .docx/.xlsx/.pptx/.pdf）", filepath.Ext(path))
	}
}

// convertDocxSmart docx→md 的路由决策：先做结构体检，脏文档（合并网格/折行碎片/
// 目录残表签名）用重建路线就地修表格，干净文档交给 markitdown（保真优先）。
// markitdown 不可用时重建产物兜底；withImages=true 时 markitdown 走 keep-data-uris
// （否则它输出的是省略号假 URI「base64...」，外置必炸）、重建路线原生保图（data URI）。
func convertDocxSmart(path string, withImages bool) (string, *DocxRebuildReport, error) {
	d, err := openDocxDoc(path)
	if err != nil {
		return "", nil, err
	}
	plain, rep := d.render(false)
	if rep.dirtyDocx() {
		md, repFull := d.render(withImages)
		return md, &repFull, nil
	}
	var converted string
	var convErr error
	if withImages {
		converted, convErr = convertViaMarkItDownKeepImages(path)
	} else {
		converted, convErr = convertViaMarkItDown(path)
	}
	if convErr == nil {
		rep.Rebuilt = false
		return converted, &rep, nil
	}
	if withImages {
		md, repFull := d.render(true)
		return md, &repFull, nil
	}
	return plain, &rep, nil
}
