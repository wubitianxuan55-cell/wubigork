package app

import (
	"os"
	"path/filepath"
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

// TestGaeaLoadConfigCorruptQuarantinedAndRecovers 持久化 config.toml 断电全零
// （2026-10-06 事故形态）时 gaeaLoadConfig 不再硬弹错：办公引擎按默认配置可
// 启动（bridge 注入保持），损坏内容逐字节隔离到 .corrupt-<ts> 后备副本。
func TestGaeaLoadConfigCorruptQuarantinedAndRecovers(t *testing.T) {
	// 指向临时用户配置目录，避免污染真实配置（同上例隔离手法）。
	oldAPPDATA := os.Getenv("APPDATA")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("APPDATA", t.TempDir())
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer func() {
		os.Setenv("APPDATA", oldAPPDATA)
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
	}()

	dir := gaeaConfig.UserConfigPath()
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	zeroed := []byte{0, 0, 0, 0, 0, 0}
	if err := os.WriteFile(dir, zeroed, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := gaeaLoadConfig()
	if err != nil {
		t.Fatalf("gaeaLoadConfig() with zeroed config: error = %v, want self-heal on defaults", err)
	}
	// 降级路径必须仍走完 bridge 注入（引擎才起得来）
	if cfg.DefaultModel != "gaea" {
		t.Errorf("DefaultModel = %q, want gaea (bridge injection must survive self-heal)", cfg.DefaultModel)
	}
	matches, _ := filepath.Glob(filepath.Dir(dir) + "/config.toml.corrupt-*")
	if len(matches) == 0 {
		t.Fatalf("no config.toml.corrupt-* backup next to %s", dir)
	}
	got, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(zeroed) {
		t.Errorf("quarantined backup content = %v, want the original zeroed bytes", got)
	}
}
