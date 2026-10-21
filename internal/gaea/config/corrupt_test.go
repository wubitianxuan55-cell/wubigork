package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolateUserConfigDir 把 os.UserConfigDir（Windows=APPDATA，Linux/macOS=
// XDG_CONFIG_HOME，兜底 HOME/USERPROFILE）重定向到临时目录，返回用户配置
// 目录（其下 gaea/config.toml 即 userConfigPath）。
func isolateUserConfigDir(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("APPDATA", home)         // Windows os.UserConfigDir
	t.Setenv("XDG_CONFIG_HOME", home) // Linux/macOS
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return filepath.Join(home, "gaea")
}

// findCorruptBackup 返回 dir 下第一个名字带 .corrupt- 前缀的文件路径；无则失败。
func findCorruptBackup(t *testing.T, dir string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "*.corrupt-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatalf("no .corrupt-* backup under %s", dir)
	}
	return matches[0]
}

// TestLoadUserConfigCorruptQuarantinedAndRecovers 用户级 config.toml 断电全零
// （2026-10-06 事故形态：NUL 填充、TOML 解析必败）时 Load 不再硬失败：按默认
// 配置启动成功，损坏内容逐字节隔离到 .corrupt-<ts> 后备副本。
func TestLoadUserConfigCorruptQuarantinedAndRecovers(t *testing.T) {
	dir := isolateUserConfigDir(t)
	zeroed := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), zeroed, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with zeroed user config: error = %v, want self-heal on defaults", err)
	}
	if cfg == nil || cfg.DefaultModel == "" {
		t.Fatalf("Load() returned unusable config: %+v", cfg)
	}
	backup := findCorruptBackup(t, dir)
	got, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(zeroed) {
		t.Errorf("quarantined backup content = %v, want the original zeroed bytes", got)
	}
	if !strings.HasPrefix(filepath.Base(backup), "config.toml.corrupt-") {
		t.Errorf("backup name = %q, want config.toml.corrupt-<ts>", filepath.Base(backup))
	}
}

// TestLoadUserConfigValidStillMerges 降级不吞正常配置：合法用户 config.toml
// 照常合并（防 mergeUserFile 把好文件也跳过的回归）。
func TestLoadUserConfigValidStillMerges(t *testing.T) {
	dir := isolateUserConfigDir(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	tomlBody := "language = \"zh\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(tomlBody), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Language != "zh" {
		t.Errorf("Language = %q, want zh (valid user config must still merge)", cfg.Language)
	}
}

// TestLoadProjectConfigCorruptStillFails 项目级 gaea.toml 损坏仍报错（开发期
// 可发现；降级只留给用户级——那里才有「断电后无法自救」的受害者）。
func TestLoadProjectConfigCorruptStillFails(t *testing.T) {
	isolateUserConfigDir(t)
	work := t.TempDir()
	t.Chdir(work)
	if err := os.WriteFile(filepath.Join(work, "gaea.toml"), []byte("language = "), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Error("Load() with corrupt project gaea.toml: want error, got nil")
	}
}
