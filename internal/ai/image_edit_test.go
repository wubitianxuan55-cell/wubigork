package ai

// 指令编辑测试：阶段一刀 C（规格 进度计划/gaea-instruct-edit-20260917.md）——
// OpenAI 兼容后端 /images/edits multipart 形状与响应解析、缺原图/非 data URL
// 诚实报错、GLM 拒绝文案；阶段二刀 A（规格 进度计划/gaea-comfyui-edit-20260922.md）
// ——ComfyUI 本地档工作流形状（Qwen-Image-Edit 2511 官方模板）与缺权重提示；
// 阶段二刀 B（规格 进度计划/gaea-mask-inpaint-20260923.md）——蒙版口径转换/
// 尺寸对齐复刻/双后端蒙版形状。

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
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

// realGrayPNGDataURL 生成真实灰度 PNG data URL（阶段二刀 B 测试用）：
// leftWhite=true 时左半白（重绘区）、右半黑（保留区）。
func realGrayPNGDataURL(t *testing.T, w, h int, leftWhite bool) string {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(0)
			if leftWhite && x < w/2 {
				v = 255
			}
			img.SetGray(x, y, color.Gray{Y: v})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// TestGrayMaskToOpenAIMask 蒙版口径转换：gaea 灰度（白=重绘）→ OpenAI
// （透明=编辑区）：白区 alpha=0、黑区 alpha=255、尺寸不变。
func TestGrayMaskToOpenAIMask(t *testing.T) {
	out, err := grayMaskToOpenAIMask(realGrayPNGDataURL(t, 2, 1, true))
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}
	m, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("转换产物不可解码: %v", err)
	}
	if m.Bounds().Dx() != 2 || m.Bounds().Dy() != 1 {
		t.Fatalf("尺寸应不变: %v", m.Bounds())
	}
	_, _, _, a0 := m.At(0, 0).RGBA()
	_, _, _, a1 := m.At(1, 0).RGBA()
	if a0 != 0 || a1 != 0xffff {
		t.Fatalf("白区应透明(0)、黑区应不透明(65535): %d %d", a0, a1)
	}
	// 非 data URL 蒙版：诚实报错
	if _, err := grayMaskToOpenAIMask("/tmp/m.png"); err == nil || !strings.Contains(err.Error(), "data URL") {
		t.Fatalf("非 data URL 应报错，得到: %v", err)
	}
}

// TestKontextScaleSize FluxKontextImageScale 尺寸选择复刻：方图→1024²；
// 2:1→1456×720（表内宽高比 2.0222 最近）；退化输入兜底 1024²。
func TestKontextScaleSize(t *testing.T) {
	for _, tc := range []struct {
		w, h, ew, eh int
	}{
		{1024, 1024, 1024, 1024},
		{200, 100, 1456, 720},
		{100, 200, 720, 1456},
		{0, 0, 1024, 1024},
		{-3, 7, 1024, 1024},
	} {
		w, h := kontextScaleSize(tc.w, tc.h)
		if w != tc.ew || h != tc.eh {
			t.Fatalf("kontextScaleSize(%d,%d) = %dx%d, 期望 %dx%d", tc.w, tc.h, w, h, tc.ew, tc.eh)
		}
	}
}

// TestOpenAIImageBackend_EditMaskMultipart 蒙版 multipart 形状：mask 字段
// 携带转换后的透明 PNG（白区 alpha=0）；image 字段原样在位。
func TestOpenAIImageBackend_EditMaskMultipart(t *testing.T) {
	var maskBytes []byte
	var hasImage bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/edits" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			return
		}
		if _, hdr, err := r.FormFile("image"); err == nil && hdr != nil {
			hasImage = true
		}
		if f, _, err := r.FormFile("mask"); err == nil {
			maskBytes, _ = io.ReadAll(f)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":1,"data":[{"b64_json":"` + "iVBORw0KGgo=" + `"}` + `]}`))
	}))
	defer srv.Close()

	b := NewOpenAIImageBackend(srv.URL+"/v1", "k1")
	maskURL := realGrayPNGDataURL(t, 4, 2, true)
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model: "qwen-image-edit", Prompt: "只改左半", Mode: "edit",
		InitImage: editDataURL(), Mask: maskURL,
	})
	if err != nil {
		t.Fatalf("edit+mask 失败: %v", err)
	}
	if !hasImage {
		t.Fatal("image 字段缺失")
	}
	if len(maskBytes) == 0 {
		t.Fatal("mask 字段缺失")
	}
	m, _, err := image.Decode(bytes.NewReader(maskBytes))
	if err != nil {
		t.Fatalf("mask 产物不可解码: %v", err)
	}
	if m.Bounds().Dx() != 4 || m.Bounds().Dy() != 2 {
		t.Fatalf("mask 尺寸应 4x2: %v", m.Bounds())
	}
	_, _, _, aLeft := m.At(0, 0).RGBA()
	_, _, _, aRight := m.At(3, 0).RGBA()
	if aLeft != 0 || aRight != 0xffff {
		t.Fatalf("mask 白区应透明、黑区不透明: %d %d", aLeft, aRight)
	}
}

// TestComfyUIBackend_EditMaskWorkflowShape 蒙版局部重绘工作流形状：
// 蒙版链 2→161(ImageScale 对齐缩放尺寸)→162(ImageToMask red)→163(SetLatentNoiseMask)；
// KSampler.latent_image 改接 163；无蒙版时零蒙版节点、latent 仍接 15（回归）。
func TestComfyUIBackend_EditMaskWorkflowShape(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	var workflow map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/upload/image":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"gaea_init_9.png"}`))
		case r.URL.Path == "/prompt":
			body, _ := io.ReadAll(r.Body)
			var req struct {
				Prompt map[string]interface{} `json:"prompt"`
			}
			_ = json.Unmarshal(body, &req)
			workflow = req.Prompt
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"prompt_id":"pid-mask"}`))
		case r.URL.Path == "/history/pid-mask":
			_, _ = w.Write([]byte(`{"pid-mask":{"outputs":{"12":{"images":[{"filename":"e.png","subfolder":"","type":"output"}]}}}}`))
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
	if _, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Mode: "edit", Prompt: "只改涂选区",
		InitImage: realGrayPNGDataURL(t, 200, 100, false), // 2:1 真图（DecodeConfig 需可解析）
		Mask:      realGrayPNGDataURL(t, 200, 100, true),
	}); err != nil {
		t.Fatalf("edit+mask 失败: %v", err)
	}
	inputs := func(id string) map[string]interface{} {
		n, _ := workflow[id].(map[string]interface{})
		if n == nil {
			t.Fatalf("工作流缺节点 %s", id)
		}
		in, _ := n["inputs"].(map[string]interface{})
		if in == nil {
			t.Fatalf("节点 %s 缺 inputs", id)
		}
		return in
	}
	if got, _ := inputs("2")["image"].(string); got != "gaea_init_9.png" {
		t.Fatalf("蒙版 LoadImage 应吃上传蒙版: %v", got)
	}
	is161 := inputs("161")
	if is161["upscale_method"] != "lanczos" || is161["width"] != float64(1456) || is161["height"] != float64(720) {
		t.Fatalf("ImageScale 应对齐 2:1 → 1456x720: %v", is161)
	}
	if n, _ := workflow["162"].(map[string]interface{}); n["class_type"] != "ImageToMask" || inputs("162")["channel"] != "red" {
		t.Fatalf("ImageToMask 形状不对: %+v", n)
	}
	if n, _ := workflow["163"].(map[string]interface{}); n["class_type"] != "SetLatentNoiseMask" {
		t.Fatalf("SetLatentNoiseMask 缺失: %+v", n)
	} else {
		if ref, _ := inputs("163")["samples"].([]interface{}); len(ref) == 0 || ref[0] != "15" {
			t.Fatalf("SetLatentNoiseMask.samples 应接 VAEEncode: %v", inputs("163")["samples"])
		}
	}
	if ref, _ := inputs("10")["latent_image"].([]interface{}); len(ref) == 0 || ref[0] != "163" {
		t.Fatalf("KSampler.latent_image 应接蒙版 latent: %v", inputs("10")["latent_image"])
	}
	// TextEncode image1 保持全图缩放图（语义参考不受蒙版限制）
	if ref, _ := inputs("7")["image1"].([]interface{}); len(ref) == 0 || ref[0] != "160" {
		t.Fatalf("TextEncode image1 应保持全图缩放图: %v", inputs("7")["image1"])
	}
}
