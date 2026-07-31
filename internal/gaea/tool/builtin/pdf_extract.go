package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/wubigork/wubigork/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(pdfExtract{}) }

type pdfExtract struct{}

func (pdfExtract) Name() string { return "pdf_extract" }

func (pdfExtract) Description() string {
	return "提取 PDF 文件文本内容（简易版）。适用于基于文本的 PDF（非扫描件），返回纯文本。扫描件请使用 OCR 软件处理。"
}

func (pdfExtract) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "pages":{"type":"string","description":"页码范围，如\"1-5\"或\"1,3,5\"（不指定则全部）"},
  "path":{"type":"string","description":"PDF文件路径"}
},
"required":["path"]
}`)
}

func (pdfExtract) ReadOnly() bool { return true }

func (pdfExtract) CompactDescription() string     { return compactDesc["pdf_extract"] }
func (pdfExtract) CompactSchema() json.RawMessage { return compactSchema["pdf_extract"] }

func (pdfExtract) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path 不能为空")
	}

	data, err := os.ReadFile(p.Path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	// 简易 PDF 文本提取：寻找 stream...endstream 之间的纯文本
	// 也尝试提取括号内的文本 (Tj 操作符)
	content := string(data)
	var texts []string

	// 方法1：提取 BT...ET 之间的文本操作
	for {
		btIdx := strings.Index(content, "BT")
		if btIdx < 0 {
			break
		}
		content = content[btIdx+2:]
		etIdx := strings.Index(content, "ET")
		if etIdx < 0 {
			break
		}
		block := content[:etIdx]
		content = content[etIdx+2:]

		// 在 BT...ET 块中提取 (text) Tj 或 [(text)] TJ
		text := extractPDFText(block)
		if strings.TrimSpace(text) != "" {
			texts = append(texts, text)
		}
	}

	result := strings.TrimSpace(strings.Join(texts, "\n"))
	if result == "" {
		// 尝试更原始的方法：提取所有可打印字符
		result = extractRawText(data)
	}

	if result == "" {
		return tool.WrapText("未能从PDF中提取到文本内容。可能原因：1) PDF为扫描件 2) PDF内容被加密 3) 纯图片PDF。建议使用OCR工具处理。"), nil
	}

	return tool.WrapText(result), nil
}

// extractPDFText 从 BT...ET 块中提取文本
func extractPDFText(block string) string {
	var parts []string

	// 处理括号文本: (text) Tj
	remaining := block
	for {
		// 查找 ( 后跟文本 )
		parenStart := -1
		for i, c := range remaining {
			if c == '(' {
				// 检查前面不是反斜杠
				if i == 0 || remaining[i-1] != '\\' {
					parenStart = i
					break
				}
			}
		}
		if parenStart < 0 {
			break
		}

		// 查找匹配的 )
		depth := 1
		parenEnd := -1
		for i := parenStart + 1; i < len(remaining); i++ {
			if remaining[i] == '\\' {
				i++ // skip escaped char
				continue
			}
			if remaining[i] == '(' {
				depth++
			} else if remaining[i] == ')' {
				depth--
				if depth == 0 {
					parenEnd = i
					break
				}
			}
		}
		if parenEnd < 0 {
			break
		}

		text := remaining[parenStart+1 : parenEnd]
		// 检查后面是否有 Tj 操作符
		tail := remaining[parenEnd+1:]
		tail = strings.TrimSpace(tail)
		if strings.HasPrefix(tail, "Tj") || strings.HasPrefix(tail, "'") || strings.HasPrefix(tail, "\"") {
			parts = append(parts, text)
		}

		remaining = remaining[parenEnd+1:]
	}

	// 处理 TJ 数组: [(text1) num (text2)] TJ
	remaining = block
	for {
		brStart := strings.Index(remaining, "[(")
		if brStart < 0 {
			break
		}
		brEnd := strings.Index(remaining[brStart:], "] TJ")
		if brEnd < 0 {
			break
		}
		arrContent := remaining[brStart+1 : brStart+brEnd]
		remaining = remaining[brStart+brEnd+4:]

		var arrParts []string
		_ = arrParts
		for {
			op := strings.Index(arrContent, "(")
			if op < 0 {
				break
			}
			cp := strings.Index(arrContent[op+1:], ")")
			if cp < 0 {
				break
			}
			arrParts = append(arrParts, arrContent[op+1:op+1+cp])
			arrContent = arrContent[op+1+cp+1:]
		}
		if len(arrParts) > 0 {
			parts = append(parts, strings.Join(arrParts, ""))
		}
	}

	// 解码转义字符
	var decoded []string
	for _, p := range parts {
		p = strings.ReplaceAll(p, "\\(", "(")
		p = strings.ReplaceAll(p, "\\)", ")")
		p = strings.ReplaceAll(p, "\\n", "\n")
		p = strings.ReplaceAll(p, "\\r", "\r")
		p = strings.ReplaceAll(p, "\\\\", "\\")
		decoded = append(decoded, p)
	}

	return strings.Join(decoded, " ")
}

// extractRawText 从PDF二进制中提取可读文本
func extractRawText(data []byte) string {
	// 移除 stream 和 endstream 之间的二进制内容
	content := string(data)

	// 提取所有可打印ASCII和中文
	var buf bytes.Buffer
	runes := []rune(content)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r >= 32 && r <= 126 {
			buf.WriteRune(r)
		} else if r >= 0x4E00 && r <= 0x9FFF {
			buf.WriteRune(r)
		} else if r == '\n' || r == '\r' || r == '\t' {
			buf.WriteRune(r)
		}
	}

	text := buf.String()
	// 移除PDF关键字行
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 跳过PDF内部指令
		if isPDFKeyword(line) {
			continue
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func isPDFKeyword(s string) bool {
	keywords := []string{"endstream", "stream", "obj", "endobj", "xref", "trailer",
		"BT", "ET", "Tj", "TJ", "Td", "Tm", "cm", "Do", "gs", "rg", "RG", "k", "K",
		"w", "J", "j", "M", "d", "ri", "sh", "EI", "BDC", "BMC", "EMC", "MP", "DP"}
	for _, kw := range keywords {
		if s == kw || strings.HasPrefix(s, kw+" ") || strings.HasSuffix(s, " "+kw) {
			return true
		}
	}
	return false
}
