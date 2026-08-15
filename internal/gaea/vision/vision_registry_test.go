package vision

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// saveVisionRuntime 快照并恢复包级 visionRuntime（config 注入）。
func saveVisionRuntime(t *testing.T) {
	t.Helper()
	old := visionRuntime
	t.Cleanup(func() { visionRuntime = old })
}

// TestVisionRegistry_AllKinds "openai" kind 经注册表构建为 *openAIProvider。
func TestVisionRegistry_AllKinds(t *testing.T) {
	kinds := VisionProviderKinds()
	if len(kinds) != 1 || kinds[0] != VisionProviderKindOpenAI {
		t.Fatalf("VisionProviderKinds = %v, want [openai]", kinds)
	}
	p, err := NewVisionProvider(VisionProviderKindOpenAI, VisionProviderConfig{BaseURL: DefaultBaseURL, Model: DefaultModel})
	if err != nil {
		t.Fatalf("NewVisionProvider(openai): %v", err)
	}
	if _, ok := p.(*openAIProvider); !ok {
		t.Fatalf("kind=openai 应返回 *openAIProvider, got %T", p)
	}
}

// TestVisionRegistry_ConfigRouting 同形配置 + 不同 kind 得到不同实现：
// 切换后端只改 kind，消费方（Provider 接口）零改动。
func TestVisionRegistry_ConfigRouting(t *testing.T) {
	var consumer func(kind string) (Provider, error)
	consumer = func(kind string) (Provider, error) {
		return NewVisionProvider(kind, VisionProviderConfig{BaseURL: "http://127.0.0.1:8080/v1"})
	}
	p, err := consumer(VisionProviderKindOpenAI)
	if err != nil {
		t.Fatalf("consumer(openai): %v", err)
	}
	if _, ok := p.(*openAIProvider); !ok {
		t.Errorf("consumer(openai) 应返回 *openAIProvider, got %T", p)
	}
}

// TestVisionRegistry_UnknownKindError 未知 kind fail-closed（附已注册列表）。
func TestVisionRegistry_UnknownKindError(t *testing.T) {
	_, err := NewVisionProvider("no-such-vision", VisionProviderConfig{BaseURL: "http://x"})
	if err == nil {
		t.Fatal("未知 kind 应报错")
	}
	if !strings.Contains(err.Error(), VisionProviderKindOpenAI) {
		t.Errorf("错误应附已注册 kind 列表: %v", err)
	}
}

// TestVisionRegistry_DuplicateKindPanics 互斥注册：重复即 panic。
func TestVisionRegistry_DuplicateKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("重复注册应 panic")
		}
	}()
	RegisterVisionProvider(VisionProviderKindOpenAI, func(cfg VisionProviderConfig) (Provider, error) { return nil, nil })
}

// TestVisionRegistry_EmptyKindPanics 空 kind 注册直接 panic。
func TestVisionRegistry_EmptyKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("空 kind 应 panic")
		}
	}()
	RegisterVisionProvider("", func(cfg VisionProviderConfig) (Provider, error) { return nil, nil })
}

// TestRecognizeImage_ConfigRouting RecognizeImage 由 SetVisionRuntime 注入的
// config 路由（不再读 GAEA_VISION_* 环境变量）；端到端打到测试服务器。
func TestRecognizeImage_ConfigRouting(t *testing.T) {
	saveVisionRuntime(t)
	path := writeTestPNG(t, t.TempDir())

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Model != "vision-test-model" {
			t.Errorf("model = %q, want vision-test-model（config 注入生效）", body.Model)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"描述成功"}}]}`))
	}))
	defer srv.Close()

	SetVisionRuntime(VisionRuntime{Kind: VisionProviderKindOpenAI, BaseURL: srv.URL, Model: "vision-test-model"})
	text, err := RecognizeImage(context.Background(), path, "这是什么")
	if err != nil {
		t.Fatalf("RecognizeImage: %v", err)
	}
	if text != "描述成功" {
		t.Errorf("text = %q, want 描述成功", text)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", gotPath)
	}
}

// TestRecognizeImage_UnknownKindFailsClosed 未知 kind fail-closed：
// 拒绝运行（返回错误），不静默降级。
func TestRecognizeImage_UnknownKindFailsClosed(t *testing.T) {
	saveVisionRuntime(t)
	path := writeTestPNG(t, t.TempDir())
	SetVisionRuntime(VisionRuntime{Kind: "no-such-vision"})
	_, err := RecognizeImage(context.Background(), path, "p")
	if err == nil {
		t.Fatal("未知 kind 应报错（fail-closed）")
	}
	if !strings.Contains(err.Error(), "no-such-vision") {
		t.Errorf("错误应点名未知 kind: %v", err)
	}
}

// TestRecognizeImage_DefaultConfig 未注入 config 时回落默认端点/模型（与旧
// GAEA_VISION_* 缺省行为一致：DefaultBaseURL + DefaultModel，kind=openai）。
func TestRecognizeImage_DefaultConfig(t *testing.T) {
	saveVisionRuntime(t)
	r := resolveVisionRuntime()
	if r.Kind != VisionProviderKindOpenAI || r.BaseURL != DefaultBaseURL || r.Model != DefaultModel {
		t.Errorf("默认 runtime = %+v, want kind=openai base=%s model=%s", r, DefaultBaseURL, DefaultModel)
	}
	// 部分注入：只注入 base 时 model/kind 仍回落默认。
	SetVisionRuntime(VisionRuntime{BaseURL: "http://127.0.0.1:9999/v1"})
	r2 := resolveVisionRuntime()
	if r2.BaseURL != "http://127.0.0.1:9999/v1" || r2.Model != DefaultModel || r2.Kind != VisionProviderKindOpenAI {
		t.Errorf("部分注入 runtime = %+v, want base 注入 + model/kind 默认", r2)
	}
}
