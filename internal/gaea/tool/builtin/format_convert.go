// Package builtin — 文档格式转换工具（docx/xlsx/pdf → Markdown）。
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/office/docmd"
)

func init() { tool.RegisterBuiltin(formatConvert{}) }

type formatConvert struct{}

func (formatConvert) Name() string { return "format_convert" }

func (formatConvert) Description() string {
	return "文档格式转换：将 docx/xlsx/pdf 文件转换为 Markdown。docx→md 保留标题层级和表格；xlsx→md 生成表格；pdf→md 提取文本（含 OCR 扫描件回退）。"
}

func (formatConvert) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"源文件路径（支持 .docx/.xlsx/.pdf）"},
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
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path 不能为空")
	}

	md, total, truncated, err := docmd.ConvertLimit(p.Path, p.Pages, docmd.DefaultMaxPDFPages)
	if err != nil {
		return "", err
	}
	if truncated {
		md += fmt.Sprintf("\n\n> 转换已截断：PDF 共 %d 页，本次仅处理前 %d 页。可指定 pages 参数分段转换，或使用 summarize_file 获取全文摘要。",
			total, docmd.DefaultMaxPDFPages)
	}
	md = fmt.Sprintf("# 文档转换: %s\n\n%s\n\n---\n*由 gaea format_convert 转换*", filepath.Base(p.Path), md)

	if p.Output != "" {
		if err := os.WriteFile(p.Output, []byte(md), 0o644); err != nil {
			return "", fmt.Errorf("写入输出文件失败: %w", err)
		}
		return tool.WrapText(fmt.Sprintf("已转换并保存为 %s（%d 字符）", p.Output, len(md))), nil
	}
	return tool.WrapText(md), nil
}
