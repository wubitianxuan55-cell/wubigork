package ai

// IN2-07（2026-10-02 全仓审计·批次二十）：图片 backendType 双套口径收敛——
// 最小收窄路线。注册表 kind（ImageBackendKind*，构造用）与运行时类型名
// （ImageBackendType*，SetImageBackend 第二参数 / GetImageBackendType 返回值 /
// 配置 image_backend 落盘值 / 前端 backend 字段）是两套口径，对照表钉在
// image_backend.go 常量块。本文件钉住：
//  1. 类型名常量 = 历史字面量（防改名漂移：改一个字符即红）；
//  2. SetImageBackend/GetImageBackendType 对每个类型名原样往返 + 空值回退 xAI；
//  3. 两套口径的边界事实：comfyui/glm 与 kind 同拼写，xai/herdsman/ollama
//     不是注册表 kind（它们共用 ImageBackendKindOpenAI 构造，引擎名只靠类型名承载）。

import "testing"

func TestImageBackendTypeConstants_LiteralConformance(t *testing.T) {
	// 历史字面量逐字对照（配置文件/前端消费的正是这些字符串，改名=外部破坏）
	want := map[string]string{
		ImageBackendTypeXAI:      "xai",
		ImageBackendTypeComfyUI:  "comfyui",
		ImageBackendTypeHerdsman: "herdsman",
		ImageBackendTypeOllama:   "ollama",
		ImageBackendTypeGLM:      "glm",
	}
	for got, exp := range want {
		if got != exp {
			t.Errorf("ImageBackendType 常量漂移: got %q, want %q", got, exp)
		}
	}
}

func TestImageBackendType_RoundTripAndDefault(t *testing.T) {
	names := []string{
		ImageBackendTypeXAI,
		ImageBackendTypeComfyUI,
		ImageBackendTypeHerdsman,
		ImageBackendTypeOllama,
		ImageBackendTypeGLM,
	}
	for _, name := range names {
		c := &Client{}
		c.SetImageBackend(nil, name)
		if got := c.GetImageBackendType(); got != name {
			t.Errorf("SetImageBackend(%q) 后 GetImageBackendType = %q", name, got)
		}
	}
	// 空值回退口径（单点在 SetImageBackend/GetImageBackendType 内部）
	c := &Client{}
	c.SetImageBackend(nil, "")
	if got := c.GetImageBackendType(); got != ImageBackendTypeXAI {
		t.Errorf("空 backendType 应回退 %q, got %q", ImageBackendTypeXAI, got)
	}
	// 全新 client（从未 Set）也回退 xAI
	if got := (&Client{}).GetImageBackendType(); got != ImageBackendTypeXAI {
		t.Errorf("零值 client GetImageBackendType = %q, want %q", got, ImageBackendTypeXAI)
	}
}

func TestImageBackendType_VsRegistryKind_TwoConventions(t *testing.T) {
	// 同拼写：comfyui/glm 类型名与注册表 kind 一致
	if ImageBackendTypeComfyUI != ImageBackendKindComfyUI {
		t.Errorf("comfyui 类型名 %q 与注册表 kind %q 应同拼写", ImageBackendTypeComfyUI, ImageBackendKindComfyUI)
	}
	if ImageBackendTypeGLM != ImageBackendKindGLM {
		t.Errorf("glm 类型名 %q 与注册表 kind %q 应同拼写", ImageBackendTypeGLM, ImageBackendKindGLM)
	}
	// 边界事实：xai/herdsman/ollama 不是注册表 kind（对照表：三者构造都归
	// ImageBackendKindOpenAI，引擎身份只由类型名承载）
	registered := map[string]bool{}
	for _, k := range ImageBackendKinds() {
		registered[k] = true
	}
	if !registered[ImageBackendKindOpenAI] {
		t.Fatalf("注册表应含 kind %q（openai 兼容构造入口），got %v", ImageBackendKindOpenAI, ImageBackendKinds())
	}
	for _, name := range []string{ImageBackendTypeXAI, ImageBackendTypeHerdsman, ImageBackendTypeOllama} {
		if registered[name] {
			t.Errorf("类型名 %q 不应同时是注册表 kind（两套口径边界被打破）", name)
		}
	}
	// 对照表的落点：openai 兼容 trio 的实例都经 ImageBackendKindOpenAI 工厂可构造
	if _, err := NewImageBackend(ImageBackendKindOpenAI, ImageBackendConfig{BaseURL: "http://127.0.0.1:1/v1"}); err != nil {
		t.Errorf("ImageBackendKindOpenAI 工厂应可构造（herdsman/ollama/xai 的 kind 归宿）: %v", err)
	}
}
