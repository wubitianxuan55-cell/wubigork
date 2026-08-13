package app

import (
	"os"
	"path/filepath"
	"testing"
)

// TestHerdsmanSecurityCheckExposedConfig 覆盖 HERDSMAN_CONFIG 指定暴露配置：
// 提取 lan_accessible=true 与 port，给出处置指引。
func TestHerdsmanSecurityCheckExposedConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	content := "api:\n    lan_accessible: true\n    port: 8443\n"
	if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
		t.Fatalf("写入样例配置失败: %v", err)
	}
	t.Setenv("HERDSMAN_CONFIG", cfg)

	got := (&App{}).HerdsmanSecurityCheck()
	if got.ConfigMissing {
		t.Errorf("ConfigMissing 应为 false，实际为 true")
	}
	if !got.Exposed {
		t.Errorf("Exposed 应为 true，实际为 false")
	}
	if got.Port != 8443 {
		t.Errorf("Port 应为 8443，实际为 %d", got.Port)
	}
	if got.Guidance == "" {
		t.Errorf("Guidance 应为非空处置指引，实际为空")
	}
}

// TestHerdsmanSecurityCheckMissingConfig 覆盖 HERDSMAN_CONFIG 指向不存在的文件。
func TestHerdsmanSecurityCheckMissingConfig(t *testing.T) {
	t.Setenv("HERDSMAN_CONFIG", filepath.Join(t.TempDir(), "no_such.yaml"))

	got := (&App{}).HerdsmanSecurityCheck()
	if got.Exposed {
		t.Errorf("配置缺失时 Exposed 应为 false，实际为 true")
	}
	if !got.ConfigMissing {
		t.Errorf("配置缺失时 ConfigMissing 应为 true，实际为 false")
	}
	if got.ParseError == "" {
		t.Errorf("配置缺失时 ParseError 应说明原因，实际为空")
	}
}

// TestHerdsmanSecurityCheckDefaultPath 覆盖默认路径回退：
// 未设置 HERDSMAN_CONFIG 时定位 %USERPROFILE%\.herdsman\config.yaml（不依赖真实 HOME）。
func TestHerdsmanSecurityCheckDefaultPath(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".herdsman", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatalf("创建 .herdsman 目录失败: %v", err)
	}
	if err := os.WriteFile(cfg, []byte("api:\n    lan_accessible: false\n"), 0o644); err != nil {
		t.Fatalf("写入默认路径配置失败: %v", err)
	}
	t.Setenv("HERDSMAN_CONFIG", "")
	t.Setenv("USERPROFILE", home)

	got := (&App{}).HerdsmanSecurityCheck()
	if got.ConfigPath != cfg {
		t.Errorf("ConfigPath 应为默认路径 %q，实际为 %q", cfg, got.ConfigPath)
	}
	if got.Exposed {
		t.Errorf("Exposed 应为 false，实际为 true")
	}
	if got.ConfigMissing {
		t.Errorf("ConfigMissing 应为 false，实际为 true")
	}
}
