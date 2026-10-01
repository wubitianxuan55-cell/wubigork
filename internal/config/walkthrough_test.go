package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaconfig "github.com/gaea/gaea/internal/gaea/config"
)

// TestLoadWalkthroughIsolatesUserData 走查第三数据面（v4.441.1）：沙箱壳内
// ①真实主目录的 .gaea_config.json 不可读（用户配置的 novels_dir 不得生效）；
// ②NovelsDir 强制落沙箱书架（硬编码默认 C:\AI\xiaoshuo 一并失效）；③Save 写
// 沙箱 home。非走查态行为不变（回归对照组）。
func TestLoadWalkthroughIsolatesUserData(t *testing.T) {
	fakeHome := t.TempDir()
	// 用户的真实配置：novels_dir 指向一个「绝不允许在走查态出现」的路径
	realConfig := `{"novels_dir": "C:\\real-user-shelf"}`
	if err := os.WriteFile(filepath.Join(fakeHome, ".gaea_config.json"), []byte(realConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("USERPROFILE", fakeHome)
	t.Setenv("HOME", fakeHome)
	t.Setenv("GAEA_WALKTHROUGH", "1")

	cfg := Load()

	wantNovels := filepath.Join(gaeaconfig.WalkthroughRootDir(), "novels")
	if cfg.NovelsDir != wantNovels {
		t.Fatalf("走查态书架必须强制指向沙箱: got %q want %q", cfg.NovelsDir, wantNovels)
	}
	if cfg.NovelsDir == `C:\real-user-shelf` || strings.Contains(cfg.NovelsDir, "xiaoshuo") {
		t.Fatalf("走查态不得解析出真实书架: %q", cfg.NovelsDir)
	}
	if _, err := os.Stat(wantNovels); err != nil {
		t.Fatalf("沙箱书架目录应已创建: %v", err)
	}

	// Save 落沙箱 home：写一个配置项，真实主目录文件必须原样未动
	if err := Save(KeyHTTPTimeoutSeconds, "99"); err != nil {
		t.Fatalf("Save(走查态): %v", err)
	}
	saved, err := os.ReadFile(filepath.Join(fakeHome, ".gaea_config.json"))
	if err != nil {
		t.Fatalf("读回真实配置: %v", err)
	}
	if string(saved) != realConfig {
		t.Fatalf("真实主目录配置被走查态污染:\nreal: %s", string(saved))
	}
	sandboxSaved, err := os.ReadFile(filepath.Join(gaeaconfig.WalkthroughRootDir(), "home", ".gaea_config.json"))
	if err != nil {
		t.Fatalf("沙箱 home 应有写入: %v", err)
	}
	if !strings.Contains(string(sandboxSaved), `"http_timeout_seconds": 99`) &&
		!strings.Contains(string(sandboxSaved), `"http_timeout_seconds":99`) {
		t.Fatalf("沙箱配置未写入目标键: %s", string(sandboxSaved))
	}
}

// TestLoadNonWalkthroughUnchanged 非走查态对照：用户配置 novels_dir 照常生效。
func TestLoadNonWalkthroughUnchanged(t *testing.T) {
	fakeHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(fakeHome, ".gaea_config.json"),
		[]byte(`{"novels_dir": "D:\\my-shelf"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("USERPROFILE", fakeHome)
	t.Setenv("HOME", fakeHome)
	t.Setenv("GAEA_WALKTHROUGH", "0")

	cfg := Load()
	if cfg.NovelsDir != `D:\my-shelf` {
		t.Fatalf("非走查态用户书架配置必须照常生效: %q", cfg.NovelsDir)
	}
}
