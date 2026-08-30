package ai

// image_glm_test.go — GLM 生图后端回归：官方 schema 干净请求体（多余字段不发）、
// Bearer 认证、URL→dataURL 统一下载、官方错误体原样透出、内容审核/图生图诚实报错。

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

func TestGLMImageBackend_RequestSchemaAndDownload(t *testing.T) {
	var gotBody map[string]interface{}
	var gotAuth string
	var gotPath string
	// 图片字节服务（响应里的 URL 会被后端下载转 data URL）
	imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gen.png" {
			t.Errorf("应下载 /gen.png, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("pngbytes"))
	}))
	defer imgSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Write([]byte(`{"created":1730000000,"data":[{"url":"` + imgSrv.URL + `/gen.png"}]}`))
	}))
	defer apiSrv.Close()

	b := NewGLMImageBackend(apiSrv.URL, "glm-key")
	resp, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model:  "cogview-4-250304",
		Prompt: "一只柯基",
		Size:   "1024x1024",
		// 通用 ImageGenerationRequest 的扩展字段：GLM 官方 schema 不收，应全部忽略
		Mode:     "txt2img",
		Negative: "blurry",
		N:        2,
		Seed:     42,
		Lora:     "x.safetensors",
	})
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if gotPath != "/images/generations" {
		t.Errorf("应请求 /images/generations, got %s", gotPath)
	}
	if gotAuth != "Bearer glm-key" {
		t.Errorf("Authorization = %q, want Bearer glm-key", gotAuth)
	}
	// 官方 schema 只有 model/prompt/size
	for k := range gotBody {
		if k != "model" && k != "prompt" && k != "size" {
			t.Errorf("请求体不应含官方 schema 外字段 %q (body=%v)", k, gotBody)
		}
	}
	if gotBody["model"] != "cogview-4-250304" || gotBody["prompt"] != "一只柯基" {
		t.Errorf("model/prompt = %v/%v", gotBody["model"], gotBody["prompt"])
	}
	// URL 已统一下载为 data URL
	if len(resp.Data) != 1 {
		t.Fatalf("data 应有 1 项, got %d", len(resp.Data))
	}
	if !strings.HasPrefix(resp.Data[0].B64JSON, "data:image/png;base64,") {
		t.Errorf("应下载为 data URL, got %q", resp.Data[0].B64JSON)
	}
	if resp.Data[0].URL != "" {
		t.Errorf("下载后应清空 URL, got %q", resp.Data[0].URL)
	}
	wantB64 := base64.StdEncoding.EncodeToString([]byte("pngbytes"))
	if !strings.HasSuffix(resp.Data[0].B64JSON, wantB64) {
		t.Errorf("data URL 内容不符")
	}
}

func TestGLMImageBackend_DefaultModel(t *testing.T) {
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		gotModel, _ = body["model"].(string)
		w.Write([]byte(`{"created":1,"data":[{"url":"http://127.0.0.1/x.png"}]}`))
	}))
	defer srv.Close()

	b := NewGLMImageBackend(srv.URL, "k")
	if _, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Prompt: "测试"}); err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if gotModel != GLMDefaultImageModel {
		t.Errorf("空模型应回默认 %s, got %q", GLMDefaultImageModel, gotModel)
	}
}

func TestGLMImageBackend_Img2ImgRejected(t *testing.T) {
	b := NewGLMImageBackend("http://127.0.0.1:1", "k")
	// 图生图应在发出请求前被拒绝（本地不可达地址：若发请求会以网络错误失败）
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Mode: "img2img", InitImage: "data:image/png;base64,xx", Prompt: "改图"})
	if err == nil || !strings.Contains(err.Error(), "仅支持文生图") {
		t.Errorf("图生图应诚实报错, got %v", err)
	}
}

func TestGLMImageBackend_OfficialErrorSurface(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":"1211","message":"模型不存在或不可用"}}`))
	}))
	defer srv.Close()

	b := NewGLMImageBackend(srv.URL, "k")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Model: "nope", Prompt: "x"})
	if err == nil || !strings.Contains(err.Error(), "1211") || !strings.Contains(err.Error(), "模型不存在或不可用") {
		t.Errorf("官方错误体应原样透出 code+message, got %v", err)
	}
}

func TestGLMImageBackend_EmptyDataContentFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"created":1,"data":[],"content_filter":[{"role":"assistant","level":0}]}`))
	}))
	defer srv.Close()

	b := NewGLMImageBackend(srv.URL, "k")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Prompt: "x"})
	if err == nil || !strings.Contains(err.Error(), "内容审核") {
		t.Errorf("200 无图应提示内容审核, got %v", err)
	}
}

func TestGLMImageBackend_RegistryRequiresKey(t *testing.T) {
	if _, err := NewImageBackend(ImageBackendKindGLM, ImageBackendConfig{BaseURL: "https://open.bigmodel.cn/api/paas/v4"}); err == nil {
		t.Error("缺 api_key 应 fail-closed")
	}
	if _, err := NewImageBackend(ImageBackendKindGLM, ImageBackendConfig{APIKey: "k"}); err == nil {
		t.Error("缺 base_url 应 fail-closed")
	}
	backend, err := NewImageBackend(ImageBackendKindGLM, ImageBackendConfig{BaseURL: "https://open.bigmodel.cn/api/paas/v4", APIKey: "k"})
	if err != nil {
		t.Fatalf("合法配置不应报错: %v", err)
	}
	if _, ok := backend.(*GLMImageBackend); !ok {
		t.Errorf("应构建 GLMImageBackend, got %T", backend)
	}
}
