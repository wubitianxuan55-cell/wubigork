package ai

import "context"

// ImageBackend 图片生成后端接口 — 支持多后端切换
type ImageBackend interface {
	GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error)
}
