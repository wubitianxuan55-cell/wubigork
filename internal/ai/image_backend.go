package ai

import (
	"context"
	"fmt"
	"sort"
)

// ImageBackend 图片生成后端接口 — 支持多后端切换
type ImageBackend interface {
	GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error)
}

// 图片后端 kind 常量（3a Image seam）：代码与配置只依赖 kind，
// 具体后端实现由各 provider 文件 init() 自注册到注册表。
const (
	// ImageBackendKindOpenAI OpenAI 兼容图片后端（/v1/images/generations）：
	// 覆盖 xAI / Herdsman / Ollama 等兼容服务。
	ImageBackendKindOpenAI = "openai"
	// ImageBackendKindComfyUI 本地 ComfyUI 后端（REST API + 工作流）。
	ImageBackendKindComfyUI = "comfyui"
)

// ImageBackendConfig 图片后端实例配置（注册表 New 入参）。
// 各 kind 按需读取字段：openai 兼容后端用 BaseURL + APIKey；
// comfyui 只用 BaseURL（本地服务无需认证）。
type ImageBackendConfig struct {
	BaseURL string // API 地址（OpenAI 兼容 /v1 后缀；ComfyUI 根地址）
	APIKey  string // 认证密钥（可为空 = 无需认证）
}

// ImageBackendFactory 按实例配置构建图片后端（kind → 实例）。
type ImageBackendFactory func(cfg ImageBackendConfig) (ImageBackend, error)

// imageBackendRegistry kind → 工厂注册表（3a Image seam，范式见
// internal/gaea/provider/provider.go 的 Register/New/Kinds）。
// 各实现 init() 自注册；互斥注册，重复即 panic（编译期接线错误）。
var imageBackendRegistry = map[string]ImageBackendFactory{}

// RegisterImageBackend 注册图片后端 kind（如 "openai" / "comfyui"）。
// 供各实现 init() 自注册；kind 为空或重复注册直接 panic。
func RegisterImageBackend(kind string, factory ImageBackendFactory) {
	if kind == "" {
		panic("ai: image backend kind must not be empty")
	}
	if _, dup := imageBackendRegistry[kind]; dup {
		panic("ai: duplicate image backend kind " + kind)
	}
	imageBackendRegistry[kind] = factory
}

// NewImageBackend 按 kind 经注册表构建图片后端实例；
// 未知 kind 返回错误（附已注册 kind 列表，fail-closed 不静默降级）。
func NewImageBackend(kind string, cfg ImageBackendConfig) (ImageBackend, error) {
	factory, ok := imageBackendRegistry[kind]
	if !ok {
		return nil, fmt.Errorf("ai: unknown image backend kind %q (registered: %v)", kind, ImageBackendKinds())
	}
	backend, err := factory(cfg)
	if err != nil {
		return nil, err
	}
	if backend == nil {
		return nil, fmt.Errorf("ai: image backend factory %q returned nil", kind)
	}
	return backend, nil
}

// ImageBackendKinds 返回已注册图片后端 kind 列表（排序，供诊断/校验）。
func ImageBackendKinds() []string {
	out := make([]string, 0, len(imageBackendRegistry))
	for k := range imageBackendRegistry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
