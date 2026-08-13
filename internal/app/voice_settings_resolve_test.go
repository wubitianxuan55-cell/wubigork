package app

import (
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/voice"
)

// newVoiceSettingsTestState 构造测试用 mediaState：engineMgr 预置 herdsman 引擎模型列表。
func newVoiceSettingsTestState(t *testing.T, models []modelengine.ModelInfo) *mediaState {
	t.Helper()
	c := &core{engineMgr: modelengine.NewManager("", "")}
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:      "herdsman",
		Enabled: true,
		Models:  models,
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	return &mediaState{core: c} // voiceManager 为 nil → VoiceGetSettings 走 DefaultVoiceConfig
}

// TestVoiceGetSettings_ResolvesHerdsmanModel_ConfiguredNotInstalled
// 默认配置 qwen3-tts-customvoice 未安装、voxcpm2 已装 → 动态解析为 voxcpm2 并暴露回退标记。
func TestVoiceGetSettings_ResolvesHerdsmanModel_ConfiguredNotInstalled(t *testing.T) {
	a := newVoiceSettingsTestState(t, []modelengine.ModelInfo{{ID: "voxcpm2"}, {ID: "edge-tts"}})

	got := a.VoiceGetSettings()
	if got["ttsHerdsmanModel"] != "voxcpm2" {
		t.Errorf("ttsHerdsmanModel = %v, want voxcpm2（默认模型未装，按优先级解析）", got["ttsHerdsmanModel"])
	}
	if got["ttsHerdsmanModelFallback"] != true {
		t.Errorf("ttsHerdsmanModelFallback = %v, want true", got["ttsHerdsmanModelFallback"])
	}
	if got["ttsHerdsmanModelFromInstalled"] != true {
		t.Errorf("ttsHerdsmanModelFromInstalled = %v, want true", got["ttsHerdsmanModelFromInstalled"])
	}
}

// TestVoiceGetSettings_ResolvesHerdsmanModel_ConfiguredInstalled
// 配置值已安装 → 原样返回，不回退。
func TestVoiceGetSettings_ResolvesHerdsmanModel_ConfiguredInstalled(t *testing.T) {
	a := newVoiceSettingsTestState(t, []modelengine.ModelInfo{{ID: "qwen3-tts-customvoice"}, {ID: "voxcpm2"}})

	got := a.VoiceGetSettings()
	if got["ttsHerdsmanModel"] != "qwen3-tts-customvoice" {
		t.Errorf("ttsHerdsmanModel = %v, want qwen3-tts-customvoice", got["ttsHerdsmanModel"])
	}
	if got["ttsHerdsmanModelFallback"] != false {
		t.Errorf("ttsHerdsmanModelFallback = %v, want false", got["ttsHerdsmanModelFallback"])
	}
	if got["ttsHerdsmanModelFromInstalled"] != false {
		t.Errorf("ttsHerdsmanModelFromInstalled = %v, want false", got["ttsHerdsmanModelFromInstalled"])
	}
}

// TestVoiceGetSettings_ResolvesHerdsmanModel_NoEngineMgr
// engineMgr 为 nil（拿不到已装列表）→ 等价于原逻辑：返回默认配置值，不标记回退。
func TestVoiceGetSettings_ResolvesHerdsmanModel_NoEngineMgr(t *testing.T) {
	a := &mediaState{core: &core{}} // engineMgr 为 nil

	got := a.VoiceGetSettings()
	if got["ttsHerdsmanModel"] != "qwen3-tts-customvoice" {
		t.Errorf("ttsHerdsmanModel = %v, want 默认 qwen3-tts-customvoice", got["ttsHerdsmanModel"])
	}
	if got["ttsHerdsmanModelFallback"] != false {
		t.Errorf("ttsHerdsmanModelFallback = %v, want false（无已装列表时不标记回退）", got["ttsHerdsmanModelFallback"])
	}
	if got["ttsHerdsmanModelFromInstalled"] != false {
		t.Errorf("ttsHerdsmanModelFromInstalled = %v, want false", got["ttsHerdsmanModelFromInstalled"])
	}
}

// TestVoiceGetSettings_ResolvesHerdsmanModel_ManagerConfig 非空 voiceManager 时同样动态解析。
func TestVoiceGetSettings_ResolvesHerdsmanModel_ManagerConfig(t *testing.T) {
	a := newVoiceSettingsTestState(t, []modelengine.ModelInfo{{ID: "voxcpm2"}})

	// 非默认配置值：配置 qwen3-tts-customvoice 未装 → 解析为 voxcpm2
	cfg := voice.DefaultVoiceConfig()
	cfg.TTSHerdsmanModel = "qwen3-tts-customvoice"
	a.voiceManager = voice.NewManager(nil, cfg)

	got := a.VoiceGetSettings()
	if got["ttsHerdsmanModel"] != "voxcpm2" {
		t.Errorf("ttsHerdsmanModel = %v, want voxcpm2", got["ttsHerdsmanModel"])
	}
	if got["ttsHerdsmanModelFallback"] != true {
		t.Errorf("ttsHerdsmanModelFallback = %v, want true", got["ttsHerdsmanModelFallback"])
	}
}
