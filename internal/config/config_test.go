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
	if !cfg.GetFeatureModelEnabled("novel") {
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

// TestSave_RoutineBindingRoundTrip 常规办公（routine）绑定持久化：
// routine_llm 工具的目标模型配置，保存后重启不丢。
func TestSave_RoutineBindingRoundTrip(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUP := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUP)
	})

	// 单 HOME 下连续保存多键：模拟模型中心一次性写入引擎/模型/开关
	if err := Save(KeyFuncRoutineEngine, "herdsman"); err != nil {
		t.Fatalf("Save routine engine 失败: %s", err)
	}
	if err := Save(KeyFuncRoutineModel, "qwen3-8b"); err != nil {
		t.Fatalf("Save routine model 失败: %s", err)
	}
	if err := Save(KeyFuncRoutineEnabled, "1"); err != nil {
		t.Fatalf("Save routine enabled 失败: %s", err)
	}
	cfg := Load()
	eng, model := cfg.GetFeatureModel("routine")
	if eng != "herdsman" || model != "qwen3-8b" {
		t.Errorf("GetFeatureModel(routine) = (%q,%q), want (herdsman,qwen3-8b)", eng, model)
	}
	if !cfg.GetFeatureModelEnabled("routine") {
		t.Error("routine 保存为启用后应保持启用")
	}

	// 同一 HOME 下停用持久化
	if err := Save(KeyFuncRoutineEnabled, "0"); err != nil {
		t.Fatalf("Save routine disabled 失败: %s", err)
	}
	cfg = Load()
	if cfg.GetFeatureModelEnabled("routine") {
		t.Error("保存 0 后 routine 应为停用")
	}
}

// TestLoad_MigratesWhisperBindingToChat 旧版 func_whisper_* 绑定合并到 func_chat：
// 老配置文件（只有 func_whisper_*）加载后 chat 绑定接管，whisper 查询别名到 chat。
func TestLoad_MigratesWhisperBindingToChat(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	legacy := `{
		"func_whisper_engine": "herdsman",
		"func_whisper_model": "qwen3-8b",
		"func_whisper_enabled": false
	}`
	if err := os.WriteFile(filepath.Join(home, ".gaea_config.json"), []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Load()
	if cfg.FuncChatEngine != "herdsman" || cfg.FuncChatModel != "qwen3-8b" {
		t.Errorf("迁移后 FuncChat = (%q,%q), want (herdsman,qwen3-8b)", cfg.FuncChatEngine, cfg.FuncChatModel)
	}
	eng, model := cfg.GetFeatureModel("whisper")
	if eng != "herdsman" || model != "qwen3-8b" {
		t.Errorf("GetFeatureModel(whisper) = (%q,%q), want chat 别名", eng, model)
	}
	if cfg.GetFeatureModelEnabled("whisper") {
		t.Error("func_whisper_enabled=false 应迁移为 chat 停用")
	}
}

// TestLoad_FuncChatWinsWhenBothSet 新旧绑定并存时 func_chat 优先（不覆盖已有 chat 配置）。
func TestLoad_FuncChatWinsWhenBothSet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	legacy := `{
		"func_chat_engine": "xai",
		"func_chat_model": "grok-4.20",
		"func_whisper_engine": "herdsman",
		"func_whisper_model": "qwen3-8b"
	}`
	if err := os.WriteFile(filepath.Join(home, ".gaea_config.json"), []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Load()
	if cfg.FuncChatEngine != "xai" || cfg.FuncChatModel != "grok-4.20" {
		t.Errorf("FuncChat = (%q,%q), want 保留已有 chat 绑定", cfg.FuncChatEngine, cfg.FuncChatModel)
	}
}

// TestSave_SensitiveLocalRoundTrip 敏感域本地化开关（S2-4/D8）持久化：
// 默认开启；显式关闭 → 保存 → 重新加载为 false；再开启恢复。
func TestSave_SensitiveLocalRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// 未配置时默认开启（D8：敏感域默认本地优先）
	if cfg := Load(); !cfg.GetSensitiveLocal() {
		t.Error("未配置时敏感域本地化默认应为开启")
	}

	// 显式关闭 → 持久化 → 读取为 false
	if err := Save(KeySensitiveLocal, "0"); err != nil {
		t.Fatalf("Save sensitive_local=0 失败: %s", err)
	}
	if cfg := Load(); cfg.GetSensitiveLocal() {
		t.Error("保存 0 后敏感域本地化应为关闭")
	}

	// 重新开启
	if err := Save(KeySensitiveLocal, "1"); err != nil {
		t.Fatalf("Save sensitive_local=1 失败: %s", err)
	}
	if cfg := Load(); !cfg.GetSensitiveLocal() {
		t.Error("保存 1 后敏感域本地化应为开启")
	}
}

// TestSave_KeepWarmRoundTrip 本地模型保活开关（T5-3a）持久化：
// 默认开启；显式关闭 → 保存 → 重新加载为 false；再开启恢复。
func TestSave_KeepWarmRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// 未配置时默认开启（T5-3a：保活开箱即用）
	if cfg := Load(); !cfg.GetKeepWarm() {
		t.Error("未配置时保活默认应为开启")
	}

	// 显式关闭 → 持久化 → 读取为 false
	if err := Save(KeyKeepWarm, "0"); err != nil {
		t.Fatalf("Save keep_warm_enabled=0 失败: %s", err)
	}
	if cfg := Load(); cfg.GetKeepWarm() {
		t.Error("保存 0 后保活应为关闭")
	}

	// 重新开启
	if err := Save(KeyKeepWarm, "1"); err != nil {
		t.Fatalf("Save keep_warm_enabled=1 失败: %s", err)
	}
	if cfg := Load(); !cfg.GetKeepWarm() {
		t.Error("保存 1 后保活应为开启")
	}
}

// TestSave_AutoPreloadRoundTrip 启动自动预载开关（T5-3b）持久化：
// 默认开启；显式关闭 → 保存 → 重新加载为 false；再开启恢复。
func TestSave_AutoPreloadRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// 未配置时默认开启（T5-3b：自动预载开箱即用）
	if cfg := Load(); !cfg.GetAutoPreload() {
		t.Error("未配置时自动预载默认应为开启")
	}

	// 显式关闭 → 持久化 → 读取为 false
	if err := Save(KeyAutoPreload, "0"); err != nil {
		t.Fatalf("Save auto_preload=0 失败: %s", err)
	}
	if cfg := Load(); cfg.GetAutoPreload() {
		t.Error("保存 0 后自动预载应为关闭")
	}

	// 重新开启
	if err := Save(KeyAutoPreload, "1"); err != nil {
		t.Fatalf("Save auto_preload=1 失败: %s", err)
	}
	if cfg := Load(); !cfg.GetAutoPreload() {
		t.Error("保存 1 后自动预载应为开启")
	}
}

// TestSave_UsdCnyRateRoundTrip 美元→人民币汇率（T6-6.2）持久化：
// 默认 7.2；保存 7.0 → 重新加载为 7.0；非法值（0/负数/非数字）拒绝写入。
func TestSave_UsdCnyRateRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// 未配置时默认 7.2
	if cfg := Load(); cfg.UsdCnyRate != 7.2 {
		t.Errorf("默认汇率 = %v, want 7.2", cfg.UsdCnyRate)
	}

	// 保存 7.0 → 重新加载为 7.0
	if err := Save(KeyUsdCnyRate, "7.0"); err != nil {
		t.Fatalf("Save usd_cny_rate=7.0 失败: %s", err)
	}
	if cfg := Load(); cfg.UsdCnyRate != 7.0 {
		t.Errorf("保存 7.0 后汇率 = %v, want 7.0", cfg.UsdCnyRate)
	}

	// 非法值拒绝：非数字 / 0 / 负数
	if err := Save(KeyUsdCnyRate, "abc"); err == nil {
		t.Error("非数字汇率应返回 error")
	}
	if err := Save(KeyUsdCnyRate, "0"); err == nil {
		t.Error("0 汇率应返回 error")
	}
	if err := Save(KeyUsdCnyRate, "-1"); err == nil {
		t.Error("负数汇率应返回 error")
	}

	// 非法值被拒绝后，文件中的合法值保持不变
	if cfg := Load(); cfg.UsdCnyRate != 7.0 {
		t.Errorf("非法写入被拒后汇率 = %v, want 保持 7.0", cfg.UsdCnyRate)
	}
}
