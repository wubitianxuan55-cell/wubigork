package ai

// 指令编辑（阶段一刀 C，规格 进度计划/gaea-instruct-edit-20260917.md）测试：
// OpenAI 兼容后端 /images/edits multipart 形状与响应解析、缺原图/非 data URL
// 诚实报错、GLM 与 ComfyUI 拒绝文案。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func editDataURL() string {
	// 8 字节 PNG 头的 base64（内容无关紧要，形状对即可）
	return "data:image/png;base64,iVBORw0KGgo="
}

func TestOpenAIImageBackend_EditMultipartShape(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	var gotPath string
	var gotForm map[string][]string
	var gotFile []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/images/edits":
			gotPath = r.URL.Path
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("ParseMultipartForm: %v", err)
				return
			}
			gotForm = r.Form
			f, hdr, err := r.FormFile("image")
			if err != nil {
				t.Errorf("FormFile(image): %v", err)
				return
			}
			defer f.Close()
			gotFile, _ = io.ReadAll(f)
			_ = hdr
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"created": 123,
				"data": []map[string]string{
					{"url": "/v1/images/cache/edited.png"},
				},
			})
		case "/v1/images/cache/edited.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(png)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b := NewOpenAIImageBackend(srv.URL+"/v1", "k1")
	resp, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model:     "qwen-image-edit",
		Prompt:    "把外套改成红色",
		N:         1,
		Size:      "1024x1024",
		Mode:      "edit",
		InitImage: editDataURL(),
	})
	if err != nil {
		t.Fatalf("edit 失败: %v", err)
	}
	if gotPath != "/v1/images/edits" {
		t.Fatalf("应打 edits 端点: %s", gotPath)
	}
	if gotForm["model"] != nil && gotForm["model"][0] != "qwen-image-edit" || gotForm["prompt"] != nil && gotForm["prompt"][0] != "把外套改成红色" {
		t.Fatalf("表单字段不对: %+v", gotForm)
	}
	if (gotForm["size"] == nil || gotForm["size"][0] != "1024x1024") || (gotForm["n"] == nil || gotForm["n"][0] != "1") {
		t.Fatalf("n/size 字段不对: %+v", gotForm)
	}
	if len(gotFile) == 0 {
		t.Fatal("image 文件为空")
	}
	if !strings.HasPrefix(resp.Data[0].B64JSON, "data:image/png;base64,") {
		t.Fatalf("响应应走 url→dataURL 归一: %s", resp.Data[0].B64JSON)
	}
}

func TestOpenAIImageBackend_EditRequiresDataURL(t *testing.T) {
	b := NewOpenAIImageBackend("http://127.0.0.1:1/v1", "")
	if _, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Mode: "edit", Prompt: "改"}); err == nil || !strings.Contains(err.Error(), "需要原图") {
		t.Fatalf("缺原图应报错，得到: %v", err)
	}
	if _, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Mode: "edit", Prompt: "改", InitImage: "/tmp/a.png"}); err == nil || !strings.Contains(err.Error(), "data URL") {
		t.Fatalf("非 data URL 应报错，得到: %v", err)
	}
}

func TestGLMImageBackend_EditRejected(t *testing.T) {
	b := NewGLMImageBackend("http://127.0.0.1:1", "k")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Mode: "edit", InitImage: editDataURL(), Prompt: "改"})
	if err == nil || !strings.Contains(err.Error(), "指令编辑") {
		t.Fatalf("GLM 应诚实拒绝编辑，得到: %v", err)
	}
}

func TestComfyUIBackend_EditRejected(t *testing.T) {
	b := NewComfyUIBackend("http://127.0.0.1:1")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Mode: "edit", InitImage: editDataURL(), Prompt: "改"})
	if err == nil || !strings.Contains(err.Error(), "指令编辑") {
		t.Fatalf("ComfyUI 应诚实拒绝编辑，得到: %v", err)
	}
}
