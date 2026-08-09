package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// planSystemPrompt 开工规划提示词：只输出可执行计划，标注资料/工具/输出物。
const planSystemPrompt = `你是 gaea 的开工规划员。用户即将开始一个办公任务，请先输出一份开工计划（Markdown）。
要求：
- 第一行给出任务理解（一句话）；随后列出步骤，每步标注将读取/查看的资料、将使用的工具、产出物；
- 需求不明确的关键点用「待确认：…」标注；
- 5-10 行，简洁直接，只输出计划本身，不要开场白。`

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
