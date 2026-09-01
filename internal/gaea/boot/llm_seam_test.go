package boot_test

import (
	"context"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
)

// llmFakeClient 是最小 ai.LLMClient：让 wubigrok bridge 工厂可构造。
type llmFakeClient struct{}

func (llmFakeClient) ChatStream(ctx context.Context, req *ai.ChatRequest) (<-chan ai.SSEChunk, error) {
	ch := make(chan ai.SSEChunk, 1)
	ch <- ai.SSEChunk{Done: true}
	close(ch)
	return ch, nil
}
func (llmFakeClient) ChatSimpleStream(ctx context.Context, model, systemPrompt, userMsg string) (string, error) {
	return "", nil
}
func (llmFakeClient) ChatSimpleStreamWithOptions(ctx context.Context, model, systemPrompt, userMsg string, opts ai.ChatSimpleOptions) (string, error) {
	return "", nil
}
func (llmFakeClient) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	return nil, nil
}

// ── LLM seam：boot 装配按 config 选提供者 ─────────────────────

// TestNewProvider_EmptyKindDefaultsToWubigrok 验证缺省行为与现状一致：
// 配置未指定 kind（providers[].kind 为空）时 NewProvider 经 LLM seam 落到
// 缺省 DefaultLLMKind（wubigrok bridge provider），成功构造且满足 seam 接口。
func TestNewProvider_EmptyKindDefaultsToWubigrok(t *testing.T) {
	bridge.SetClient(llmFakeClient{})
	t.Cleanup(func() { bridge.SetClient(nil) })

	p, err := boot.NewProvider(&config.ProviderEntry{Name: "gaea"})
	if err != nil {
		t.Fatalf("NewProvider(empty kind): %v", err)
	}
	if p.Name() != "gaea" {
		t.Errorf("Name = %q, want gaea", p.Name())
	}
	// 装配结果必须满足 LLM seam 定义接口（调用方只依赖接口）。
	var _ provider.LLMProvider = p
}

// TestNewProvider_UnknownKindFailsClosed 验证未知 kind 在装配链上 fail-closed：
// 配置写了不存在的 kind → 立即报错，绝不静默降级。
func TestNewProvider_UnknownKindFailsClosed(t *testing.T) {
	_, err := boot.NewProvider(&config.ProviderEntry{Name: "x", Kind: "llm-test-no-such-kind"})
	if err == nil {
		t.Fatal("unknown kind must fail closed at boot assembly")
	}
}

// TestNewProvider_SelectsByKind 验证按 config 的 kind 切换后端：同一装配函数，
// 只改配置（kind）即得到不同提供者——「切换后端只改配置，代码零改动」。
func TestNewProvider_SelectsByKind(t *testing.T) {
	kindA := testKind("boot-llm-a-" + t.Name())
	kindB := testKind("boot-llm-b-" + t.Name())
	provider.Register(kindA, func(cfg provider.Config) (provider.Provider, error) {
		return &bootNamedProvider{name: "backend-A"}, nil
	})
	provider.Register(kindB, func(cfg provider.Config) (provider.Provider, error) {
		return &bootNamedProvider{name: "backend-B"}, nil
	})

	pa, err := boot.NewProvider(&config.ProviderEntry{Name: "m1", Kind: kindA})
	if err != nil {
		t.Fatalf("NewProvider(kindA): %v", err)
	}
	if pa.Name() != "backend-A" {
		t.Errorf("kindA provider Name = %q, want backend-A", pa.Name())
	}
	pb, err := boot.NewProvider(&config.ProviderEntry{Name: "m2", Kind: kindB})
	if err != nil {
		t.Fatalf("NewProvider(kindB): %v", err)
	}
	if pb.Name() != "backend-B" {
		t.Errorf("kindB provider Name = %q, want backend-B", pb.Name())
	}
	// 未原生实现 Chat 的提供者被 seam 自动适配为 LLMProvider。
	var _ provider.LLMProvider = pa
	var _ provider.LLMProvider = pb
}

// bootNamedProvider 是最小 Provider（无原生 Chat；seam 自动适配）。
type bootNamedProvider struct{ name string }

func (p *bootNamedProvider) Name() string { return p.name }
func (p *bootNamedProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 1)
	ch <- provider.Chunk{Type: provider.ChunkDone}
	close(ch)
	return ch, nil
}
