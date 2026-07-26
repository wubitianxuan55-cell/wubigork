package ai

import "context"

// LLMClient 是 Agent 包依赖的 AI 调用接口。
// 当前由 *Client 实现，测试时可用 mock 替换。
type LLMClient interface {
	// ChatStream 流式对话，通过 channel 返回 SSEChunk
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan SSEChunk, error)

	// ChatSimpleStream 简化流式对话（默认参数）
	ChatSimpleStream(ctx context.Context, model, systemPrompt, userMsg string) (string, error)

	// ChatSimpleStreamWithOptions 简化流式对话，支持参数覆盖
	ChatSimpleStreamWithOptions(ctx context.Context, model, systemPrompt, userMsg string, opts ChatSimpleOptions) (string, error)

	// GenerateImage 生成图片
	GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error)
}
