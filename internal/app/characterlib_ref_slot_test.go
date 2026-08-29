package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/modelengine"
)

// TestBuildPortraitImageRequest_RefSlot 生图参考槽请求构造契约（纯函数）：
//   - 无参考图 → Mode 为空（txt2img 默认）、InitImage 空、Denoise 0；
//   - 有参考图 → Mode=img2img、InitImage 透传、Denoise=0.55（保留角色特征）；
//   - comfyui 后端带 1024x1024，其它后端尺寸留空（与既有行为一致）。
func TestBuildPortraitImageRequest_RefSlot(t *testing.T) {
	c := characterlib.Character{Name: "苏念", Gender: "female"}
	ref := "data:image/png;base64,REVGRklNQUdF"

	// 无参考图：txt2img 路径
	noRef := buildPortraitImageRequest(c, "krea2", "comfyui", "")
	if noRef.Mode != "" || noRef.InitImage != "" || noRef.Denoise != 0 {
		t.Fatalf("无参考图应保持 txt2img: %+v", noRef)
	}
	if noRef.Size != "1024x1024" {
		t.Fatalf("comfyui 应带尺寸: %s", noRef.Size)
	}

	// 有参考图：img2img 路径
	withRef := buildPortraitImageRequest(c, "krea2", "comfyui", ref)
	if withRef.Mode != "img2img" {
		t.Fatalf("有参考图应为 img2img: %+v", withRef)
	}
	if withRef.InitImage != ref {
		t.Fatalf("InitImage 应透传参考图: %s", withRef.InitImage)
	}
	if withRef.Denoise != portraitRefDenoise || withRef.Denoise < 0.5 || withRef.Denoise > 0.65 {
		t.Fatalf("denoise 应在 0.5-0.65 区间: %v", withRef.Denoise)
	}
	if withRef.Size != "1024x1024" {
		t.Fatalf("comfyui img2img 应带尺寸: %s", withRef.Size)
	}

	// 非 comfyui 后端尺寸留空
	herdsman := buildPortraitImageRequest(c, "krea2", "herdsman", ref)
	if herdsman.Size != "" {
		t.Fatalf("herdsman 不应带尺寸: %s", herdsman.Size)
	}
}

// TestCheckPortraitRefSupport_ModelGate 参考图模型门禁：comfyui 仅
// krea2 / z-image-turbo 可用（与 image_comfyui.go img2img 白名单一致），
// 其它后端不提前拦截（交给后端/API 裁决）。
func TestCheckPortraitRefSupport_ModelGate(t *testing.T) {
	if err := checkPortraitRefSupport("comfyui", "flux"); err == nil {
		t.Fatal("comfyui + flux 应报错")
	} else if !strings.Contains(err.Error(), "暂不支持参考图生成") {
		t.Fatalf("错误文案不符: %v", err)
	}
	if err := checkPortraitRefSupport("comfyui", "krea2"); err != nil {
		t.Fatalf("comfyui + krea2 应放行: %v", err)
	}
	if err := checkPortraitRefSupport("comfyui", "krea2-flux-8"); err != nil {
		t.Fatalf("comfyui + krea2 前缀模型应放行: %v", err)
	}
	if err := checkPortraitRefSupport("comfyui", "z-image-turbo"); err != nil {
		t.Fatalf("comfyui + z-image-turbo 应放行: %v", err)
	}
	// 非 comfyui 后端不拦截（herdsman/xai 由 API 自行裁决）
	if err := checkPortraitRefSupport("herdsman", "flux"); err != nil {
		t.Fatalf("herdsman 不应拦截模型: %v", err)
	}
	if err := checkPortraitRefSupport("", "flux"); err != nil {
		t.Fatalf("空后端（跟随绘梦）不应拦截: %v", err)
	}
}

// newRefSlotTestServer 记录最后一次请求路径与请求体，并回包一张 b64 图。
func newRefSlotTestServer(t *testing.T) (*httptest.Server, *sync.Mutex, *string, *[]byte) {
	t.Helper()
	var mu sync.Mutex
	lastPath := ""
	var lastBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		lastPath = r.URL.Path
		lastBody = body
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"data:image/png;base64,QUJD"}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &mu, &lastPath, &lastBody
}

// newRefSlotTestApp 构造走 herdsman 引擎（httptest 后端）的剧照生成 App。
func newRefSlotTestApp(t *testing.T, srv *httptest.Server) *App {
	t.Helper()
	a := newCharacterLibTestApp(t)
	a.cfg.PortraitBackend = "herdsman"
	a.cfg.PortraitModel = "krea2"
	a.engineMgr = modelengine.NewManager("", "")
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{ID: "herdsman", BaseURL: srv.URL, Enabled: true}); err != nil {
		t.Fatalf("配置 herdsman 引擎: %v", err)
	}
	return a
}

// TestCharacterGeneratePortraitWithRef_Img2ImgPath 有参考图路径：
// CharacterGeneratePortraitWithRef 带参考图 → 请求走 /images/img2img，
// 请求体 image 字段为参考图 data URL、model 透传。
func TestCharacterGeneratePortraitWithRef_Img2ImgPath(t *testing.T) {
	srv, mu, lastPath, lastBody := newRefSlotTestServer(t)
	a := newRefSlotTestApp(t, srv)

	chJSON := `{"id":"c1","name":"苏念","gender":"female","appearance":"长发"}`
	ref := "data:image/png;base64,UkVGRklNQUdF"

	img, err := a.CharacterGeneratePortraitWithRef(chJSON, "", ref)
	if err != nil {
		t.Fatalf("参考图生成失败: %v", err)
	}
	if img != "data:image/png;base64,QUJD" {
		t.Fatalf("返回图片不符: %s", img)
	}

	mu.Lock()
	defer mu.Unlock()
	if *lastPath != "/images/img2img" {
		t.Fatalf("应走 img2img 端点, got %s", *lastPath)
	}
	var req map[string]interface{}
	if err := json.Unmarshal(*lastBody, &req); err != nil {
		t.Fatalf("解析请求体: %v (%s)", err, string(*lastBody))
	}
	if req["image"] != ref {
		t.Fatalf("image 字段应为参考图 data URL: %v", req["image"])
	}
	if req["model"] != "krea2" {
		t.Fatalf("model 字段应透传: %v", req["model"])
	}
}

// TestCharacterGeneratePortraitWithRef_Txt2ImgWhenNoRef 无参考图路径：
// 参考图为空 → 与 CharacterGeneratePortrait 一致走 /images/generations
// （txt2img 默认），请求体不含 image 字段。
func TestCharacterGeneratePortraitWithRef_Txt2ImgWhenNoRef(t *testing.T) {
	srv, mu, lastPath, lastBody := newRefSlotTestServer(t)
	a := newRefSlotTestApp(t, srv)

	chJSON := `{"id":"c1","name":"苏念","gender":"female"}`
	if _, err := a.CharacterGeneratePortraitWithRef(chJSON, "", "  "); err != nil {
		t.Fatalf("无参考图生成失败: %v", err)
	}
	if _, err := a.CharacterGeneratePortrait(chJSON, ""); err != nil {
		t.Fatalf("原 CharacterGeneratePortrait 失败: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if *lastPath != "/images/generations" {
		t.Fatalf("无参考图应走 txt2img 端点, got %s", *lastPath)
	}
	if strings.Contains(string(*lastBody), `"image"`) {
		t.Fatalf("txt2img 请求体不应含 image 字段: %s", string(*lastBody))
	}
	if strings.Contains(string(*lastBody), `"mode"`) {
		t.Fatalf("txt2img 请求体不应含 mode 字段: %s", string(*lastBody))
	}
}

// TestCharacterGeneratePortraitWithRef_ComfyModelGate comfyui 后端 +
// 不支持图生图的模型 → 前置门禁拦截，不发起请求。
func TestCharacterGeneratePortraitWithRef_ComfyModelGate(t *testing.T) {
	srv, _, _, _ := newRefSlotTestServer(t)
	a := newRefSlotTestApp(t, srv)
	a.cfg.PortraitBackend = "comfyui"
	a.cfg.ComfyUIURL = srv.URL
	a.cfg.PortraitModel = "flux"

	chJSON := `{"id":"c1","name":"苏念"}`
	_, err := a.CharacterGeneratePortraitWithRef(chJSON, "", "data:image/png;base64,UkVGRg==")
	if err == nil {
		t.Fatal("comfyui + flux + 参考图应被门禁拦截")
	}
	if !strings.Contains(err.Error(), "暂不支持参考图生成") {
		t.Fatalf("门禁错误文案不符: %v", err)
	}
}
