// 外部测试包：验证 LLM seam 的缺省选择路由到 wubigrok（bridge）provider。
// 放外部包是为了 import bridge（bridge 的 init() 注册 wubigrok kind；
// 内部测试包无法 import bridge，否则构成 import cycle）。
package provider_test

import (
	"context"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
)

// fakeClient 是最小 ai.LLMClient 实现：返回一段文本 + Done。
type fakeClient struct{}

func (fakeClient) ChatStream(ctx context.Context, req *ai.ChatRequest) (<-chan ai.SSEChunk, error) {
	ch := make(chan ai.SSEChunk, 2)
	ch <- ai.SSEChunk{Content: "bridge-ok"}
	ch <- ai.SSEChunk{Done: true}
	close(ch)
	return ch, nil
}
func (fakeClient) ChatSimpleStream(ctx context.Context, model, systemPrompt, userMsg string) (string, error) {
	return "", nil
}
func (fakeClient) ChatSimpleStreamWithOptions(ctx context.Context, model, systemPrompt, userMsg string, opts ai.ChatSimpleOptions) (string, error) {
	return "", nil
}
func (fakeClient) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	return nil, nil
}

// TestNewLLM_EmptyKindDefaultsToWubigrok 验证「缺省行为与现状一致」：
// 空 kind 经 NewLLM 路由到缺省 DefaultLLMKind（wubigrok），并成功构造
// bridge provider（配置驱动选择，未配置 = 旧行为）。
func TestNewLLM_EmptyKindDefaultsToWubigrok(t *testing.T) {
	bridge.SetClient(fakeClient{})
	t.Cleanup(func() { bridge.SetClient(nil) })

	p, err := provider.NewLLM("", provider.Config{Name: "gaea"})
	if err != nil {
		t.Fatalf("NewLLM(empty kind): %v", err)
	}
	if p.Name() != "gaea" {
		t.Errorf("Name = %q, want gaea", p.Name())
	}
	// 构造结果必须是 LLM seam 接口（含 Chat），且 Chat 与 Stream 同一条路径。
	if _, ok := p.(provider.LLMProvider); !ok {
		t.Fatalf("NewLLM(empty kind) = %T, want provider.LLMProvider", p)
	}
	c, err := p.Chat(context.Background(), provider.Request{
		Messages: []provider.Message{{Role: provider.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if c.Content != "bridge-ok" {
		t.Errorf("Chat Content = %q, want bridge-ok", c.Content)
	}
}

// TestNewLLM_ExplicitWubigrokSameAsDefault 验证显式 "wubigrok" 与缺省等价
// （两者都走 bridge 工厂）。
func TestNewLLM_ExplicitWubigrokSameAsDefault(t *testing.T) {
	bridge.SetClient(fakeClient{})
	t.Cleanup(func() { bridge.SetClient(nil) })

	explicit, err := provider.NewLLM(provider.LLMKindWubigrok, provider.Config{Name: "x"})
	if err != nil {
		t.Fatalf("NewLLM(wubigrok): %v", err)
	}
	if explicit.Name() != "x" {
		t.Errorf("Name = %q, want x", explicit.Name())
	}
}
