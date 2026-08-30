package app

import (
	"testing"

	appconfig "github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/secure"
)

// S1 Realtime 配置接线测试（initVoice + VoiceHealth.realtimeReady）：
// 落盘三项（provider/model + secure.EncryptString 密文 Key）→ initVoice 注入
// VoiceRuntimeConfig；解密失败降级不崩；未配置零变化。

// TestInitVoice_RealtimeConfigWiring 配置 provider=openai + model + 真实 DPAPI
// 密文 → initVoice 后 VoiceRuntimeConfig 三字段就位且 APIKey = 明文；
// VoiceHealth.realtimeReady = true（配置了 provider 且 seam 构造成功）。
func TestInitVoice_RealtimeConfigWiring(t *testing.T) {
	const plainKey = "sk-realtime-wiring-sample"
	enc, err := secure.EncryptString(plainKey)
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	cfg := &appconfig.Config{
		RealtimeProvider: "openai",
		RealtimeModel:    "gpt-4o-realtime-preview",
		RealtimeAPIKey:   enc,
	}
	a := &mediaState{core: &core{cfg: cfg}}
	a.initVoice()

	got := a.voiceManager.GetConfig()
	if got.RealtimeProvider != "openai" {
		t.Errorf("RealtimeProvider = %q, want openai", got.RealtimeProvider)
	}
	if got.RealtimeModel != "gpt-4o-realtime-preview" {
		t.Errorf("RealtimeModel = %q, want gpt-4o-realtime-preview", got.RealtimeModel)
	}
	if got.RealtimeAPIKey != plainKey {
		t.Errorf("RealtimeAPIKey = %q, want 解密后明文 %q", got.RealtimeAPIKey, plainKey)
	}
	if health := a.VoiceHealth(); health["realtimeReady"] != true {
		t.Errorf("realtimeReady = %v, want true（配置了 provider 且 seam 构造成功）", health["realtimeReady"])
	}
}

// TestInitVoice_RealtimeDecryptFailure 解密失败降级：密文损坏（合法 base64 但
// 非 DPAPI blob）→ initVoice 不崩；provider/model 照常注入、APIKey 置空；
// realtimeReady = false（seam 构造因缺 Key 失败）。
func TestInitVoice_RealtimeDecryptFailure(t *testing.T) {
	cfg := &appconfig.Config{
		RealtimeProvider: "openai",
		RealtimeModel:    "gpt-4o-realtime-preview",
		RealtimeAPIKey:   "dpapi:bm90LWEtcmVhbC1ibG9i", // 解码为 "not-a-real-blob"，非合法密文
	}
	a := &mediaState{core: &core{cfg: cfg}}
	a.initVoice() // 不崩即通过（降级路径内部 slog.Warn）

	got := a.voiceManager.GetConfig()
	if got.RealtimeProvider != "openai" || got.RealtimeModel != "gpt-4o-realtime-preview" {
		t.Errorf("降级后 provider/model = (%q,%q), want 照常注入", got.RealtimeProvider, got.RealtimeModel)
	}
	if got.RealtimeAPIKey != "" {
		t.Errorf("解密失败后 APIKey = %q, want 空", got.RealtimeAPIKey)
	}
	if health := a.VoiceHealth(); health["realtimeReady"] != false {
		t.Errorf("realtimeReady = %v, want false（解密失败降级）", health["realtimeReady"])
	}
}

// TestInitVoice_RealtimeUnconfigured 未配置（Provider 空）→ 现状零变化：
// Realtime 三项保持零值，realtimeReady = false（probeRealtime 未配置跳过）。
func TestInitVoice_RealtimeUnconfigured(t *testing.T) {
	a := &mediaState{core: &core{cfg: &appconfig.Config{}}}
	a.initVoice()

	got := a.voiceManager.GetConfig()
	if got.RealtimeProvider != "" || got.RealtimeModel != "" || got.RealtimeAPIKey != "" {
		t.Errorf("未配置时 Realtime 三项应保持零值，实际 (%q,%q,%q)",
			got.RealtimeProvider, got.RealtimeModel, got.RealtimeAPIKey)
	}
	if health := a.VoiceHealth(); health["realtimeReady"] != false {
		t.Errorf("realtimeReady = %v, want false（未配置恒为 false）", health["realtimeReady"])
	}
}
