package control

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/event"
)

var planAskSeq atomic.Int64

// planGate 开工前计划确认（对标 WorkBuddy Plan 模式）：用 agent 的 provider
// 生成开工计划 → 以 ask 卡片询问「确认执行 / 先调整」。计划生成失败或为空时
// 不阻塞，直接执行。
func (c *Controller) planGate(ctx context.Context, input string) (bool, error) {
	if c.executor == nil {
		return true, nil
	}
	plan, err := c.executor.Plan(ctx, c.systemPrompt, input)
	if err != nil {
		slog.Debug("plan gate: 计划生成失败，直接执行", "err", err)
		return true, nil
	}
	if strings.TrimSpace(plan) == "" {
		return true, nil
	}
	// 尽力把模型计划解析为结构化 PlanCard；失败则回退纯文本计划卡。
	prompt := plan
	var structured *event.Plan
	if p := agent.ParsePlan(plan); p != nil {
		structured = p
		prompt = agent.RenderPlanMarkdown(p)
	}
	ok, err := c.askPlanApproval(ctx, prompt, structured)
	if err != nil {
		return false, err
	}
	if !ok {
		c.notice("已按你的选择取消本轮执行；可补充说明后重发，或直接要求执行。")
	}
	return ok, nil
}

// askPlanApproval 复用 ask 机制：发 AskRequest（可选携带结构化 Plan）并阻塞到
// AnswerQuestion。
func (c *Controller) askPlanApproval(ctx context.Context, prompt string, plan *event.Plan) (bool, error) {
	id := fmt.Sprintf("plan-%d", planAskSeq.Add(1))
	reply := make(chan []event.AskAnswer, 1)
	c.mu.Lock()
	c.asks[id] = reply
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.asks, id)
		c.mu.Unlock()
	}()

	c.sink.Emit(event.Event{Kind: event.AskRequest, Ask: event.Ask{
		ID:   id,
		Plan: plan,
		Questions: []event.AskQuestion{{
			ID:     "plan",
			Header: "开工计划",
			Prompt: prompt,
			Options: []event.AskOption{
				{Label: "确认执行", Description: "按此计划开始干活"},
				{Label: "先调整", Description: "取消本轮，补充说明后重发"},
			},
		}},
	}})

	select {
	case ans := <-reply:
		for _, a := range ans {
			if a.QuestionID != "plan" {
				continue
			}
			for _, s := range a.Selected {
				if strings.Contains(s, "确认") || strings.Contains(s, "执行") {
					return true, nil
				}
			}
		}
		return false, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
