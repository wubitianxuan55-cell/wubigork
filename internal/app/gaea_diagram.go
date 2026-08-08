package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// diagramTypePrompts 图表类型 → 生成约束（与方案编写模块保持一致）。
var diagramTypePrompts = map[string]string{
	"flowchart":    "生成 Mermaid flowchart，节点5-12个",
	"sequence":     "生成 Mermaid sequenceDiagram，参与者3-5个",
	"gantt":        "生成 Mermaid gantt，至少5个任务",
	"pie":          "生成 Mermaid pie，5-8个分类",
	"graph":        "生成 Mermaid graph，节点5-10个",
	"mindmap":      "生成 Mermaid mindmap，3-5个一级分支",
	"process-flow": "生成 Mermaid flowchart LR 工艺流程图：预处理→主处理→后处理→监测，含主要参数节点",
	"org-chart":    "生成 Mermaid flowchart TD 组织架构图：层级清晰（如 经理→各部门/岗位）",
}

var diagramTypeLabels = map[string]string{
	"flowchart":    "流程图",
	"sequence":     "时序图",
	"gantt":        "甘特图",
	"pie":          "饼图",
	"graph":        "架构图",
	"mindmap":      "思维导图",
	"process-flow": "工艺流程图",
	"org-chart":    "组织架构图",
}

// mermaidKeywords 用于轻量校验生成结果是否为有效 Mermaid 首行。
var mermaidKeywords = []string{
	"flowchart", "graph", "sequenceDiagram", "classDiagram",
	"stateDiagram", "stateDiagram-v2", "erDiagram", "journey",
	"gantt", "pie", "mindmap", "timeline", "gitGraph", "requirementDiagram",
}

// diagramTool 画图工具：让 gaea 智能体像 Codex 一样自行绘制办公图表。
// 复用当前活跃引擎生成 Mermaid 代码，保存 .mmd 到工作区并返回。
type diagramTool struct {
	a *App
}

func (t diagramTool) Name() string { return "diagram" }

func (t diagramTool) Description() string {
	return "生成办公图表（Mermaid 代码）：流程图 flowchart、时序图 sequence、甘特图 gantt、组织架构图 org-chart、架构图 graph、思维导图 mindmap、工艺流程图 process-flow、饼图 pie。根据图表类型与主题生成 Mermaid 代码，保存为 .mmd 文件并返回，聊天中可直接渲染。"
}

func (t diagramTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "type":{"type":"string","enum":["flowchart","sequence","gantt","org-chart","graph","mindmap","process-flow","pie"],"description":"图表类型"},
  "topic":{"type":"string","description":"图表主题与内容描述，例如\"项目立项审批流程\"、\"公司组织架构\"、\"产品发布计划甘特图\""},
  "extra":{"type":"string","description":"可选：额外要求，如具体节点、层级数量、包含的环节"}
},
"required":["type","topic"]
}`)
}

func (t diagramTool) ReadOnly() bool { return false }

func (t diagramTool) CompactDescription() string {
	return "生成办公图表(流程图/时序/甘特/组织架构/架构/思维导图)Mermaid代码,保存.mmd并返回"
}

func (t diagramTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"type":{"type":"string"},"topic":{"type":"string"},"extra":{"type":"string"}},"required":["type","topic"]}`)
}

func (t diagramTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Type  string `json:"type"`
		Topic string `json:"topic"`
		Extra string `json:"extra"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	typePrompt, ok := diagramTypePrompts[p.Type]
	if !ok {
		return "", fmt.Errorf("不支持的图表类型: %q（可选：flowchart/sequence/gantt/org-chart/graph/mindmap/process-flow/pie）", p.Type)
	}
	if strings.TrimSpace(p.Topic) == "" {
		return "", fmt.Errorf("topic 不能为空")
	}

	label := diagramTypeLabels[p.Type]
	systemPrompt := "你是专业图表设计师，擅长用 Mermaid 绘制清晰、规范、层级分明的办公图表（流程图、时序图、甘特图、组织架构图、架构图、思维导图等）。"
	var b strings.Builder
	fmt.Fprintf(&b, "图表类型：%s（%s）\n主题：%s\n", label, p.Type, p.Topic)
	b.WriteString(typePrompt)
	if strings.TrimSpace(p.Extra) != "" {
		b.WriteString("\n额外要求：" + p.Extra)
	}
	b.WriteString("\n只输出 Mermaid 代码：以 ```mermaid 开头、``` 结尾，不要输出任何解释文字。")

	reply, err := t.a.client.ChatSimpleStreamWithOptions(ctx, "", systemPrompt, b.String(), ai.ChatSimpleOptions{Temperature: 0.4})
	if err != nil {
		return "", fmt.Errorf("生成图表失败: %w", err)
	}
	code := extractDiagramMermaid(reply)
	if code == "" {
		return "", fmt.Errorf("AI 未能生成有效的 Mermaid 代码：%s", truncateStr(reply, 200))
	}
	if !validMermaidStart(code) {
		return "", fmt.Errorf("生成的代码不是有效的 Mermaid（首行应为 flowchart/graph/sequenceDiagram/gantt/pie/mindmap 等）：%s", truncateStr(code, 120))
	}

	rel, err := saveDiagramMermaid(gaeaCwd(), code)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("已生成%s并保存：[%s](%s)\n\n```mermaid\n%s\n```\n提示：该图表在对话中会自动导出 PNG 图片到工作区（图表下方会显示图片文件链接）。", label, filepath.Base(rel), rel, code), nil
}

// extractDiagramMermaid 从模型回复中提取 Mermaid 代码（兼容围栏与裸代码）。
func extractDiagramMermaid(raw string) string {
	raw = strings.TrimSpace(raw)
	for _, fence := range []string{"```mermaid", "```"} {
		if idx := strings.Index(raw, fence); idx >= 0 {
			start := idx + len(fence)
			start = skipLineBreaks(start, raw)
			if end := strings.Index(raw[start:], "```"); end >= 0 {
				return strings.TrimSpace(raw[start : start+end])
			}
		}
	}
	return raw
}

func skipLineBreaks(pos int, s string) int {
	for pos < len(s) && (s[pos] == '\n' || s[pos] == '\r') {
		pos++
	}
	return pos
}

// validMermaidStart 校验代码首行是否为已知 Mermaid 图关键字。
func validMermaidStart(code string) bool {
	for _, line := range strings.Split(code, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		for _, kw := range mermaidKeywords {
			if strings.HasPrefix(line, kw) && (len(line) == len(kw) || line[len(kw)] == ' ' || line[len(kw)] == '\t') {
				return true
			}
		}
		return false
	}
	return false
}

// saveDiagramMermaid 保存 Mermaid 代码到工作区 .gaea/uploads/，
// 返回相对工作区的路径（如 .gaea/uploads/diagram-xxx.mmd）。
func saveDiagramMermaid(baseDir, code string) (string, error) {
	dir := filepath.Join(baseDir, ".gaea", "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("diagram-%d.mmd", time.Now().UnixNano())
	rel := filepath.ToSlash(filepath.Join(".gaea", "uploads", name))
	if err := os.WriteFile(filepath.Join(dir, name), []byte(code), 0o644); err != nil {
		return "", err
	}
	return rel, nil
}

func truncateStr(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

var _ tool.Tool = diagramTool{}
