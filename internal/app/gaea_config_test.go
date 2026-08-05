package app

import (
	"os"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// TestGaeaConfigPersistRoundTrip 验证办公引擎配置持久化往返：
// 默认配置 → 修改 Agent 参数 → Save → 重新加载 → 值保持。
func TestGaeaConfigPersistRoundTrip(t *testing.T) {
	// 指向临时用户配置目录，避免污染真实配置。
	// os.UserConfigDir() 在 Windows 读 APPDATA（不吃 XDG_CONFIG_HOME），
	// 两个都必须重定向；Linux/macOS 兜底 XDG_CONFIG_HOME。
	oldAPPDATA := os.Getenv("APPDATA")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("APPDATA", t.TempDir())
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer func() {
		os.Setenv("APPDATA", oldAPPDATA)
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}()

	cfg, err := gaeaLoadConfig()
	if err != nil {
		t.Fatalf("gaeaLoadConfig() 失败: %v", err)
	}
	// 修改设置
	cfg.Agent.Temperature = 0.42
	cfg.Agent.MaxSteps = 7
	cfg.Permissions.Mode = "auto"
	cfg.Sandbox.Network = false

	if err := gaeaConfig.Save(cfg); err != nil {
		t.Fatalf("Save() 失败: %v", err)
	}

	// 重新加载（模拟下次启动）
	got, err := gaeaLoadConfig()
	if err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}
	if got.Agent.Temperature != 0.42 {
		t.Errorf("Temperature = %v, want 0.42", got.Agent.Temperature)
	}
	if got.Agent.MaxSteps != 7 {
		t.Errorf("MaxSteps = %d, want 7", got.Agent.MaxSteps)
	}
	if got.Permissions.Mode != "auto" {
		t.Errorf("Permissions.Mode = %q, want auto", got.Permissions.Mode)
	}
	if got.Sandbox.Network {
		t.Errorf("Sandbox.Network = true, want false")
	}
	// bridge provider 注入必须保持
	if got.DefaultModel != "gaea" {
		t.Errorf("DefaultModel = %q, want gaea", got.DefaultModel)
	}
}
