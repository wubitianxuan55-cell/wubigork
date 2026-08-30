package app

import (
	"strings"
	"testing"

	appconfig "github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/secure"
)

// S1 Realtime 设置应用测试（VoiceApplySettings 三键 + VoiceGetSettings 回读）：
// 明文 Key 经 DPAPI 密文落盘、内存持明文；读侧只回 hasKey 布尔不回明文；
// 非法 provider 报错不落盘；空 Key=清除。
func TestVoiceApplySettings_RealtimeKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	cfg := &appconfig.Config{}
	a := &mediaState{core: &core{cfg: cfg}}
	a.initVoice()

	// 应用：provider + model + 明文 Key
	err := a.VoiceApplySettings(map[string]interface{}{
		"realtimeProvider": "openai",
		"realtimeModel":    "gpt-realtime",
		"realtimeAPIKey":   "  sk-live-sample-123  ", // 带空白：后端 trim
	})
	if err != nil {
		t.Fatalf("VoiceApplySettings: %v", err)
	}

	// 内存：明文（trim 后）
	got := a.voiceManager.GetConfig()
	if got.RealtimeAPIKey != "sk-live-sample-123" {
		t.Errorf("内存 APIKey = %q, want trim 后明文", got.RealtimeAPIKey)
	}
	// 落盘：密文（Load 后非明文且可解回）
	loaded := appconfig.Load()
	if loaded.GetRealtimeAPIKey() == "sk-live-sample-123" {
		t.Fatal("落盘值不应为明文")
	}
	plain, err := secure.DecryptString(loaded.GetRealtimeAPIKey())
	if err != nil || plain != "sk-live-sample-123" {
		t.Fatalf("落盘密文解密 = %q, err=%v, want 明文", plain, err)
	}
	if loaded.GetRealtimeProvider() != "openai" || loaded.GetRealtimeModel() != "gpt-realtime" {
		t.Errorf("落盘 provider/model = (%q,%q)", loaded.GetRealtimeProvider(), loaded.GetRealtimeModel())
	}

	// 读侧：hasKey=true 且全 map 不含明文
	view := a.VoiceGetSettings()
	if view["realtimeHasKey"] != true {
		t.Errorf("realtimeHasKey = %v, want true", view["realtimeHasKey"])
	}
	for k, v := range view {
		if s, ok := v.(string); ok && strings.Contains(s, "sk-live-sample") {
			t.Errorf("设置视图 %s 泄漏明文 Key: %q", k, s)
		}
	}

	// 清除：空串 Key → 落盘清空 + hasKey=false
	if err := a.VoiceApplySettings(map[string]interface{}{"realtimeAPIKey": ""}); err != nil {
		t.Fatalf("清除 Key: %v", err)
	}
	if a.voiceManager.GetConfig().RealtimeAPIKey != "" {
		t.Error("清除后内存 Key 应为空")
	}
	if v := a.VoiceGetSettings(); v["realtimeHasKey"] != false {
		t.Errorf("清除后 hasKey = %v, want false", v["realtimeHasKey"])
	}
}

// 非法 provider：报错且不落盘（provider 校验在 config.Save 层，宁拒勿存）。
func TestVoiceApplySettings_RealtimeInvalidProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	a := &mediaState{core: &core{cfg: &appconfig.Config{}}}
	a.initVoice()

	err := a.VoiceApplySettings(map[string]interface{}{"realtimeProvider": "anthropic"})
	if err == nil {
		t.Fatal("非法 provider 应报错")
	}
	if p := appconfig.Load().GetRealtimeProvider(); p == "anthropic" {
		t.Fatal("非法 provider 不应落盘")
	}
}
