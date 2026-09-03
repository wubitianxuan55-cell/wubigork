package largefile

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// summarizeFile is the model-facing summarize_file tool: it runs the map-reduce
// pipeline over a large local file so the model never has to read it whole.
type summarizeFile struct {
	prov    provider.Provider
	workDir string
}

// NewSummarizeTool builds the summarize_file tool with the session provider.
// The provider is injected at boot time (a builtin cannot self-provide one).
func NewSummarizeTool(prov provider.Provider) tool.Tool {
	return &summarizeFile{prov: prov}
}

func (t *summarizeFile) Name() string { return "summarize_file" }

func (t *summarizeFile) Description() string {
	return "Summarize one or more large local files (docx/xls/pdf/txt/md/csv etc.) without reading them all: each file is chunked and summarized (map-reduce), then per-file summaries are merged into one document summary (摘要的摘要). Use for @-referenced or workspace files too large to read fully; pass focus to emphasize e.g. key data or tables."
}

func (t *summarizeFile) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"文件路径（支持 docx/xls/xlsx/pdf/txt/md/csv 等）"},
  "paths":{"type":"array","items":{"type":"string"},"description":"可选：多个文件路径——逐文件摘要后合并为总览（摘要的摘要）"},
  "focus":{"type":"string","description":"可选：摘要侧重，如 \"关键数据与表格清单\" 或 \"结论与建议\""}
}
}`)
}

func (t *summarizeFile) ReadOnly() bool { return true }

// ModelBacked 声明该工具内部调用会话模型做分块摘要——桌面端把一次调用当作
// 「变相子代理」打开 mt_ 运行记录（与子代理同一会话 UI）。
func (t *summarizeFile) ModelBacked() bool { return true }

func (t *summarizeFile) CompactDescription() string {
	return "大文件分块摘要：docx/xls/pdf/txt/md 等过大文件 → 分块摘要后合并（免整读）"
}

func (t *summarizeFile) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"paths":{"type":"array","items":{"type":"string"}},"focus":{"type":"string"}}}`)
}

func (t *summarizeFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path  string   `json:"path"`
		Paths []string `json:"paths,omitempty"`
		Focus string   `json:"focus,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	paths := p.Paths
	if len(paths) == 0 {
		if p.Path == "" {
			return "", fmt.Errorf("path 不能为空")
		}
		paths = []string{p.Path}
	}
	for i := range paths {
		paths[i] = resolveIn(t.workDir, paths[i])
	}

	var b strings.Builder
	if len(paths) == 1 {
		res, err := SummarizeFile(ctx, t.prov, paths[0], Options{Focus: p.Focus})
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "文件：%s", res.Path)
		if res.TotalPages > 0 {
			fmt.Fprintf(&b, "（PDF 共 %d 页）", res.TotalPages)
		}
		fmt.Fprintf(&b, "\n共 %d 字符，分 %d 块摘要：\n\n%s", res.Chars, res.Chunks, res.Summary)
	} else {
		res, err := SummarizeFiles(ctx, t.prov, paths, Options{Focus: p.Focus})
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "文件：%s 等 %d 个文件\n共分 %d 块摘要，合并总览：\n\n%s",
			filepath.Base(res.Paths[0]), res.Files, res.Chunks, res.Summary)
	}
	return tool.WrapText(b.String()), nil
}

func resolveIn(workDir, p string) string {
	if workDir == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(workDir, p)
}
