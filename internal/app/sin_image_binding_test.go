package app

// v4.388 原罪插图独立生图绑定测试：级联语义（绑定>全局，可单项回退）、
// GetSinImageConfig 读映射、buildImageClientFor 分支（不碰真实网络/进程）、
// generateImageInternal 的 override 客户端通道（绑定后端 ≠ 全局后端时走
// 独立客户端）。刻意不调 SetSinImageConfig——config.Save 直写真实用户
// ~/.gaea_config.json，无测试隔离钩子（同 hub*Store 系的坑：真实面必须绕开）。

import (
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// TestSinImageBindingCascade 级联契约：任一项空即各自回退全局，非空即覆盖。
func TestSinImageBindingCascade(t *testing.T) {
	cases := []struct {
		name                      string
		sinBackend, globalBackend string
		sinModel, globalModel     string
		wantBackend, wantModel    string
	}{
		{"全空回退全局", "", "xai", "", "grok-imagine-image-quality", "xai", "grok-imagine-image-quality"},
		{"只绑后端", "herdsman", "xai", "", "grok-imagine-image-quality", "herdsman", "grok-imagine-image-quality"},
		{"只绑模型", "", "comfyui", "krea2", "z-image-turbo", "comfyui", "krea2"},
		{"双绑全覆盖", "glm", "comfyui", "glm-image", "krea2", "glm", "glm-image"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, m := sinImageBinding(&config.Config{
				ImageBackend: c.globalBackend, ImageModel: c.globalModel,
				SinImageBackend: c.sinBackend, SinImageModel: c.sinModel,
			})
			if b != c.wantBackend || m != c.wantModel {
				t.Fatalf("cascade = (%s, %s), want (%s, %s)", b, m, c.wantBackend, c.wantModel)
			}
		})
	}
}

// TestGetSinImageConfig 读映射（空绑定如实回空串，前端据此显示「跟随全局」）。
func TestGetSinImageConfig(t *testing.T) {
	a := &App{core: &core{cfg: &config.Config{}}}
	if r := a.GetSinImageConfig(); r["backend"] != "" || r["model"] != "" {
		t.Fatalf("空绑定应回空串, got %v", r)
	}
	a.cfg.SinImageBackend = "herdsman"
	a.cfg.SinImageModel = "qwen-image"
	if r := a.GetSinImageConfig(); r["backend"] != "herdsman" || r["model"] != "qwen-image" {
		t.Fatalf("绑定值应如实回读, got %v", r)
	}
}

// TestBuildImageClientFor 分支：xai/comfyui 构建成功；本地引擎未启用时报错
// 且文案点名功能（空管理器种子目录 herdsman Enabled 但 BaseURL 空 —— 不发
// 真实请求，只构建客户端实例）。
func TestBuildImageClientFor(t *testing.T) {
	a := &App{core: &core{
		cfg:       &config.Config{ComfyUIURL: "http://127.0.0.1:18188"},
		engineMgr: modelengine.NewManager("", ""),
	}}
	if c, err := a.buildImageClientFor("xai", "原罪插图"); err != nil || c == nil {
		t.Fatalf("xai 分支应构建成功: %v", err)
	}
	if c, err := a.buildImageClientFor("comfyui", "原罪插图"); err != nil || c == nil {
		t.Fatalf("comfyui 分支应构建成功（不拨号）: %v", err)
	}
	// 种子目录 ollama 恒 Enabled（构建即成功）——禁用后走「未启用」分支。
	if eng, ok := a.engineMgr.GetEngine("ollama"); ok {
		eng.Enabled = false
		if err := a.engineMgr.SaveEngine(*eng); err != nil {
			t.Fatalf("disable ollama: %v", err)
		}
	}
	if _, err := a.buildImageClientFor("ollama", "原罪插图"); err == nil {
		t.Fatal("ollama 未启用应报错")
	}
	// ComfyUIURL 未配置 → 明确报错。
	a.cfg.ComfyUIURL = ""
	if _, err := a.buildImageClientFor("comfyui", "原罪插图"); err == nil {
		t.Fatal("未配置 ComfyUI 地址应报错")
	}
}

// TestGenerateImageInternal_OverrideClient：override 客户端优先于全局
// （绑定后端≠全局后端的路由通道）；走 fake 后端成功路径，零真实网络。
func TestGenerateImageInternal_OverrideClient(t *testing.T) {
	// 全局客户端：comfyui 假后端（若被误用会记到 globalFake.calls）。
	globalFake := &flakyImageBackend{}
	gc := &ai.Client{}
	gc.SetImageBackend(globalFake, "comfyui")
	// override 客户端：herdsman 假后端（本链应走它）。
	overFake := &flakyImageBackend{}
	oc := &ai.Client{}
	oc.SetImageBackend(overFake, "herdsman")

	ms := &mediaState{core: &core{cfg: &config.Config{
		ImageBackend: "comfyui",
		ImageModel:   "krea2",
		ImageSaveDir: t.TempDir(),
	}, client: gc}}

	res, err := ms.generateImageInternal(imageGenInternal{
		prompt: "夜景", n: 1,
		clientOverride: oc, backendOverride: "herdsman",
	})
	if err != nil {
		t.Fatalf("generateImageInternal: %v", err)
	}
	if res["error"] != nil {
		t.Fatalf("不应失败: %v", res["error"])
	}
	if overFake.calls != 1 {
		t.Fatalf("override 客户端调用次数 = %d, want 1", overFake.calls)
	}
	if globalFake.calls != 0 {
		t.Fatalf("全局客户端不应被触碰, calls = %d", globalFake.calls)
	}
}
