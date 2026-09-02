package config

// config_dashscope_test.go — 百炼 DashScope Key 配置回归：saveSetters 登记
// （v4.9.1 GLM 教训）、密文样例原样往返（存储口径 = secure.EncryptString
// 密文，config 层只存取不做加解密，同 realtime_api_key / glm_api_key）。

import "testing"

// TestSaveSetters_CoverDashScopeAPIKey 显式锚定：dashscope_api_key 漏登记
// saveSetters 时，设置页保存百炼 Key 会直接报「不支持的配置项」。
func TestSaveSetters_CoverDashScopeAPIKey(t *testing.T) {
	if _, ok := saveSetters[KeyDashScopeAPIKey]; !ok {
		t.Fatal("KeyDashScopeAPIKey 未登记 saveSetters——保存百炼 Key 会报「不支持的配置项」")
	}
}

// TestSave_DashScopeAPIKeyRoundTrip 密文样例 Save → Load 原样取回；空串清空。
// 加解密由 app 层 secure 负责（先例 GLMAPIKey），config 层只验证字符串存取。
func TestSave_DashScopeAPIKeyRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if cfg := Load(); cfg.DashScopeAPIKey != "" {
		t.Error("未配置时 DashScopeAPIKey 默认应为空")
	}

	// 密文样例（"dpapi:" 前缀 + base64 blob 的形态，仅验证字符串原样存取）
	encSample := "dpapi:RFNfU0VDUkVUX0NJUEhFUlRFWFQ="
	if err := Save(KeyDashScopeAPIKey, encSample); err != nil {
		t.Fatalf("Save dashscope_api_key 失败: %s", err)
	}

	cfg := Load()
	if cfg.DashScopeAPIKey != encSample {
		t.Errorf("DashScopeAPIKey = %q, want 密文样例原样往返", cfg.DashScopeAPIKey)
	}

	// 空串清空（用户删除 Key 场景）
	if err := Save(KeyDashScopeAPIKey, ""); err != nil {
		t.Fatalf("Save dashscope_api_key=\"\" 失败: %s", err)
	}
	if cfg := Load(); cfg.DashScopeAPIKey != "" {
		t.Errorf("清空后 DashScopeAPIKey = %q, want 空", cfg.DashScopeAPIKey)
	}
}
