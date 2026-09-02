package ai

// image_dashscope_test.go — 百炼改图后端回归：官方 schema 干净请求体（多余字段
// 不发）、Bearer 认证、1 图 1 文 content、URL→dataURL 统一下载、官方扁平错误体
// 原样透出、txt2img/t2v 诚实拒绝、size 星号转换、注册表 fail-closed。

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashScopeImageBackend_RequestSchemaAndDownload(t *testing.T) {
	var gotBody map[string]interface{}
	var gotAuth string
	var gotPath string
	// 图片字节服务（响应里的 URL 会被后端下载转 data URL）
	imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/edit.png" {
			t.Errorf("应下载 /edit.png, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("editedbytes"))
	}))
	defer imgSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Write([]byte(`{"output":{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":[{"image":"` + imgSrv.URL + `/edit.png"}]}}]},"usage":{"image_count":1},"request_id":"req-1"}`))
	}))
	defer apiSrv.Close()

	initImage := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("imgbytes"))
	b := NewDashScopeImageBackend(apiSrv.URL, "ds-key")
	resp, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model:     "qwen-image-edit-plus",
		Prompt:    "把背景换成海滩",
		Mode:      "img2img",
		InitImage: initImage,
		// 通用 ImageGenerationRequest 的扩展字段：百炼官方 schema 不收，应全部忽略
		Negative: "blurry",
		Seed:     42,
		Lora:     "x.safetensors",
		Denoise:  0.7,
	})
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if gotPath != "/api/v1/services/aigc/multimodal-generation/generation" {
		t.Errorf("应请求官方多模态生成端点, got %s", gotPath)
	}
	if gotAuth != "Bearer ds-key" {
		t.Errorf("Authorization = %q, want Bearer ds-key", gotAuth)
	}
	// 官方 schema 只有 model/input/parameters
	for k := range gotBody {
		if k != "model" && k != "input" && k != "parameters" {
			t.Errorf("请求体不应含官方 schema 外字段 %q (body=%v)", k, gotBody)
		}
	}
	input, _ := gotBody["input"].(map[string]interface{})
	msgs, _ := input["messages"].([]interface{})
	if len(msgs) != 1 {
		t.Fatalf("messages 应恰好 1 条, got %d", len(msgs))
	}
	msg, _ := msgs[0].(map[string]interface{})
	if msg["role"] != "user" {
		t.Errorf("role = %v, want user", msg["role"])
	}
	content, _ := msg["content"].([]interface{})
	if len(content) != 2 {
		t.Fatalf("content 应为 1 图 + 1 文, got %d 项", len(content))
	}
	imgPart, _ := content[0].(map[string]interface{})
	txtPart, _ := content[1].(map[string]interface{})
	if imgPart["image"] != initImage {
		t.Errorf("content[0].image 应为参考图 data URL 原样, got %v", imgPart["image"])
	}
	if txtPart["text"] != "把背景换成海滩" {
		t.Errorf("content[1].text = %v, want 编辑指令", txtPart["text"])
	}
	// parameters：n=1、watermark=false 恒发；negative_prompt 只在配置时以
	// 官方字段名出现（顶层 Negative 等非官方字段不透传）；无 size（改图默认）
	params, _ := gotBody["parameters"].(map[string]interface{})
	if n, _ := params["n"].(float64); n != 1 {
		t.Errorf("parameters.n = %v, want 1", params["n"])
	}
	if wm, _ := params["watermark"].(bool); wm {
		t.Errorf("parameters.watermark 应为 false")
	}
	if np, _ := params["negative_prompt"].(string); np != "blurry" {
		t.Errorf("parameters.negative_prompt = %v, want blurry", params["negative_prompt"])
	}
	if _, ok := params["size"]; ok {
		t.Errorf("未配置时 parameters 不应含 %q", "size")
	}
	// 结果 URL 已统一下载为 data URL（URL 24h 过期 → 本地 data URL 可显示可落盘）
	if len(resp.Data) != 1 {
		t.Fatalf("data 应有 1 项, got %d", len(resp.Data))
	}
	if !strings.HasPrefix(resp.Data[0].B64JSON, "data:image/png;base64,") {
		t.Errorf("应下载为 data URL, got %q", resp.Data[0].B64JSON)
	}
	if resp.Data[0].URL != "" {
		t.Errorf("下载后应清空 URL, got %q", resp.Data[0].URL)
	}
	wantB64 := base64.StdEncoding.EncodeToString([]byte("editedbytes"))
	if !strings.HasSuffix(resp.Data[0].B64JSON, wantB64) {
		t.Errorf("data URL 内容不符")
	}
}

func TestDashScopeImageBackend_Txt2ImgAndT2VRejected(t *testing.T) {
	b := NewDashScopeImageBackend("http://127.0.0.1:1", "k")
	for _, mode := range []string{"txt2img", "t2v"} {
		// 本地不可达地址：若发出请求会以网络错误失败，这里应在请求前拒绝
		_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Mode: mode, Prompt: "x"})
		if err == nil || !strings.Contains(err.Error(), "仅支持改图") {
			t.Errorf("%s 应诚实报错「仅支持改图」, got %v", mode, err)
		}
	}
	// InitImage 非空即使 Mode 未显式声明也按改图放行（请求前拦截，不会触网成功）
	if _, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		InitImage: "data:image/png;base64,eA==", Prompt: "改",
	}); err != nil && strings.Contains(err.Error(), "仅支持改图") {
		t.Errorf("InitImage 非空应放行, got %v", err)
	}
}

func TestDashScopeImageBackend_ErrorSurface(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"code":"InvalidParameter","message":"输入图片格式不支持","request_id":"req-2"}`))
	}))
	defer srv.Close()

	b := NewDashScopeImageBackend(srv.URL, "k")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Mode:      "img2img",
		InitImage: "data:image/png;base64,eA==",
		Prompt:    "x",
	})
	if err == nil || !strings.Contains(err.Error(), "InvalidParameter") || !strings.Contains(err.Error(), "输入图片格式不支持") {
		t.Errorf("官方错误体应原样透出 code+message, got %v", err)
	}
}

func TestDashScopeImageBackend_SizeStarConversion(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Write([]byte(`{"output":{"choices":[{"message":{"content":[{"image":"http://127.0.0.1:1/placeholder.png"}]}}]}}`))
	}))
	defer srv.Close()

	b := NewDashScopeImageBackend(srv.URL, "k")
	if _, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Mode: "img2img", InitImage: "data:image/png;base64,eA==", Prompt: "x",
		Size:  "1024x1536",
		Model: "qwen-image-edit",
	}); err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	params, _ := gotBody["parameters"].(map[string]interface{})
	if params["size"] != "1024*1536" {
		t.Errorf("size 应转换为官方星号格式 1024*1536, got %v", params["size"])
	}

	// 空 size（改图默认）：不传，输出宽高比随参考图
	gotBody = nil
	if _, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Mode: "img2img", InitImage: "data:image/png;base64,eA==", Prompt: "x",
	}); err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	params, _ = gotBody["parameters"].(map[string]interface{})
	if _, ok := params["size"]; ok {
		t.Errorf("改图默认不应传 size, got %v", params["size"])
	}
}

func TestDashScopeImageBackend_DefaultModel(t *testing.T) {
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		gotModel, _ = body["model"].(string)
		w.Write([]byte(`{"output":{"choices":[{"message":{"content":[{"image":"http://127.0.0.1/x.png"}]}}]}}`))
	}))
	defer srv.Close()

	b := NewDashScopeImageBackend(srv.URL, "k")
	if _, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Mode: "img2img", InitImage: "data:image/png;base64,eA==", Prompt: "x",
	}); err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if gotModel != DashScopeDefaultImageModel {
		t.Errorf("空模型应回默认 %s, got %q", DashScopeDefaultImageModel, gotModel)
	}
}

func TestDashScopeImageBackend_OversizeInitImage(t *testing.T) {
	// >10MB 输入图：请求前诚实拒绝（不发起上传）
	big := strings.Repeat("A", DashScopeMaxInputImageBytes+1)
	b := NewDashScopeImageBackend("http://127.0.0.1:1", "k")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Mode:      "img2img",
		InitImage: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(big)),
		Prompt:    "x",
	})
	if err == nil || !strings.Contains(err.Error(), "10MB") {
		t.Errorf("超限参考图应在请求前报 10MB 限制, got %v", err)
	}
}

func TestDashScopeImageBackend_RegistryRequiresKey(t *testing.T) {
	if _, err := NewImageBackend(ImageBackendKindDashScope, ImageBackendConfig{BaseURL: DashScopeBaseURL}); err == nil {
		t.Error("缺 api_key 应 fail-closed")
	}
	if _, err := NewImageBackend(ImageBackendKindDashScope, ImageBackendConfig{APIKey: "k"}); err == nil {
		t.Error("缺 base_url 应 fail-closed")
	}
	backend, err := NewImageBackend(ImageBackendKindDashScope, ImageBackendConfig{BaseURL: DashScopeBaseURL, APIKey: "k"})
	if err != nil {
		t.Fatalf("合法配置不应报错: %v", err)
	}
	if _, ok := backend.(*DashScopeImageBackend); !ok {
		t.Errorf("应构建 DashScopeImageBackend, got %T", backend)
	}
}
