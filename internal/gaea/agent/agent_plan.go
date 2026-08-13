package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// planSystemPrompt 开工规划提示词：只输出严格 JSON，标注资料/工具/产出物/待确认。
const planSystemPrompt = `你是 gaea 的开工规划员。用户即将开始一个办公任务，请先输出一份开工计划。
输出必须是严格 JSON（不要 Markdown 代码围栏，不要任何额外文字、不要开场白），结构如下：
{
  "goal": "任务理解（一句话）",
  "steps": [
    {
      "title": "步骤名",
      "detail": "该步骤做什么",
      "resources": ["将读取/查看的资料路径或名称"],
      "tools": ["将使用的工具名"],
      "deliverable": "产出物"
    }
  ],
  "questions": ["需求不明确的关键点"]
}
要求：
- 3-8 步，简洁直接；resources/tools 使用 gaea 现有工具与工作区文件的实际名称；
- 需求不明确的关键点放入 questions（无则给空数组 []）；
- 只输出 JSON 本身。`

// Plan 用同一 provider 一次性生成开工计划（非流式聚合）。
// 计划失败返回错误，由调用方决定是否回退为直接执行。
func (a *AgentRunner) Plan(ctx context.Context, systemPrompt, input string) (string, error) {
	if a.prov == nil {
		return "", fmt.Errorf("provider unavailable")
	}
	userMsg := provider.Message{Role: provider.RoleUser, Content: input}
	msgs := []provider.Message{userMsg}
	if strings.TrimSpace(systemPrompt) != "" {
		msgs = append([]provider.Message{{
			Role:    provider.RoleSystem,
			Content: systemPrompt + "\n\n" + planSystemPrompt,
		}}, msgs...)
	}
	ch, err := a.prov.Stream(ctx, provider.Request{
		Messages:    msgs,
		Temperature: 0.2,
		MaxTokens:   900,
	})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for c := range ch {
		if c.Err != nil {
			return "", c.Err
		}
		if c.Type == provider.ChunkText {
			b.WriteString(c.Text)
		}
	}
	return strings.TrimSpace(b.String()), nil
}
