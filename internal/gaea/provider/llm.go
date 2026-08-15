// LLM seam (3.0 Step 3b): 定义 / 提供者 / 消费者 三元组。
//
//	定义   LLMProvider{ Chat, Stream }——流式（Stream）继承自现有 Provider
//	      能力面（Name/Stream/Message/Request/Chunk），一次性补全（Chat）在
//	      其上扩展；事件词汇（Chunk*/ChunkType/Usage/Completion）集中于此。
//	提供者 各 kind 实现 init() 自注册到同一注册表（provider.Register 互斥注册，
//	      重复即 panic）；bridge 的 "wubigrok" 是缺省提供者（现状行为）。
//	消费者 boot 装配与 agent 聊天调用只依赖 LLMProvider 接口；切换后端 = 改
//	      gaea.toml providers[].kind，代码零改动。
//
// 三纪律：定义含事件词汇（本文件）；提供者互斥注册（Register 重复 panic）；
// 不可用即 fail-closed（New/NewLLM 对未知 kind 报错，绝不静默降级）。
package provider

import (
	"context"
	"errors"
)

// LLMKindWubigrok 是 gaea 模型中心桥接 provider（bridge）的注册 kind。
// 办公 agent 经它走 gaea 模型中心（ai.LLMClient），不直接连接任何外部 API。
const LLMKindWubigrok = "wubigrok"

// DefaultLLMKind 是配置未选择 kind 时的缺省 LLM provider——与现状一致
// （wubigrok bridge provider）。gaea.toml 的 providers[].kind 为空时经
// NewLLM 落到此处，保证「不配置 = 旧行为」。
const DefaultLLMKind = LLMKindWubigrok

// Completion 是一次性（非流式）补全的结果，与 Stream 的 Chunk 事件词汇对应：
// Content ↔ ChunkText、ReasoningContent/Signature ↔ ChunkReasoning、
// ToolCalls ↔ ChunkToolCall、Usage ↔ ChunkUsage、FinishReason ↔ Usage.FinishReason。
type Completion struct {
	Content            string
	ReasoningContent   string
	ReasoningSignature string
	ToolCalls          []ToolCall
	Usage              *Usage
	FinishReason       string
}

// LLMProvider 是 LLM seam 的定义接口：一个可聊天的模型后端，同时支持流式
// （Stream，复用 Provider 能力面）与一次性（Chat）语义。提供者按 kind 注册，
// 消费者只依赖本接口。
type LLMProvider interface {
	Provider
	// Chat 执行一次非流式补全，返回完整结果。
	Chat(ctx context.Context, req Request) (*Completion, error)
}

// errNilProvider 是 ChatFromStream 收到 nil provider 的错误。
var errNilProvider = errors.New("provider: nil provider")

// ChatFromStream 把一次 Stream 补全聚合为单个 Completion——任意 Provider 的
// 通用 Chat 实现。注册的 provider 若未原生实现 Chat，会由 NewLLM 自动包一层
// 该聚合适配器，因此每个注册 kind 都具备完整 LLM 能力面。
func ChatFromStream(ctx context.Context, p Provider, req Request) (*Completion, error) {
	if p == nil {
		return nil, errNilProvider
	}
	ch, err := p.Stream(ctx, req)
	if err != nil {
		return nil, err
	}
	c := &Completion{}
	var firstErr error
	for chunk := range ch {
		switch chunk.Type {
		case ChunkText:
			c.Content += chunk.Text
		case ChunkReasoning:
			c.ReasoningContent += chunk.Text
			if chunk.Signature != "" {
				c.ReasoningSignature = chunk.Signature
			}
		case ChunkToolCall:
			if chunk.ToolCall != nil {
				c.ToolCalls = append(c.ToolCalls, *chunk.ToolCall)
			}
		case ChunkUsage:
			if chunk.Usage != nil {
				c.Usage = chunk.Usage
				if c.FinishReason == "" {
					c.FinishReason = chunk.Usage.FinishReason
				}
			}
		case ChunkError:
			if chunk.Err != nil && firstErr == nil {
				firstErr = chunk.Err
			}
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return c, nil
}

// streamChatAdapter 把任意 Provider 适配为 LLMProvider：Chat 由 Stream 聚合
// 派生（ChatFromStream）。保证「注册即 LLM 能力」——提供者不必各自重复实现
// 一次性补全；有原生一次性调用的 kind 可直接实现 Chat 走捷径。
type streamChatAdapter struct {
	Provider
}

// Chat 聚合 Stream 为 Completion（通用实现）。
func (a *streamChatAdapter) Chat(ctx context.Context, req Request) (*Completion, error) {
	return ChatFromStream(ctx, a.Provider, req)
}

// NewLLM 按 kind 经注册表构建 LLM seam provider：
//   - kind 为空 → 缺省 DefaultLLMKind（"wubigrok"，现状行为）；
//   - 未知 kind → fail-closed 错误（附已注册 kind 列表），不静默降级；
//   - 工厂返回 nil → 错误（Register/New 既有防线）；
//   - 实现未原生带 Chat → 自动包 Stream 聚合适配器，保证 LLM 能力面完整。
func NewLLM(kind string, cfg Config) (LLMProvider, error) {
	if kind == "" {
		kind = DefaultLLMKind
	}
	p, err := New(kind, cfg)
	if err != nil {
		return nil, err
	}
	if llm, ok := p.(LLMProvider); ok {
		return llm, nil
	}
	return &streamChatAdapter{Provider: p}, nil
}

// _ 编译期断言：streamChatAdapter 满足 LLMProvider。
var _ LLMProvider = (*streamChatAdapter)(nil)
