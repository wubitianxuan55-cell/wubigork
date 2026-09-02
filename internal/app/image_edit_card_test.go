package app

// image_edit_card_test.go — 对话式改图引擎线回归：
//   - editImageFromCard 仅 img2img 能力后端放行，其余诚实报错「当前生图引擎不支持改图」；
//   - 请求锚定 Mode=img2img / N=1 / 不传 size，落盘走 GenerateMedia 同口径；
//   - SetImageBackend(dashscope) 百炼 Key 经 secure 加密落盘、重启读回可解密。

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/secure"
)

// editCardImageBackend 记录收到的请求，供断言改图参数锚定。
type editCardImageBackend struct {
	req    *ai.ImageGenerationRequest
	result *ai.ImageGenerationResponse
	err    error
}

func (f *editCardImageBackend) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	f.req = req
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

// TestEditImageFromCard_UnsupportedEngine 非 img2img 能力后端诚实报错（xai 默认 /
// glm / ollama 均无改图参数），不静默降级为文生图。
func TestEditImageFromCard_UnsupportedEngine(t *testing.T) {
	for _, backend := range []string{"", "xai", "glm", "ollama"} {
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: backend}, client: &ai.Client{}}}
		cardPath, err := ms.editImageFromCard(pngDataURLApp("ref"), "把背景换成海滩")
		if err == nil || !strings.Contains(err.Error(), "当前生图引擎不支持改图") {
			t.Errorf("backend=%q 应报「当前生图引擎不支持改图」, got cardPath=%q err=%v", backend, cardPath, err)
		}
	}
}

// TestEditImageFromCard_NoClient 未登录（无客户端）时友好报错。
func TestEditImageFromCard_NoClient(t *testing.T) {
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "dashscope"}}}
	if _, err := ms.editImageFromCard(pngDataURLApp("ref"), "改"); err == nil {
		t.Fatal("无客户端应报错")
	}
}

// TestEditImageFromCard_EmptyInputs 空参考图 / 空指令在触网前拒绝。
func TestEditImageFromCard_EmptyInputs(t *testing.T) {
	rec := &editCardImageBackend{}
	c := &ai.Client{}
	c.SetImageBackend(rec, "dashscope")
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "dashscope"}, client: c}}

	if _, err := ms.editImageFromCard("", "改"); err == nil || !strings.Contains(err.Error(), "参考图") {
		t.Errorf("空参考图应报错, got %v", err)
	}
	if _, err := ms.editImageFromCard(pngDataURLApp("ref"), "  "); err == nil || !strings.Contains(err.Error(), "编辑指令") {
		t.Errorf("空指令应报错, got %v", err)
	}
	if rec.req != nil {
		t.Fatal("参数缺失时不应发出请求")
	}
}

// TestEditImageFromCard_Img2ImgAndPersists 请求锚定 img2img/N=1/不传 size，
// 结果与 GenerateMedia 同口径落盘并返回本地路径。
func TestEditImageFromCard_Img2ImgAndPersists(t *testing.T) {
	dir := t.TempDir()
	rec := &editCardImageBackend{result: &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: pngDataURLApp("edited")}},
	}}
	c := &ai.Client{}
	c.SetImageBackend(rec, "dashscope")
	ms := &mediaState{core: &core{cfg: &config.Config{
		ImageBackend: "dashscope",
		ImageModel:   "qwen-image-edit-max",
		ImageSaveDir: dir,
	}, client: c}}

	initImage := pngDataURLApp("ref")
	cardPath, err := ms.editImageFromCard(initImage, "把背景换成海滩")
	if err != nil {
		t.Fatalf("editImageFromCard: %v", err)
	}
	// 请求契约：img2img + 参考图原样 + N=1 + 不传 size（输出比例随参考图）
	if rec.req == nil {
		t.Fatal("未发出生成请求")
	}
	if rec.req.Mode != "img2img" {
		t.Errorf("Mode = %q, want img2img", rec.req.Mode)
	}
	if rec.req.InitImage != initImage {
		t.Errorf("InitImage 未原样透传: %q", rec.req.InitImage)
	}
	if rec.req.Prompt != "把背景换成海滩" {
		t.Errorf("Prompt = %q", rec.req.Prompt)
	}
	if rec.req.N != 1 {
		t.Errorf("N = %d, want 1", rec.req.N)
	}
	if rec.req.Size != "" {
		t.Errorf("改图不应传 Size, got %q", rec.req.Size)
	}
	if rec.req.Model != "qwen-image-edit-max" {
		t.Errorf("Model 应取 cfg.ImageModel, got %q", rec.req.Model)
	}
	// 落盘：返回本地路径且文件存在
	if cardPath == "" {
		t.Fatal("cardPath 为空")
	}
	if !strings.HasPrefix(cardPath, dir) {
		t.Errorf("cardPath 应落在 ImageSaveDir 下: %q", cardPath)
	}
	if _, err := os.Stat(cardPath); err != nil {
		t.Fatalf("cardPath 文件不存在: %v", err)
	}
}

// TestEditImageFromCard_BackendErrorPassthrough 后端错误原样返回。
func TestEditImageFromCard_BackendErrorPassthrough(t *testing.T) {
	wantErr := errors.New("百炼生图错误（HTTP 400）：[InvalidParameter] 输入图片格式不支持")
	rec := &editCardImageBackend{err: wantErr}
	c := &ai.Client{}
	c.SetImageBackend(rec, "dashscope")
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "dashscope"}, client: c}}

	cardPath, err := ms.editImageFromCard(pngDataURLApp("ref"), "改")
	if !errors.Is(err, wantErr) && err == nil {
		t.Fatalf("后端错误应原样返回, got cardPath=%q err=%v", cardPath, err)
	}
	if cardPath != "" {
		t.Errorf("失败时 cardPath 应为空, got %q", cardPath)
	}
}

// TestSetImageBackend_DashScopeKeyEncrypted 百炼后端切换链路：Key 明文入参 →
// cfg 内为 secure 密文 → 落盘 ~/.gaea_config.json（dashscope_api_key 字段）→
// 解密回明文一致；后端实例就绪（GetImageBackendType=dashscope）。
func TestSetImageBackend_DashScopeKeyEncrypted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	ms := &mediaState{core: &core{
		cfg:    &config.Config{ImageSaveDir: t.TempDir()},
		client: &ai.Client{},
	}}
	if err := ms.SetImageBackend("dashscope", "", "qwen-image-edit-plus", "", "sk-ds-test-123"); err != nil {
		t.Fatalf("SetImageBackend(dashscope): %v", err)
	}
	if ms.cfg.ImageBackend != "dashscope" || ms.cfg.ImageModel != "qwen-image-edit-plus" {
		t.Fatalf("配置未落地: backend=%s model=%s", ms.cfg.ImageBackend, ms.cfg.ImageModel)
	}
	// cfg 内必须是 secure 密文（"dpapi:" 前缀），不能落明文
	if !strings.HasPrefix(ms.cfg.DashScopeAPIKey, "dpapi:") {
		t.Fatalf("cfg.DashScopeAPIKey 应为 secure 密文, got %q", ms.cfg.DashScopeAPIKey)
	}
	if plain, err := secure.DecryptString(ms.cfg.DashScopeAPIKey); err != nil || plain != "sk-ds-test-123" {
		t.Fatalf("解密回读不符: plain=%q err=%v", plain, err)
	}
	// 落盘文件含 dashscope_api_key 字段且不落明文
	raw, err := os.ReadFile(filepath.Join(home, ".gaea_config.json"))
	if err != nil {
		t.Fatalf("读取配置文件: %v", err)
	}
	var cf map[string]map[string]string
	_ = json.Unmarshal(raw, &cf)
	var stored string
	var m map[string]string
	_ = json.Unmarshal(raw, &m)
	if m != nil {
		stored = m["dashscope_api_key"]
	}
	if !strings.HasPrefix(stored, "dpapi:") {
		t.Fatalf("落盘 dashscope_api_key 应为密文, got %q (raw=%s)", stored, trimJSON(raw))
	}
	if strings.Contains(string(raw), "sk-ds-test-123") {
		t.Fatal("配置文件泄漏明文 Key")
	}
	// 后端实例就绪
	if got := ms.client.GetImageBackendType(); got != "dashscope" {
		t.Fatalf("GetImageBackendType = %q, want dashscope", got)
	}
}

// TestSetImageBackend_DashScopeRequiresKey 未传 Key 且无存量 Key 时诚实拒绝。
func TestSetImageBackend_DashScopeRequiresKey(t *testing.T) {
	ms := &mediaState{core: &core{cfg: &config.Config{ImageSaveDir: t.TempDir()}, client: &ai.Client{}}}
	err := ms.SetImageBackend("dashscope", "", "", "", "")
	if err == nil || !strings.Contains(err.Error(), "Key 未配置") {
		t.Fatalf("无 Key 应诚实拒绝, got %v", err)
	}
	if ms.cfg.ImageBackend == "dashscope" {
		t.Fatal("拒绝时不应切换后端")
	}
}

// TestSetImageBackend_DashScopeResetsStaleModel 切到百炼时空模型/残留上一后端
// 模型（grok-imagine-* / krea2）都归位百炼默认编辑模型，避免带残留名请求
// 官方报 model 不存在；手填 qwen-image-edit 系原样保留。
func TestSetImageBackend_DashScopeResetsStaleModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	for _, tc := range []struct {
		imageModel string
		want       string
	}{
		{"", ai.DashScopeDefaultImageModel},            // 空：前端引擎列表无 dashscope，拿不到官方名
		{"grok-imagine-image-quality", ai.DashScopeDefaultImageModel}, // xAI 残留
		{"krea2", ai.DashScopeDefaultImageModel},                     // ComfyUI 残留
		{"qwen-image-edit-max", "qwen-image-edit-max"},               // 手填官方模型保留
	} {
		ms := &mediaState{core: &core{
			cfg:    &config.Config{ImageSaveDir: t.TempDir()},
			client: &ai.Client{},
		}}
		if err := ms.SetImageBackend("dashscope", "", tc.imageModel, "", "sk-ds-test"); err != nil {
			t.Fatalf("imageModel=%q SetImageBackend: %v", tc.imageModel, err)
		}
		if ms.cfg.ImageModel != tc.want {
			t.Fatalf("imageModel=%q → cfg.ImageModel = %q, want %q", tc.imageModel, ms.cfg.ImageModel, tc.want)
		}
	}
}

func trimJSON(raw []byte) string {
	s := string(raw)
	if len(s) > 300 {
		return s[:300]
	}
	return s
}
