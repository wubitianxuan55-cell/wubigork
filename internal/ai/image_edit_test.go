package ai

// 指令编辑测试：OpenAI 兼容后端 /images/edits multipart 形状与响应解析、
// 缺原图/非 data URL 诚实报错、GLM 拒绝文案，以及 OpenAI 蒙版口径。
// ComfyUI 的 Qwen Image 2.1 Turbo Edit 工作流与拒绝路径见 image_qwen21_edit_test.go。

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
		Model:     "cloud-edit-model",
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
	if gotForm["model"] != nil && gotForm["model"][0] != "cloud-edit-model" || gotForm["prompt"] != nil && gotForm["prompt"][0] != "把外套改成红色" {
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
		Model: "cloud-edit-model", Prompt: "只改左半", Mode: "edit",
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

// TestComfyUIResolveRefModeQedit 一致性方法路由（阶段三刀 A）：qedit 新分支、
// ipadapter/pulid 改口指向 qedit、img2img 模式优先。
func TestComfyUIResolveRefModeQedit(t *testing.T) {
	if m, err := comfyResolveRefMode("txt2img", "qedit"); err != nil || m != "qedit" {
		t.Fatalf("txt2img+qedit 应路由 qedit: %s %v", m, err)
	}
	if m, err := comfyResolveRefMode("txt2img", ""); err != nil || m != "img2img" {
		t.Fatalf("缺省应 img2img 近似: %s %v", m, err)
	}
	if m, err := comfyResolveRefMode("img2img", "qedit"); err != nil || m != "img2img" {
		t.Fatalf("img2img 模式应优先自身: %s %v", m, err)
	}
	_, err := comfyResolveRefMode("txt2img", "ipadapter")
	if err == nil || !strings.Contains(err.Error(), "qedit") {
		t.Fatalf("ipadapter 应改口指向 qedit: %v", err)
	}
}
