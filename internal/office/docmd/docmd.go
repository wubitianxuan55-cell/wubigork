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
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx", ".doc":
		if converted, convErr := convertViaMarkItDown(path); convErr == nil {
			return converted, 0, false, nil
		}
		md, err := docxToMarkdown(path)
		return md, 0, false, err
	case ".xlsx", ".xls":
		md, err := xlsxToMarkdown(path)
		return md, 0, false, err
	case ".pptx", ".ppt":
		converted, convErr := convertViaMarkItDown(path)
		if convErr != nil {
			return "", 0, false, fmt.Errorf("不支持的文件格式: %s（pptx 转换需要 markitdown: pip install markitdown）", filepath.Ext(path))
		}
		return converted, 0, false, nil
	case ".pdf":
		return pdfToMarkdownLimit(path, pages, maxPages, progress)
	default:
		return "", 0, false, fmt.Errorf("不支持的文件格式: %s（支持 .docx/.xlsx/.pptx/.pdf）", filepath.Ext(path))
	}
}
