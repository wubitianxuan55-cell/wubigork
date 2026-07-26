package ai

import (
	"context"
	"fmt"
	"log/slog"
)

// ── 多模型路由 ───────────────────────────────────────────────

// TaskType AI 任务类型
type TaskType string

const (
	TaskGenerate     TaskType = "generate"     // 创意生成（章节、场景）
	TaskRewrite      TaskType = "rewrite"      // 文笔润色、改写
	TaskAnalyze      TaskType = "analyze"      // 结构化分析、审稿
	TaskComplete     TaskType = "complete"     // 快速补全（Ghost Text）
	TaskSummarize    TaskType = "summarize"    // 摘要生成
	TaskBrainstorm   TaskType = "brainstorm"   // 脑暴点子
	TaskConsistency  TaskType = "consistency"  // 一致性检查
	TaskEdit         TaskType = "edit"         // 指令编辑（Cmd+K）
)

// ModelPreference 模型选择偏好
type ModelPreference struct {
	Model          string  `json:"model"`           // 模型名称
	Temperature    float64 `json:"temperature"`     // 温度
	MaxTokens      int     `json:"max_tokens"`      // 最大 token
	ReasoningEffort string `json:"reasoning_effort"` // 推理深度
}

// Router 多模型路由器
// 当前所有任务均使用 Grok（单后端），但接口预留多后端扩展
type Router struct {
	defaultModel string
	preferences  map[TaskType]ModelPreference
}

// NewRouter 创建路由器
func NewRouter(defaultModel string) *Router {
	r := &Router{
		defaultModel: defaultModel,
		preferences:  make(map[TaskType]ModelPreference),
	}

	// 默认偏好：不同任务类型有不同温度和参数
	r.preferences[TaskGenerate] = ModelPreference{
		Temperature:    0.75,
		MaxTokens:      8192,
		ReasoningEffort: "medium",
	}
	r.preferences[TaskRewrite] = ModelPreference{
		Temperature:    0.6,
		MaxTokens:      4096,
		ReasoningEffort: "",
	}
	r.preferences[TaskAnalyze] = ModelPreference{
		Temperature:    0.15,
		MaxTokens:      2048,
		ReasoningEffort: "high",
	}
	r.preferences[TaskComplete] = ModelPreference{
		Temperature:    0.8,
		MaxTokens:      256,
		ReasoningEffort: "",
	}
	r.preferences[TaskSummarize] = ModelPreference{
		Temperature:    0.15,
		MaxTokens:      2048,
		ReasoningEffort: "",
	}
	r.preferences[TaskBrainstorm] = ModelPreference{
		Temperature:    0.9,
		MaxTokens:      4096,
		ReasoningEffort: "high",
	}
	r.preferences[TaskConsistency] = ModelPreference{
		Temperature:    0.1,
		MaxTokens:      1024,
		ReasoningEffort: "high",
	}
	r.preferences[TaskEdit] = ModelPreference{
		Temperature:    0.6,
		MaxTokens:      4096,
		ReasoningEffort: "",
	}

	return r
}

// Route 根据任务类型返回最佳模型和参数
func (r *Router) Route(task TaskType) ModelPreference {
	pref, ok := r.preferences[task]
	if !ok {
		return ModelPreference{
			Model:       r.defaultModel,
			Temperature: 0.7,
			MaxTokens:   4096,
		}
	}
	if pref.Model == "" {
		pref.Model = r.defaultModel
	}
	return pref
}

// ChatWithRoute 使用路由偏好简化调用
// 自动根据 task 类型选择最优参数
func (c *Client) ChatWithRoute(ctx context.Context, task TaskType, systemPrompt, userMsg string) (string, error) {
	router := NewRouter(c.cfg.Model)
	pref := router.Route(task)

	slog.Info("模型路由",
		"task", task,
		"model", pref.Model,
		"temperature", pref.Temperature,
		"reasoning", pref.ReasoningEffort,
	)

	reply, err := c.ChatSimpleStreamWithOptions(ctx, pref.Model, systemPrompt, userMsg, ChatSimpleOptions{
		Temperature:     pref.Temperature,
		MaxTokens:       pref.MaxTokens,
		ReasoningEffort: pref.ReasoningEffort,
	})
	if err != nil {
		return "", fmt.Errorf("路由调用失败 [%s]: %w", task, err)
	}
	return reply, nil
}

// ChatStreamWithRoute 流式版本的路由调用
func (c *Client) ChatStreamWithRoute(ctx context.Context, task TaskType, systemPrompt, userMsg string) (<-chan SSEChunk, error) {
	router := NewRouter(c.cfg.Model)
	pref := router.Route(task)

	slog.Info("模型路由(流式)",
		"task", task,
		"model", pref.Model,
		"temperature", pref.Temperature,
	)

	req := &ChatRequest{
		Model:           pref.Model,
		Messages:        []ChatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userMsg}},
		MaxTokens:       pref.MaxTokens,
		Temperature:     pref.Temperature,
		ReasoningEffort: pref.ReasoningEffort,
	}

	return c.ChatStream(ctx, req)
}
