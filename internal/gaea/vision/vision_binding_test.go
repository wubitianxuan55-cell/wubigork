package vision

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// saveResolver 快照并恢复包级动态解析钩子（功能绑定注入）。
func saveResolver(t *testing.T) {
	t.Helper()
	old := visionRuntimeResolver
	t.Cleanup(func() { visionRuntimeResolver = old })
}

// TestResolveVisionRuntime_ResolverPrecedence 动态解析钩子（办公识图功能绑定）
// 的优先级：钩子命中 → 覆盖静态注入；钩子弃权/未注入 → 回落静态注入值；
// 空字段统一补默认。
func TestResolveVisionRuntime_ResolverPrecedence(t *testing.T) {
	saveVisionRuntime(t)
	saveResolver(t)
	SetVisionRuntime(VisionRuntime{})

	// 未注入钩子：静态零值 → 全默认（本机 herdsman Qwen3.6 视觉模型）
	r := resolveVisionRuntime()
	if r.BaseURL != DefaultBaseURL || r.Model != DefaultModel || r.Kind != VisionProviderKindOpenAI {
		t.Errorf("未注入钩子 = %+v, want 全默认", r)
	}

	// 钩子命中：覆盖静态注入（改绑即生效）
	SetVisionRuntimeResolver(func() (VisionRuntime, bool) {
		return VisionRuntime{BaseURL: "http://10.0.0.2:9999/v1", Model: "bound-model", APIKey: "sk-b"}, true
	})
	r = resolveVisionRuntime()
	if r.BaseURL != "http://10.0.0.2:9999/v1" || r.Model != "bound-model" || r.APIKey != "sk-b" {
		t.Errorf("钩子命中 = %+v, want 绑定值", r)
	}
	if r.Kind != VisionProviderKindOpenAI {
		t.Errorf("钩子命中 kind = %q, want 默认补全 openai", r.Kind)
	}

	// 钩子弃权（绑定停用/引擎不可用）：回落静态注入
	SetVisionRuntimeResolver(func() (VisionRuntime, bool) { return VisionRuntime{}, false })
	r = resolveVisionRuntime()
	if r.BaseURL != DefaultBaseURL || r.Model != DefaultModel {
		t.Errorf("钩子弃权 = %+v, want 回落静态默认", r)
	}

	// 撤销钩子：与未注入等价
	SetVisionRuntimeResolver(nil)
	r = resolveVisionRuntime()
	if r.BaseURL != DefaultBaseURL || r.Model != DefaultModel {
		t.Errorf("撤销钩子 = %+v, want 静态默认", r)
	}
}

// TestRecognizeImage_BoundResolverAPIKey 绑定链路端到端：钩子返回的
// BaseURL/Model/APIKey 落到真实请求（Bearer 头 + model 字段）。
func TestRecognizeImage_BoundResolverAPIKey(t *testing.T) {
	saveVisionRuntime(t)
	saveResolver(t)
	SetVisionRuntime(VisionRuntime{})

	var gotAuth, gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotModel = body.Model
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"识图结果"}}]}`))
	}))
	defer srv.Close()

	SetVisionRuntimeResolver(func() (VisionRuntime, bool) {
		return VisionRuntime{BaseURL: srv.URL, Model: "bound-vision", APIKey: "sk-test-1"}, true
	})

	path := writeTestPNG(t, t.TempDir())
	text, err := RecognizeImage(context.Background(), path, "描述")
	if err != nil {
		t.Fatalf("RecognizeImage: %v", err)
	}
	if text != "识图结果" {
		t.Errorf("text = %q", text)
	}
	if gotAuth != "Bearer sk-test-1" {
		t.Errorf("Authorization = %q, want Bearer sk-test-1", gotAuth)
	}
	if gotModel != "bound-vision" {
		t.Errorf("model = %q, want bound-vision", gotModel)
	}
}

// TestRecognizeImage_UnboundNoAuthHeader 未绑定（钩子弃权）走本地默认：不带
// Authorization 头（本地服务免鉴权口径不回归）。
func TestRecognizeImage_UnboundNoAuthHeader(t *testing.T) {
	saveVisionRuntime(t)
	saveResolver(t)
	SetVisionRuntime(VisionRuntime{})

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	SetVisionRuntimeResolver(func() (VisionRuntime, bool) { return VisionRuntime{}, false })
	SetVisionRuntime(VisionRuntime{BaseURL: srv.URL, Model: "local-m"})

	path := writeTestPNG(t, t.TempDir())
	if _, err := RecognizeImage(context.Background(), path, "p"); err != nil {
		t.Fatalf("RecognizeImage: %v", err)
	}
	if gotAuth != "" {
		t.Errorf("Authorization = %q, want 空（本地免鉴权不带头）", gotAuth)
	}
}
