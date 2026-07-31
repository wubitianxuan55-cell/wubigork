// Package bridge 实现 gaea provider.Provider 接口，通过 gaea 的 ai.LLMClient
// （模型引擎中心）驱动 gaea agent 引擎。这是 gaeaW 移植到 gaea 的模型适配层：
// gaeaW 自身的 openai/anthropic/xai 实现（模型引擎）不移植，统一走 gaea 模型中心。
package bridge

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// Provider 是 gaea provider.Provider 的 gaea 实现。
type Provider struct {
	name   string
	model  string
	client ai.LLMClient
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
	}
	raw, err := p.client.ChatStream(ctx, creq)
	if err != nil {
		return nil, fmt.Errorf("bridge: chat stream: %w", err)
	}

	out := make(chan provider.Chunk, 64)
	go func() {
		defer close(out)
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

func init() {
	provider.Register("wubigrok", func(cfg provider.Config) (provider.Provider, error) {
		if client == nil {
			return nil, errors.New("bridge: ai.LLMClient 未注入，请先调用 bridge.SetClient")
		}
		return &Provider{name: cfg.Name, model: cfg.Model, client: client}, nil
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
			Name:        t.Name,
			Description: strings.TrimSpace(t.Description),
			Parameters:  t.Parameters,
		})
	}
	return out
}
