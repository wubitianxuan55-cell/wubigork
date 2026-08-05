// Package bridge 实现 gaea provider.Provider 接口，通过 gaea 的 ai.LLMClient
// （模型引擎中心）驱动 gaea agent 引擎。这是 gaeaW 移植到 gaea 的模型适配层：
// gaeaW 自身的 openai/anthropic/xai 实现（模型引擎）不移植，统一走 gaea 模型中心。
package bridge

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// Provider 是 gaea provider.Provider 的 gaea 实现。
type Provider struct {
	name   string
	model  string
	client ai.LLMClient
	engine string // 办公功能级引擎（空 = 由 ai.Client 按活跃引擎解析）
}

// Name 返回 provider 实例名。
func (p *Provider) Name() string { return p.name }

// Stream 将 gaea 请求转发到 gaea 模型中心，并把流式响应转换为 gaea Chunk。
func (p *Provider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	creq := &ai.ChatRequest{
		Model:       p.model,
		Messages:    toChatMessages(req.Messages),
		Tools:       toChatTools(req.Tools),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		EngineID:    p.engine,
	}
	raw, err := p.client.ChatStream(ctx, creq)
	if err != nil {
		return nil, fmt.Errorf("bridge: chat stream: %w", err)
	}

	out := make(chan provider.Chunk, 64)
	go func() {
		defer close(out)
		defer func() {
			if r := recover(); r != nil {
				slog.Error("bridge: stream relay panic recovered", "panic", r)
				out <- provider.Chunk{Type: provider.ChunkError, Err: fmt.Errorf("stream relay panic: %v", r)}
			}
		}()
		// send 在 ctx 取消时停止发送并返回 false，避免下游退出后阻塞泄漏
		send := func(ch provider.Chunk) bool {
			select {
			case out <- ch:
				return true
			case <-ctx.Done():
				return false
			}
		}
		for c := range raw {
			if c.Error != "" {
				if !send(provider.Chunk{Type: provider.ChunkError, Err: errors.New(c.Error)}) {
					return
				}
				return
			}
			for _, tc := range c.ToolCalls {
				if !send(provider.Chunk{
					Type: provider.ChunkToolCall,
					ToolCall: &provider.ToolCall{
						ID:        tc.ID,
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}) {
					return
				}
			}
			if c.Content != "" {
				if !send(provider.Chunk{Type: provider.ChunkText, Text: c.Content}) {
					return
				}
			}
			if c.Done {
				if !send(provider.Chunk{Type: provider.ChunkDone}) {
					return
				}
				return
			}
		}
		if !send(provider.Chunk{Type: provider.ChunkDone}) {
			return
		}
	}()
	return out, nil
}

// client 持有注入的 gaea AI 客户端（模型引擎中心入口）。
var client ai.LLMClient

// SetClient 注入 gaea ai.LLMClient。须在创建任何 Provider 前调用。
func SetClient(c ai.LLMClient) { client = c }

// featureEngine/featureModel 是办公功能级模型绑定（老栈 func_gaea_engine/
// func_gaea_model），由 app 层在 GaeaInit 时注入。它让办公 agent 走指定
// 引擎（如 deepseek），而不是全局活跃引擎——避免活跃引擎为 xai 时把
// 其他模型名发到 xAI 导致 404。空值 = 跟随全局活跃引擎（原行为）。
var featureEngine, featureModel string

// SetFeature 注入办公功能级引擎与模型（空 = 跟随全局活跃引擎）。
func SetFeature(engine, model string) {
	featureEngine, featureModel = engine, model
}

func init() {
	provider.Register("wubigrok", func(cfg provider.Config) (provider.Provider, error) {
		if client == nil {
			return nil, errors.New("bridge: ai.LLMClient 未注入，请先调用 bridge.SetClient")
		}
		model := cfg.Model
		if model == "" {
			model = featureModel // 功能级模型（未绑定则为空，由 ai.Client 动态解析）
		}
		return &Provider{name: cfg.Name, model: model, engine: featureEngine, client: client}, nil
	})
}

// toChatMessages 转换 gaea 消息为 OpenAI 兼容消息。
func toChatMessages(msgs []provider.Message) []ai.ChatMessage {
	out := make([]ai.ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		cm := ai.ChatMessage{
			Role:       string(m.Role),
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			Name:       m.Name,
		}
		for _, tc := range m.ToolCalls {
			cm.ToolCalls = append(cm.ToolCalls, ai.ChatToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: ai.ChatToolFunction{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}
		out = append(out, cm)
	}
	return out
}

// toChatTools 转换 gaea 工具定义为 OpenAI 兼容 tools。
func toChatTools(tools []provider.ToolSchema) []ai.ChatToolSchema {
	if len(tools) == 0 {
		return nil
	}
	out := make([]ai.ChatToolSchema, 0, len(tools))
	for _, t := range tools {
		out = append(out, ai.ChatToolSchema{
			Type: "function",
			Function: ai.ChatToolFunctionSpec{
				Name:        t.Name,
				Description: strings.TrimSpace(t.Description),
				Parameters:  t.Parameters,
			},
		})
	}
	return out
}
