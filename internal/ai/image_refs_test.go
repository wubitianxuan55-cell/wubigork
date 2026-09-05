package ai

import (
	"context"
	"strings"
	"testing"
)

// ── T2 参考槽数据层测试（无调用方 = 现状行为零变化，语义可测）──

func TestComfyResolveRefMode(t *testing.T) {
	cases := []struct {
		mode, ref string
		want      string
		wantErr   bool
	}{
		{mode: "txt2img", ref: "", want: "img2img"},
		{mode: "txt2img", ref: "img2img", want: "img2img"},
		{mode: "img2img", ref: "ipadapter", want: "img2img"}, // 已图生图：方法不影响主流程
		{mode: "txt2img", ref: "ipadapter", wantErr: true},
		{mode: "txt2img", ref: "pulid", wantErr: true},
		{mode: "txt2img", ref: "no-such-method", wantErr: true},
	}
	for _, c := range cases {
		got, err := comfyResolveRefMode(c.mode, c.ref)
		if c.wantErr {
			if err == nil {
				t.Fatalf("%s/%s 应报错", c.mode, c.ref)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Fatalf("%s/%s = %q err=%v, want %q", c.mode, c.ref, got, err, c.want)
		}
	}
}

func TestGLMImageBackendRejectsRefImages(t *testing.T) {
	b := NewGLMImageBackend("http://example.invalid/v4", "test-key")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model:     "glm-image",
		Prompt:    "p",
		RefImages: []string{"data:image/png;base64,AAAA"},
	})
	if err == nil || !strings.Contains(err.Error(), "参考图") {
		t.Fatalf("GLM 带参考图应诚实报错，got %v", err)
	}
}

func TestOpenAIImageBackendRejectsTxtImgRefs(t *testing.T) {
	b := NewOpenAIImageBackend("http://example.invalid/v1", "")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model:     "some-cloud-model",
		Prompt:    "p",
		RefImages: []string{"data:image/png;base64,AAAA"},
	})
	if err == nil || !strings.Contains(err.Error(), "参考槽") {
		t.Fatalf("OpenAI 兼容文生图带参考图应诚实报错，got %v", err)
	}
}
