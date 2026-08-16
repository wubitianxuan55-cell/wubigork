package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/ai"
)

// NovelReadingAsk AI 阅读伴读：summary = 章节摘要，ask = 划线提问。
// 只使用当前章节本地文本，不写任何文件；模型走 novel 功能路由。
func (a *writingState) NovelReadingAsk(kind, title, chapterText, selection, question string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("AI 客户端未初始化")
	}
	kind = strings.TrimSpace(kind)
	if title == "" {
		title = "未命名章节"
	}
	const maxChapterRunes = 9000
	trunc := func(s string) string {
		r := []rune(s)
		if len(r) > maxChapterRunes {
			return string(r[:maxChapterRunes]) + "……（正文过长已截断）"
		}
		return s
	}

	var system, user string
	switch kind {
	case "summary":
		body := strings.TrimSpace(chapterText)
		if body == "" {
			return "", fmt.Errorf("本章暂无内容")
		}
		system = "你是一位专业的小说阅读伴读。用简洁、有信息量的中文概括本章内容，不复述原文，不用套话开头。"
		user = fmt.Sprintf("章节：%s\n\n【正文】\n%s\n\n请给出本章摘要：3-5 条要点，每条不超过 60 字，使用“- ”列表。",
			title, trunc(body))
	case "ask":
		sel := strings.TrimSpace(selection)
		q := strings.TrimSpace(question)
		if sel == "" {
			return "", fmt.Errorf("请先摘选原文")
		}
		if q == "" {
			return "", fmt.Errorf("请输入问题")
		}
		system = "你是一位小说阅读伴读助手。基于用户摘选的原文片段回答问题，语言精炼、贴合上下文；若原文信息不足，明确说明而非编造。"
		user = fmt.Sprintf("章节：%s\n\n【摘选原文】\n%s\n\n【问题】%s", title, trunc(sel), q)
	default:
		return "", fmt.Errorf("未知的伴读类型：%s", kind)
	}

	eng, model, _ := a.routeModel("novel")
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reply, err := a.client.ChatSimpleStreamWithOptions(ctx, model, system, user, ai.ChatSimpleOptions{
		EngineID:    eng,
		Temperature: 0.6,
		MaxTokens:   1024,
	})
	if err != nil {
		return "", fmt.Errorf("AI 伴读失败: %w", err)
	}
	return strings.TrimSpace(reply), nil
}
