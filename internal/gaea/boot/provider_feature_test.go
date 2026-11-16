package boot

// provider_feature_test.go — 账目 feature 标签管道锁（2026-11-17 审查批5 刀1）：
// newProviderWithFeature 把标签送进 provider.Config.Extra，bridge 工厂转成
// 每请求 ai.ChatRequest.Feature（modelengine PerFeature 分桶的消费形态）。
// 图文 Build 传 opts.Feature="wenxue"；办公 NewProvider 不覆盖（"gaea" 现状）。

import (
	"context"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

type featureCaptureClient struct{ got *ai.ChatRequest }

func (c *featureCaptureClient) ChatStream(ctx context.Context, req *ai.ChatRequest) (<-chan ai.SSEChunk, error) {
	c.got = req
	ch := make(chan ai.SSEChunk, 1)
	ch <- ai.SSEChunk{Done: true}
	close(ch)
	return ch, nil
}
func (c *featureCaptureClient) ChatSimpleStream(ctx context.Context, model, system, user string) (string, error) {
	return "", nil
}
func (c *featureCaptureClient) ChatSimpleStreamWithOptions(ctx context.Context, model, system, user string, opts ai.ChatSimpleOptions) (string, error) {
	return "", nil
}
func (c *featureCaptureClient) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	return nil, nil
}

func TestBootProviderFeaturePlumbing(t *testing.T) {
	entry := &gaeaConfig.ProviderEntry{Name: "t", Kind: "wubigrok", Model: "m", ContextWindow: 1}

	t.Run("wenxue 标签随管道进请求", func(t *testing.T) {
		mc := &featureCaptureClient{}
		bridge.SetClient(mc)
		t.Cleanup(func() { bridge.SetClient(nil) })
		p, err := newProviderWithFeature(entry, "wenxue")
		if err != nil {
			t.Fatalf("newProviderWithFeature: %v", err)
		}
		if _, err := p.Stream(context.Background(), provider.Request{
			Messages: []provider.Message{{Role: provider.RoleUser, Content: "hi"}}, MaxTokens: 16,
		}); err != nil {
			t.Fatalf("Stream: %v", err)
		}
		if mc.got == nil || mc.got.Feature != "wenxue" {
			feature := ""
			if mc.got != nil {
				feature = mc.got.Feature
			}
			t.Fatalf("ChatRequest.Feature = %q, want wenxue", feature)
		}
	})

	t.Run("办公 NewProvider 缺省 gaea 逐字节", func(t *testing.T) {
		mc := &featureCaptureClient{}
		bridge.SetClient(mc)
		t.Cleanup(func() { bridge.SetClient(nil) })
		p, err := NewProvider(entry)
		if err != nil {
			t.Fatalf("NewProvider: %v", err)
		}
		if _, err := p.Stream(context.Background(), provider.Request{
			Messages: []provider.Message{{Role: provider.RoleUser, Content: "hi"}}, MaxTokens: 16,
		}); err != nil {
			t.Fatalf("Stream: %v", err)
		}
		if mc.got == nil || mc.got.Feature != "gaea" {
			feature := ""
			if mc.got != nil {
				feature = mc.got.Feature
			}
			t.Fatalf("办公缺省标签漂移: %q", feature)
		}
	})
}
