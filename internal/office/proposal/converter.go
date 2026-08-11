// Package proposal — 文档格式转换（优先使用 Microsoft MarkItDown）
package proposal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/carmel/gooxml/document"
	"github.com/gaea/gaea/internal/gaea/proc"
	"github.com/ledongthuc/pdf"
)

// ConvertToMarkdown 将文件转换为 Markdown
// 优先使用 Microsoft MarkItDown（pip install markitdown），回退到内置转换器
func ConvertToMarkdown(filePath string) (string, error) {
	// 尝试 MarkItDown
	if markitdownAvailable() {
		result, err := convertWithMarkItDown(filePath)
		if err == nil && result != "" && !looksBinary(result) {
			return result, nil
		}
		// MarkItDown 失败，回退到内置
	}

	// 内置转换器
	md, err := builtinConvert(filePath)
	if err != nil {
		return "", err
	}
	if looksBinary(md) {
		return "", fmt.Errorf("转换结果异常：输出仍是原始文件字节（可能是编码异常的扫描件），请重试或更换文件")
	}
	return md, nil
}

// looksBinary 判断转换结果是否仍是原始文件字节（历史版本曾把 PDF 原始
// 字节当 markdown 存进方案 JSON，导致文件膨胀且 AI 无法使用）。
// 命中 PDF 魔数或含 NUL 字节即视为未真正转换。
func looksBinary(s string) bool {
	if strings.HasPrefix(s, "%PDF-") {
		return true
	}
	return strings.IndexByte(s, 0) >= 0
}

// convertFile 转换文件：文本提取优先，扫描件/文字过少时回退 OCR。
// 返回 (markdown, 是否使用OCR, 错误)。
func (s *Service) convertFile(ctx context.Context, filePath string) (string, bool, error) {
	md, err := ConvertToMarkdown(filePath)
	if err == nil && len([]rune(md)) >= 30 {
		return md, false, nil
	}
	if isOCRFile(filePath) {
		s.ensureOCR()
		if s.ocr == nil {
			reason := "提取文字过少"
			if err != nil {
				reason = err.Error()
			}
			return "", false, fmt.Errorf("未提取到文字（可能是扫描件），OCR 不可用：%s；请安装 Python 依赖（pip install rapidocr_onnxruntime pymupdf）", reason)
		}
		text, oerr := s.ocr.OCR(ctx, filePath)
		if oerr != nil {
			return "", false, fmt.Errorf("文字提取失败且 OCR 失败：%v", oerr)
		}
		return "# 招标文件（OCR 识别）\n\n" + text, true, nil
	}
	if err != nil {
		return "", false, err
	}
	return "", false, fmt.Errorf("未提取到文字")
}

func markitdownAvailable() bool {
	// 检查 python 和 markitdown 是否可用
	cmd := exec.Command("python", "-c", "from markitdown import MarkItDown")
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
	return cmd.Run() == nil
}

func convertWithMarkItDown(filePath string) (string, error) {
	// 使用 Python 脚本调用 MarkItDown
	script := `
import sys
from markitdown import MarkItDown
md = MarkItDown()
result = md.convert(sys.argv[1])
print(result.text_content)
`
	cmd := exec.Command("python", "-c", script, filePath)
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("MarkItDown 转换失败: %w\n输出: %s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// ─── 内置转换器（回退方案）─────────────────────────────────

func builtinConvert(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		return convertPdfToMD(filePath)
	case ".docx":
		return convertDocxToMD(filePath)
	case ".doc":
		return "", fmt.Errorf(".doc 格式暂不支持，请另存为 .docx 或安装 markitdown (pip install markitdown)")
	case ".txt", ".md":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("不支持的文件格式: %s", ext)
	}
}

func convertPdfToMD(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 PDF 失败: %w（扫描件PDF请安装 markitdown: pip install markitdown）", err)
	}
	defer f.Close()

	var buf strings.Builder
	totalPage := r.NumPage()
	buf.WriteString(fmt.Sprintf("# 招标文件\n\n> 来源：%s | 共 %d 页\n\n", filepath.Base(filePath), totalPage))
	textFound := false

	for i := 1; i <= totalPage; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		md := pdfTextToMD(text)
		if strings.TrimSpace(md) != "" {
			textFound = true
			buf.WriteString(md + "\n\n")
		}
	}

	result := buf.String()
	if !textFound {
		return "", fmt.Errorf("PDF 无可提取文本（可能是扫描件，将尝试 OCR）")
	}
	return result, nil
}

func pdfTextToMD(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isHeading(trimmed) {
			result = append(result, "## "+trimmed)
		} else {
			result = append(result, trimmed)
		}
	}
	return strings.Join(result, "\n")
}

func isHeading(line string) bool {
	if len([]rune(line)) > 40 {
		return false
	}
	patterns := []string{"第", "一、", "二、", "三、", "1.", "招标", "投标", "项目", "技术"}
	for _, p := range patterns {
		if strings.Contains(line, p) {
			return true
		}
	}
	return false
}

func convertDocxToMD(filePath string) (string, error) {
	doc, err := document.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 docx 失败: %w", err)
	}
	defer func() { _ = recover() }()

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("# 招标文件\n\n> 来源：%s\n\n", filepath.Base(filePath)))
	for _, para := range doc.Paragraphs() {
		var parts []string
		for _, run := range para.Runs() {
			if t := run.Text(); t != "" {
				parts = append(parts, t)
			}
		}
		text := strings.TrimSpace(strings.Join(parts, ""))
		if text == "" {
			buf.WriteString("\n")
			continue
		}
		style := para.Properties().Style()
		isBold := false
		for _, run := range para.Runs() {
			if run.Properties().IsBold() {
				isBold = true
				break
			}
		}
		if strings.Contains(style, "Heading") || strings.Contains(style, "标题") || (isBold && len([]rune(text)) < 60) {
			buf.WriteString("## " + text + "\n\n")
		} else {
			buf.WriteString(text + "\n\n")
		}
	}
	return buf.String(), nil
}
