package app

// IN2-07（2026-10-02 全仓审计·批次二十）app 侧对照：image_backend 判据全部
// 走 ai.ImageBackendType* 单点常量后，钉住前端/配置可观测面的取值——
//  1. GetImageBackend 绑定（下发前端）对五个类型名原样透传，无 client 时回退 xai；
//  2. GetImageBackendInfo["backend"] 透传 cfg.ImageBackend，且 comfyui/glm 的
//     默认模型归位分支（判据曾以字面量散落，错改判据本组先红）；
//  3. mediaState.SetImageBackend xai 分支落盘值 = ImageBackendTypeXAI。

import (
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
)

func TestImageBackendTypeBindingPassthrough(t *testing.T) {
	names := []string{
		ai.ImageBackendTypeXAI,
		ai.ImageBackendTypeComfyUI,
		ai.ImageBackendTypeHerdsman,
		ai.ImageBackendTypeOllama,
		ai.ImageBackendTypeGLM,
	}
	for _, name := range names {
		c := &ai.Client{}
		c.SetImageBackend(nil, name)
		ms := &mediaState{core: &core{cfg: &config.Config{}, client: c}}
		if got := ms.GetImageBackend(); got != name {
			t.Errorf("GetImageBackend 透传: 注入 %q, got %q", name, got)
		}
	}
	// 无 client：绑定回退 xai（前端默认出图依赖该值）
	ms := &mediaState{core: &core{cfg: &config.Config{}}}
	if got := ms.GetImageBackend(); got != ai.ImageBackendTypeXAI {
		t.Errorf("无 client 回退: want %q, got %q", ai.ImageBackendTypeXAI, got)
	}
}

func TestImageBackendInfoBackendJudgments(t *testing.T) {
	t.Run("backend 原样透传", func(t *testing.T) {
		// 生产流程：类型名经 client.SetImageBackend 注入，绑定的 backend 字段
		// 读 GetImageBackendType()；cfg.ImageBackend 只供模型归位等判据。
		for _, name := range []string{
			ai.ImageBackendTypeXAI,
			ai.ImageBackendTypeComfyUI,
			ai.ImageBackendTypeHerdsman,
			ai.ImageBackendTypeOllama,
			ai.ImageBackendTypeGLM,
		} {
			c := &ai.Client{}
			c.SetImageBackend(nil, name)
			ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: name, ImageModel: "m"}, client: c}}
			if got := ms.GetImageBackendInfo()["backend"]; got != name {
				t.Errorf("GetImageBackendInfo backend 透传: 注入 %q, got %q", name, got)
			}
		}
	})
	t.Run("comfyui 空模型归位 krea2", func(t *testing.T) {
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: ai.ImageBackendTypeComfyUI}}}
		if got := ms.GetImageBackendInfo()["image_model"]; got != "krea2" {
			t.Errorf("comfyui 空模型应归位 krea2, got %q", got)
		}
	})
	t.Run("非 comfyui 空模型不归位 krea2", func(t *testing.T) {
		// 反向判据锚点：同一空模型在 xai 下走默认高质量模型（判据错改时此处红）
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: ai.ImageBackendTypeXAI}}}
		if got := ms.GetImageBackendInfo()["image_model"]; got != "grok-imagine-image-quality" {
			t.Errorf("xai 空模型应为 grok-imagine-image-quality, got %q", got)
		}
	})
	t.Run("glm 非官方模型归位默认", func(t *testing.T) {
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: ai.ImageBackendTypeGLM, ImageModel: "grok-imagine-image"}}}
		if got := ms.GetImageBackendInfo()["image_model"]; got != ai.GLMDefaultImageModel {
			t.Errorf("glm 非官方模型应归位 %s, got %q", ai.GLMDefaultImageModel, got)
		}
	})
}

func TestMediaSetImageBackend_XAIBranchPersistsTypeName(t *testing.T) {
	ms := &mediaState{core: &core{cfg: &config.Config{ImageSaveDir: t.TempDir()}, client: &ai.Client{}}}
	if err := ms.SetImageBackend(ai.ImageBackendTypeXAI, "", "", ""); err != nil {
		t.Fatalf("SetImageBackend(xai): %v", err)
	}
	if ms.cfg.ImageBackend != ai.ImageBackendTypeXAI {
		t.Errorf("cfg.ImageBackend = %q, want %q", ms.cfg.ImageBackend, ai.ImageBackendTypeXAI)
	}
	if got := ms.clientRef().GetImageBackendType(); got != ai.ImageBackendTypeXAI {
		t.Errorf("GetImageBackendType = %q, want %q", got, ai.ImageBackendTypeXAI)
	}
}

// TestWarmComfyUIArmedGate :54 暖启武装位闸（round-22 线 1 登记收口）：未武装
// （服务启动未完成）恒 "unarmed"；武装后按后端/引擎状态分流（非 comfyui 后端
// → "backend-not-comfyui"；comfyui 后端引擎不在跑 → "comfyui-not-running"，
// mock 环境无引擎即走此分支）。包级 atomic 复位保序，防污染其他用例。
func TestWarmComfyUIArmedGate(t *testing.T) {
	prev := comfyWarmArmed.Load()
	t.Cleanup(func() { comfyWarmArmed.Store(prev) })

	comfyWarmArmed.Store(false)
	ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: ai.ImageBackendTypeComfyUI, ImageModel: "krea2"}}}
	if got := ms.WarmComfyUI(); got["started"] != false || got["reason"] != "unarmed" {
		t.Fatalf("未武装应 unarmed: %v", got)
	}

	comfyWarmArmed.Store(true)
	if got := ms.WarmComfyUI(); got["started"] != false || got["reason"] != "comfyui-not-running" {
		t.Fatalf("武装+comfyui 后端+引擎不在跑应 comfyui-not-running: %v", got)
	}
	xai := &mediaState{core: &core{cfg: &config.Config{ImageBackend: ai.ImageBackendTypeXAI, ImageModel: "grok-imagine-image-quality"}}}
	if got := xai.WarmComfyUI(); got["started"] != false || got["reason"] != "backend-not-comfyui" {
		t.Fatalf("武装+非 comfyui 后端应 backend-not-comfyui: %v", got)
	}
}
