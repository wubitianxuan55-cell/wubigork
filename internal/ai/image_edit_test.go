package ai

// 指令编辑测试：阶段一刀 C（规格 进度计划/gaea-instruct-edit-20260917.md）——
// OpenAI 兼容后端 /images/edits multipart 形状与响应解析、缺原图/非 data URL
// 诚实报错、GLM 拒绝文案；阶段二刀 A（规格 进度计划/gaea-comfyui-edit-20260922.md）
// ——ComfyUI 本地档工作流形状（Qwen-Image-Edit 2511 官方模板）与缺权重提示。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Mode: "edit", Prompt: "改"})
	if err == nil || !strings.Contains(err.Error(), "指令编辑需要原图") {
		t.Fatalf("ComfyUI 缺原图应报错，得到: %v", err)
	}
}

// TestComfyUIBackend_EditWorkflowShape 指令编辑本地档（阶段二刀 A）：
// 全链 httptest——上传原图 → 提交工作流（捕获形状）→ history → view 出图。
// 断言按官方 image_qwen_image_edit_2511 模板：缩放图同时进 TextEncode×2 与
// VAEEncode；UNET/CLIP/VAE 文件名与 type；KSampler 20/4.0/1.0（denoise 恒 1）。
func TestComfyUIBackend_EditWorkflowShape(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	var workflow map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload/image":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"gaea_init_1.png"}`))
		case r.URL.Path == "/prompt":
			body, _ := io.ReadAll(r.Body)
			var req struct {
				Prompt map[string]interface{} `json:"prompt"`
			}
			if err := json.Unmarshal(body, &req); err != nil || req.Prompt == nil {
				t.Errorf("提交体形状异常: %v", err)
				http.Error(w, "bad", 400)
				return
			}
			workflow = req.Prompt
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"prompt_id":"pid-edit"}`))
		case r.URL.Path == "/history/pid-edit":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"pid-edit":{"outputs":{"12":{"images":[{"filename":"edited.png","subfolder":"","type":"output"}]}}}}`))
		case r.URL.Path == "/view":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(png)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b := NewComfyUIBackend(srv.URL)
	b.pollInterval = 5 * time.Millisecond
	resp, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Mode:      "edit",
		Prompt:    "把外套改成红色",
		Model:     "krea2", // 请求模型名在 edit 分支被忽略（编辑引擎≠生图模型）
		InitImage: editDataURL(),
	})
	if err != nil {
		t.Fatalf("edit 失败: %v", err)
	}
	if !strings.HasPrefix(resp.Data[0].B64JSON, "data:image/png;base64,") {
		t.Fatalf("响应应为 data URL: %s", resp.Data[0].B64JSON[:40])
	}
	node := func(id string) map[string]interface{} {
		n, _ := workflow[id].(map[string]interface{})
		if n == nil {
			t.Fatalf("工作流缺节点 %s", id)
		}
		return n
	}
	inputs := func(id string) map[string]interface{} {
		in, _ := node(id)["inputs"].(map[string]interface{})
		if in == nil {
			t.Fatalf("节点 %s 缺 inputs", id)
		}
		return in
	}
	// 原图上传后进 LoadImage；FluxKontextImageScale 吃 LoadImage
	if got, _ := inputs("1")["image"].(string); got != "gaea_init_1.png" {
		t.Fatalf("LoadImage 应吃上传图: %v", got)
	}
	if got := inputs("160"); got["image"].([]interface{})[0] != "1" {
		t.Fatalf("FluxKontextImageScale 应接 LoadImage: %v", got["image"])
	}
	// 正/负 TextEncodeQwenImageEditPlus 都吃缩放图（官方图：image1=160）
	for _, id := range []string{"7", "8"} {
		n := node(id)
		if n["class_type"] != "TextEncodeQwenImageEditPlus" {
			t.Fatalf("节点 %s 应为 TextEncodeQwenImageEditPlus: %v", id, n["class_type"])
		}
		in := inputs(id)
		if ref, _ := in["image1"].([]interface{}); len(ref) == 0 || ref[0] != "160" {
			t.Fatalf("节点 %s image1 应接缩放图: %v", id, in["image1"])
		}
		if ref, _ := in["vae"].([]interface{}); len(ref) == 0 || ref[0] != "6" {
			t.Fatalf("节点 %s vae 应接 VAELoader: %v", id, in["vae"])
		}
	}
	if got, _ := inputs("7")["prompt"].(string); got != "把外套改成红色" {
		t.Fatalf("正向提示词应透传: %v", got)
	}
	// 模型链：UNETLoader → AuraFlow(3.1) → CFGNorm(1) → KSampler
	if got, _ := inputs("4")["unet_name"].(string); got != "qwen_image_edit_2511_fp8mixed.safetensors" {
		t.Fatalf("UNET 文件名不对: %v", got)
	}
	if got, _ := inputs("5")["type"].(string); got != "qwen_image" {
		t.Fatalf("CLIP type 应为 qwen_image: %v", got)
	}
	if got, _ := inputs("6")["vae_name"].(string); got != "qwen_image_vae.safetensors" {
		t.Fatalf("VAE 文件名不对: %v", got)
	}
	if got, _ := inputs("145")["shift"].(float64); got != 3.1 {
		t.Fatalf("AuraFlow shift 应 3.1: %v", got)
	}
	if ref, _ := inputs("152")["model"].([]interface{}); len(ref) == 0 || ref[0] != "145" {
		t.Fatalf("CFGNorm 应接 AuraFlow: %v", inputs("152")["model"])
	}
	// KSampler：官方 Comfy 列参数 20 步 CFG 4.0，denoise 恒 1.0（语义编辑）
	ks := inputs("10")
	if ks["steps"] != float64(20) || ks["cfg"] != float64(4.0) || ks["denoise"] != float64(1.0) {
		t.Fatalf("KSampler 参数不对: %v", ks)
	}
	for _, pair := range [][2]string{{"model", "152"}, {"positive", "7"}, {"negative", "8"}, {"latent_image", "15"}} {
		if ref, _ := ks[pair[0]].([]interface{}); len(ref) == 0 || ref[0] != pair[1] {
			t.Fatalf("KSampler.%s 应接 %s: %v", pair[0], pair[1], ks[pair[0]])
		}
	}
	// VAEEncode 吃缩放图（不是原始 LoadImage——官方图 pixels=160）
	if ref, _ := inputs("15")["pixels"].([]interface{}); len(ref) == 0 || ref[0] != "160" {
		t.Fatalf("VAEEncode pixels 应接缩放图: %v", inputs("15")["pixels"])
	}
}

// TestComfyUIBackend_EditMissingModelHint 缺权重提示：ComfyUI 回
// value_not_in_list 时错误追加三件文件名 + HF 链接；普通错误不追加。
func TestComfyUIBackend_EditMissingModelHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/upload/image":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"gaea_init_1.png"}`))
		case "/prompt":
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"PromptOutputsFailedValidation: 'qwen_image_edit_2511_fp8mixed.safetensors' value_not_in_list"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b := NewComfyUIBackend(srv.URL)
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Mode: "edit", Prompt: "改", InitImage: editDataURL()})
	if err == nil {
		t.Fatal("缺模型应报错")
	}
	msg := err.Error()
	for _, want := range []string{"指令编辑本地档", "qwen_image_edit_2511_fp8mixed.safetensors", "huggingface.co/Comfy-Org/Qwen-Image-Edit_ComfyUI", "qwen_2.5_vl_7b_fp8_scaled.safetensors", "models/vae/qwen_image_vae.safetensors"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("缺权重提示应含 %q:\n%s", want, msg)
		}
	}
	// 普通 400（非 value_not_in_list/模型名）不追加提示
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/upload/image":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"gaea_init_1.png"}`))
		case "/prompt":
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`boom`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv2.Close()
	b2 := NewComfyUIBackend(srv2.URL)
	_, err2 := b2.GenerateImage(context.Background(), &ImageGenerationRequest{Mode: "edit", Prompt: "改", InitImage: editDataURL()})
	if err2 == nil || strings.Contains(err2.Error(), "huggingface") {
		t.Fatalf("普通错误不应追加缺权重提示: %v", err2)
	}
}
