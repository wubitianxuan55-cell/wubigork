package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 3a Image seam 注册表测试：注册/构建/互斥/未知 kind/切换只改 kind。

func TestImageBackendRegistry_NewOpenAI(t *testing.T) {
	b, err := NewImageBackend(ImageBackendKindOpenAI, ImageBackendConfig{
		BaseURL: "http://localhost:8080/v1",
		APIKey:  "secret",
	})
	if err != nil {
		t.Fatalf("NewImageBackend(openai) 失败: %v", err)
	}
	if _, ok := b.(*OpenAIImageBackend); !ok {
		t.Fatalf("kind=openai 应返回 *OpenAIImageBackend，实际 %T", b)
	}
}

func TestImageBackendRegistry_NewComfyUI(t *testing.T) {
	b, err := NewImageBackend(ImageBackendKindComfyUI, ImageBackendConfig{
		BaseURL: "http://127.0.0.1:8188",
	})
	if err != nil {
		t.Fatalf("NewImageBackend(comfyui) 失败: %v", err)
	}
	if _, ok := b.(*ComfyUIBackend); !ok {
		t.Fatalf("kind=comfyui 应返回 *ComfyUIBackend，实际 %T", b)
	}
}

func TestImageBackendRegistry_FactoryRequiresBaseURL(t *testing.T) {
	for _, kind := range []string{ImageBackendKindOpenAI, ImageBackendKindComfyUI} {
		if _, err := NewImageBackend(kind, ImageBackendConfig{}); err == nil {
			t.Errorf("kind=%s 缺 BaseURL 应报错", kind)
		}
	}
}

func TestImageBackendRegistry_DuplicateKindPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("重复注册相同 kind 应 panic")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "duplicate image backend kind") {
			t.Fatalf("panic 信息异常: %v", r)
		}
	}()
	// openai 已被 init() 注册，重复注册触发互斥 panic
	RegisterImageBackend(ImageBackendKindOpenAI, func(cfg ImageBackendConfig) (ImageBackend, error) {
		return nil, nil
	})
}

func TestImageBackendRegistry_EmptyKindPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("空 kind 注册应 panic")
		}
	}()
	RegisterImageBackend("", func(cfg ImageBackendConfig) (ImageBackend, error) { return nil, nil })
}

func TestImageBackendRegistry_UnknownKindError(t *testing.T) {
	_, err := NewImageBackend("no-such-backend", ImageBackendConfig{BaseURL: "http://x"})
	if err == nil {
		t.Fatal("未知 kind 应返回错误")
	}
	if !strings.Contains(err.Error(), "no-such-backend") {
		t.Errorf("错误应包含未知 kind，实际: %v", err)
	}
	if !strings.Contains(err.Error(), ImageBackendKindOpenAI) {
		t.Errorf("错误应提示已注册 kind，实际: %v", err)
	}
}

func TestImageBackendRegistry_KindsSorted(t *testing.T) {
	kinds := ImageBackendKinds()
	// openai 与 comfyui 均由 init() 注册，且列表排序稳定
	joined := strings.Join(kinds, ",")
	if !strings.Contains(joined, ImageBackendKindOpenAI) || !strings.Contains(joined, ImageBackendKindComfyUI) {
		t.Fatalf("已注册 kinds = %v，缺 openai/comfyui", kinds)
	}
	for i := 1; i < len(kinds); i++ {
		if kinds[i-1] > kinds[i] {
			t.Fatalf("kinds 未排序: %v", kinds)
		}
	}
}

// TestImageBackendRegistry_SwitchKindOnly 验证「切换生图后端只改 kind 配置」：
// 消费方只依赖 ImageBackend 接口，同形配置 + 不同 kind 即得到不同后端实现，
// 代码零 switch 造实例。
func TestImageBackendRegistry_SwitchKindOnly(t *testing.T) {
	var consumer func(kind string) (ImageBackend, error)
	consumer = func(kind string) (ImageBackend, error) {
		return NewImageBackend(kind, ImageBackendConfig{BaseURL: "http://127.0.0.1:8188"})
	}

	openai, err := consumer(ImageBackendKindOpenAI)
	if err != nil {
		t.Fatalf("切换 kind=openai 失败: %v", err)
	}
	comfy, err := consumer(ImageBackendKindComfyUI)
	if err != nil {
		t.Fatalf("切换 kind=comfyui 失败: %v", err)
	}
	if _, ok := openai.(*OpenAIImageBackend); !ok {
		t.Errorf("kind=openai 应得 OpenAI 兼容后端，实际 %T", openai)
	}
	if _, ok := comfy.(*ComfyUIBackend); !ok {
		t.Errorf("kind=comfyui 应得 ComfyUI 后端，实际 %T", comfy)
	}
}

// TestImageBackendRegistry_NewOpenAI_Generates 端到端证明注册表构建的后端可用：
// 模拟 xAI/herdsman 兼容的 /v1/images/generations 服务（即 client.generateImageXAI
// 现在的走法）。
func TestImageBackendRegistry_NewOpenAI_Generates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("Authorization 头 = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"created": 123,
			"data":    []map[string]string{{"b64_json": "data:image/png;base64,AAAA"}},
		})
	}))
	defer srv.Close()

	backend, err := NewImageBackend(ImageBackendKindOpenAI, ImageBackendConfig{
		BaseURL: srv.URL + "/v1",
		APIKey:  "tok",
	})
	if err != nil {
		t.Fatalf("NewImageBackend(openai) 失败: %v", err)
	}
	resp, err := backend.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model:  "grok-imagine-image",
		Prompt: "test",
	})
	if err != nil {
		t.Fatalf("GenerateImage 失败: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].B64JSON != "data:image/png;base64,AAAA" {
		t.Fatalf("响应异常: %+v", resp.Data)
	}
}
