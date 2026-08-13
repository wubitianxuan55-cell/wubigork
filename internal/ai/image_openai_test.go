package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestOpenAIImageBackend_ConvertsRelativeURLToDataURL 验证 herdsman 返回相对 url
// （如 /v1/images/cache/xxx.png）时，客户端会基于 baseURL 解析并下载为 data URL。
func TestOpenAIImageBackend_ConvertsRelativeURLToDataURL(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	gotGen := ""
	gotCache := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/images/generations":
			gotGen = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"created": 123,
				"data": []map[string]string{
					{"url": "/v1/images/cache/abc.png"},
				},
			})
		case "/v1/images/cache/abc.png":
			gotCache = r.URL.Path
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(png)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b := NewOpenAIImageBackend(srv.URL+"/v1", "")
	resp, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Model: "zimage-turbo", Prompt: "test"})
	if err != nil {
		t.Fatalf("GenerateImage 失败: %v", err)
	}
	if gotGen != "/v1/images/generations" {
		t.Errorf("generations 路径 = %q", gotGen)
	}
	if gotCache != "/v1/images/cache/abc.png" {
		t.Errorf("cache 路径 = %q", gotCache)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data 数量 = %d", len(resp.Data))
	}
	d := resp.Data[0]
	if !strings.HasPrefix(d.B64JSON, "data:image/png;base64,") {
		t.Errorf("B64JSON 前缀异常: %s", d.B64JSON)
	}
	if d.URL != "" {
		t.Errorf("转换后应清空 URL，实际 %q", d.URL)
	}
}

// TestOpenAIImageBackend_B64JSONPassthrough 验证服务端直接返回 b64_json 时不做额外下载。
func TestOpenAIImageBackend_B64JSONPassthrough(t *testing.T) {
	cacheHits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/images/generations":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"created": 123,
				"data": []map[string]string{
					{"b64_json": "data:image/png;base64,AAAA"},
				},
			})
		default:
			cacheHits++
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b := NewOpenAIImageBackend(srv.URL+"/v1", "")
	resp, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Model: "zimage-turbo", Prompt: "test"})
	if err != nil {
		t.Fatalf("GenerateImage 失败: %v", err)
	}
	if resp.Data[0].B64JSON != "data:image/png;base64,AAAA" {
		t.Errorf("B64JSON 被改写: %q", resp.Data[0].B64JSON)
	}
	if cacheHits != 0 {
		t.Errorf("不应触发额外下载，实际命中 %d 次", cacheHits)
	}
}

// TestOpenAIImageBackend_Img2Img 验证图生图走 /v1/images/img2img（JSON + image 字段），
// 并将 URL 响应转换为 data URL。
func TestOpenAIImageBackend_Img2Img(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/images/img2img":
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"created": 123,
				"data": []map[string]string{
					{"url": "/v1/images/cache/out.png"},
				},
			})
		case "/v1/images/cache/out.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(png)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b := NewOpenAIImageBackend(srv.URL+"/v1", "")
	resp, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model:     "zimage-turbo",
		Prompt:    "green apple",
		N:         1,
		Size:      "512x512",
		Mode:      "img2img",
		InitImage: "data:image/png;base64,AAAA",
	})
	if err != nil {
		t.Fatalf("GenerateImage 失败: %v", err)
	}
	if gotBody == nil {
		t.Fatal("未收到 img2img 请求体")
	}
	if gotBody["image"] != "data:image/png;base64,AAAA" {
		t.Errorf("image 字段 = %v", gotBody["image"])
	}
	if gotBody["size"] != "512x512" {
		t.Errorf("size 字段 = %v", gotBody["size"])
	}
	if gotBody["model"] != "zimage-turbo" {
		t.Errorf("model 字段 = %v", gotBody["model"])
	}
	if !strings.HasPrefix(resp.Data[0].B64JSON, "data:image/png;base64,") {
		t.Errorf("B64JSON 前缀异常: %s", resp.Data[0].B64JSON)
	}
}

// TestOpenAIImageBackend_Img2ImgRequiresInitImage 验证缺参考图时直接报错。
func TestOpenAIImageBackend_Img2ImgRequiresInitImage(t *testing.T) {
	b := NewOpenAIImageBackend("http://example.invalid/v1", "")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model:  "zimage-turbo",
		Prompt: "green apple",
		Mode:   "img2img",
	})
	if err == nil || !strings.Contains(err.Error(), "参考图") {
		t.Fatalf("应提示缺少参考图，实际: %v", err)
	}
}
