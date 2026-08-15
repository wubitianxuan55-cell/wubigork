package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// gaeaSpecialistTools 需要 App 服务的专业工具清单（ExtraTools 的补充部分）。
// 3.0 Step 3d #6：ocr/semantic_search 工具此前已定义但未注册进 ExtraTools
// （gaea_handler.go 装配列表漂移，死代码）；Wave 4 在此集中注册（决策：纳入——
// semantic_search 实现完整且有 E2E 测试，能力面板「本地专业模型」分组也声明了它），
// gaea_handler.go 装配时展开进 ExtraTools，后续新增专业工具只需改这一处。
func gaeaSpecialistTools(a *App) []tool.Tool {
	return []tool.Tool{
		ocrTool{a: a},
		semanticSearchTool{a: a},
	}
}

// ocrTool 专业 OCR 工具：图片/扫描件文字提取走本地 OvisOCR2 常驻服务。
// 与 vision（整图理解）分工：要"看懂图"用 vision，要"提取图中文字"用 ocr。
type ocrTool struct {
	a *App
}

func (t ocrTool) Name() string { return "ocr" }

func (t ocrTool) Description() string {
	return "提取图片/扫描件中的文字（OCR）：读取本地图片文件，调用本地 OCR 模型返回识别出的文本。" +
		"适合截图、扫描 PDF 页面、票据、表格照片中的文字提取。通常 2-5 秒/页，冷启动可能更久；不消耗主模型 token。"
}

func (t ocrTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "image_path":{"type":"string","description":"图片文件路径（相对工作区或绝对路径），支持 png/jpg 等"},
  "language":{"type":"string","description":"可选：语言提示，如 chs/eng，默认自动"}
},
"required":["image_path"]
}`)
}

func (t ocrTool) ReadOnly() bool { return true }

func (t ocrTool) CompactDescription() string {
	return "提取图片/扫描件中的文字（本地 OCR 模型）"
}

func (t ocrTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"image_path":{"type":"string"}},"required":["image_path"]}`)
}

func (t ocrTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if t.a == nil {
		return "", fmt.Errorf("ocr: 应用实例不可用")
	}
	var p struct {
		ImagePath string `json:"image_path"`
		Language  string `json:"language"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("ocr: 参数无效: %w", err)
	}
	if strings.TrimSpace(p.ImagePath) == "" {
		return "", fmt.Errorf("ocr: image_path 不能为空")
	}
	text, err := t.a.GaeaOCRText(p.ImagePath)
	if err != nil {
		return "", fmt.Errorf("ocr: %w", err)
	}
	return tool.WrapText(text), nil
}

var _ tool.Tool = ocrTool{}

// semanticSearchTool 跨库语义检索工具：本地 bge-m3 向量化 + 重排，
// 默认在成本库/工程知识库/办公记忆中按语义找相关内容；scope=file 时检索工作区已索引文件。
type semanticSearchTool struct {
	a *App
}

func (t semanticSearchTool) Name() string { return "semantic_search" }

func (t semanticSearchTool) Description() string {
	return "跨库语义检索（本地 bge-m3）：默认在成本库/工程知识库/办公记忆中按语义查找相关内容；scope=file 时在工作区已索引文件（md/txt/docx/xlsx/pdf/pptx 等）中查找。返回命中条目与相似度。" +
		"适合记不清关键词、用自然语言描述想找的内容。通常 1-3 秒；不消耗主模型 token。"
}

func (t semanticSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "query":{"type":"string","description":"自然语言检索语句，例如\"不锈钢驳接爪 90 度 价格\""},
  "scope":{"type":"string","enum":["all","file"],"description":"可选：all=成本库/知识库/办公记忆（默认）；file=工作区已索引文件"}
},
"required":["query"]
}`)
}

func (t semanticSearchTool) ReadOnly() bool { return true }

func (t semanticSearchTool) CompactDescription() string {
	return "跨库/工作区文件语义检索（本地 bge-m3 向量检索）"
}

func (t semanticSearchTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"scope":{"type":"string"}},"required":["query"]}`)
}

func (t semanticSearchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if t.a == nil {
		return "", fmt.Errorf("semantic_search: 应用实例不可用")
	}
	var p struct {
		Query string `json:"query"`
		Scope string `json:"scope"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("semantic_search: 参数无效: %w", err)
	}
	if strings.TrimSpace(p.Query) == "" {
		return "", fmt.Errorf("semantic_search: query 不能为空")
	}
	if p.Scope == "file" {
		return semanticSearchFileResult(t.a, p.Query)
	}
	hits, err := t.a.GaeaSemanticSearch(p.Query)
	if err != nil {
		return "", fmt.Errorf("semantic_search: %w", err)
	}
	if len(hits) == 0 {
		return tool.WrapText("没有找到相关内容"), nil
	}
	var sb strings.Builder
	for i, h := range hits {
		if i > 0 {
			sb.WriteString("\n---\n")
		}
		fmt.Fprintf(&sb, "[%s] %s (相似度 %.3f)\n%s", h.Kind, h.Name, h.Score, h.Text)
	}
	return tool.WrapText(sb.String()), nil
}

var _ tool.Tool = semanticSearchTool{}

// semanticSearchFileResult 工作区已索引文件的语义检索结果。
func semanticSearchFileResult(a *App, query string) (string, error) {
	hits, err := a.GaeaFileSemanticSearch(query, 10)
	if err != nil {
		return "", fmt.Errorf("semantic_search: %w", err)
	}
	if len(hits) == 0 {
		return tool.WrapText("工作区没有找到相关内容（可先确认文件索引已构建）"), nil
	}
	var sb strings.Builder
	for i, h := range hits {
		if i > 0 {
			sb.WriteString("\n---\n")
		}
		fmt.Fprintf(&sb, "[文件] %s (相似度 %.3f)\n%s", h.Path, h.Score, h.Snippet)
	}
	return tool.WrapText(sb.String()), nil
}
