package proposal

import (
	"context"
	"fmt"
	"strings"
)

var chartTypeDescriptions = map[string]string{
	"flowchart": "流程图",
	"sequence":  "时序图",
	"gantt":     "甘特图",
	"pie":       "饼图",
	"graph":     "架构图",
	"mindmap":   "思维导图",
}

var chartTypePrompts = map[string]string{
	"flowchart": "生成 Mermaid flowchart，节点5-12个",
	"sequence":  "生成 Mermaid sequenceDiagram，参与者3-5个",
	"gantt":     "生成 Mermaid gantt，至少5个任务",
	"pie":       "生成 Mermaid pie，5-8个分类",
	"graph":     "生成 Mermaid graph，节点5-10个",
	"mindmap":   "生成 Mermaid mindmap，3-5个一级分支",
}

func (s *Service) GenerateChart(ctx context.Context, proposalID, sectionID, chartType string) (*Proposal, string, error) {
	if s.ai == nil { return nil, "", fmt.Errorf("AI 客户端未初始化") }
	prompt, ok := chartTypePrompts[chartType]
	if !ok { return nil, "", fmt.Errorf("不支持的图表类型: %s", chartType) }
	p, err := s.store.Get(proposalID)
	if err != nil { return nil, "", err }
	var targetSec *ProposalSection
	for _, sec := range flattenSections(p.Sections) { if sec.ID == sectionID { targetSec = sec; break } }
	if targetSec == nil { return nil, "", fmt.Errorf("章节未找到: %s", sectionID) }
	ctxParts := []string{fmt.Sprintf("方案：%s", p.Title), fmt.Sprintf("章节：%s", targetSec.Title)}
	if targetSec.Content != "" { ctxParts = append(ctxParts, "内容摘要："+truncate(targetSec.Content, 800)) }
	sp := fmt.Sprintf("你是专业文档图表设计师。%s。只输出 Mermaid 代码，以```mermaid开头，以```结尾。", prompt)
	reply, err := s.ai.ChatSimpleStream(ctx, "", sp, strings.Join(ctxParts, "\n"))
	if err != nil { return nil, "", err }
	mc := extractMermaidCode(reply)
	if mc == "" { return nil, "", fmt.Errorf("AI 未能生成有效的 Mermaid 代码\n%s", reply) }
	return p, mc, nil
}

func extractMermaidCode(raw string) string {
	raw = strings.TrimSpace(raw)
	if idx := strings.Index(raw, "```mermaid"); idx >= 0 {
		start := idx + len("```mermaid"); start = skipLineBreak2(raw, start)
		if end := strings.Index(raw[start:], "```"); end >= 0 { return strings.TrimSpace(raw[start : start+end]) }
	}
	if idx := strings.Index(raw, "```"); idx >= 0 {
		start := idx + 3; start = skipLineBreak2(raw, start)
		if end := strings.Index(raw[start:], "```"); end >= 0 { return strings.TrimSpace(raw[start : start+end]) }
	}
	return strings.TrimSpace(raw)
}

func skipLineBreak2(s string, pos int) int {
	for pos < len(s) && (s[pos] == '\n' || s[pos] == '\r') { pos++ }
	return pos
}
