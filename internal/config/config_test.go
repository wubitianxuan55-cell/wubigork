package config

import (
	"os"
	"path/filepath"
	"testing"
)

// saveWithTempHome 在临时 HOME 下执行 Save，避免污染真实配置文件
func saveWithTempHome(t *testing.T, key, value string) error {
	t.Helper()
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	})
	return Save(key, value)
}

func TestSave_InvalidKey(t *testing.T) {
	err := saveWithTempHome(t, "nonexistent_key", "value")
	if err == nil {
		t.Fatal("不支持的 key 应返回 error")
	}
}

func TestSave_StringKeyRoundTrip(t *testing.T) {
	err := saveWithTempHome(t, KeyNovelsDir, "/tmp/test_novels")
	if err != nil {
		t.Fatalf("Save 失败: %s", err)
	}
	cfg := Load()
	if cfg.NovelsDir != "/tmp/test_novels" {
		t.Errorf("NovelsDir = %q, 期望 /tmp/test_novels", cfg.NovelsDir)
	}
}

func TestSave_XaiClientID(t *testing.T) {
	err := saveWithTempHome(t, KeyXaiClientID, "test_client_123")
	if err != nil {
		t.Fatalf("Save 失败: %s", err)
	}
	cfg := Load()
	if cfg.XaiClientID != "test_client_123" {
		t.Errorf("XaiClientID = %q, 期望 test_client_123", cfg.XaiClientID)
	}
}

func TestSave_IntKeyValid(t *testing.T) {
	err := saveWithTempHome(t, KeyHTTPTimeoutSeconds, "30")
	if err != nil {
		t.Fatalf("Save 失败: %s", err)
	}
	cfg := Load()
	if cfg.HTTPTimeoutSeconds != 30 {
		t.Errorf("HTTPTimeoutSeconds = %d, 期望 30", cfg.HTTPTimeoutSeconds)
	}
}

func TestSave_IntKeyInvalidInput(t *testing.T) {
	err := saveWithTempHome(t, KeyHTTPTimeoutSeconds, "abc")
	if err == nil {
		t.Fatal("非法的整数值应返回 error")
	}
}

func TestSave_FloatKeyValid(t *testing.T) {
	err := saveWithTempHome(t, KeyDefaultTemperature, "0.8")
	if err != nil {
		t.Fatalf("Save 失败: %s", err)
	}
	cfg := Load()
	if cfg.DefaultTemperature != 0.8 {
		t.Errorf("DefaultTemperature = %f, 期望 0.8", cfg.DefaultTemperature)
	}
}

func TestSave_FloatKeyInvalidInput(t *testing.T) {
	err := saveWithTempHome(t, KeyDefaultTemperature, "not_a_float")
	if err == nil {
		t.Fatal("非法的浮点数值应返回 error")
	}
}

func TestSave_ReasoningEffort(t *testing.T) {
	err := saveWithTempHome(t, KeyReasoningEffort, "high")
	if err != nil {
		t.Fatalf("Save 失败: %s", err)
	}
	cfg := Load()
	if cfg.ReasoningEffort != "high" {
		t.Errorf("ReasoningEffort = %q, 期望 high", cfg.ReasoningEffort)
	}
}

func TestSave_QualityThreshold(t *testing.T) {
	err := saveWithTempHome(t, KeyQualityThreshold, "80")
	if err != nil {
		t.Fatalf("Save 失败: %s", err)
	}
	cfg := Load()
	if cfg.QualityThreshold != 80 {
		t.Errorf("QualityThreshold = %d, 期望 80", cfg.QualityThreshold)
	}
}

func TestSave_MultipleKeysRoundTrip(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUP := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUP)
	})

	// 保存多个键
	if err := Save(KeyNovelsDir, "/multi/novels"); err != nil {
		t.Fatalf("Save NovelsDir 失败: %s", err)
	}
	if err := Save(KeyHTTPTimeoutSeconds, "99"); err != nil {
		t.Fatalf("Save HTTPTimeoutSeconds 失败: %s", err)
	}
	if err := Save(KeyDefaultTemperature, "1.5"); err != nil {
		t.Fatalf("Save DefaultTemperature 失败: %s", err)
	}

	// 验证文件存在
	configPath := filepath.Join(tmpHome, ".gaea_config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Save 后配置文件不存在")
	}

	// Load 时读新配置
	cfg := Load()
	if cfg.NovelsDir != "/multi/novels" {
		t.Errorf("NovelsDir = %q", cfg.NovelsDir)
	}
	if cfg.HTTPTimeoutSeconds != 99 {
		t.Errorf("HTTPTimeoutSeconds = %d", cfg.HTTPTimeoutSeconds)
	}
	if cfg.DefaultTemperature != 1.5 {
		t.Errorf("DefaultTemperature = %f", cfg.DefaultTemperature)
	}
}

func TestSave_ActiveEngineIDRoundTrip(t *testing.T) {
	err := saveWithTempHome(t, KeyActiveEngineID, "ollama")
	if err != nil {
		t.Fatalf("Save 失败: %s", err)
	}
	cfg := Load()
	if cfg.ActiveEngineID != "ollama" {
		t.Errorf("ActiveEngineID = %q, 期望 ollama（保存后重启必须恢复全局活跃引擎）", cfg.ActiveEngineID)
	}
}

func TestSave_FuncEnabledRoundTrip(t *testing.T) {
	// 未配置时默认启用（绑定即生效）
	cfg := Load()
	if !cfg.GetFeatureModelEnabled("chat") {
		t.Error("未配置时 chat 功能模型默认应启用")
	}

	// 显式停用 → 持久化 → 读取为 false
	err := saveWithTempHome(t, KeyFuncChatEnabled, "0")
	if err != nil {
		t.Fatalf("Save 失败: %s", err)
	}
	cfg = Load()
	if cfg.GetFeatureModelEnabled("chat") {
		t.Error("保存 0 后 chat 功能模型应为停用")
	}
	if !cfg.GetFeatureModelEnabled("whisper") {
		t.Error("未写入的功能应保持默认启用")
	}

	// 重新启用
	if err := saveWithTempHome(t, KeyFuncChatEnabled, "1"); err != nil {
		t.Fatalf("Save 失败: %s", err)
	}
	cfg = Load()
	if !cfg.GetFeatureModelEnabled("chat") {
		t.Error("保存 1 后 chat 功能模型应为启用")
	}
}
