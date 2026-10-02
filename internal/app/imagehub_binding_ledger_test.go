package app

// ── AP7-05 台账 backend/model 失真收口用例 ────────────────────────
//
// 审计原文：recordImageHubGeneratedFor 固定取全局 m.cfg.ImageBackend，但生成侧
// 存在独立绑定（原罪插图走 clientOverride/backendOverride，见 sin_handler.go →
// image_handler.go），另有传空 backend（character_portrait_register.go、
// novel_bookcover.go）与硬编码 model（chapter_handler.go 写死
// grok-imagine-image-quality）。imagehub_usage.go 按 {model,backend} 分组 ⇒
// 同一后端被拆成多行 / 原罪插图记成全局后端 / 消耗报表失真。
//
// 本用例钉住生效值通道：走 override 客户端生成后，台账记录的 backend/model
// 必须是 override 值，而不是全局配置值。

import (
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
)

// TestGenerateImageInternalLedgerRecordsEffectiveBinding override 通道生成：
// backendType 取 override 后端、model 取本轮下发模型，台账必须如实落这两者，
// 而不是全局 ImageBackend/ImageModel。
func TestGenerateImageInternalLedgerRecordsEffectiveBinding(t *testing.T) {
	ws, restore := s4SpaceIsolate(t, "play") // 隔离 ga.cfg（gaeaCwd 锚点）到临时工作区 + play 空间
	defer restore()

	origGate := imageHubLedgerRuntimeCheck
	imageHubLedgerRuntimeCheck = func() bool { return true }
	defer func() { imageHubLedgerRuntimeCheck = origGate }()

	// 全局客户端：comfyui 假后端（若被误用会记到 globalFake.calls）。
	globalFake := &flakyImageBackend{}
	gc := &ai.Client{}
	gc.SetImageBackend(globalFake, "comfyui")
	// override 客户端：herdsman 假后端（本链必须走它）。
	overFake := &flakyImageBackend{}
	oc := &ai.Client{}
	oc.SetImageBackend(overFake, "herdsman")

	ms := &mediaState{core: &core{cfg: &config.Config{
		ImageBackend: "comfyui",       // 全局后端（不得进台账）
		ImageModel:   "z-image-turbo", // 全局模型（显式 model 非空时不得进台账）
		ImageSaveDir: t.TempDir(),
	}, client: gc}}

	res, err := ms.generateImageInternal(imageGenInternal{
		prompt: "夜景", n: 1,
		model:           "qwen-image", // 本轮显式下发模型
		sourceBoard:     "imagegen",   // 与绘梦侧调用点同口径
		clientOverride:  oc,
		backendOverride: "herdsman",
	})
	if err != nil {
		t.Fatalf("generateImageInternal: %v", err)
	}
	if res["error"] != nil {
		t.Fatalf("不应失败: %v", res["error"])
	}
	if overFake.calls != 1 || globalFake.calls != 0 {
		t.Fatalf("override 通道应只走 override 客户端: over=%d global=%d", overFake.calls, globalFake.calls)
	}

	recs := newImageHubLedger(ws).list("play", 0)
	if len(recs) != 1 {
		t.Fatalf("应登记 1 条台账，got %d", len(recs))
	}
	rec := recs[0]
	if rec.Meta.Backend != "herdsman" {
		t.Fatalf("台账 backend = %q, want override 值 herdsman（改前=全局 comfyui）", rec.Meta.Backend)
	}
	if rec.Meta.Model != "qwen-image" {
		t.Fatalf("台账 model = %q, want 本轮下发模型 qwen-image（改前=全局 z-image-turbo）", rec.Meta.Model)
	}
	if rec.Meta.SourceBoard != "imagegen" {
		t.Fatalf("来源板块 = %q, want imagegen", rec.Meta.SourceBoard)
	}
}

// TestGenerateImageInternalLedgerFallsBackToGlobal 无 override 时回落全局配置：
// backend=全局 ImageBackend、model=全局 ImageModel（生效绑定解析的唯一分支）。
func TestGenerateImageInternalLedgerFallsBackToGlobal(t *testing.T) {
	ws, restore := s4SpaceIsolate(t, "play")
	defer restore()

	origGate := imageHubLedgerRuntimeCheck
	imageHubLedgerRuntimeCheck = func() bool { return true }
	defer func() { imageHubLedgerRuntimeCheck = origGate }()

	fake := &flakyImageBackend{}
	c := &ai.Client{}
	c.SetImageBackend(fake, "comfyui")

	ms := &mediaState{core: &core{cfg: &config.Config{
		ImageBackend: "comfyui",
		ImageModel:   "krea2",
		ImageSaveDir: t.TempDir(),
	}, client: c}}

	res, err := ms.generateImageInternal(imageGenInternal{prompt: "外景", n: 1})
	if err != nil {
		t.Fatalf("generateImageInternal: %v", err)
	}
	if res["error"] != nil {
		t.Fatalf("不应失败: %v", res["error"])
	}
	recs := newImageHubLedger(ws).list("play", 0)
	if len(recs) != 1 {
		t.Fatalf("应登记 1 条台账，got %d", len(recs))
	}
	if recs[0].Meta.Backend != "comfyui" || recs[0].Meta.Model != "krea2" {
		t.Fatalf("无 override 应回落全局: backend=%q model=%q, want comfyui/krea2",
			recs[0].Meta.Backend, recs[0].Meta.Model)
	}
}
