package config

import "testing"

// TestSave_EngineFailoverRoundTrip 引擎故障转移开关（C 刀 v0）持久化：
// 默认关闭；开启 → 保存 → 重新加载为 true；再关闭恢复。
func TestSave_EngineFailoverRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if cfg := Load(); cfg.GetEngineFailover() {
		t.Error("未配置时引擎故障转移默认应为关闭")
	}
	if err := Save(KeyEngineFailover, "1"); err != nil {
		t.Fatalf("Save engine_failover_enabled=1 失败: %s", err)
	}
	if cfg := Load(); !cfg.GetEngineFailover() {
		t.Error("保存 1 后引擎故障转移应为开启")
	}
	if err := Save(KeyEngineFailover, "0"); err != nil {
		t.Fatalf("Save engine_failover_enabled=0 失败: %s", err)
	}
	if cfg := Load(); cfg.GetEngineFailover() {
		t.Error("保存 0 后引擎故障转移应为关闭")
	}
}
