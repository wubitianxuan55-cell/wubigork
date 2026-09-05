// Package builtin — 文档格式转换工具（docx/xlsx/pdf → Markdown）。
package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/office/docmd"
)

func init() { tool.RegisterBuiltin(formatConvert{}) }

type formatConvert struct{}

func (formatConvert) Name() string { return "format_convert" }

func (formatConvert) Description() string {
	return "文档格式转换：将 docx/xlsx/pptx/pdf 文件转换为 Markdown。docx→md 保留标题层级和表格；xlsx→md 生成表格；pptx→md 提取每页文本与备注；pdf→md 提取文本（含 OCR 扫描件回退）。按文档大小数秒到数十秒；不消耗主模型 token。"
}

func (formatConvert) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"源文件路径（支持 .docx/.xlsx/.pptx/.pdf）"},
  "output":{"type":"string","description":"输出 Markdown 文件路径（可选，不指定则返回文本）"},
  "pages":{"type":"string","description":"PDF页码范围，如\"1-5\"或\"1,3,5\"（仅PDF有效）"}
},
"required":["path"]
}`)
}

func (formatConvert) ReadOnly() bool { return true }

func (formatConvert) CompactDescription() string     { return compactDesc["format_convert"] }
func (formatConvert) CompactSchema() json.RawMessage { return compactSchema["format_convert"] }

type fcInput struct {
	Path   string `json:"path"`
	Output string `json:"output,omitempty"`
	Pages  string `json:"pages,omitempty"`
}

func (formatConvert) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p fcInput
	if err := json.Unmarshal(args, &p); err != nil {
		return "", codedError("FORMAT_INVALID_ARGS", "参数无效: %v", err)
	}
	if p.Path == "" {
		return "", codedError("FORMAT_INVALID_ARGS", "path 不能为空")
	}

	md, total, truncated, err := docmd.ConvertLimit(p.Path, p.Pages, docmd.DefaultMaxPDFPages)
	if err != nil {
		// U1 错误码：模型按 code 路由恢复（换路径/如实告知不支持），不解析散文
		switch {
		case strings.Contains(err.Error(), "不支持的文件格式"):
			return "", codedError("FORMAT_UNSUPPORTED", "%v", err)
		case errors.Is(err, os.ErrNotExist) || os.IsNotExist(err):
			return "", codedError("FORMAT_SOURCE_MISSING", "源文件不存在: %s（%v）", p.Path, err)
		default:
			return "", codedError("FORMAT_CONVERT_FAILED", "%v", err)
		}
	}
	if truncated {
		md += fmt.Sprintf("\n\n> 转换已截断：PDF 共 %d 页，本次仅处理前 %d 页。可指定 pages 参数分段转换，或使用 summarize_file 获取全文摘要。",
			total, docmd.DefaultMaxPDFPages)
	}
	md = fmt.Sprintf("# 文档转换: %s\n\n%s\n\n---\n*由 gaea format_convert 转换*", filepath.Base(p.Path), md)

	if p.Output != "" {
		// 自动创建输出文件的父目录，与 write_file 行为一致；否则输出路径
		// 指向不存在的目录时 os.WriteFile 会失败，导致“没有生成文件”。
		if dir := filepath.Dir(p.Output); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return "", codedError("FORMAT_OUTPUT_WRITE_FAILED", "创建输出目录失败: %v", err)
			}
		}
		if err := os.WriteFile(p.Output, []byte(md), 0o644); err != nil {
			return "", codedError("FORMAT_OUTPUT_WRITE_FAILED", "写入输出文件失败: %v", err)
		}
		return tool.WrapText(fmt.Sprintf("已转换并保存为 %s（%d 字符）", p.Output, len(md))), nil
	}
	return tool.WrapText(md), nil
}
