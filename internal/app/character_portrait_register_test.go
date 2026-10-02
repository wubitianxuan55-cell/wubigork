package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/types"
)

func TestRegisterCharacterPortraitAsset(t *testing.T) {
	origGate := imageHubLedgerRuntimeCheck
	imageHubLedgerRuntimeCheck = func() bool { return true }
	defer func() { imageHubLedgerRuntimeCheck = origGate }()

	cwd := t.TempDir()
	img := filepath.Join(cwd, "c1.png")
	if err := os.WriteFile(img, []byte("png"), 0o644); err != nil {
		t.Fatalf("write img: %v", err)
	}
	registerCharacterPortraitAsset(cwd, []types.Character{
		{ID: "c1", PortraitURL: img},
		{ID: "c2", PortraitURL: "data:image/png;base64,AAAA"},
	}, "c1", "herdsman", "qwen-image")

	got := newImageHubLedger(cwd).list("play", 0)
	if len(got) != 1 {
		t.Fatalf("应登记 1 条剧照，got %d", len(got))
	}
	rec := got[0]
	if rec.Meta.SourceBoard != "characterlib" || rec.Asset.Path != img {
		t.Fatalf("剧照登记错误: %+v", rec)
	}
	if rec.Meta.Params["character_id"] != "c1" {
		t.Fatalf("缺 character_id: %+v", rec.Meta.Params)
	}
	// AP7-05：生效后端/模型必须如实登记（此前恒为空串 ⇒ 消耗报表多一行）。
	if rec.Meta.Backend != "herdsman" || rec.Meta.Model != "qwen-image" {
		t.Fatalf("剧照台账应记生效绑定: backend=%q model=%q, want herdsman/qwen-image",
			rec.Meta.Backend, rec.Meta.Model)
	}
}

func TestRegisterCharacterPortraitAssetSkipsRemoteOrMissing(t *testing.T) {
	origGate := imageHubLedgerRuntimeCheck
	imageHubLedgerRuntimeCheck = func() bool { return true }
	defer func() { imageHubLedgerRuntimeCheck = origGate }()

	cwd := t.TempDir()
	registerCharacterPortraitAsset(cwd, []types.Character{
		{ID: "c1", PortraitURL: "data:image/png;base64,AAAA"},
		{ID: "c2", PortraitURL: filepath.Join(cwd, "missing.png")},
		{ID: "c3", PortraitURL: ""},
	}, "c1", "comfyui", "krea2")
	registerCharacterPortraitAsset(cwd, []types.Character{{ID: "c1", PortraitURL: "data:image/png;base64,AAAA"}}, "c2", "comfyui", "krea2")
	if got := newImageHubLedger(cwd).list("play", 0); len(got) != 0 {
		t.Fatalf("远程/缺失/空路径不应登记: %+v", got)
	}
}

// TestPortraitImageBindingCascade 剧照生效绑定级联（AP7-05 唯一解析点）：
// 剧照绑定 > 全局绘梦；显式 model 入参优先于两者；空配置不 panic。
func TestPortraitImageBindingCascade(t *testing.T) {
	cases := []struct {
		name                   string
		portraitBackend        string
		portraitModel          string
		globalBackend          string
		globalModel            string
		explicitModel          string
		wantBackend, wantModel string
	}{
		{"全空", "", "", "", "", "", "", ""},
		{"跟随全局", "", "", "xai", "grok-imagine-image-quality", "", "xai", "grok-imagine-image-quality"},
		{"只绑后端", "herdsman", "", "xai", "grok-imagine-image-quality", "", "herdsman", "grok-imagine-image-quality"},
		{"只绑模型", "", "krea2", "comfyui", "z-image-turbo", "", "comfyui", "krea2"},
		{"双绑", "comfyui", "qwen-image-edit", "xai", "grok-imagine-image-quality", "", "comfyui", "qwen-image-edit"},
		{"显式模型优先", "", "krea2", "comfyui", "z-image-turbo", "flux", "comfyui", "flux"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, m := portraitImageBinding(&config.Config{
				PortraitBackend: c.portraitBackend, PortraitModel: c.portraitModel,
				ImageBackend: c.globalBackend, ImageModel: c.globalModel,
			}, c.explicitModel)
			if b != c.wantBackend || m != c.wantModel {
				t.Fatalf("portraitImageBinding = (%q, %q), want (%q, %q)", b, m, c.wantBackend, c.wantModel)
			}
		})
	}
	// nil 配置不 panic（防御）。
	if b, m := portraitImageBinding(nil, ""); b != "" || m != "" {
		t.Fatalf("nil 配置应回空串, got (%q, %q)", b, m)
	}
}

// TestEffectiveSceneIllustrationModel 章节配图/书封生效模型：配置优先、
// 未配置回落 Aurora 档（登记与请求同源的前提）。
func TestEffectiveSceneIllustrationModel(t *testing.T) {
	cfg := &config.Config{}
	if got := effectiveSceneIllustrationModel(cfg); got != auroraSceneImageModel {
		t.Fatalf("未配置应回落 Aurora 档, got %q", got)
	}
	cfg.ImageModel = "krea2"
	if got := effectiveSceneIllustrationModel(cfg); got != "krea2" {
		t.Fatalf("配置优先, got %q", got)
	}
	cfg.ImageModel = "   "
	if got := effectiveSceneIllustrationModel(cfg); got != auroraSceneImageModel {
		t.Fatalf("纯空白应回落 Aurora 档, got %q", got)
	}
	if got := effectiveSceneIllustrationModel(nil); got != auroraSceneImageModel {
		t.Fatalf("nil 配置应回落 Aurora 档, got %q", got)
	}
}
